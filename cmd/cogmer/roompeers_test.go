package main

import "testing"

// A room's page reports the guests of that room. A colleague who is a guest of another
// room only is not a member of this one, and reporting on them says something false
// about the room.
func TestARoomPageReportsOnlyThatRoomsGuests(t *testing.T) {
	d, first := testDaemon(t)
	second, err := d.members.CreateRoom(d.id.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	inFirst, inSecond := testIdentity(t), testIdentity(t)
	for _, c := range []struct {
		peer *Identity
		name string
		room Room
		addr string
	}{
		{inFirst, "arielle", first, "127.0.0.1:59001"},
		{inSecond, "gareth", second, "127.0.0.1:59002"},
	} {
		if err := d.members.Allow(c.peer.PeerID, c.name); err != nil {
			t.Fatal(err)
		}
		if err := d.members.Invite(c.room.RoomID, c.peer.PeerID); err != nil {
			t.Fatal(err)
		}
		if err := d.members.SetPeerEndpoint(c.peer.PeerID, c.addr); err != nil {
			t.Fatal(err)
		}
	}
	// An address configured for the machine is asked about every room.
	t.Setenv("COGMER_PEERS", "127.0.0.1:59003")

	names := func(room Room) map[string]bool {
		st, err := d.snapshot(room)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, p := range st.Peers {
			got[p.Name] = true
		}
		return got
	}
	one, two := names(first), names(second)
	if !one["arielle"] || one["gareth"] {
		t.Errorf("the first room's page reports %v, want its own guest and not the other room's", one)
	}
	if !two["gareth"] || two["arielle"] {
		t.Errorf("the second room's page reports %v, want its own guest and not the other room's", two)
	}
	if !one["127.0.0.1:59003"] || !two["127.0.0.1:59003"] {
		t.Errorf("an address configured for the machine is missing from a room's page: %v, %v", one, two)
	}

	// The command that tells a person whom they can reach is about every peer.
	all := map[string]bool{}
	for _, p := range d.peerStatus() {
		all[p.Name] = true
	}
	if !all["arielle"] || !all["gareth"] {
		t.Errorf("the status of every peer lacks one of them: %v", all)
	}
}
