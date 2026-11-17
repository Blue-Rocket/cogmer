package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// D-217: a cloud session carries its relay connection as a WebSocket, and nothing
// else does.
func TestACloudSessionCarriesTheRelayAsAWebSocket(t *testing.T) {
	cases := []struct {
		name   string
		env    map[string]string
		wantOn bool
	}{
		{"a cloud session", map[string]string{"CLAUDE_CODE_REMOTE": "true"}, true},
		{"outside a cloud session", map[string]string{}, false},
		{"a value other than true", map[string]string{"CLAUDE_CODE_REMOTE": "false"}, false},
		{"a cloud session where the user chose", map[string]string{"CLAUDE_CODE_REMOTE": "true", relayWebSocketVar: "0"}, false},
	}
	for _, c := range cases {
		env := map[string]string{}
		for k, v := range c.env {
			env[k] = v
		}
		set := relayAsWebSocketInCloud(
			func(k string) string { return env[k] },
			func(k, v string) error { env[k] = v; return nil },
		)
		if set != c.wantOn {
			t.Errorf("%s: set %v, want %v", c.name, set, c.wantOn)
		}
		if c.wantOn && env[relayWebSocketVar] != "1" {
			t.Errorf("%s: %s is %q, want 1", c.name, relayWebSocketVar, env[relayWebSocketVar])
		}
		if !c.wantOn && env[relayWebSocketVar] != c.env[relayWebSocketVar] {
			t.Errorf("%s: %s changed to %q", c.name, relayWebSocketVar, env[relayWebSocketVar])
		}
	}
}

// The variable does nothing in a binary built without the tag, so a release without
// it would leave a cloud daemon unable to reach the relay, with no error to say so.
func TestTheReleaseBuildCompilesInTheWebSocketRelayClient(t *testing.T) {
	src, err := os.ReadFile("../../scripts/release.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`(?m)^tags="[^"]*\bts_debug_websockets\b[^"]*"$`).Match(src) {
		t.Error(`scripts/release.sh does not set tags="ts_debug_websockets"`)
	}
	if !regexp.MustCompile(`go build -tags "\$tags"`).Match(src) {
		t.Error(`scripts/release.sh does not pass -tags "$tags" to go build`)
	}
}

// The tag and the variable are Tailscale debug settings with no stability promise, and
// an upgrade of the module that renamed or removed either would still build, leaving a
// cloud daemon on the relay's own protocol, which the session's proxy breaks (D-217).
// This reads the Tailscale version in go.mod, so the commit that bumps it fails here.
func TestTheTailscaleVersionInUseStillSwitchesTheRelayToAWebSocket(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go command to list the package with")
	}
	const pkg = "tailscale.com/derp/derphttp"
	list := func(tags string) (string, []string) {
		t.Helper()
		args := []string{"list", "-f", "{{.Dir}}\n{{range .GoFiles}}{{.}} {{end}}"}
		if tags != "" {
			args = append(args, "-tags", tags)
		}
		cmd := exec.Command("go", append(args, pkg)...)
		// The client is compiled only for Linux and macOS, so the list is made for Linux
		// whatever this machine runs.
		cmd.Env = append(append(os.Environ(), realGoCaches...), "GOOS=linux", "GOARCH=amd64")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("go list %s: %v", pkg, err)
		}
		dir, files, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
		return dir, strings.Fields(files)
	}

	dir, tagged := list("ts_debug_websockets")
	_, plain := list("")
	inPlain := map[string]bool{}
	for _, f := range plain {
		inPlain[f] = true
	}
	var added []string
	for _, f := range tagged {
		if !inPlain[f] {
			added = append(added, f)
		}
	}
	if len(added) == 0 {
		t.Fatalf("the ts_debug_websockets tag adds no file to %s in this Tailscale version, so a release compiles in no WebSocket relay client; find the tag that does and change scripts/release.sh and D-217", pkg)
	}

	// The files the tag adds decide nothing by themselves: the client dials a WebSocket
	// only when the variable cogmer sets is read somewhere in the package.
	read := false
	for _, f := range tagged {
		src, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), `"`+relayWebSocketVar+`"`) {
			read = true
		}
	}
	if !read {
		t.Errorf("%s no longer reads %s in this Tailscale version, so a cloud daemon would speak the relay's own protocol, which the session's proxy breaks; find what now switches the client to a WebSocket and change relayWebSocketVar and D-217", pkg, relayWebSocketVar)
	}
}
