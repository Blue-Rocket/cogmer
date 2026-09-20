package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// testDaemon builds a daemon the way the real one is now built: no room of its
// own, a membership store, and rooms opened on demand.
func testDaemon(t *testing.T) (*Daemon, Room) {
	t.Helper()
	m := testMembership(t)
	id := testIdentity(t)
	room, err := m.CreateRoom(id.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	d := &Daemon{id: id, members: m}
	t.Cleanup(d.closeStores)
	return d, room
}

func testMembership(t *testing.T) *Membership {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	return m
}

// Admission is the decision authentication cannot make. A stranger with a valid
// key authenticates correctly and must still be refused (D-044).
func TestGuestListDecidesAdmission(t *testing.T) {
	m := testMembership(t)
	host, guest, stranger := testIdentity(t), testIdentity(t), testIdentity(t)

	room, err := m.CreateRoom(host.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	if !m.IsGuest(room.RoomID, host.PeerID) {
		t.Error("a room's creator is not its first guest")
	}
	if m.IsGuest(room.RoomID, stranger.PeerID) {
		t.Error("an uninvited peer is a guest")
	}
	if err := m.Invite(room.RoomID, guest.PeerID); err != nil {
		t.Fatal(err)
	}
	if !m.IsGuest(room.RoomID, guest.PeerID) {
		t.Error("an invited peer is not a guest")
	}
	if m.IsGuest(room.RoomID, stranger.PeerID) {
		t.Error("inviting one peer admitted another")
	}
}

// An identifier that names no key cannot be admitted: nothing could ever prove
// possession of it, so recording it would be recording a hope.
func TestOnlyKeysCanBeAdmitted(t *testing.T) {
	m := testMembership(t)
	room, _ := m.CreateRoom(testIdentity(t).PeerID)
	for _, bad := range []string{"peer-852ffe6408339bcada91", "alice", "ed25519:notbase64!!"} {
		if err := m.Invite(room.RoomID, bad); err == nil {
			t.Errorf("admitted %q, which names no key", bad)
		}
		if err := m.Allow(bad, "x"); err == nil {
			t.Errorf("recorded %q as a known peer", bad)
		}
	}
}

// Revoking withdraws admission to one room; forgetting discards the identity so a
// later meeting is a first meeting (§12). They are not the same act.
func TestRevokeAndForgetDiffer(t *testing.T) {
	m := testMembership(t)
	host, guest := testIdentity(t), testIdentity(t)
	a, _ := m.CreateRoom(host.PeerID)
	b, _ := m.CreateRoom(host.PeerID)
	_ = m.Allow(guest.PeerID, "alice")
	_ = m.Invite(a.RoomID, guest.PeerID)
	_ = m.Invite(b.RoomID, guest.PeerID)

	if err := m.Revoke(a.RoomID, guest.PeerID); err != nil {
		t.Fatal(err)
	}
	if m.IsGuest(a.RoomID, guest.PeerID) {
		t.Error("revoke did not withdraw admission")
	}
	if !m.IsGuest(b.RoomID, guest.PeerID) {
		t.Error("revoking one room withdrew admission to another")
	}
	known, _ := m.KnownPeers()
	if len(known) != 1 {
		t.Error("revoking a room forgot the peer entirely")
	}

	if err := m.Forget(guest.PeerID); err != nil {
		t.Fatal(err)
	}
	if known, _ = m.KnownPeers(); len(known) != 0 {
		t.Error("forget did not discard the identity")
	}
}

// Room names are generated and need be unique only among this peer's rooms (§12).
func TestRoomNamesAreGeneratedAndLocallyUnique(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		r, err := m.CreateRoom(self.PeerID)
		if err != nil {
			t.Fatalf("could not create room %d: %v", i, err)
		}
		if seen[r.RoomName] {
			t.Fatalf("two rooms on one machine share the name %q", r.RoomName)
		}
		seen[r.RoomName] = true
		if r.RoomID == r.RoomName {
			t.Error("identity and name are the same value")
		}
	}
}

// A person typing a room should not have to know whether they hold its name or
// its identity (§3.2).
func TestRoomResolvesByEitherIdentifier(t *testing.T) {
	m := testMembership(t)
	r, _ := m.CreateRoom(testIdentity(t).PeerID)
	for _, key := range []string{r.RoomName, r.RoomID} {
		got, err := m.FindRoom(key)
		if err != nil || got.RoomID != r.RoomID {
			t.Errorf("room did not resolve by %q: %v", key, err)
		}
	}
	if _, err := m.FindRoom("no-such-room"); !errors.Is(err, errNoSuchRoom) {
		t.Errorf("an unknown name gave %v, want errNoSuchRoom", err)
	}
}

// The bug this pair exists for: joining a room named the same as one already held
// hit a UNIQUE index and failed the join. A name is a mnemonic drawn from 7,656
// combinations, the rooms are named by whoever created them, and this table
// accumulates every room ever recorded — so the clash was a question of when.
func TestJoiningTwoRoomsWithOneNameSucceeds(t *testing.T) {
	m := testMembership(t)
	first, err := m.CreateRoom(testIdentity(t).PeerID)
	if err != nil {
		t.Fatal(err)
	}
	// A different room, hosted elsewhere, that happens to carry the same name.
	other := uuidV4()
	if err := m.RecordRoom(other, first.RoomName); err != nil {
		t.Fatalf("recording a second room named %q failed: %v", first.RoomName, err)
	}
	for _, id := range []string{first.RoomID, other} {
		if _, ok := m.RoomByID(id); !ok {
			t.Errorf("room %s was not recorded", id)
		}
	}
}

// Having allowed the clash, the name must stop being an answer. Picking one
// silently would act on a room the person did not name.
func TestAnAmbiguousNameIsReportedNotGuessed(t *testing.T) {
	m := testMembership(t)
	first, _ := m.CreateRoom(testIdentity(t).PeerID)
	other := uuidV4()
	if err := m.RecordRoom(other, first.RoomName); err != nil {
		t.Fatal(err)
	}

	_, err := m.FindRoom(first.RoomName)
	if err == nil {
		t.Fatal("an ambiguous name resolved to a room")
	}
	if errors.Is(err, errNoSuchRoom) {
		t.Fatal("an ambiguous name was reported as unknown, which sends someone looking for a room they hold")
	}
	// The message has to be actionable: both identities, so the person can say
	// which they meant.
	for _, want := range []string{first.RoomID, other} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message does not name %s: %s", want, err)
		}
	}

	// And each is still reachable by the identifier that does not collide.
	for _, id := range []string{first.RoomID, other} {
		got, err := m.FindRoom(id)
		if err != nil || got.RoomID != id {
			t.Errorf("room %s did not resolve by id: %v", id, err)
		}
	}
}

