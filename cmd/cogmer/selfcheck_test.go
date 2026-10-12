package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tailscale/tailcat"
)

// A pairing string carries an overlay address whatever state the daemon is in, so
// the string was reported reachable by its shape, and a daemon that never attached
// to its relay handed out an address that led nowhere. Whether it is reachable is
// asked of the running daemon, and these tests hold the answers a person is given
// (D-196).

var healthyOverlay = SelfCheck{Overlay: true, Relay: "sfo", Reached: true, TookMs: 294}

// overlayAnswers makes the daemon answer a self-check as given, or not answer.
func overlayAnswers(t *testing.T, sc SelfCheck, up bool) {
	t.Helper()
	old := queryDaemonSelf
	queryDaemonSelf = func() (SelfCheck, bool) { return sc, up }
	t.Cleanup(func() { queryDaemonSelf = old })
}

func TestAnOverlayAddressIsReachableOnlyWhenTheDaemonCanBeReachedThroughItsRelay(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	const ep = "tc://tcpSOMELONGNEGOTIATEDVALUE"
	cases := []struct {
		name   string
		sc     SelfCheck
		up     bool
		ok     bool
		inWhy  []string
		inNote []string
	}{
		{"healthy", healthyOverlay, true, true, nil, []string{"Checked just now", "relay at sfo", "294ms"}},
		{"no daemon", SelfCheck{}, false, false, []string{"daemon is not running", "/cogmer:self-restart"}, nil},
		{"daemon without an overlay", SelfCheck{}, true, false, []string{"not running the relay connection", "leads nowhere"}, nil},
		{"not attached to a relay", SelfCheck{Overlay: true}, true, false, []string{"not attached to a relay", "/cogmer:self-restart", "daemon.log"}, nil},
		{"attached, but not reachable from outside", SelfCheck{Overlay: true, Relay: "sfo", Error: "context deadline exceeded"}, true, false,
			[]string{"attached to the relay (sfo)", "did not get through", "no answer", "/cogmer:self-restart"}, nil},
	}
	for _, c := range cases {
		overlayAnswers(t, c.sc, c.up)
		ok, why, note := checkReachable(ep)
		if ok != c.ok {
			t.Errorf("%s: reachable = %v, want %v (%q)", c.name, ok, c.ok, why)
		}
		for _, w := range c.inWhy {
			if !strings.Contains(why, w) {
				t.Errorf("%s: the reason lacks %q: %q", c.name, w, why)
			}
		}
		for _, w := range c.inNote {
			if !strings.Contains(note, w) {
				t.Errorf("%s: the note lacks %q: %q", c.name, w, note)
			}
		}
		if !c.ok && note != "" {
			t.Errorf("%s: a note is printed beside a string nobody can use: %q", c.name, note)
		}
	}
}

func TestNoStringIsPrintedForAnOverlayAddressNobodyCanReach(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	id := testIdentity(t)
	overlayAnswers(t, SelfCheck{Overlay: true}, true)
	var out bytes.Buffer
	writePairingInvitation(&out, id, "tc://SOMENEGOTIATEDVALUE")
	if strings.Contains(out.String(), id.PeerID) || !strings.Contains(out.String(), "no string to send yet") {
		t.Errorf("a string was printed for a daemon that is not attached to a relay:\n%s", out.String())
	}
	overlayAnswers(t, healthyOverlay, true)
	out.Reset()
	writePairingInvitation(&out, id, "tc://SOMENEGOTIATEDVALUE")
	if !strings.Contains(out.String(), id.PeerID) || !strings.Contains(out.String(), "Checked just now") {
		t.Errorf("a reachable string is not printed with the check:\n%s", out.String())
	}
}

// The daemon asks its overlay and keeps a pass for a short while, so a command run
// twice in a row opens one connection through the relay. A failure is asked again,
// because the next ask follows a fix.
func TestTheDaemonKeepsAPassAndAsksAgainAfterAFailure(t *testing.T) {
	d, _ := testDaemon(t)
	rec := httptest.NewRecorder()
	d.handleSelf(rec, httptest.NewRequest(http.MethodGet, "/self", nil))
	var none SelfCheck
	if err := json.Unmarshal(rec.Body.Bytes(), &none); err != nil || none.Overlay {
		t.Fatalf("a daemon with no overlay answered %s", rec.Body.String())
	}

	calls := 0
	results := []SelfCheck{{Overlay: true, Error: "not attached to a relay"}, healthyOverlay}
	d.selfCheck = func(context.Context) SelfCheck {
		r := results[calls]
		calls++
		return r
	}
	ask := func() SelfCheck {
		rec := httptest.NewRecorder()
		d.handleSelf(rec, httptest.NewRequest(http.MethodGet, "/self", nil))
		var sc SelfCheck
		if err := json.Unmarshal(rec.Body.Bytes(), &sc); err != nil {
			t.Fatal(err)
		}
		return sc
	}
	if got := ask(); got.Reached {
		t.Fatalf("a failed check read as a pass: %+v", got)
	}
	if got := ask(); !got.Reached || calls != 2 {
		t.Fatalf("a failure was kept instead of asked again: %+v after %d checks", got, calls)
	}
	if ask(); calls != 2 {
		t.Errorf("a pass was asked again at once, which opens a connection through the relay per command: %d checks", calls)
	}
	d.selfAt = time.Now().Add(-time.Minute)
	results = append(results, healthyOverlay)
	if ask(); calls != 3 {
		t.Errorf("a stale pass was not asked again: %d checks", calls)
	}
}

