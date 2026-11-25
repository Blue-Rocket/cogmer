package main

import (
	"crypto/ed25519"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// countedServer serves a daemon's peer routes over TLS and counts the connections it
// accepts, each of which is a handshake. Closing it frees the address, so a test can
// serve again at the same address.
type countedServer struct {
	addr   string
	conns  atomic.Int32
	server *http.Server
}

func serveCounted(t *testing.T, d *Daemon, addr string) *countedServer {
	t.Helper()
	conf, err := d.serverConfig()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	cs := &countedServer{addr: raw.Addr().String()}
	cs.server = &http.Server{Handler: d.PeerRoutes(), ConnState: func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			cs.conns.Add(1)
		}
	}}
	go func() { _ = cs.server.Serve(tls.NewListener(raw, conf)) }()
	t.Cleanup(func() { _ = cs.server.Close() })
	return cs
}

// Polls of one address use the connection the first poll made, so the address costs
// one handshake and not one each second (D-303, a machine keeps one connection to each
// address it polls).
func TestPollsOfAnAddressUseTheConnectionTheFirstPollMade(t *testing.T) {
	a := newOfflinePeer(t, "room-keep", "keep")
	b := newOfflinePeer(t, "room-keep", "keep")
	introduce(t, a, b, "room-keep")
	b.say(t, "room-keep", "s", EventUserPrompt, "said once")
	srv := serveCounted(t, b.d, "127.0.0.1:0")

	for i := 0; i < 5; i++ {
		if _, err := a.d.pullFrom(srv.addr); err != nil {
			t.Fatal(err)
		}
	}
	if got := contents(heldEvents(t, a, "room-keep")); got != "said once" {
		t.Fatalf("the polls took %q", got)
	}
	if n := srv.conns.Load(); n != 1 {
		t.Errorf("five polls made %d connections; want the first poll's alone", n)
	}
}

// A kept connection to a machine that has been suspended since it was made is not used
// again: the next poll makes a new connection, whose handshake refuses the machine, as
// a handshake does for a machine this machine's records no longer accept (D-101, peer
// connections are TLS pinned to a key this machine has recorded).
func TestAKeptConnectionToAMachineSuspendedSinceIsNotUsedAgain(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	oldID := ana.machine(t, "old")
	ana.rotate(t)
	keptID := ana.machine(t, "kept")
	kept := newUserPeer(t, trustRoom, trustRoomName, keptID, ana)
	old := newUserPeer(t, trustRoom, trustRoomName, oldID, nil)
	colleague := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, kept, colleague, trustRoom)
	introduceUsers(t, old, colleague, trustRoom)
	srv := serveCounted(t, old.d, "127.0.0.1:0")

	if _, err := colleague.d.pullFrom(srv.addr); err != nil {
		t.Fatalf("before the colleague learns of the new shared key, the old machine is accepted: %v", err)
	}
	// The kept machine asks the colleague, which tells it of shared key 2, under which
	// the old machine is not certified and so is suspended.
	if _, err := kept.d.pullFrom(colleague.addr); err != nil {
		t.Fatal(err)
	}
	_, err := colleague.d.pullFrom(srv.addr)
	if err == nil {
		t.Fatal("a poll of the suspended machine succeeded on the connection made before its suspension")
	}
	if n := srv.conns.Load(); n != 2 {
		t.Errorf("the suspended machine saw %d connections; want a new one after the suspension (%v)", n, err)
	}
}

// A kept client is made again when this machine's certificate has changed, as it does
// when a user identity is made or a machine is certified again, so that the other side
// sees the certificate this machine presents now.
func TestAKeptClientIsMadeAgainWhenThisMachinesCertificateChanges(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	laptop := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	colleague := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, laptop, colleague, trustRoom)
	srv := serveCounted(t, colleague.d, "127.0.0.1:0")

	if _, err := laptop.d.pullFrom(srv.addr); err != nil {
		t.Fatal(err)
	}
	if _, err := laptop.d.pullFrom(srv.addr); err != nil {
		t.Fatal(err)
	}
	chain := ana.chainFor(laptop.d.id.private.Public().(ed25519.PublicKey), "laptop")
	laptop.d.id.Chain = &chain
	laptop.d.tls.reset()
	if _, err := laptop.d.pullFrom(srv.addr); err != nil {
		t.Fatal(err)
	}
	if n := srv.conns.Load(); n != 2 {
		t.Errorf("polls before and after the machine was certified again made %d connections; want 2", n)
	}
}

// A connection the other side closed is not used again: after a poll of an address that
// went away fails, the next poll makes a new connection, so the address is reached when
// it comes back.
func TestAPollAfterAConnectionBrokeMakesANewOne(t *testing.T) {
	a := newOfflinePeer(t, "room-keep", "keep")
	b := newOfflinePeer(t, "room-keep", "keep")
	introduce(t, a, b, "room-keep")
	srv := serveCounted(t, b.d, "127.0.0.1:0")

	if _, err := a.d.pullFrom(srv.addr); err != nil {
		t.Fatal(err)
	}
	_ = srv.server.Close()
	if _, err := a.d.pullFrom(srv.addr); err == nil {
		t.Fatal("a poll of a closed address succeeded")
	}
	again := serveCounted(t, b.d, srv.addr)
	b.say(t, "room-keep", "s", EventUserPrompt, "back again")
	if _, err := a.d.pullFrom(again.addr); err != nil {
		t.Fatalf("the address came back and the poll still failed: %v", err)
	}
	if got := contents(heldEvents(t, a, "room-keep")); !strings.Contains(got, "back again") {
		t.Errorf("the poll after the address came back took %q", got)
	}
	if n := again.conns.Load(); n != 1 {
		t.Errorf("the address that came back saw %d connections; want 1", n)
	}
}
