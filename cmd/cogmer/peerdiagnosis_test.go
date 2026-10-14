package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tailscale/tailcat"
)

// fakeTransport stands in for the overlay, which diagnoses and discards.
type fakeTransport struct {
	mu        sync.Mutex
	diagnosed int
	resets    int
	hold      chan struct{} // a diagnosis waits on it when it is set
}

func (f *fakeTransport) Dial(context.Context, Endpoint) (net.Conn, error) {
	return nil, errors.New("no")
}

func (f *fakeTransport) Diagnose(_ context.Context, e Endpoint) string {
	f.mu.Lock()
	f.diagnosed++
	hold := f.hold
	f.mu.Unlock()
	if hold != nil {
		<-hold
	}
	return "FAKE-DIAGNOSIS of " + e.Value
}

func (f *fakeTransport) Reset(Endpoint) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resets++
	return true
}

func (f *fakeTransport) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.diagnosed, f.resets
}

func withFakeOverlay(t *testing.T) (*fakeTransport, *bytes.Buffer) {
	t.Helper()
	f := &fakeTransport{}
	old := dialers[schemeTailcat]
	dialers[schemeTailcat] = f
	t.Cleanup(func() { dialers[schemeTailcat] = old })
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return f, &logged
}

// When a peer stops answering, the log says where the connection fails, once at the start of
// the outage and again at each repeat, and never two at once for one address. The only other
// line was that the request timed out, which does not say which stage did.
func TestAPeerThatStopsAnsweringIsDiagnosedAtTheStartAndAtEachRepeat(t *testing.T) {
	f, logged := withFakeOverlay(t)
	d := &Daemon{}
	const addr = "tc://SOMEADDRESS"
	fail := func() { d.markPeerUnreachable(addr, errors.New("context deadline exceeded")); d.background.Wait() }

	fail()
	if n, _ := f.counts(); n != 1 || !strings.Contains(logged.String(), "why it cannot be reached: FAKE-DIAGNOSIS of SOMEADDRESS") {
		t.Fatalf("the first failure was diagnosed %d times: %s", n, logged.String())
	}
	fail()
	fail()
	if n, _ := f.counts(); n != 1 {
		t.Errorf("a failure within the repeat interval was diagnosed again: %d", n)
	}
	d.peerMu.Lock()
	d.peerStateLocked(addr).reported = time.Now().Add(-downRepeat - time.Minute)
	d.peerMu.Unlock()
	fail()
	if n, _ := f.counts(); n != 2 {
		t.Errorf("a repeat of the outage was diagnosed %d times in all, want 2", n)
	}

	// a diagnosis that is still running is not started again
	f.mu.Lock()
	f.hold = make(chan struct{})
	f.mu.Unlock()
	d.peerMu.Lock()
	d.peerStateLocked(addr).reported = time.Now().Add(-downRepeat - time.Minute)
	d.peerMu.Unlock()
	d.markPeerUnreachable(addr, errors.New("x"))
	d.peerMu.Lock()
	d.peerStateLocked(addr).reported = time.Now().Add(-downRepeat - time.Minute)
	d.peerMu.Unlock()
	d.markPeerUnreachable(addr, errors.New("x"))
	time.Sleep(50 * time.Millisecond)
	if n, _ := f.counts(); n != 3 {
		t.Errorf("a second diagnosis started for an address with one running: %d", n)
	}
	close(f.hold)
	d.background.Wait()
}

