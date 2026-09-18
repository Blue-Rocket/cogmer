package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func nonce(t *testing.T) []byte {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

// Both sides compute the same words from the same exchange, whichever order they
// hold the two identities in. Without this the ceremony fails for everyone.
func TestSASIsSymmetric(t *testing.T) {
	a, b := testIdentity(t), testIdentity(t)
	na, nb := nonce(t), nonce(t)

	x := SAS(a.PeerID, na, b.PeerID, nb)
	y := SAS(b.PeerID, nb, a.PeerID, na)
	if x != y {
		t.Fatalf("the two sides computed different words: %q and %q", x, y)
	}
	if len(strings.Fields(x)) != sasWords {
		t.Errorf("got %q, want %d words", x, sasWords)
	}
}

// The property the whole scheme exists for. An attacker relaying between two
// people substitutes a key in each direction; each side then derives its words
// from a different pair of identities, and the two strings disagree. That
// disagreement is what the people hear.
func TestAnInterceptorProducesDifferentWords(t *testing.T) {
	alice, david, mallory := testIdentity(t), testIdentity(t), testIdentity(t)

	// David believes he is exchanging with Alice; he is exchanging with Mallory.
	nDavid, nToDavid := nonce(t), nonce(t)
	davidSees := SAS(david.PeerID, nDavid, mallory.PeerID, nToDavid)

	// Alice likewise.
	nAlice, nToAlice := nonce(t), nonce(t)
	aliceSees := SAS(mallory.PeerID, nToAlice, alice.PeerID, nAlice)

	if davidSees == aliceSees {
		t.Fatal("an interceptor produced matching words on both sides")
	}
}

// Freshness: the same two peers verifying twice must not produce the same words,
// or an attacker could replay a string overheard once.
func TestWordsDifferPerExchange(t *testing.T) {
	a, b := testIdentity(t), testIdentity(t)
	first := SAS(a.PeerID, nonce(t), b.PeerID, nonce(t))
	same := 0
	for i := 0; i < 8; i++ {
		if SAS(a.PeerID, nonce(t), b.PeerID, nonce(t)) == first {
			same++
		}
	}
	if same > 1 {
		t.Errorf("%d of 8 exchanges reproduced the same words", same)
	}
}

// Alternation is what makes a transposition detectable: the two positions draw on
// different vocabularies, so words said in the wrong order are not a valid pair.
func TestTheTwoWordsComeFromDifferentLists(t *testing.T) {
	even := map[string]bool{}
	for _, w := range sasEven {
		even[w] = true
	}
	for _, w := range sasOdd {
		if even[w] {
			t.Fatalf("%q is in both lists, so its position carries no information", w)
		}
	}
	for i := 0; i < 32; i++ {
		f := strings.Fields(SAS(testIdentity(t).PeerID, nonce(t), testIdentity(t).PeerID, nonce(t)))
		if !even[f[0]] {
			t.Errorf("first word %q is not from the even list", f[0])
		}
		if even[f[1]] {
			t.Errorf("second word %q is not from the odd list", f[1])
		}
	}
}

// The wordlists must be exactly a byte each, or the mapping from digest to words
// is not the one the security argument assumes.
func TestWordlistsAreWholeBytes(t *testing.T) {
	for name, list := range map[string][256]string{"even": sasEven, "odd": sasOdd} {
		seen := map[string]bool{}
		for i, w := range list {
			if w == "" {
				t.Fatalf("%s[%d] is empty", name, i)
			}
			if seen[w] {
				t.Errorf("%s contains %q twice, so two byte values render alike", name, w)
			}
			seen[w] = true
		}
	}
}

// --- the exchange ---

// A reveal that arrives before a commitment must be refused. This is the ordering
// the security rests on: accept a reveal from a peer that never committed and it
// may choose its nonce after seeing ours, which is the entire attack.
func TestRevealBeforeCommitIsRefused(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t)
	if err := d.members.Allow(peer.PeerID, "peer"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.startVerify(peer.PeerID); err != nil {
		t.Fatal(err)
	}

	w := postVerify(t, d, peer, "reveal", nonce(t))
	if w.Code != http.StatusBadRequest {
		t.Errorf("a reveal with no commitment was answered %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// And a reveal that does not open the commitment already held.
func TestARevealMustOpenItsCommitment(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t)
	if err := d.members.Allow(peer.PeerID, "peer"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.startVerify(peer.PeerID); err != nil {
		t.Fatal(err)
	}

	committed := nonce(t)
	if w := postVerify(t, d, peer, "commit", sasCommitment(peer.PeerID, committed)); w.Code != http.StatusOK {
		t.Fatalf("commit was refused: %d %s", w.Code, w.Body)
	}
	// Reveal a different nonce than the one committed to.
	if w := postVerify(t, d, peer, "reveal", nonce(t)); w.Code != http.StatusBadRequest {
		t.Errorf("a nonce that does not open the commitment was answered %d, want %d",
			w.Code, http.StatusBadRequest)
	}
	if w := postVerify(t, d, peer, "reveal", committed); w.Code != http.StatusOK {
		t.Errorf("the committed nonce was refused: %d %s", w.Code, w.Body)
	}
}

// No inbound surface (§12a leaves this open, and this closes it by construction):
// a peer whose user has not asked to verify answers 409 and displays nothing.
func TestVerificationCannotBeStartedByAPeer(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t)
	if err := d.members.Allow(peer.PeerID, "peer"); err != nil {
		t.Fatal(err)
	}
	// Known, correctly signed, and still refused: no session was asked for here.
	w := postVerify(t, d, peer, "commit", sasCommitment(peer.PeerID, nonce(t)))
	if w.Code != http.StatusConflict {
		t.Errorf("an unsolicited verification was answered %d, want %d", w.Code, http.StatusConflict)
	}
}

// A stranger has no key on file, so there is nothing to confirm.
func TestAnUnknownPeerCannotVerify(t *testing.T) {
	d, _ := testDaemon(t)
	stranger := testIdentity(t)
	if _, err := d.startVerify(stranger.PeerID); err != nil {
		t.Fatal(err)
	}
	w := postVerify(t, d, stranger, "commit", sasCommitment(stranger.PeerID, nonce(t)))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("a peer this machine does not know was answered %d, want %d",
			w.Code, http.StatusUnauthorized)
	}
}

// A message signed by some other key, claiming an identity it does not hold.
func TestVerificationMessagesMustBeSigned(t *testing.T) {
	d, _ := testDaemon(t)
	peer, impostor := testIdentity(t), testIdentity(t)
	if err := d.members.Allow(peer.PeerID, "peer"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.startVerify(peer.PeerID); err != nil {
		t.Fatal(err)
	}

	payload := sasCommitment(peer.PeerID, nonce(t))
	msg := verifyMessage{
		Step: "commit", PeerID: peer.PeerID,
		Payload:   b64(payload),
		Signature: b64(signVerify(impostor.private, "commit", peer.PeerID, payload)),
	}
	if w := post(t, d.handleVerify, msg); w.Code != http.StatusUnauthorized {
		t.Errorf("a message signed by another key was answered %d, want %d",
			w.Code, http.StatusUnauthorized)
	}
}

func postVerify(t *testing.T, d *Daemon, from *Identity, step string, payload []byte) *httptest.ResponseRecorder {
	t.Helper()
	return post(t, d.handleVerify, verifyMessage{
		Step: step, PeerID: from.PeerID,
		Payload:   b64(payload),
		Signature: b64(signVerify(from.private, step, from.PeerID, payload)),
	})
}

func post(t *testing.T, h http.HandlerFunc, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("POST", "/verify", strings.NewReader(string(raw))))
	return w
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// The whole ceremony over real HTTP: two daemons, each told by its own user to
// verify the other, arriving at the same two words.
func TestTwoDaemonsReachTheSameWords(t *testing.T) {
	a, aAddr := verifiableDaemon(t)
	b, bAddr := verifiableDaemon(t)

	// Each knows the other -- verification confirms a key already on file.
	if err := a.members.Allow(b.id.PeerID, "b"); err != nil {
		t.Fatal(err)
	}
	if err := b.members.Allow(a.id.PeerID, "a"); err != nil {
		t.Fatal(err)
	}

	type result struct {
		words string
		err   error
	}
	done := make(chan result, 2)
	// Both sides run at once, which is what the two people do on the call.
	go func() {
		w, err := a.RunVerification(b.id.PeerID, []string{bAddr})
		done <- result{w, err}
	}()
	go func() {
		w, err := b.RunVerification(a.id.PeerID, []string{aAddr})
		done <- result{w, err}
	}()

	first, second := <-done, <-done
	if first.err != nil || second.err != nil {
		t.Fatalf("exchange failed: %v / %v", first.err, second.err)
	}
	if first.words != second.words {
		t.Fatalf("the two sides saw different words: %q and %q", first.words, second.words)
	}
	if len(strings.Fields(first.words)) != sasWords {
		t.Errorf("got %q, want %d words", first.words, sasWords)
	}

	// Verification is not admission and is recorded separately.
	if a.members.IsVerified(b.id.PeerID) {
		t.Error("the exchange recorded a verification on its own; only a person may")
	}
	if err := a.members.MarkVerified(b.id.PeerID); err != nil {
		t.Fatal(err)
	}
	if !a.members.IsVerified(b.id.PeerID) {
		t.Error("a confirmed verification was not recorded")
	}
}

// The marker §25 requires inside injected text must be a fact, not a constant.
func TestTheUnverifiedMarkerTracksVerification(t *testing.T) {
	d, _ := testDaemon(t)
	peer := testIdentity(t)
	if err := d.members.Allow(peer.PeerID, "peer"); err != nil {
		t.Fatal(err)
	}
	ev := []Event{{PeerID: peer.PeerID, UserDisplayName: "Alice",
		EventType: EventUserPrompt, Content: "hello", Timestamp: "2026-09-17T00:00:00Z"}}

	if out := FormatTeamContext(ev, d.members.IsVerified); !strings.Contains(out, "unverified") {
		t.Error("a peer nobody has verified is not marked unverified")
	}
	if err := d.members.MarkVerified(peer.PeerID); err != nil {
		t.Fatal(err)
	}
	if out := FormatTeamContext(ev, d.members.IsVerified); strings.Contains(out, "unverified") {
		t.Error("a verified peer is still marked unverified, so the marker says nothing")
	}
}

func verifiableDaemon(t *testing.T) (*Daemon, string) {
	t.Helper()
	d, _ := testDaemon(t)
	srv := httptest.NewServer(d.PeerRoutes())
	t.Cleanup(srv.Close)
	return d, strings.TrimPrefix(srv.URL, "http://")
}
