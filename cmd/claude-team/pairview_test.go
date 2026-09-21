package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Every pairing must get its own address. This is not tidiness: a page left open
// from an earlier attempt must never quietly become a different pairing, because
// the whole security property is that the person knows which key they vouched for.
// It is also what makes a second pairing visible -- a fresh address is a fresh tab,
// which the browser raises, rather than a silent rewrite of a background one.
func TestEachPairingGetsItsOwnURL(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t).PeerID

	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id := d.newPairing(peer, "", true)
		if id == "" {
			t.Fatal("newPairing returned no id")
		}
		if seen[id] {
			t.Fatalf("pairing id repeated after %d pairings: %q", i, id)
		}
		seen[id] = true
	}
}

func TestPairingResolvesToItsPeer(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t).PeerID

	id := d.newPairing(peer, "", true)
	got, ok := d.peerForPairing(id)
	if !ok || got != peer {
		t.Fatalf("peerForPairing(%q) = %q,%v; want %q,true", id, got, ok, peer)
	}

	if _, ok := d.peerForPairing("never-issued"); ok {
		t.Fatal("an id that was never issued resolved to a peer")
	}
}

// A link that outlives its ceremony must stop working. Otherwise a stale tab could
// begin a verification the person is no longer present for.
func TestExpiredPairingIsRefused(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t).PeerID
	id := d.newPairing(peer, "", true)

	d.pairs.mu.Lock()
	d.pairs.byID[id].created = time.Now().Add(-pairTTL - time.Minute)
	d.pairs.mu.Unlock()

	if _, ok := d.peerForPairing(id); ok {
		t.Fatal("an expired pairing still resolved")
	}
}