// A membership.db written before the constraint was relaxed still enforces it:
// CREATE TABLE IF NOT EXISTS leaves the old table exactly as it was.
func TestMigrationDropsTheNameConstraint(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(homeDir(), "membership.db")
	if err := os.MkdirAll(homeDir(), 0o700); err != nil {
		t.Fatal(err)
	}

	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(`CREATE TABLE rooms (
	  room_id         TEXT PRIMARY KEY,
	  room_name       TEXT NOT NULL UNIQUE,
	  state           TEXT NOT NULL,
	  issued_sequence INTEGER NOT NULL DEFAULT 0,
	  created_at      TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(`INSERT INTO rooms (room_id, room_name, state, created_at)
		VALUES ('room-a','misty-canyon','joined','2026-09-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	old.Close()

	m, err := OpenMembership()
	if err != nil {
		t.Fatalf("opening a database from before the change: %v", err)
	}
	t.Cleanup(func() { m.Close() })

	// Migrated, not orphaned: what was there is still there.
	if r, ok := m.RoomByID("room-a"); !ok || r.RoomName != "misty-canyon" {
		t.Fatalf("the existing room did not survive: %+v", r)
	}
	if err := m.RecordRoom("room-b", "misty-canyon"); err != nil {
		t.Errorf("the constraint is still enforced after migration: %v", err)
	}
}

// Pairing is machine scope and outlives every room; a guest list is room scope.
// Collapsing them would make it impossible to know six colleagues and admit two
// to a room about customer data (§12).
func TestPairingOutlivesTheRoomsItEnables(t *testing.T) {
	m := testMembership(t)
	self, alice := testIdentity(t), testIdentity(t)
	if err := m.Allow(alice.PeerID, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := m.MarkVerified(alice.PeerID); err != nil {
		t.Fatal(err)
	}

	room, err := m.CreateRoom(self.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Invite(room.RoomID, alice.PeerID); err != nil {
		t.Fatal(err)
	}
	// Withdrawing admission to a room says nothing about knowing the person.
	if err := m.Revoke(room.RoomID, alice.PeerID); err != nil {
		t.Fatal(err)
	}
	if m.IsGuest(room.RoomID, alice.PeerID) {
		t.Error("revoke did not withdraw admission")
	}
	if !m.Knows(alice.PeerID) {
		t.Error("revoking a room's guest discarded the pairing")
	}
	if !m.IsVerified(alice.PeerID) {
		t.Error("revoking a room's guest discarded the verification")
	}

	// Forgetting the peer is the other direction, and does discard it.
	if err := m.Forget(alice.PeerID); err != nil {
		t.Fatal(err)
	}
	if m.Knows(alice.PeerID) || m.IsVerified(alice.PeerID) {
		t.Error("forget left the peer or its verification behind")
	}
}

// A paired peer is reachable before any room exists, which is what lets
// verification precede admission rather than follow it.
func TestAPairedPeerIsReachableWithoutARoom(t *testing.T) {
	m := testMembership(t)
	alice := testIdentity(t)
	if err := m.Allow(alice.PeerID, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := m.SetPeerEndpoint(alice.PeerID, "198.51.100.7:4783"); err != nil {
		t.Fatal(err)
	}

	d := &Daemon{id: testIdentity(t), members: m}
	t.Cleanup(d.closeStores)
	found := false
	for _, a := range d.syncTargets() {
		if a == "198.51.100.7:4783" {
			found = true
		}
	}
	if !found {
		t.Error("a paired peer's address is not reachable until a room exists")
	}
}

// A session is in a room because somebody put it there, and never because a
// machine-level setting happened to point somewhere when it started (D-064).
//
// The behaviour this replaces bound any session on its first prompt to whatever
// room was last created or joined. A room created and forgotten was therefore
// silently joined weeks later by a session in an unrelated repository, which began
// capturing and publishing without anyone doing anything — and nothing derives a
// room from a directory (D-015), so nothing else would have caught it.
func TestASessionJoinsARoomOnlyWhenPutInOne(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	room, err := m.CreateRoom(self.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	// Current room is set, as `create` sets it for command-line convenience.

	if got, ok := m.RoomForSession("session-a"); ok {
		t.Errorf("a session nobody put in a room is in %s", got.RoomName)
	}

	if err := m.BindSession("session-a", room.RoomID); err != nil {
		t.Fatal(err)
	}
	if got, ok := m.RoomForSession("session-a"); !ok || got.RoomID != room.RoomID {
		t.Fatalf("a session that was put in a room is not in it: %+v", got)
	}
	// Still nothing happens to a session nobody touched.
	if _, ok := m.RoomForSession("session-b"); ok {
		t.Error("binding one session bound another")
	}
}

// §12a forbids moving a session once it has been told something, and what it was
// told cannot be withdrawn — so the refusal is the mechanism, not a warning.
func TestABoundSessionCannotBeMoved(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	first, _ := m.CreateRoom(self.PeerID)
	second, _ := m.CreateRoom(self.PeerID)

	if err := m.BindSession("session-a", first.RoomID); err != nil {
		t.Fatal(err)
	}
	// Binding it to the same room again is not a move, and must not be an error:
	// running /team-join twice in one session is an ordinary thing to do.
	if err := m.BindSession("session-a", first.RoomID); err != nil {
		t.Errorf("rebinding a session to the room it is already in failed: %v", err)
	}

	err := m.BindSession("session-a", second.RoomID)
	if err == nil {
		t.Fatal("a session was moved to a second room")
	}
	if !strings.Contains(err.Error(), first.RoomName) {
		t.Errorf("the refusal does not say which room it is in: %v", err)
	}
	if got, _ := m.RoomForSession("session-a"); got.RoomID != first.RoomID {
		t.Error("a refused move changed the binding anyway")
	}
}

// The column is removed from databases that predate the change, not merely ignored.
func TestMigrationDropsTheInjectedColumn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(homeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	old, err := sql.Open("sqlite", filepath.Join(homeDir(), "membership.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(`CREATE TABLE session_rooms (
	  session_id TEXT PRIMARY KEY,
	  room_id    TEXT NOT NULL,
	  joined_at  TEXT NOT NULL,
	  injected   INTEGER NOT NULL DEFAULT 0
	);
	INSERT INTO session_rooms (session_id, room_id, joined_at, injected)
	  VALUES ('s1','r1','2026-09-01T00:00:00Z',1)`); err != nil {
		t.Fatal(err)
	}
	old.Close()

	m, err := OpenMembership()
	if err != nil {
		t.Fatalf("opening a database from before the change: %v", err)
	}
	t.Cleanup(func() { m.Close() })

	var n int
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('session_rooms') WHERE name = 'injected'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("the injected column survived migration")
	}
	// Migrated, not orphaned: the binding it carried is still there.
	var room string
	if err := m.db.QueryRow(`SELECT room_id FROM session_rooms WHERE session_id = 's1'`).Scan(&room); err != nil {
		t.Fatalf("the existing binding did not survive: %v", err)
	}
	if room != "r1" {
		t.Errorf("binding became %q, want r1", room)
	}
}

// The hazard D-064 removes, stated as a test so it cannot come back: a room that
// was created and forgotten must not collect sessions.
func TestAForgottenRoomDoesNotCollectSessions(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	if _, err := m.CreateRoom(self.PeerID); err != nil {
		t.Fatal(err)
	}

	// Weeks pass. A session starts in an unrelated repository and submits prompts.
	// Nothing can put it in a room now that the pointer is gone (D-080), which is
	// the same guarantee D-064 made by refusing to consult one.
	for _, s := range []string{"much-later-1", "much-later-2", "much-later-3"} {
		if got, ok := m.RoomForSession(s); ok {
			t.Fatalf("session %s was silently put in %s and would begin publishing", s, got.RoomName)
		}
	}
}

// Leaving is a pause, not a door closing behind you (D-071). Three things it must
// not do, each stated because each was either wrong or nearly wrong.
func TestLeavingDoesNotForecloseReturn(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	room, err := m.CreateRoom(self.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.BindSession("s1", room.RoomID); err != nil {
		t.Fatal(err)
	}

	left, err := m.LeaveSession("s1")
	if err != nil {
		t.Fatal(err)
	}
	if left.RoomID != room.RoomID {
		t.Errorf("leave reported %s, want %s", left.RoomName, room.RoomName)
	}
	// Stops participating.
	if got, ok := m.RoomForSession("s1"); ok {
		t.Errorf("a session that left is still in %s", got.RoomName)
	}
	// And is not counted as present.
	if n := m.SessionsInRoom(room.RoomID); n != 0 {
		t.Errorf("%d sessions present after the only one left", n)
	}

	// May return to the SAME room: §12a forbids moving to another, and coming
	// back to the one you were in is not a move.
	if err := m.BindSession("s1", room.RoomID); err != nil {
		t.Fatalf("a session could not rejoin the room it left: %v", err)
	}
	if got, ok := m.RoomForSession("s1"); !ok || got.RoomID != room.RoomID {
		t.Error("rejoining did not restore membership")
	}
}

// The trap that makes the tombstone necessary: if leaving deleted the row, then
// leave-then-join would be the move §12a exists to prevent, with two extra
// keystrokes. The session still holds the first room's turns in its context.
func TestLeavingIsNotALaunderedMove(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	first, _ := m.CreateRoom(self.PeerID)
	second, _ := m.CreateRoom(self.PeerID)

	if err := m.BindSession("s1", first.RoomID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.LeaveSession("s1"); err != nil {
		t.Fatal(err)
	}

	err := m.BindSession("s1", second.RoomID)
	if err == nil {
		t.Fatal("a session left one room and joined another; leaving laundered the move")
	}
	if !strings.Contains(err.Error(), first.RoomName) {
		t.Errorf("the refusal does not name the room it had been in: %v", err)
	}
	if got, ok := m.RoomForSession("s1"); ok {
		t.Errorf("the refused join bound the session to %s anyway", got.RoomName)
	}
}

// Leaving must not obscure the history, and must not change what the browser view
// or the command line are looking at. Clearing the current room used to be the
// whole of leaving, and it blanked the view.
func TestLeavingLeavesTheRoomVisible(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	room, _ := m.CreateRoom(self.PeerID)
	if err := m.BindSession("s1", room.RoomID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.LeaveSession("s1"); err != nil {
		t.Fatal(err)
	}

	// Leaving must not remove the room or its history; there is no longer a
	// pointer to disturb, which is the simpler form of the same guarantee.
	if _, ok := m.RoomByID(room.RoomID); !ok {
		t.Error("leaving removed the room")
	}
}

// The refusal has to be explainable, so it is a typed error carrying both rooms
// rather than a formatted string. A caller that cannot name the room a person was
// trying to reach cannot tell them what to do instead (D-072).
func TestARefusedJoinCanBeExplained(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	first, _ := m.CreateRoom(self.PeerID)
	second, _ := m.CreateRoom(self.PeerID)

	if err := m.BindSession("s1", first.RoomID); err != nil {
		t.Fatal(err)
	}
	err := m.BindSession("s1", second.RoomID)

	var bound *BoundElsewhereError
	if !errors.As(err, &bound) {
		t.Fatalf("refusal is not distinguishable from any other failure: %v", err)
	}
	if bound.Was.RoomID != first.RoomID {
		t.Errorf("the error names %s as the prior room, want %s", bound.Was.RoomName, first.RoomName)
	}
	if bound.Wanted.RoomID != second.RoomID {
		t.Errorf("the error names %s as the wanted room, want %s", bound.Wanted.RoomName, second.RoomName)
	}
	// Both names are what let the explanation say "start a second session to work
	// in X" and "this one can still rejoin Y".
	if bound.Was.RoomName == "" || bound.Wanted.RoomName == "" {
		t.Error("a room name is missing, so the explanation cannot name both rooms")
	}
}

// §12: "Forgetting discards the identity itself, so a later meeting is a first
// meeting again." That was not true — forget deleted the peer and left their room
// admissions behind, so meeting them again readmitted them to every room they had
// ever been in, without the host inviting them to any of it (D-073).
func TestForgettingDiscardsAdmissionToo(t *testing.T) {
	m := testMembership(t)
	self, peer := testIdentity(t), testIdentity(t)
	a, _ := m.CreateRoom(self.PeerID)
	b, _ := m.CreateRoom(self.PeerID)

	if err := m.Allow(peer.PeerID, "colleague"); err != nil {
		t.Fatal(err)
	}
	if err := m.MarkVerified(peer.PeerID); err != nil {
		t.Fatal(err)
	}
	for _, r := range []Room{a, b} {
		if err := m.Invite(r.RoomID, peer.PeerID); err != nil {
			t.Fatal(err)
		}
	}

	if err := m.Forget(peer.PeerID); err != nil {
		t.Fatal(err)
	}
	for _, r := range []Room{a, b} {
		if m.IsGuest(r.RoomID, peer.PeerID) {
			t.Errorf("a forgotten peer is still a guest of %s", r.RoomName)
		}
	}

	// A later meeting is a first meeting: known again, unverified again, and
	// admitted to nothing until a host says so.
	if err := m.Allow(peer.PeerID, "colleague"); err != nil {
		t.Fatal(err)
	}
	if m.IsVerified(peer.PeerID) {
		t.Error("meeting again carried the old verification forward")
	}
	for _, r := range []Room{a, b} {
		if m.IsGuest(r.RoomID, peer.PeerID) {
			t.Errorf("meeting again readmitted them to %s without an invitation", r.RoomName)
		}
	}
}

// And forgetting one peer must not disturb another, including the peer's own
// membership of its own rooms.
func TestForgettingIsNarrow(t *testing.T) {
	m := testMembership(t)
	self, one, two := testIdentity(t), testIdentity(t), testIdentity(t)
	room, _ := m.CreateRoom(self.PeerID)
	for _, p := range []*Identity{one, two} {
		if err := m.Allow(p.PeerID, ""); err != nil {
			t.Fatal(err)
		}
		if err := m.Invite(room.RoomID, p.PeerID); err != nil {
			t.Fatal(err)
		}
	}

	if err := m.Forget(one.PeerID); err != nil {
		t.Fatal(err)
	}
	if !m.Knows(two.PeerID) || !m.IsGuest(room.RoomID, two.PeerID) {
		t.Error("forgetting one peer disturbed another")
	}
	// The room's creator is its first guest and must survive.
	if !m.IsGuest(room.RoomID, self.PeerID) {
		t.Error("forgetting a peer removed the room's own creator")
	}
}

// A peerId IS a key (D-042), so "the same person with a new key" is not
// expressible — to this machine it is a peer it has never seen. The one moment the
// two can be connected is when somebody records the new key under a name they
// already use, which is exactly what a substitution looks like from the host's
// side (D-074).
func TestANameMeansOneKey(t *testing.T) {
	m := testMembership(t)
	alice, impostor := testIdentity(t), testIdentity(t)

	if err := m.Allow(alice.PeerID, "alice"); err != nil {
		t.Fatal(err)
	}

	err := m.Allow(impostor.PeerID, "alice")
	var taken *NameTakenError
	if !errors.As(err, &taken) {
		t.Fatalf("a second key was recorded under a name already in use: %v", err)
	}
	// Both keys, because the explanation has to show what changed.
	if taken.Existing != alice.PeerID || taken.Offered != impostor.PeerID {
		t.Errorf("the error does not carry both keys: %+v", taken)
	}
	// And nothing was written.
	if m.Knows(impostor.PeerID) {
		t.Error("the refused key was recorded anyway")
	}

	// Renaming the peer already known is not a collision with itself.
	if err := m.Allow(alice.PeerID, "alice"); err != nil {
		t.Errorf("re-recording the same peer under its own name failed: %v", err)
	}
	// And forgetting frees the name, which is the documented way to accept a
	// genuinely new key.
	if err := m.Forget(alice.PeerID); err != nil {
		t.Fatal(err)
	}
	if err := m.Allow(impostor.PeerID, "alice"); err != nil {
		t.Errorf("the name was not freed by forgetting: %v", err)
	}
}

// The watch line names a room or admits there is none. "Watch this room" was
// printed on a machine with no rooms at all — an invented fact in the one line
// whose job is to send somebody somewhere.
func TestTheWatchLineDoesNotInventARoom(t *testing.T) {
	none := watchLine("")
	if strings.Contains(none, "this room") {
		t.Errorf("with no room, the line still refers to one: %s", none)
	}
	if !strings.Contains(none, "http://") {
		t.Errorf("the address is missing, which is the only part that matters: %s", none)
	}

	named := watchLine("misty-canyon")
	if !strings.Contains(named, "misty-canyon") {
		t.Errorf("a named room is not named: %s", named)
	}
}

// A command run from inside a session acts on that session's room, before
// anything else is consulted. currentRoom went straight to the machine-level
// pointer, so /room-invite inside a session in one room admitted people to
// whichever room was last created — silently, and to the wrong people (D-076).
//
// A room-scoped command resolves one way: the room the invoking session is in.
// There is nowhere else to look since D-080 removed the machine-level pointer,
// which is what answered with the wrong room three separate times.
//
// The resolvers exit on failure, so the rule is asserted on what they depend on.
func TestARoomResolvesOnlyThroughTheSession(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	mine, _ := m.CreateRoom(self.PeerID)
	if _, err := m.CreateRoom(self.PeerID); err != nil {
		t.Fatal(err)
	}

	if err := m.BindSession("sess-A", mine.RoomID); err != nil {
		t.Fatal(err)
	}
	got, ok := m.RoomForSession("sess-A")
	if !ok || got.RoomID != mine.RoomID {
		t.Fatalf("the session's room resolved to %+v, want %s", got, mine.RoomName)
	}

	// A session in no room has no room, however many rooms the machine holds.
	if r, ok := m.RoomForSession("sess-Z"); ok {
		t.Errorf("a session in no room resolved to %s", r.RoomName)
	}
}

// An environment variable is ambient: exported once in a profile, or inherited by
// every session a machine starts, it answers for sessions that are in other rooms.
// CLAUDE_TEAM_ROOM was consulted ahead of the session's own room, which quietly
// reinstated the failure D-064 removed, at higher precedence (D-077).
func TestNoAmbientRoomOverride(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	// It may be named in a comment explaining its absence; it may not be read.
	if strings.Contains(string(src), `os.Getenv("CLAUDE_TEAM_ROOM")`) {
		t.Error("CLAUDE_TEAM_ROOM is read again; an ambient value must not decide which room a command acts on")
	}
	for _, f := range []string{"identity.go", "daemon.go", "sync.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), `os.Getenv("CLAUDE_TEAM_ROOM")`) {
			t.Errorf("%s reads CLAUDE_TEAM_ROOM", f)
		}
	}
}

// Standing, not choice. A room's guest list is its members' concern, and §22 puts
// membership in a session — so a terminal has no room to admit anybody to, and
// naming one would be reaching into a room that belongs to a session you are not
// in (D-079).
//
// roomToChange exits rather than returning an error, so the rule is asserted on
// what it depends on: a session id, and that session being in a room.
func TestOnlyASessionInTheRoomCanChangeIt(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	room, err := m.CreateRoom(self.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.BindSession("in-the-room", room.RoomID); err != nil {
		t.Fatal(err)
	}

	// The session that is in it has standing.
	if got, ok := m.RoomForSession("in-the-room"); !ok || got.RoomID != room.RoomID {
		t.Fatalf("the room's own session cannot resolve it: %+v", got)
	}
	// A session that is not in it has none, and the pointer must not lend it any.
	if got, ok := m.RoomForSession("elsewhere"); ok {
		t.Errorf("a session in no room resolved to %s", got.RoomName)
	}
}

// Every room-scoped slash command must map to a subcommand that exists, and every
// subcommand a person could want must be reachable from a session.
//
// The plugin and the binary version together (D-075), so a command file naming a
// subcommand the binary lacks is a release that half works — which is exactly what
// v0.2.0 shipped, where /room-status called `where` and got usage back.
func TestEverySlashCommandNamesARealSubcommand(t *testing.T) {
	files, err := filepath.Glob("../../plugin/commands/*.md")
	if err != nil || len(files) == 0 {
		t.Skipf("plugin commands not found from here: %v", err)
	}
	known := map[string]bool{}
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`case "([a-z-]+)":`).FindAllStringSubmatch(string(src), -1) {
		known[m[1]] = true
	}

	sub := regexp.MustCompile(`cli\.sh" ([a-z]+)`)
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range sub.FindAllStringSubmatch(string(body), -1) {
			if !known[m[1]] {
				t.Errorf("%s invokes %q, which is not a subcommand", filepath.Base(f), m[1])
			}
		}
	}
}
