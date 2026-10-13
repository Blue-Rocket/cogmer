package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A room can be created or joined show-only: its conversation appears in the room
// view, and none of it is given to this machine's Claude sessions. The choice is the
// person's own, made when the room is first recorded here, and it never changes for
// that room (D-200).

func TestAShowOnlyRoomIsRecordedAsSuchAndOthersAreNot(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	normal, err := m.CreateRoom(id.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	quiet, err := m.CreateRoom(id.PeerID, true)
	if err != nil {
		t.Fatal(err)
	}
	if normal.ShowOnly || !quiet.ShowOnly {
		t.Fatalf("created rooms read back as %v and %v", normal.ShowOnly, quiet.ShowOnly)
	}
	if got, ok := m.RoomByID(quiet.RoomID); !ok || !got.ShowOnly {
		t.Errorf("RoomByID lost the setting: %+v", got)
	}
	if got, err := m.FindRoom(quiet.RoomName); err != nil || !got.ShowOnly {
		t.Errorf("FindRoom lost the setting: %+v %v", got, err)
	}
	rooms, _ := m.Rooms()
	seen := map[string]bool{}
	for _, r := range rooms {
		seen[r.RoomID] = r.ShowOnly
	}
	if seen[normal.RoomID] || !seen[quiet.RoomID] {
		t.Errorf("Rooms lost the setting: %+v", seen)
	}
	// A Room built without saying injects, as every room did before the setting.
	if (Room{}).ShowOnly {
		t.Error("the zero Room is show-only, so any code that builds one would silently stop injecting")
	}
}

func TestTheSettingCannotChangeForARoom(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	room, _ := m.CreateRoom(id.PeerID, false)

	// The store itself refuses, so no code path can change it by mistake.
	if _, err := m.db.Exec(`UPDATE rooms SET show_only = 1 WHERE room_id = ?`, room.RoomID); err == nil {
		t.Fatal("a room's setting was changed by an UPDATE")
	} else if !strings.Contains(err.Error(), "fixed when the room is created or joined") {
		t.Errorf("the refusal does not say why: %v", err)
	}
	// Writing the same value is not a change.
	if _, err := m.db.Exec(`UPDATE rooms SET show_only = 0, state = 'joined' WHERE room_id = ?`, room.RoomID); err != nil {
		t.Errorf("an update that leaves the setting as it was was refused: %v", err)
	}

	// Recording the room again with the other setting leaves it as it was.
	if err := m.RecordRoom(room.RoomID, room.RoomName, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := m.RoomByID(room.RoomID); got.ShowOnly {
		t.Error("joining again with --show-only changed a room that was not")
	}

	// And a request for the other setting is refused in words, both ways round.
	var fixed *ShowOnlyFixedError
	if err := m.CheckShowOnly(room.RoomID, true); !errors.As(err, &fixed) || !strings.Contains(err.Error(), "without --show-only") {
		t.Errorf("asking a normal room to be show-only was not refused: %v", err)
	}
	if err := m.CheckShowOnly(room.RoomID, false); err != nil {
		t.Errorf("asking for the setting a room has was refused: %v", err)
	}
	quiet, _ := m.CreateRoom(id.PeerID, true)
	if err := m.CheckShowOnly(quiet.RoomID, false); !errors.As(err, &fixed) || !strings.Contains(err.Error(), "show-only on this machine") {
		t.Errorf("asking a show-only room to inject was not refused: %v", err)
	}
	if err := m.CheckShowOnly("a-room-not-recorded-here", true); err != nil {
		t.Errorf("a room not yet recorded has no setting to change: %v", err)
	}
}

// Every store that already exists is migrated: a membership database written before
// the setting has rooms with no such column, and each must read back as injecting.
func TestAMembershipDatabaseFromBeforeTheSettingKeepsInjecting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".cogmer"), 0o700); err != nil {
		t.Fatal(err)
	}
	old, err := sql.Open("sqlite", filepath.Join(home, ".cogmer", "membership.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(`CREATE TABLE rooms (
		room_id TEXT PRIMARY KEY, room_name TEXT NOT NULL, state TEXT NOT NULL,
		issued_sequence INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL);
		INSERT INTO rooms (room_id, room_name, state, created_at) VALUES ('r1', 'old-room', 'joined', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	old.Close()

	m, err := OpenMembership()
	if err != nil {
		t.Fatalf("an older membership database could not be opened: %v", err)
	}
	defer m.Close()
	room, ok := m.RoomByID("r1")
	if !ok || room.ShowOnly {
		t.Fatalf("an older room reads back as %+v, want one that injects", room)
	}
	if _, err := m.db.Exec(`UPDATE rooms SET show_only = 1 WHERE room_id = 'r1'`); err == nil {
		t.Error("the migrated database does not fix the setting")
	}
}

// The prompt of a session in a show-only room is captured, since the room's members
// read it in the view, and nothing is offered to the session. The same colleague
// turn is offered to a session in a room that is not show-only.
func TestAShowOnlyRoomGivesNothingToTheSession(t *testing.T) {
	d, normal := testDaemon(t)
	quiet, err := d.members.CreateRoom(d.id.PeerID, true)
	if err != nil {
		t.Fatal(err)
	}
	alice := testIdentity(t)
	if err := d.members.Allow(alice.PeerID, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := d.members.MarkVerified(alice.PeerID); err != nil {
		t.Fatal(err)
	}
	for _, r := range []Room{normal, quiet} {
		store, err := d.storeFor(r.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Append(1, alice, r.RoomID, "alice-session", EventUserPrompt, "WHAT ALICE SAID", nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.members.BindSession("normal-session", normal.RoomID); err != nil {
		t.Fatal(err)
	}
	if err := d.members.BindSession("quiet-session", quiet.RoomID); err != nil {
		t.Fatal(err)
	}

	prompt := func(session string) string {
		body, _ := json.Marshal(promptReq{SessionID: session, Prompt: "MY OWN PROMPT", CWD: "/tmp", PromptID: "p-" + session})
		req := httptest.NewRequest(http.MethodPost, "/hook/prompt", bytes.NewReader(body))
		req.Header.Set(localGuardHeader, localGuardValue)
		rec := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("the prompt hook answered %d: %s", rec.Code, rec.Body.String())
		}
		var out struct{ Context string }
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out.Context
	}

	if got := prompt("normal-session"); !strings.Contains(got, "WHAT ALICE SAID") {
		t.Fatalf("a room that is not show-only gave its session nothing: %q", got)
	}
	if got := prompt("quiet-session"); got != "" {
		t.Errorf("a show-only room gave its session context: %q", got)
	}

	store, _ := d.storeFor(quiet.RoomID)
	evs, _ := store.ListRoom(quiet.RoomID)
	captured := false
	for _, e := range evs {
		if e.EventType == EventUserPrompt && e.Content == "MY OWN PROMPT" {
			captured = true
		}
	}
	if !captured {
		t.Error("the prompt of a session in a show-only room was not captured, so the room's members cannot read it")
	}
	// Nothing was marked offered, so the room's turns are still there for the view.
	if left, _ := store.UndeliveredFor(quiet.RoomID, "quiet-session"); len(left) != 1 {
		t.Errorf("%d colleague turns are undelivered in the show-only room, want the one that was never offered", len(left))
	}
}

// The command line: the flag is read, a room is listed as show-only, and asking for
// the other setting for a room that has one is refused with a non-zero exit. A real
// build, because the refusal is an exit.
func TestTheCommandsCreateAndJoinShowOnlyAndRefuseToChangeIt(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	dir := t.TempDir()
	bin := buildVersion(t, dir, "0.0.1")
	home := filepath.Join(dir, "home")
	run := func(session string, args ...string) (string, int) {
		cmd := exec.Command(bin, args...)
		cmd.Env = append(os.Environ(), "COGMER_HOME="+home, "COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off",
			"COGMER_ADDR=127.0.0.1:1", "CLAUDE_CODE_SESSION_ID="+session)
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return string(out), code
	}
	field := func(out, key string) string {
		for _, l := range strings.Split(out, "\n") {
			if strings.HasPrefix(l, key) {
				return strings.TrimSpace(strings.TrimPrefix(l, key))
			}
		}
		return ""
	}

	out, code := run("s-quiet", "create", "--show-only")
	if code != 0 || !strings.Contains(flat(out), "show-only on this machine") || !strings.Contains(flat(out), "none of it is given to your agent") {
		t.Fatalf("creating show-only: exit %d\n%s", code, out)
	}
	quietName := strings.TrimSpace(strings.Fields(field(out, "created"))[0])

	out, code = run("s-normal", "create")
	if code != 0 || strings.Contains(out, "show-only") {
		t.Fatalf("creating a normal room: exit %d\n%s", code, out)
	}
	normalName := strings.Fields(field(out, "created"))[0]

	rooms, _ := run("s-quiet", "rooms")
	for _, l := range strings.Split(rooms, "\n") {
		if strings.Contains(l, quietName) && !strings.Contains(l, "show-only") {
			t.Errorf("the show-only room is not listed as such: %q", l)
		}
		if strings.Contains(l, normalName) && strings.Contains(l, "show-only") {
			t.Errorf("a normal room is listed as show-only: %q", l)
		}
	}

	out, code = run("s-other", "join", normalName, "--show-only")
	if code == 0 || !strings.Contains(flat(out), "cannot change") {
		t.Errorf("joining a normal room show-only was not refused: exit %d\n%s", code, out)
	}
	out, code = run("s-other", "join", quietName)
	if code != 0 || !strings.Contains(flat(out), "show-only on this machine") {
		t.Errorf("joining a show-only room without the flag should keep it show-only: exit %d\n%s", code, out)
	}
	if out, code := run("s-bad", "create", "--nonsense"); code == 0 {
		t.Errorf("create accepted an argument it does not know:\n%s", out)
	}
}