func TestPairPageRendersAndExpires(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t).PeerID
	id := d.newPairing(peer, "", true)

	rec := httptest.NewRecorder()
	d.handlePairPage(rec, httptest.NewRequest(http.MethodGet, "/pair/"+id, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("page returned %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, id) {
		t.Fatal("page does not carry its own pairing id, so it cannot start the exchange")
	}
	if !strings.Contains(body, PeerName(peer)) {
		t.Fatal("page does not name the peer being vouched for")
	}
	// The page talks to our own daemon and must present the guard header, or the
	// ceremony silently stops working the moment D-087 is enforced.
	if !strings.Contains(body, localGuardHeader) {
		t.Fatalf("page does not send %s; its requests would be refused", localGuardHeader)
	}

	// An unknown link explains itself rather than showing a browser error for a
	// URL this program handed out minutes earlier.
	rec = httptest.NewRecorder()
	d.handlePairPage(rec, httptest.NewRequest(http.MethodGet, "/pair/not-a-real-id", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown pairing returned %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "expired") {
		t.Fatal("unknown pairing did not explain itself")
	}
}

// The page names a pairing, never a peer, so a page cannot start an exchange for
// an arbitrary identifier even if it reached the endpoint.
func TestVerifyStartRefusesAnExpiredPairing(t *testing.T) {
	d, _ := testDaemon(t)
	if _, ok := d.peerFromRequest("", "no-such-pairing"); ok {
		t.Fatal("an unknown pairing id resolved; the page could name any peer")
	}
	// The CLI's shape still works: it names the peer directly.
	peer := testIdentity(t).PeerID
	if got, ok := d.peerFromRequest(peer, ""); !ok || got != peer {
		t.Fatal("naming a peer directly stopped working; the terminal path is broken")
	}
}

// The warning this replaces asked whether the BIND address was loopback, which it
// is by default and stays even when tailcat has negotiated a routable endpoint. So
// it fired on every pairing string ever printed, including every working one. These
// assert it now fires on exactly the strings a colleague cannot use.
func TestPairingReachabilityWarnsOnlyWhenItShould(t *testing.T) {
	t.Setenv("CLAUDE_TEAM_HOME", t.TempDir())

	// The ordinary case. A false warning here is the regression being guarded
	// against: it trains somebody to ignore the true one.
	if ok, why := pairingReachable("tc://tcpSOMELONGNEGOTIATEDVALUE"); !ok {
		t.Fatalf("a negotiated tailcat endpoint warned: %q", why)
	}
	if ok, why := pairingReachable("192.168.86.31:4783"); !ok {
		t.Fatalf("a routable address warned: %q", why)
	}
	if ok, why := pairingReachable("198.51.100.7:4783"); !ok {
		t.Fatalf("a public address warned: %q", why)
	}

	// Loopback in any spelling is unusable by somebody else.
	for _, bad := range []string{"127.0.0.1:4783", "localhost:4783", "[::1]:4783"} {
		if ok, _ := pairingReachable(bad); ok {
			t.Fatalf("%s was reported as reachable by a colleague", bad)
		}
	}

	// Unparseable is not reachable, and must not be reported as fine.
	if ok, _ := pairingReachable("not an endpoint at all"); ok {
		t.Fatal("an unparseable endpoint was reported as reachable")
	}
}

// "No daemon has run" and "a daemon ran and has no route out" need different
// answers: one is fixed by starting a session, the other by setting an address.
func TestPairingReachabilityDistinguishesItsTwoCauses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CLAUDE_TEAM_HOME", home)

	_, guessed := pairingReachable("127.0.0.1:4783")
	if !strings.Contains(guessed, "no daemon has published") {
		t.Fatalf("with nothing recorded, the advice was %q; it should say no daemon has published one", guessed)
	}

	if err := recordEndpoint("127.0.0.1:4783"); err != nil {
		t.Fatal(err)
	}
	_, noRoute := pairingReachable("127.0.0.1:4783")
	if !strings.Contains(noRoute, "CLAUDE_TEAM_PEER_ADDR") {
		t.Fatalf("with a daemon-recorded loopback address, the advice was %q; it should name the setting that fixes it", noRoute)
	}
	if guessed == noRoute {
		t.Fatal("both causes give the same advice, so one of them sends the person somewhere useless")
	}
}

// confirm drives the confirm step the way the page does.
func confirm(t *testing.T, d *Daemon, pairID string, matched bool) {
	t.Helper()
	body, _ := json.Marshal(verifyConfirmRequest{PairID: pairID, Matched: matched})
	rec := httptest.NewRecorder()
	d.handleVerifyConfirm(rec, httptest.NewRequest(http.MethodPost, "/verify/confirm", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm returned %d: %s", rec.Code, rec.Body.String())
	}
}

func startPairing(t *testing.T, d *Daemon, peer, name string) string {
	t.Helper()
	body, _ := json.Marshal(pairNewRequest{Peer: peer, Name: name})
	rec := httptest.NewRecorder()
	d.handlePairNew(rec, httptest.NewRequest(http.MethodPost, "/pair/new", bytes.NewReader(body)))
	var res pairNewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("pair/new: %v", err)
	}
	if res.Error != "" {
		t.Fatalf("pair/new refused: %s", res.Error)
	}
	return pairIDFromURL(res.URL)
}

func peerLabel(t *testing.T, d *Daemon, peerID string) (string, bool) {
	t.Helper()
	known, err := d.members.KnownPeers()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range known {
		if p.PeerID == peerID {
			return p.Name, true
		}
	}
	return "", false
}

// A flow with two interactions has to choose what an unfinished second one means,
// and these three endings are genuinely different. Abandoning is not mismatching,
// and neither used to be distinguishable from the other (D-093).
func TestTheThreeEndingsOfAPairing(t *testing.T) {
	t.Run("abandoned leaves a nameless, unverified row", func(t *testing.T) {
		d, _ := testDaemon(t)
		peer := testIdentity(t).PeerID
		startPairing(t, d, peer, "alice")

		label, present := peerLabel(t, d, peer)
		if !present {
			t.Fatal("the key was not recorded, so an inbound exchange has nothing to reach")
		}
		if label == "alice" {
			t.Error("the chosen name was written before the words were compared")
		}
		if label != PeerName(peer) {
			t.Errorf("label is %q, want the derived placeholder %q", label, PeerName(peer))
		}
		if d.members.IsVerified(peer) {
			t.Error("an abandoned pairing recorded a verification")
		}
	})

	t.Run("matched writes the name and the verification", func(t *testing.T) {
		d, _ := testDaemon(t)
		peer := testIdentity(t).PeerID
		id := startPairing(t, d, peer, "alice")
		confirm(t, d, id, true)

		if !d.members.IsVerified(peer) {
			t.Error("a matched comparison did not record the verification")
		}
		if label, _ := peerLabel(t, d, peer); label != "alice" {
			t.Errorf("label is %q, want alice", label)
		}
	})

	t.Run("differed removes what the pairing created", func(t *testing.T) {
		d, _ := testDaemon(t)
		peer := testIdentity(t).PeerID
		id := startPairing(t, d, peer, "alice")
		confirm(t, d, id, false)

		if _, present := peerLabel(t, d, peer); present {
			t.Error("an intercepted key stayed on the list, wearing the name meant for somebody else")
		}
		// And the name is free again, so pairing with the real person is not
		// blocked by the attempt that detected the attack.
		if err := d.members.NameFree("alice", "ed25519:someone-else"); err != nil {
			t.Errorf("the name is still taken after a mismatch: %v", err)
		}
	})

	t.Run("differed does NOT discard a peer it did not create", func(t *testing.T) {
		d, _ := testDaemon(t)
		peer := testIdentity(t).PeerID
		// A colleague of long standing, already known and named.
		if err := d.members.Allow(peer, "bob"); err != nil {
			t.Fatal(err)
		}
		id := startPairing(t, d, peer, "")
		confirm(t, d, id, false)

		label, present := peerLabel(t, d, peer)
		if !present {
			t.Fatal("re-verifying an existing peer and seeing different words discarded them; " +
				"that is an alarm about a relationship, not a reason to end it")
		}
		if label != "bob" {
			t.Errorf("label became %q, want bob", label)
		}
	})
}

// A label already meaning another key must be refused BEFORE the ceremony, or the
// clash surfaces once the words have matched -- leaving a verified peer and an
// unusable name, which is one more thing for somebody to finish.
func TestATakenNameIsRefusedBeforeTheCeremony(t *testing.T) {
	d, _ := testDaemon(t)
	first := testIdentity(t).PeerID
	second := testIdentity(t).PeerID
	if err := d.members.Allow(first, "alice"); err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(pairNewRequest{Peer: second, Name: "alice"})
	rec := httptest.NewRecorder()
	d.handlePairNew(rec, httptest.NewRequest(http.MethodPost, "/pair/new", bytes.NewReader(body)))
	var res pairNewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Error == "" {
		t.Fatal("a name already meaning another key was accepted")
	}
	if _, present := peerLabel(t, d, second); present {
		t.Error("the refused pairing recorded the peer anyway")
	}
}
