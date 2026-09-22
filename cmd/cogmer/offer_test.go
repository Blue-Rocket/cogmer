package main

import (
	"testing"
)

// introduceDaemons makes two daemons know, verify and be able to reach each other,
// which is the state pairing leaves them in.
func introduceDaemons(t *testing.T, a, b *Daemon, aAddr, bAddr string) {
	t.Helper()
	for _, p := range []struct {
		from, to *Daemon
		addr     string
	}{{a, b, bAddr}, {b, a, aAddr}} {
		if err := p.from.members.Allow(p.to.id.PeerID, PeerName(p.to.id.PeerID)); err != nil {
			t.Fatal(err)
		}
		if err := p.from.members.SetPeerEndpoint(p.to.id.PeerID, p.addr); err != nil {
			t.Fatal(err)
		}
		if err := p.from.members.MarkVerified(p.to.id.PeerID); err != nil {
			t.Fatal(err)
		}
	}
}

// An invitation reaches the guest over the channel pairing established, so forming
// a room costs one command rather than a command plus a pasted string (D-105).
func TestAnInvitationArrivesOverTheChannel(t *testing.T) {
	host, _ := testDaemon(t)
	guest, _ := testDaemon(t)
	hostAddr := servePeerTLS(t, host)
	guestAddr := servePeerTLS(t, guest)
	introduceDaemons(t, host, guest, hostAddr, guestAddr)

	room, err := host.members.CreateRoom(host.id.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	if err := host.members.Invite(room.RoomID, guest.id.PeerID); err != nil {
		t.Fatal(err)
	}
	if err := host.deliverOffer(guest.id.PeerID, room); err != nil {
		t.Fatalf("delivery failed: %v", err)
	}

	offers := guest.members.Offers()
	if len(offers) != 1 {
		t.Fatalf("guest has %d offers, want 1", len(offers))
	}
	if offers[0].RoomName != room.RoomName || offers[0].Host != host.id.PeerID {
		t.Errorf("offer is %+v; want %s from the host", offers[0], room.RoomName)
	}

	// An offer is not a room. The offered room must not be recorded, synchronized
	// or displayed until the guest accepts, because none of that has been agreed
	// to. (The guest has a room of its own from the fixture; it is this one that
	// must be absent.)
	if _, ok := guest.members.RoomByID(room.RoomID); ok {
		t.Error("the offered room was recorded before anybody accepted it")
	}
	if _, ok := guest.members.FindOffer(room.RoomName); !ok {
		t.Error("the offer cannot be found by the name the guest was shown")
	}
}

// A host withholds until verification, and an offer that arrives early is refused
// rather than stored: accepting it would produce the silent room the withholding
// exists to prevent (D-106).
func TestAnOfferFromAnUnverifiedPeerIsRefused(t *testing.T) {
	host, _ := testDaemon(t)
	guest, _ := testDaemon(t)
	hostAddr := servePeerTLS(t, host)
	guestAddr := servePeerTLS(t, guest)

	// Each records the other — enough to connect, since the pin is on recorded
	// rather than verified (D-101) — but the guest never verifies the host, which
	// is the state an abandoned pairing leaves behind (D-093).
	if err := host.members.Allow(guest.id.PeerID, "g"); err != nil {
		t.Fatal(err)
	}
	if err := host.members.SetPeerEndpoint(guest.id.PeerID, guestAddr); err != nil {
		t.Fatal(err)
	}
	if err := host.members.MarkVerified(guest.id.PeerID); err != nil {
		t.Fatal(err)
	}
	if err := guest.members.Allow(host.id.PeerID, "h"); err != nil {
		t.Fatal(err)
	}
	if err := guest.members.SetPeerEndpoint(host.id.PeerID, hostAddr); err != nil {
		t.Fatal(err)
	}

	room, _ := host.members.CreateRoom(host.id.PeerID)
	_ = host.members.Invite(room.RoomID, guest.id.PeerID)
	if err := host.deliverOffer(guest.id.PeerID, room); err == nil {
		t.Fatal("an offer from an unverified peer was accepted")
	}
	if len(guest.members.Offers()) != 0 {
		t.Error("it was stored anyway")
	}
}

// Completing a verification delivers whatever was waiting, so invite-then-verify
// ends with a working room rather than a silent one (D-106).
func TestVerifyingDeliversWhatWasWaiting(t *testing.T) {
	host, _ := testDaemon(t)
	guest, _ := testDaemon(t)
	hostAddr := servePeerTLS(t, host)
	guestAddr := servePeerTLS(t, guest)
	introduceDaemons(t, host, guest, hostAddr, guestAddr)

	// Two rooms admitted before the guest could be told about either.
	var names []string
	for i := 0; i < 2; i++ {
		r, err := host.members.CreateRoom(host.id.PeerID)
		if err != nil {
			t.Fatal(err)
		}
		if err := host.members.Invite(r.RoomID, guest.id.PeerID); err != nil {
			t.Fatal(err)
		}
		names = append(names, r.RoomName)
	}

	host.deliverPending(guest.id.PeerID)

	if got := len(guest.members.Offers()); got != 2 {
		t.Fatalf("%d offers delivered, want %d", got, len(names))
	}
	for _, n := range names {
		if _, ok := guest.members.FindOffer(n); !ok {
			t.Errorf("%s was admitted and never offered", n)
		}
	}
}

// An unverified peer is a state; one holding up an invitation is a reason. The
// count is what lets /cogmer:peer-list say what verifying would release (D-106).
func TestWithheldInvitationsAreCounted(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	peer := testIdentity(t).PeerID
	if err := m.Allow(peer, "alice"); err != nil {
		t.Fatal(err)
	}
	if n := m.WithheldFor(peer); n != 0 {
		t.Fatalf("nothing admitted yet and %d withheld", n)
	}

	for i := 0; i < 2; i++ {
		r, err := m.CreateRoom(self.PeerID)
		if err != nil {
			t.Fatal(err)
		}
		if err := m.Invite(r.RoomID, peer); err != nil {
			t.Fatal(err)
		}
	}
	if n := m.WithheldFor(peer); n != 2 {
		t.Errorf("withheld = %d, want 2", n)
	}

	// Verifying releases them, so there is nothing left to report. A count that
	// survived verification would be a standing reproach for work already done.
	if err := m.MarkVerified(peer); err != nil {
		t.Fatal(err)
	}
	if n := m.WithheldFor(peer); n != 0 {
		t.Errorf("withheld = %d after verifying, want 0", n)
	}
}
