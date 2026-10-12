package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A colleague this machine cannot reach looks the same as a quiet room, and the
// only trace used to be a line in a log nobody reads. These tests hold the words a
// person is told, and the facts they are told them from.

func TestReachLineSaysWhetherAColleagueCanBeReached(t *testing.T) {
	cases := []struct {
		name string
		p    *uiPeer
		up   bool
		want []string
	}{
		{"answering", &uiPeer{Online: true}, true, []string{"reachable"}},
		{"never reached, still trying", &uiPeer{Since: "never reached", Trying: "12m", Error: `Post "https://peer/sync": tailcat: context deadline exceeded`}, true,
			[]string{"NOT REACHABLE", "tried for 12m", "never reached them", "no answer"}},
		{"was reached, now gone", &uiPeer{Since: "9m ago", Trying: "9m", Error: "connection refused"}, true,
			[]string{"NOT REACHABLE for 9m", "last reached 9m ago", "refused"}},
		{"first attempt not finished", &uiPeer{Since: "never reached"}, true, []string{"not reached yet"}},
		{"no address", nil, true, []string{"no address is recorded"}},
		{"no daemon", nil, false, []string{"unknown", "daemon is not running"}},
	}
	for _, c := range cases {
		got := reachLine(c.p, c.up)
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: %q lacks %q", c.name, got, w)
			}
		}
	}
	if got := reachLine(&uiPeer{Online: true}, true); strings.Contains(got, "NOT") {
		t.Errorf("a colleague who answers is described as unreachable: %q", got)
	}
}

func TestTheGuestListSaysWhoCannotBeReachedAndWhatThatMeans(t *testing.T) {
	rows := []guestRow{
		{Name: "wonka", ID: "ed25519:" + strings.Repeat("a", 43), Self: true},
		{Name: "vivid-gannet", ID: "ed25519:" + strings.Repeat("b", 43)},
		{Name: "dusky-heron", ID: "ed25519:" + strings.Repeat("c", 43)},
	}
	reach := []uiPeer{
		{PeerID: rows[1].ID, Since: "never reached", Trying: "12m", Error: "tailcat: context deadline exceeded"},
		{PeerID: rows[2].ID, Online: true},
	}
	var out bytes.Buffer
	renderGuests(&out, "moonlit-oasis", rows, reach, true)
	text := out.String()
	for _, want := range []string{"moonlit-oasis", "vivid-gannet", "NOT REACHABLE", "tried for 12m", "the room looks quiet", "/cogmer:self-status", "dusky-heron", "reachable"} {
		if !strings.Contains(text, want) {
			t.Errorf("the guest list lacks %q:\n%s", want, text)
		}
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if strings.Contains(l, "(you)") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "    ") && lines[i+1] != "" {
			t.Errorf("the list says whether this machine can reach itself: %q", lines[i+1])
		}
	}

	// Everyone answering: nothing to explain.
	out.Reset()
	renderGuests(&out, "moonlit-oasis", rows[:1], nil, true)
	renderGuests(&out, "r", []guestRow{rows[2]}, []uiPeer{{PeerID: rows[2].ID, Online: true}}, true)
	if strings.Contains(out.String(), "looks quiet") {
		t.Errorf("the explanation appears when nobody is unreachable:\n%s", out.String())
	}
}

// The daemon keeps what it knows per address, and a person is a name.
func TestAColleagueIsNamedAndTimedWhileUnreachable(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t)
	if err := d.members.Allow(peer.PeerID, "vivid-gannet"); err != nil {
		t.Fatal(err)
	}
	const addr = "127.0.0.1:59999"
	if err := d.members.SetPeerEndpoint(peer.PeerID, addr); err != nil {
		t.Fatal(err)
	}

	d.markPeerUnreachable(addr, errors.New("tailcat: context deadline exceeded"))
	d.peerMu.Lock()
	d.peerSeen[addr].downSince = time.Now().Add(-12 * time.Minute)
	d.peerMu.Unlock()

	var got *uiPeer
	for _, p := range d.peerStatus() {
		if p.PeerID == peer.PeerID {
			p := p
			got = &p
		}
	}
	if got == nil {
		t.Fatalf("the colleague is not among the peers: %+v", d.peerStatus())
	}
	if got.Name != "vivid-gannet" {
		t.Errorf("the colleague is named %q, not for the person", got.Name)
	}
	if got.Online || got.Trying != "12m" || !strings.Contains(got.Error, "deadline exceeded") {
		t.Errorf("unreachable for twelve minutes reads as %+v", got)
	}

	// The command asks the daemon over its own address.
	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/peers", nil))
	var body struct{ Peers []uiPeer }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Peers) == 0 || body.Peers[0].Trying != "12m" {
		t.Errorf("/peers answered %d %s", rec.Code, rec.Body.String())
	}

	// Answered five seconds ago and failed once since: still online, no outage.
	d.peerMu.Lock()
	d.peerSeen[addr].lastSeen = time.Now().Add(-5 * time.Second)
	d.peerSeen[addr].downSince = time.Now()
	d.peerMu.Unlock()
	for _, p := range d.peerStatus() {
		if p.PeerID == peer.PeerID && (!p.Online || p.Trying != "") {
			t.Errorf("one failed poll after an answer five seconds ago reads as an outage: %+v", p)
		}
	}

	// And it stops being said when they answer.
	d.markPeerSeen(addr)
	for _, p := range d.peerStatus() {
		if p.PeerID == peer.PeerID && (!p.Online || p.Trying != "" || p.Error != "") {
			t.Errorf("a colleague who answered is still described as unreachable: %+v", p)
		}
	}
}