// What a colleague last asked of this machine says whether their daemon is running,
// which tells a person where to look when this machine cannot reach them.
func TestAColleagueWhoReachedUsIsSaidToHaveDone(t *testing.T) {
	d, _, _ := authDaemon(t)
	peer := testIdentity(t)
	if err := d.members.Allow(peer.PeerID, "vivid-gannet"); err != nil {
		t.Fatal(err)
	}
	if err := d.members.SetPeerEndpoint(peer.PeerID, "127.0.0.1:59998"); err != nil {
		t.Fatal(err)
	}
	find := func() *uiPeer {
		for _, p := range d.peerStatus() {
			if p.PeerID == peer.PeerID {
				p := p
				return &p
			}
		}
		return nil
	}
	if p := find(); p == nil || p.HeardFrom != "" {
		t.Fatalf("a colleague who has not made a request is described as having: %+v", p)
	}
	d.markPeerHeardFrom(peer.PeerID)
	d.peerMu.Lock()
	d.heard[peer.PeerID] = time.Now().Add(-3 * time.Minute)
	d.peerMu.Unlock()
	if p := find(); p == nil || p.HeardFrom != "3m" {
		t.Fatalf("a request three minutes ago reads as %+v", p)
	}

	line := reachLine(&uiPeer{Since: "never reached", Trying: "12m", Error: "deadline exceeded", HeardFrom: "3m"}, true)
	if !strings.Contains(line, "last reached this machine 3m ago") {
		t.Errorf("the line does not say they reached this machine: %q", line)
	}
	line = reachLine(&uiPeer{Since: "never reached", Trying: "12m", Error: "deadline exceeded"}, true)
	if !strings.Contains(line, "not reached this machine since its daemon started") {
		t.Errorf("the line does not say they have not: %q", line)
	}
}

// Only a request that proves its key counts. One that names a peer and cannot sign
// as it names nobody.
func TestOnlyAnAuthenticatedRequestCountsAsHearingFromAColleague(t *testing.T) {
	d, guest, room := authDaemon(t)
	send := func(req syncRequest) int {
		body, _ := json.Marshal(req)
		rec := httptest.NewRecorder()
		d.handleSync(rec, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body)))
		return rec.Code
	}

	forged := credentials(t, guest, room.RoomID)
	forged.Signature = ""
	if code := send(forged); code != http.StatusUnauthorized {
		t.Fatalf("an unsigned request got %d", code)
	}
	d.peerMu.Lock()
	_, heard := d.heard[guest.PeerID]
	d.peerMu.Unlock()
	if heard {
		t.Error("an unsigned request that named a colleague counted as hearing from them")
	}

	if code := send(credentials(t, guest, room.RoomID)); code != http.StatusOK {
		t.Fatalf("a genuine request got %d", code)
	}
	d.peerMu.Lock()
	_, heard = d.heard[guest.PeerID]
	d.peerMu.Unlock()
	if !heard {
		t.Error("an authenticated request was not recorded")
	}
}

// The check against a real overlay server and the real relay: a live server is
// reached through the relay, and an address nobody serves is not. Skipped where the
// relay cannot be reached, since a test that needs a network says so and does not
// fail for want of one.
func TestTheRealOverlayCheckPassesForALiveServerAndFailsForADeadOne(t *testing.T) {
	if testing.Short() {
		t.Skip("uses the relay")
	}
	if c, err := net.DialTimeout("tcp", "tailcat.dev:443", 3*time.Second); err != nil {
		t.Skipf("cannot reach tailcat.dev, so the relay is out of reach: %v", err)
	} else {
		c.Close()
	}

	start := func() (string, *tailcat.Server, func()) {
		t.Setenv("COGMER_HOME", t.TempDir())
		l, ep, srv, err := StartTailcat()
		if err != nil {
			t.Skipf("the overlay could not start here: %v", err)
		}
		return ep, srv, func() { l.Close(); srv.Close() }
	}

	liveEndpoint, live, closeLive := start()
	defer closeLive()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	got := newSelfCheck(live, liveEndpoint)(ctx)
	if !got.Overlay || got.Relay == "" || !got.Reached {
		t.Fatalf("a live overlay server was not reached through the relay: %+v", got)
	}

	deadEndpoint, _, closeDead := start()
	closeDead()
	_, otherSrv, closeOther := start()
	defer closeOther()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel2()
	got = newSelfCheck(otherSrv, deadEndpoint)(ctx2)
	if got.Reached || got.Error == "" {
		t.Errorf("an address nobody serves was reported reached: %+v", got)
	}
}
