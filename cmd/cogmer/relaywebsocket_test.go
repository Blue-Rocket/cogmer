package main

import (
	"os"
	"regexp"
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
