package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A failed upgrade or copy writes the daemon-state mark the session-start hook reads,
// so that a user who runs no command still learns why nothing is captured (D-256, the
// databases are copied before the state directory is migrated, and a program that
// cannot migrate it stops).

func markLine(t *testing.T, home string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, "daemon-state"))
	if err != nil {
		t.Fatalf("no mark was written: %v", err)
	}
	f := strings.Split(strings.TrimRight(string(raw), "\n"), "\t")
	if len(f) != 3 {
		t.Fatalf("the mark is not state, time and detail: %q", raw)
	}
	return f
}

func TestTheMarkNamesBothCausesAndQuotesNothing(t *testing.T) {
	home := useHome(t)
	secret := `room "kept" \ turn: attempt to write a readonly database`
	for _, tc := range []struct {
		err  error
		want string
	}{
		{errors.New(secret), "upgrade of its data stopped partway"},
		{&backupError{}, "copy of its databases"},
	} {
		recordUpgradeFailed(tc.err)
		f := markLine(t, home)
		if f[0] != "upgrade-failed" {
			t.Errorf("state is %q", f[0])
		}
		if !strings.Contains(f[2], tc.want) {
			t.Errorf("detail %q does not say %q", f[2], tc.want)
		}
		if strings.ContainsAny(f[2], "\"\\\n") || strings.Contains(f[2], "kept") || strings.Contains(f[2], home) {
			t.Errorf("the detail quotes something it must not: %q", f[2])
		}
	}
}

func TestAnUpgradeThatSucceedsClearsItsOwnMarkAndNoOther(t *testing.T) {
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	home := useHome(t)
	oldSignerRoom(t, "room-a", "kept")
	recordUpgradeFailed(errors.New("x"))
	guardStateDirectory("status")
	if _, err := os.Stat(filepath.Join(home, "daemon-state")); err == nil {
		t.Error("a mark of a failed upgrade outlived an upgrade that succeeded")
	}
	recordDaemonState("blocked", "peer sync cannot be served")
	guardStateDirectory("status")
	if f := markLine(t, home); f[0] != "blocked" {
		t.Errorf("a failed bind's mark was cleared by an upgrade check: %v", f)
	}
}

// The daemon that cannot migrate writes the mark, and a hook run afterward in the same
// home says so to the model, correctly for this cause.
func TestAFailedUpgradeReachesTheSessionStartNote(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	bin := buildVersion(t, t.TempDir(), "0.0.1-failed-mark")
	home := useHome(t)
	oldSignerRoom(t, "room-a", "kept")
	readOnlyRoom(t, "room-b")
	env := append(os.Environ(), "COGMER_HOME="+home, "COGMER_ADDR="+freeAddr(t), "COGMER_PEER_ADDR="+freeAddr(t),
		"COGMER_TAILCAT=off", "COGMER_HOST=claude", "CLAUDE_CODE_SESSION_ID=s-1")
	daemon := exec.Command(bin, "daemon")
	daemon.Env = env
	done := make(chan struct{})
	go func() { _, _ = daemon.CombinedOutput(); close(done) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		_ = daemon.Process.Kill()
		t.Fatal("the daemon started on data it could not migrate")
	}
	f := markLine(t, home)
	if f[0] != "upgrade-failed" {
		t.Fatalf("the failed upgrade wrote %v", f)
	}

	note := runSessionStart(t, home, "127.0.0.1:1")
	for _, want := range []string{"daemon could not start", "upgrade of its data stopped partway", "until that is fixed"} {
		if !strings.Contains(note, want) {
			t.Errorf("the note lacks %q:\n%s", want, note)
		}
	}
	if strings.Contains(note, "address is free") {
		t.Errorf("the note blames an address for an upgrade failure:\n%s", note)
	}

	// A daemon that answers means the fault is gone, whatever the mark says.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	if note := runSessionStart(t, home, strings.TrimPrefix(srv.URL, "http://")); strings.Contains(note, "could not start") {
		t.Errorf("a stale mark repeated a fault that a running daemon disproves:\n%s", note)
	}
}

func runSessionStart(t *testing.T, home, addr string) string {
	t.Helper()
	root, _ := cliHarness(t)
	src, err := os.ReadFile("../../plugin/hooks-handlers/session-start.sh")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "hooks-handlers", "session-start.sh")
	if err := os.WriteFile(path, src, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", path)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root,
		"COGMER_BIN=/missing", "COGMER_ADDR=" + addr}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		H struct {
			C string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("the hook did not emit one JSON value: %v\n%s", err, out)
	}
	return v.H.C
}
