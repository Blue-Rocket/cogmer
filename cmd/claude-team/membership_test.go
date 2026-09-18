package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
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
	if err := m.SetCurrentRoom(room.RoomID); err != nil {
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
