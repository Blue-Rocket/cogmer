package main

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"
)

// The carrying of a machine key and its rooms into the user identity (D-242, a
// machine key from before the user identity carries its verification, label and
// turns to the first user it proves it belongs to; D-243, a room from before user
// identities takes one guest list, founded by the users whose machines wrote its
// turns).

const oldRoom, oldRoomName = "11111111-2222-4333-8444-555555555555", "old-harbor"

// becomeTestUser makes p's machine one of a new user's with the key it has, as the
// identity page does, and returns the user.
func becomeTestUser(t *testing.T, p *offlinePeer, name string) *testUser {
	t.Helper()
	u := newTestUser(t)
	chain := u.chainFor(p.d.id.private.Public().(ed25519.PublicKey), name)
	p.d.home = t.TempDir()
	if err := p.d.becomeUser(u.id, name, chain, u.shared, u.version); err != nil {
		t.Fatal(err)
	}
	return u
}

// named records each side's label for the other, as pairing with a name does.
func named(t *testing.T, p *offlinePeer, other *offlinePeer, label string) {
	t.Helper()
	if err := p.d.members.Allow(other.d.id.PeerID, label); err != nil {
		t.Fatal(err)
	}
}

func viewHas(t *testing.T, p *offlinePeer, text string) bool {
	t.Helper()
	return strings.Contains(contents(heldEvents(t, p, oldRoom)), text)
}

// Two machines that paired and shared a room before either had a user identity
// upgrade one at a time. Nobody pairs again: each key carries its verification and
// label to its user, the room takes one list founded by the users who wrote its
// turns, and every turn, from before, from while only one side had upgraded and from
// after, reaches both.
func TestAPairedRoomFromBeforeUserIdentitiesSurvivesBothUpgrades(t *testing.T) {
	a := newOfflinePeer(t, oldRoom, oldRoomName)
	b := newOfflinePeer(t, oldRoom, oldRoomName)
	introduce(t, a, b, oldRoom)
	named(t, a, b, "bob")
	named(t, b, a, "alice")
	a.say(t, oldRoom, "s", EventUserPrompt, "before a")
	b.say(t, oldRoom, "s", EventUserPrompt, "before b")
	settle(t, a, b, Room{RoomID: oldRoom})
	bKey := b.d.id.PeerID

	ub := becomeTestUser(t, b, "bob-desktop")
	if !b.d.members.EventListed(oldRoom) {
		t.Fatal("the upgraded machine did not carry the room into one list")
	}
	b.say(t, oldRoom, "s", EventUserPrompt, "during b")
	a.say(t, oldRoom, "s", EventUserPrompt, "during a")
	settle(t, a, b, Room{RoomID: oldRoom})
	for _, p := range []*offlinePeer{a, b} {
		if !viewHas(t, p, "during a") || !viewHas(t, p, "during b") {
			t.Fatalf("while only one side had upgraded, a machine holds %q", contents(heldEvents(t, p, oldRoom)))
		}
	}
	if got := a.d.members.Label(ub.id); got != "bob" {
		t.Errorf("alice's machine calls bob's user %q, want the label she gave his machine", got)
	}
	if !a.d.members.IsVerified(ub.id) || a.d.members.Knows(bKey) {
		t.Error("the verification of bob's machine key did not pass to his user")
	}
	if got := a.d.members.Label(bKey); got != "bob" {
		t.Errorf("a turn bob's key signed before his user existed is labeled %q", got)
	}

	ua := becomeTestUser(t, a, "alice-laptop")
	a.say(t, oldRoom, "s", EventUserPrompt, "after a")
	b.say(t, oldRoom, "s", EventUserPrompt, "after b")
	settle(t, a, b, Room{RoomID: oldRoom})
	for _, p := range []*offlinePeer{a, b} {
		for _, text := range []string{"before a", "before b", "during a", "during b", "after a", "after b"} {
			if !viewHas(t, p, text) {
				t.Errorf("%s does not hold %q", PeerName(p.d.id.PeerID), text)
			}
		}
		g := p.d.guestList(oldRoom)
		if g == nil || !g.Admitted[ua.id] || !g.Admitted[ub.id] || len(g.Admitted) != 2 {
			t.Fatalf("the carried room's list is %+v", g)
		}
		if !g.Counts(a.d.id.PeerID, a.d.id.PeerID, 1) || !g.Counts(bKey, bKey, 1) {
			t.Error("a turn a key signed before its user existed does not count as the user's")
		}
	}
	if got := b.d.members.Label(ua.id); got != "alice" || !b.d.members.IsVerified(ua.id) {
		t.Errorf("bob's machine holds alice's user as %q, verified %v", got, b.d.members.IsVerified(ua.id))
	}
	// The room's sync targets survive the machines moving to their users.
	if got := a.d.members.RoomPeers(oldRoom, a.d.id.PeerID); len(got) != 1 {
		t.Errorf("alice's machine has %v as the room's targets", got)
	}
}