// A connection is built once and kept, so one that is stuck fails for as long as the daemon
// runs. After a few minutes of failures it is discarded, at most every few minutes.
func TestAConnectionHeldToAPeerThatKeepsFailingIsDiscarded(t *testing.T) {
	f, logged := withFakeOverlay(t)
	d := &Daemon{}
	const addr = "tc://SOMEADDRESS"
	failFor := func(down time.Duration) {
		d.peerMu.Lock()
		d.peerStateLocked(addr).downSince = time.Now().Add(-down)
		d.peerMu.Unlock()
		d.markPeerUnreachable(addr, errors.New("e"))
		d.background.Wait()
	}
	failFor(0) // the first failure starts the outage
	failFor(2 * time.Minute)
	if _, r := f.counts(); r != 0 {
		t.Fatalf("the connection was discarded after two minutes of failures: %d", r)
	}
	failFor(4 * time.Minute)
	if _, r := f.counts(); r != 1 || !strings.Contains(logged.String(), "discarded the connection held to it after 4m") {
		t.Fatalf("after four minutes the connection was discarded %d times: %s", r, logged.String())
	}
	failFor(4*time.Minute + 30*time.Second)
	if _, r := f.counts(); r != 1 {
		t.Errorf("the connection was discarded again within five minutes: %d", r)
	}
	d.peerMu.Lock()
	d.peerStateLocked(addr).rebuilt = time.Now().Add(-rebuildEvery - time.Minute)
	d.peerMu.Unlock()
	failFor(10 * time.Minute)
	if _, r := f.counts(); r != 2 {
		t.Errorf("a connection still failing after the interval was discarded %d times in all, want 2", r)
	}
}

// A plain TCP address holds nothing to discard and has no diagnosis, and costs nothing.
func TestAPlainTCPAddressIsNeitherDiagnosedNorDiscarded(t *testing.T) {
	f, _ := withFakeOverlay(t)
	d := &Daemon{}
	d.markPeerUnreachable("127.0.0.1:59999", errors.New("refused"))
	d.peerMu.Lock()
	d.peerStateLocked("127.0.0.1:59999").downSince = time.Now().Add(-time.Hour)
	d.peerStateLocked("127.0.0.1:59999").reported = time.Now().Add(-time.Hour)
	d.peerMu.Unlock()
	d.markPeerUnreachable("127.0.0.1:59999", errors.New("refused"))
	d.background.Wait()
	if n, r := f.counts(); n != 0 || r != 0 {
		t.Errorf("a TCP address was diagnosed %d times and discarded %d", n, r)
	}
}

func TestADiagnosisReadsTheThreePingsAsTheyAre(t *testing.T) {
	failed := pingOutcome{Took: 10 * time.Second, Err: errors.New("context deadline exceeded")}
	ok := pingOutcome{Took: 120 * time.Millisecond}
	for _, c := range []struct {
		name        string
		held, fresh pingOutcome
		wantReading string
	}{
		{"both fail", failed, failed, "does not answer through that relay"},
		{"only the held one fails", failed, ok, "the held one is stuck and is replaced"},
		{"only the fresh one fails", ok, failed, "unusual"},
		{"both answer", ok, ok, "the failure is above it"},
	} {
		text := tailcatDiagnosis{Region: "relay region 1 (nyc)", Held: c.held, Fresh: c.fresh, Path: "direct, to 1.2.3.4:5 in 12ms"}.describe()
		for _, want := range []string{"relay region 1 (nyc)", "Relay ping, connection held by this daemon", "Relay ping, a new connection", "Path: direct, to 1.2.3.4:5", "Reading: ", c.wantReading} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: %q lacks %q", c.name, text, want)
			}
		}
	}
	if !strings.Contains(tailcatDiagnosis{Held: failed, Fresh: failed}.describe(), "failed after 10s (context deadline exceeded)") {
		t.Error("a failed ping does not say how long it took and why")
	}
}

func TestTheOverlayDialerDiscardsAHeldConnectionOnce(t *testing.T) {
	d := &tailcatDialer{clients: map[string]*tailcat.Client{"held": tailcat.NewClient("held")}}
	if !d.Reset(Endpoint{Scheme: schemeTailcat, Value: "held"}) {
		t.Error("a held connection was not discarded")
	}
	if d.Reset(Endpoint{Scheme: schemeTailcat, Value: "held"}) {
		t.Error("a connection already discarded was reported discarded again")
	}
	if d.Reset(Endpoint{Scheme: schemeTailcat, Value: "never-held"}) {
		t.Error("an address nothing was held for was reported discarded")
	}
}
