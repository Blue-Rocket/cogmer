package main

import (
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
		got, ok := m.FindRoom(key)
		if !ok || got.RoomID != r.RoomID {
			t.Errorf("room did not resolve by %q", key)
		}
	}
	if _, ok := m.FindRoom("no-such-room"); ok {
		t.Error("an unknown name resolved to a room")
	}
}
