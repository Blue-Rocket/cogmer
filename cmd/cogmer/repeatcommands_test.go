package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A session that left and leaves again is told it left, naming the room, and one that
// never had a room is told it has none.
func TestLeavingTwiceSaysTheSessionAlreadyLeft(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	room, err := m.CreateRoom(self.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.BindSession("s1", room.RoomID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.LeaveSession("s1"); err != nil {
		t.Fatal(err)
	}
	_, err = m.LeaveSession("s1")
	if err == nil || !strings.Contains(err.Error(), "already left "+room.RoomName) {
		t.Fatalf("second leave said %v, want that the session already left %s", err, room.RoomName)
	}
	_, err = m.LeaveSession("never")
	if err == nil || !strings.Contains(err.Error(), "not in a room") || strings.Contains(err.Error(), "already left") {
		t.Fatalf("a session that was never in a room was told %v", err)
	}
}

// create names the room a session already has and makes nothing, until the user gives
// --another; a session with no room is not asked.
func TestCreateFromASessionWithARoomIsRefusedUnlessAnotherIsGiven(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	dir := t.TempDir()
	bin := buildVersion(t, dir, testBuildVersion)
	home := filepath.Join(dir, "home")
	run := func(sid string, args ...string) (string, int) {
		cmd := exec.Command(bin, args...)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "COGMER_HOME=" + home,
			"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_IDENTITY_PAGE=off", "COGMER_ADDR=127.0.0.1:1",
			"COGMER_HOST=claude", "CLAUDE_CODE_SESSION_ID=" + sid}
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return string(out), code
	}
	roomLines := func() int {
		out, _ := run("observer", "rooms")
		return strings.Count(out, "\n")
	}
	first, code := run("one", "create")
	if code != 0 || !strings.Contains(first, "created") {
		t.Fatalf("first create: exit %d\n%s", code, first)
	}
	name := strings.Fields(strings.TrimPrefix(first, "created "))[0]
	before := roomLines()

	out, code := run("one", "create")
	if code == 0 || !strings.Contains(out, "already in "+name) || !strings.Contains(out, "Nothing was created") || !strings.Contains(out, "--another") {
		t.Fatalf("a repeated create was not refused with the room named: exit %d\n%s", code, out)
	}
	if got := roomLines(); got != before {
		t.Fatalf("the refused create changed the room list from %d to %d lines", before, got)
	}

	if out, code := run("two", "create"); code != 0 || !strings.Contains(out, "created") {
		t.Fatalf("a session with no room was refused: exit %d\n%s", code, out)
	}

	if out, _ := run("one", "leave"); !strings.Contains(out, "has left "+name) {
		t.Fatalf("leave: %s", out)
	}
	if out, _ := run("one", "leave"); !strings.Contains(out, "already left "+name) {
		t.Fatalf("a second leave said: %s", out)
	}
	if out, code := run("one", "create"); code == 0 || !strings.Contains(out, "left "+name) || !strings.Contains(out, "Nothing was created") {
		t.Fatalf("a session that left was not refused: exit %d\n%s", code, out)
	}

	// On purpose, it makes the room, though this session cannot enter it.
	before = roomLines()
	out, _ = run("one", "create", "--another")
	if !strings.Contains(out, "created") {
		t.Fatalf("--another made no room:\n%s", out)
	}
	if got := roomLines(); got <= before {
		t.Fatalf("--another left the room list at %d lines, was %d", got, before)
	}
}