// A guest who joined a room and never spoke is on the carried list, invited by the
// machine whose earlier list admitted them, and a room with no turns is not carried.
func TestASilentGuestIsCarriedAndARoomWithNoTurnsIsNot(t *testing.T) {
	a := newOfflinePeer(t, oldRoom, oldRoomName)
	silent := testIdentity(t)
	if err := a.d.members.Allow(silent.PeerID, "sam"); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.Invite(oldRoom, silent.PeerID); err != nil {
		t.Fatal(err)
	}
	becomeTestUser(t, a, "alice-laptop")
	if a.d.members.EventListed(oldRoom) {
		t.Fatal("a room with no turns was carried")
	}

	a.d.id.User, a.d.id.Chain = "", nil
	a.say(t, oldRoom, "s", EventUserPrompt, "a turn")
	ua := becomeTestUser(t, a, "alice-laptop")
	g := a.d.guestList(oldRoom)
	if g == nil || !g.Admitted[ua.id] || !g.Admitted[silent.PeerID] {
		t.Fatalf("the carried list is %+v; want the founder and the silent guest", g)
	}
	if !a.d.members.IsGuest(oldRoom, silent.PeerID) {
		t.Error("the silent guest's machine is refused")
	}

	// Carrying again invites nobody twice.
	before := len(heldEvents(t, a, oldRoom))
	a.d.mu.Lock()
	a.d.carryRooms()
	a.d.mu.Unlock()
	if after := len(heldEvents(t, a, oldRoom)); after != before {
		t.Errorf("carrying the room again wrote %d events", after-before)
	}
}

// The first user a key proves it belongs to is its user. A second user it proves
// it belongs to passes nothing over and is reported as a conflict (D-242).
func TestASecondUserClaimingAKeyCarriesNothing(t *testing.T) {
	m := testMembership(t)
	key := testIdentity(t)
	if err := m.Allow(key.PeerID, "bob"); err != nil {
		t.Fatal(err)
	}
	if err := m.MarkVerified(key.PeerID); err != nil {
		t.Fatal(err)
	}
	first, second := newTestUser(t), newTestUser(t)
	if linked, err := m.LinkMachineKey(key.PeerID, first.id); !linked || err != nil {
		t.Fatalf("the first user did not take the key: %v %v", linked, err)
	}
	if linked, err := m.LinkMachineKey(key.PeerID, first.id); linked || err != nil {
		t.Fatalf("the same user again: %v %v", linked, err)
	}
	_, err := m.LinkMachineKey(key.PeerID, second.id)
	var conflict *LinkConflictError
	if !errors.As(err, &conflict) || conflict.First != first.id || conflict.Second != second.id {
		t.Fatalf("a second user's claim returned %v", err)
	}
	if m.Knows(second.id) || m.IsVerified(second.id) || m.Label(first.id) != "bob" || !m.IsVerified(first.id) {
		t.Error("the second user's claim moved something")
	}
	if u, _ := m.LinkedUser(key.PeerID); u != first.id {
		t.Errorf("the key is linked to %s", u)
	}
	if got := m.LinkConflicts(); len(got) != 1 {
		t.Errorf("the conflict is not reported: %+v", got)
	}
	if err := m.AssignMachineUser(key.PeerID, second.id); err != nil {
		t.Fatal(err)
	}
	if ms, _ := m.UserMachines(second.id); len(ms) != 0 {
		t.Error("the machine was given to the second user")
	}
	// A key this machine does not know as a colleague carries nothing.
	stranger := testIdentity(t)
	if linked, err := m.LinkMachineKey(stranger.PeerID, second.id); linked || err != nil {
		t.Errorf("a stranger's key linked: %v %v", linked, err)
	}
}

// A machine given to its user keeps the rooms it joined as sync targets, in a room
// whose list is room_guests and in one whose list is kept as events.
func TestAMachineGivenToItsUserStaysARoomsSyncTarget(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	room, err := m.CreateRoom(self.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	key := testIdentity(t)
	if err := m.Allow(key.PeerID, "bob"); err != nil {
		t.Fatal(err)
	}
	if err := m.Invite(room.RoomID, key.PeerID); err != nil {
		t.Fatal(err)
	}
	if err := m.SetPeerEndpoint(key.PeerID, "tc://bob"); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordRoomMachine(room.RoomID, key.PeerID, key.PeerID); err != nil {
		t.Fatal(err)
	}
	u := newTestUser(t)
	if _, err := m.LinkMachineKey(key.PeerID, u.id); err != nil {
		t.Fatal(err)
	}
	if got := m.RoomPeers(room.RoomID, self.PeerID); len(got) != 1 || got[0] != "tc://bob" {
		t.Fatalf("the room's targets are %v once the machine is its user's", got)
	}
	if !m.IsGuest(room.RoomID, u.id) {
		t.Fatal("the user is not a guest where their machine key is")
	}
	if err := m.CacheGuestList(&GuestList{Room: room.RoomID, Admitted: map[string]bool{u.id: true, self.PeerID: true}}); err != nil {
		t.Fatal(err)
	}
	if got := m.RoomPeers(room.RoomID, self.PeerID); len(got) != 1 || got[0] != "tc://bob" {
		t.Fatalf("the room's targets are %v once its list is kept as events", got)
	}
}

// Recording a peer with no name keeps the name the user gave them, so joining a room
// from an invitation leaves the inviter's label.
func TestAllowWithNoNameKeepsTheLabel(t *testing.T) {
	m := testMembership(t)
	p := testIdentity(t)
	if err := m.Allow(p.PeerID, "bob"); err != nil {
		t.Fatal(err)
	}
	if err := m.Allow(p.PeerID, ""); err != nil {
		t.Fatal(err)
	}
	if got := m.Label(p.PeerID); got != "bob" {
		t.Fatalf("the label is %q after recording the peer again with no name", got)
	}
	if err := m.Allow(p.PeerID, "robert"); err != nil {
		t.Fatal(err)
	}
	if got := m.Label(p.PeerID); got != "robert" {
		t.Fatalf("a name given replaced nothing: %q", got)
	}
}
