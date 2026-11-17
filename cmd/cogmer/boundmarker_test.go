package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A session in no room reaches nothing of cogmer's but the commands its user types
// (D-284). These tests run plugin/hooks-handlers/run.sh against bindings made through
// Membership, so the file a binding writes and the file the script looks for cannot
// drift apart without one of them failing.

// hookHarness is a plugin root holding run.sh and common.sh, and a state directory
// whose program only records that it ran and what it was given.
func hookHarness(t *testing.T) (root, home string) {
	t.Helper()
	root, home = t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "hooks-handlers"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"hooks-handlers/run.sh", "hooks-handlers/common.sh"} {
		src, err := os.ReadFile(filepath.Join("../../plugin", f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, f), src, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	stub := "#!/usr/bin/env bash\ncat > \"$COGMER_HOME/ran\"\n"
	if err := os.WriteFile(filepath.Join(home, "bin", "cogmer"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COGMER_HOME", home)
	return root, home
}

// hookEnv is added to the script's environment by a test that needs more than the
// harness gives it.
var hookEnv []string

// runHookScript runs the script as a host would, and reports whether the program ran
// and, if it did, what it read.
func runHookScript(t *testing.T, root, home, payload string, args ...string) (ran bool, got string) {
	t.Helper()
	_ = os.Remove(filepath.Join(home, "ran"))
	cmd := exec.Command("bash", append([]string{filepath.Join(root, "hooks-handlers", "run.sh")}, args...)...)
	cmd.Env = append([]string{"PATH=/usr/bin:/bin", "HOME=" + home, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root}, hookEnv...)
	cmd.Stdin = strings.NewReader(payload)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("run.sh %v failed: %v", args, err)
	}
	if len(out) != 0 {
		t.Fatalf("run.sh %v wrote %q to stdout, which the host would inject", args, out)
	}
	b, err := os.ReadFile(filepath.Join(home, "ran"))
	if err != nil {
		return false, ""
	}
	return true, string(b)
}

func openHookMembership(t *testing.T) (*Membership, Room) {
	t.Helper()
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	room, err := m.CreateRoom(testIdentity(t).PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	return m, room
}

func TestAHookForASessionInNoRoomNeverStartsTheProgram(t *testing.T) {
	root, home := hookHarness(t)
	openHookMembership(t)

	for _, args := range [][]string{{"hook", "prompt"}, {"hook", "stop"}} {
		if ran, _ := runHookScript(t, root, home, `{"session_id":"sess-1","prompt":"hello"}`, args...); ran {
			t.Errorf("%v for a session in no room started the program", args)
		}
	}
	if ran, _ := runHookScript(t, root, home, `{"session_id":"thread-1","turn_id":"t","prompt":"hello"}`,
		"hook", "--host", "codex", "prompt"); ran {
		t.Error("a Codex prompt from a thread in no room started the program")
	}
}

func TestAHookForABoundSessionHandsTheProgramItsPayloadUnchanged(t *testing.T) {
	root, home := hookHarness(t)
	m, room := openHookMembership(t)
	if err := m.BindSession("sess-1", room.RoomID); err != nil {
		t.Fatal(err)
	}
	payload := `{"session_id":"sess-1","prompt":"hello"}`
	for _, kind := range []string{"prompt", "stop"} {
		ran, got := runHookScript(t, root, home, payload, "hook", kind)
		if !ran {
			t.Errorf("hook %s for a session in a room did not start the program", kind)
		}
		if got != payload {
			t.Errorf("hook %s handed the program %q, want %q", kind, got, payload)
		}
	}

	// A Codex thread is stored under the host's own form of its identifier, and the
	// script finds it from the bare thread its hook carries.
	if err := m.BindSession(codexHost{}.SessionID("thread-1"), room.RoomID); err != nil {
		t.Fatal(err)
	}
	if ran, _ := runHookScript(t, root, home, `{"session_id":"thread-1","turn_id":"t"}`,
		"hook", "--host", "codex", "stop"); !ran {
		t.Error("a Codex stop from a thread in a room did not start the program")
	}
}

func TestACommandFromASessionInNoRoomReachesTheProgram(t *testing.T) {
	root, home := hookHarness(t)
	openHookMembership(t)
	if ran, _ := runHookScript(t, root, home, `{"session_id":"sess-1","prompt":"/cogmer:room-join amber-fox"}`,
		"hook", "prompt"); !ran {
		t.Error("a typed command from a session in no room never reached the program, so the reply to it would be captured once it binds the session (D-209)")
	}
	if ran, _ := runHookScript(t, root, home, `{"session_id":"thread-1","turn_id":"t","prompt":"$cogmer room-join amber-fox"}`,
		"hook", "--host", "codex", "prompt"); !ran {
		t.Error("a typed Codex command from a thread in no room never reached the program")
	}
}

func TestEveryDoubtAboutASessionCallsTheProgram(t *testing.T) {
	root, home := hookHarness(t)
	cases := map[string]struct {
		payload string
		args    []string
	}{
		"no session identifier":      {`{"prompt":"hello"}`, []string{"hook", "prompt"}},
		"an identifier with a dot":   {`{"session_id":"../membership.db","prompt":"hello"}`, []string{"hook", "prompt"}},
		"an escaped identifier":      {`{"session_id":"a\"b","prompt":"hello"}`, []string{"hook", "prompt"}},
		"a host it does not know":    {`{"session_id":"sess-1"}`, []string{"hook", "--host", "elsewhere", "prompt"}},
		"arguments that are no hook": {`{"session_id":"sess-1"}`, []string{"version"}},
	}

	// With no directory of bound sessions, nothing can be known, so every hook calls.
	if ran, _ := runHookScript(t, root, home, `{"session_id":"sess-1","prompt":"hello"}`, "hook", "prompt"); !ran {
		t.Error("with no directory of bound sessions, a hook did not call the program")
	}

	openHookMembership(t)
	for name, c := range cases {
		if ran, _ := runHookScript(t, root, home, c.payload, c.args...); !ran {
			t.Errorf("%s: the hook ended without calling the program", name)
		}
	}
}

func TestLeavingARoomEndsTheHooksCalls(t *testing.T) {
	root, home := hookHarness(t)
	m, room := openHookMembership(t)
	if err := m.BindSession("sess-1", room.RoomID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.LeaveSession("sess-1"); err != nil {
		t.Fatal(err)
	}
	if ran, _ := runHookScript(t, root, home, `{"session_id":"sess-1","prompt":"hello"}`, "hook", "prompt"); ran {
		t.Error("a session that left its room still started the program")
	}
	// Rejoining is a binding like any other.
	if err := m.BindSession("sess-1", room.RoomID); err != nil {
		t.Fatal(err)
	}
	if ran, _ := runHookScript(t, root, home, `{"session_id":"sess-1","prompt":"hello"}`, "hook", "prompt"); !ran {
		t.Error("a session that rejoined its room did not start the program")
	}
}

// A store written before these files existed has bound sessions and no directory.
// Opening it must give every one of them a file before the directory appears.
func TestAStoreFromBeforeTheFilesGetsOneForEveryBoundSession(t *testing.T) {
	root, home := hookHarness(t)
	m, room := openHookMembership(t)
	for _, s := range []string{"sess-1", codexHost{}.SessionID("thread-1")} {
		if err := m.BindSession(s, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	m.Close()
	if err := os.RemoveAll(filepath.Join(home, boundDirName)); err != nil {
		t.Fatal(err)
	}

	m2, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	defer m2.Close()
	if ran, _ := runHookScript(t, root, home, `{"session_id":"sess-1"}`, "hook", "stop"); !ran {
		t.Error("a session bound before the files existed lost its calls when the store was opened")
	}
	if ran, _ := runHookScript(t, root, home, `{"session_id":"thread-1","turn_id":"t"}`, "hook", "--host", "codex", "stop"); !ran {
		t.Error("a Codex thread bound before the files existed lost its calls when the store was opened")
	}
}

// A cloud session in no room is ended in the script like any other. Its daemon takes a
// cloud value uploaded after the session began when a command asks for the identity
// (D-279, a cloud session's daemon reads its cloud value from the plugin files it
// finds), not at a prompt.
func TestACloudSessionInNoRoomNeverStartsTheProgram(t *testing.T) {
	root, home := hookHarness(t)
	openHookMembership(t)
	hookEnv = []string{"CLAUDE_CODE_REMOTE=true"}
	t.Cleanup(func() { hookEnv = nil })
	if ran, _ := runHookScript(t, root, home, `{"session_id":"sess-1","prompt":"hello"}`, "hook", "prompt"); ran {
		t.Error("a prompt from a cloud session in no room started the program")
	}
}
