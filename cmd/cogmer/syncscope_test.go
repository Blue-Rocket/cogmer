package main

import (
	"sort"
	"strings"
	"testing"
)

// A peer that is known and is in no room is not asked for anything. Asking sends it a signed
// request naming a room it is not a guest of, and a pairing that was never finished leaves a
// record that was asked every second for 18 hours (D-103).
func TestAPeerInNoRoomIsNotAskedAndAGuestIsAskedOnlyAboutItsOwnRooms(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	roomA, _ := m.CreateRoom(self.PeerID, false)
	roomB, _ := m.CreateRoom(self.PeerID, false)
	alice, bob, owl := testIdentity(t).PeerID, testIdentity(t).PeerID, testIdentity(t).PeerID
	for name, p := range map[string]string{"alice": alice, "bob": bob, "owl": owl} {
		if err := m.Allow(p, name); err != nil {
			t.Fatal(err)
		}
		if err := m.SetPeerEndpoint(p, "tc://"+name); err != nil {
			t.Fatal(err)
		}
	}
	for _, g := range []struct{ room, peer string }{{roomA.RoomID, alice}, {roomA.RoomID, bob}, {roomB.RoomID, bob}} {
		if err := m.Invite(g.room, g.peer); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("COGMER_PEERS", "")
	d := &Daemon{id: self, members: m}
	t.Cleanup(d.closeStores)

	targets := d.syncTargets()
	sort.Strings(targets)
	if got := strings.Join(targets, ","); got != "tc://alice,tc://bob" {
		t.Errorf("the addresses asked are %q; a known peer who is in no room must not be one", got)
	}
	rooms, _ := m.Rooms()
	names := func(rs []Room) string {
		var out []string
		for _, r := range rs {
			out = append(out, r.RoomName)
		}
		sort.Strings(out)
		return strings.Join(out, ",")
	}
	want := func(r ...Room) string { return names(r) }
	if got := names(d.roomsToAsk("tc://alice", rooms)); got != want(roomA) {
		t.Errorf("alice is asked about %q, want only the room she is a guest of", got)
	}
	if got := names(d.roomsToAsk("tc://bob", rooms)); got != want(roomA, roomB) {
		t.Errorf("bob is asked about %q, want both his rooms", got)
	}
	if got := d.roomsToAsk("tc://owl", rooms); len(got) != 0 {
		t.Errorf("a peer in no room is asked about %d rooms", len(got))
	}
	// an address configured for this machine names no peer, so it is asked about every room
	t.Setenv("COGMER_PEERS", "tc://configured")
	if got := len(d.roomsToAsk("tc://configured", rooms)); got != 2 {
		t.Errorf("a configured address is asked about %d rooms, want 2", got)
	}
	if got := d.syncTargets(); len(got) != 3 {
		t.Errorf("the addresses asked are %v, want the two guests and the configured one", got)
	}
}

// shareARoom makes a peer a guest of a room of this daemon's, which is what makes its
// address one that is asked and reported (D-103).
func shareARoom(t *testing.T, d *Daemon, peer string) {
	t.Helper()
	room, err := d.members.CreateRoom(d.id.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.members.Invite(room.RoomID, peer); err != nil {
		t.Fatal(err)
	}
}
