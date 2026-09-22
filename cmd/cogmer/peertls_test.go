package main

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// servePeerTLS runs a daemon's peer routes behind the TLS a real peer listener
// would present, and returns the address to dial.
func servePeerTLS(t *testing.T, d *Daemon) string {
	t.Helper()
	return servePeerTLSHandler(t, d, d.PeerRoutes())
}

// servePeerTLSHandler is the same for a harness that wraps the routes — the
// offline tests gate them behind a reachability switch.
func servePeerTLSHandler(t *testing.T, d *Daemon, h http.Handler) string {
	t.Helper()
	conf, err := d.serverConfig()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l := tls.NewListener(raw, conf)
	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })
	return l.Addr().String()
}

func get(t *testing.T, d *Daemon, addr, expect string) (int, error) {
	t.Helper()
	conf, err := d.clientConfig(expect)
	if err != nil {
		t.Fatal(err)
	}
	c, err := clientFor(addr, 3*time.Second, conf)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Get(peerURL("/healthz"))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// Every peer connection is TLS pinned to a key this machine has RECORDED — not to
// one it has verified. Verification is what this connection carries, so requiring
// it would be a deadlock (D-101). What keeps an unverified peer harmless is a layer
// up: D-054 refuses its events either way.
func TestARecordedButUnverifiedPeerCanConnect(t *testing.T) {
	server, _ := testDaemon(t)
	client, _ := testDaemon(t)
	if err := server.members.Allow(client.id.PeerID, "c"); err != nil {
		t.Fatal(err)
	}
	if err := client.members.Allow(server.id.PeerID, "s"); err != nil {
		t.Fatal(err)
	}

	// Neither has verified the other, and that is the case under test rather than
	// an oversight: verification happens OVER this connection, so a pin that
	// required it would make the only route to it unreachable and nobody could
	// ever pair. If this test is ever seen failing, the fix is not to verify the
	// peers here (D-101).
	if server.members.IsVerified(client.id.PeerID) || client.members.IsVerified(server.id.PeerID) {
		t.Fatal("this test is meaningless unless both peers are unverified")
	}

	addr := servePeerTLS(t, server)
	code, err := get(t, client, addr, server.id.PeerID)
	if err != nil {
		t.Fatalf("a recorded but unverified peer could not connect, so pairing can never complete: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("got %d, want 200", code)
	}
}

// A stranger has nothing to pin to. Refusing at the handshake is earlier and
// cheaper than refusing after a body has been read, and it means an unknown peer
// never reaches a route at all.
func TestAStrangerCompletesNoHandshake(t *testing.T) {
	server, _ := testDaemon(t)
	stranger, _ := testDaemon(t)
	// The stranger knows the server, so the refusal can only come from the
	// server's side of the pin.
	if err := stranger.members.Allow(server.id.PeerID, "s"); err != nil {
		t.Fatal(err)
	}

	addr := servePeerTLS(t, server)
	if _, err := get(t, stranger, addr, server.id.PeerID); err == nil {
		t.Fatal("a peer the server does not know completed a connection")
	}
}

// When a dial is for one named peer, reaching a different key at that address is
// refused — which is what an address quietly changing hands looks like.
func TestAnAddressAnsweringForAnotherKeyIsRefused(t *testing.T) {
	server, _ := testDaemon(t)
	client, _ := testDaemon(t)
	other, _ := testDaemon(t)
	for _, p := range []string{client.id.PeerID, other.id.PeerID} {
		if err := server.members.Allow(p, PeerName(p)); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{server.id.PeerID, other.id.PeerID} {
		if err := client.members.Allow(p, PeerName(p)); err != nil {
			t.Fatal(err)
		}
	}

	addr := servePeerTLS(t, server)
	// Both ends know each other, so only the expectation can refuse this.
	_, err := get(t, client, addr, other.id.PeerID)
	if err == nil {
		t.Fatal("an address answering for a different key was accepted")
	}
	if !strings.Contains(err.Error(), "different key") {
		t.Errorf("refused for the wrong reason: %v", err)
	}
}

// The scheme is what makes http.Transport perform the handshake. A peer URL that
// reverted to http would disable every check above and nothing else would fail.
func TestPeerURLsAreHTTPS(t *testing.T) {
	if !strings.HasPrefix(peerURL("/sync"), "https://") {
		t.Fatalf("peerURL is %q; TLS would be skipped entirely", peerURL("/sync"))
	}
}
