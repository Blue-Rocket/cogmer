package main

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

// A colleague's daemon dials with a client key this machine has never been told,
// because the client generates its own. So the listener must let in a dialer it has
// never seen, or no two machines on different networks ever reach each other. The
// TLS pin and the signed request decide who is a peer, above this.
//
// It needs the public relay, so it runs only when COGMER_NETWORK_TESTS is set.
func TestATunnelAdmitsADialerItHasNeverSeen(t *testing.T) {
	if os.Getenv("COGMER_NETWORK_TESTS") == "" {
		t.Skip("needs the public relay; set COGMER_NETWORK_TESTS=1 to run it")
	}
	t.Setenv("COGMER_HOME", t.TempDir())

	l, endpoint, srv, err := StartTailcat()
	if err != nil {
		t.Fatalf("StartTailcat: %v", err)
	}
	defer srv.Close()
	defer l.Close()

	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		_, _ = io.WriteString(c, "ok")
		c.Close()
	}()

	e, err := ParseEndpoint(endpoint)
	if err != nil {
		t.Fatalf("ParseEndpoint(%q): %v", endpoint, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := (&tailcatDialer{}).Dial(ctx, e)
	if err != nil {
		t.Fatalf("a dialer the listener had never seen could not open a tunnel: %v", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	got, err := io.ReadAll(conn)
	if err != nil || string(got) != "ok" {
		t.Fatalf("the tunnel opened but carried nothing back: %q, %v", got, err)
	}
}
