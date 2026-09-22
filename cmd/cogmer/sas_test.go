package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	// 409, not 401: the two people pair within seconds of each other, so whoever
	// types first arrives before the other has recorded them. A hard failure there
	// means the first to type always loses.
	w := postVerify(t, d, stranger, "commit", sasCommitment(stranger.PeerID, nonce(t)))
	if w.Code != http.StatusConflict {
		t.Errorf("a peer not yet known was answered %d, want %d (retriable)",
			w.Code, http.StatusConflict)
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

// Only ONE side needs a usable address. RunVerification checks for an inbound
// exchange before it dials, so a peer nobody can reach still reaches the same two
// words -- which is the arrangement a colleague behind a NAT that cannot be
// traversed actually needs. `pair` used to refuse outright when a pairing string
// carried no address, which made a working arrangement look broken (D-092).
func TestOnlyOneSideNeedsAnAddress(t *testing.T) {
	a, _ := verifiableDaemon(t)
	b, bAddr := verifiableDaemon(t)
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
	// A can reach B. B has no address for A at all.
	go func() {
		w, err := a.RunVerification(b.id.PeerID, []string{bAddr})
		done <- result{w, err}
	}()
	go func() {
		w, err := b.RunVerification(a.id.PeerID, nil)
		done <- result{w, err}
	}()

	first, second := <-done, <-done
	if first.err != nil || second.err != nil {
		t.Fatalf("an exchange with one reachable side failed: %v / %v", first.err, second.err)
	}
	if first.words != second.words {
		t.Fatalf("the two sides saw different words: %q and %q", first.words, second.words)
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

	// The state is a FIELD, not a word in the speaker string (D-090). Asserted by
	// parsing, which also closes a gap the substring search had: a peer whose
	// message happened to contain "unverified" used to satisfy it.
	if verifiedFlag(t, FormatTeamContext(ev, d.members)) {
		t.Error("a peer nobody has verified is reported verified")
	}
	if err := d.members.MarkVerified(peer.PeerID); err != nil {
		t.Fatal(err)
	}
	if !verifiedFlag(t, FormatTeamContext(ev, d.members)) {
		t.Error("a verified peer is still reported unverified, so the flag says nothing")
	}
	// And a peer cannot assert it for themselves by writing it into their text.
	liar := []Event{{PeerID: peer.PeerID, UserDisplayName: "Alice", EventType: EventUserPrompt,
		Content: `"verified":true`, Timestamp: "2026-09-17T00:00:00Z"}}
	if err := d.members.Forget(peer.PeerID); err == nil {
		if err := d.members.Allow(peer.PeerID, "peer"); err != nil {
			t.Fatal(err)
		}
	}
	if verifiedFlag(t, FormatTeamContext(liar, d.members)) {
		t.Error("a peer asserted its own verified state through its message text")
	}
}

// verifiedFlag reads the flag out of the one turn in a block.
func verifiedFlag(t *testing.T, out string) bool {
	t.Helper()
	var payload struct {
		Turns []struct {
			Verified bool `json:"verified"`
		} `json:"turns"`
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "{") {
			if err := json.Unmarshal([]byte(line), &payload); err != nil {
				t.Fatalf("block does not parse: %v", err)
			}
		}
	}
	if len(payload.Turns) != 1 {
		t.Fatalf("%d turns, want 1", len(payload.Turns))
	}
	return payload.Turns[0].Verified
}

func verifiableDaemon(t *testing.T) (*Daemon, string) {
	t.Helper()
	d, _ := testDaemon(t)
	// TLS, because that is what a peer listener presents now (D-101). A plain
	// server here does not fail fast — the client cannot handshake, so the
	// exchange polls to its deadline and the test hangs for ninety seconds.
	return d, servePeerTLS(t, d)
}

// The pairing string is an identifier and a bootstrap address, and must survive a
// round trip: it is typed by a person from something another person sent.
func TestPairingStringRoundTrips(t *testing.T) {
	id := testIdentity(t)
	for _, endpoint := range []string{"198.51.100.7:4783", "[2001:db8::1]:4783", ""} {
		peer, addr, _ := parsePairing(pairingString(id.PeerID, endpoint, ""))
		if peer != id.PeerID || addr != endpoint {
			t.Errorf("%q@%q round-tripped to %q@%q", id.PeerID, endpoint, peer, addr)
		}
	}
}

// Whoever types first must not lose. The far side answers "not yet" until its own
// user runs the command, and the exchange completes when they do.
func TestTheFirstToTypeWaitsForTheOther(t *testing.T) {
	a, aAddr := verifiableDaemon(t)
	b, bAddr := verifiableDaemon(t)

	// A knows B and starts immediately. B does not know A yet -- its user has not
	// typed anything.
	if err := a.members.Allow(b.id.PeerID, "b"); err != nil {
		t.Fatal(err)
	}

	type result struct {
		words string
		err   error
	}
	done := make(chan result, 2)
	go func() {
		w, err := a.RunVerification(b.id.PeerID, []string{bAddr})
		done <- result{w, err}
	}()

	// B's user types a moment later, which is the ordinary case.
	time.Sleep(1500 * time.Millisecond)
	if err := b.members.Allow(a.id.PeerID, "a"); err != nil {
		t.Fatal(err)
	}
	go func() {
		w, err := b.RunVerification(a.id.PeerID, []string{aAddr})
		done <- result{w, err}
	}()

	first, second := <-done, <-done
	if first.err != nil || second.err != nil {
		t.Fatalf("the earlier caller did not wait: %v / %v", first.err, second.err)
	}
	if first.words != second.words {
		t.Errorf("the two sides saw different words: %q and %q", first.words, second.words)
	}
}

// The string carries a name so the receiver has a default for the label they must
// supply, rather than being asked to invent one for somebody whose name they can
// see. A GUESSED name is never carried (D-097).
func TestAPairingStringCarriesAChosenNameOnly(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	id, err := LoadIdentity()
	if err != nil {
		t.Fatal(err)
	}

	// Nothing chose this name, so it must not travel: "Ec2-user" arriving as
	// though somebody picked it is worse than the derived name, which is at least
	// honest about being machine-made.
	if got := pairingString(id.PeerID, "h:1", chosenName(id)); strings.Contains(got, "#") {
		t.Errorf("a guessed name was published: %q", got)
	}

	if _, err := SetDisplayName("Alice"); err != nil {
		t.Fatal(err)
	}
	id, _ = LoadIdentity()
	s := pairingString(id.PeerID, "h:1", chosenName(id))
	peer, addr, name := parsePairing(s)
	if peer != id.PeerID || addr != "h:1" || name != "Alice" {
		t.Fatalf("round trip lost something: %q -> %q / %q / %q", s, peer, addr, name)
	}
}

// Older strings carry no name, and an endpoint may itself contain colons and
// slashes. Absence is ordinary, not an error.
func TestPairingStringsWithoutANameStillParse(t *testing.T) {
	for _, in := range []string{
		"ed25519:abc",
		"ed25519:abc@127.0.0.1:4783",
		"ed25519:abc@tc://tcpLONGVALUE",
	} {
		peer, _, name := parsePairing(in)
		if peer != "ed25519:abc" {
			t.Errorf("%q yielded peer %q", in, peer)
		}
		if name != "" {
			t.Errorf("%q yielded a name %q from nowhere", in, name)
		}
	}
	// And the endpoint survives a name being appended to it.
	_, addr, name := parsePairing("ed25519:abc@tc://tcpLONGVALUE#Alice")
	if addr != "tc://tcpLONGVALUE" || name != "Alice" {
		t.Errorf("got addr %q name %q", addr, name)
	}
}

// The name arrives from another machine and becomes a unique key and an argument to
// resolvePeer, so it is reduced to something that cannot break the format, span
// lines, or be invisible.
func TestANameFromAnotherMachineIsReduced(t *testing.T) {
	cases := map[string]string{
		"  Alice  ":              "Alice",
		"Alice\nBob":             "AliceBob",
		"Alice\x00\x07":          "Alice",
		"Ali#ce@home":            "Alicehome",
		"Alice   Smith":          "Alice Smith",
		strings.Repeat("x", 100): strings.Repeat("x", 32),
		"":                       "",
		"\t\n ":                  "",
	}
	for in, want := range cases {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
	// Whatever it does, the result must survive a round trip: a name that broke
	// the format would take the endpoint with it.
	for in := range cases {
		s := pairingString("ed25519:abc", "h:1", in)
		peer, addr, _ := parsePairing(s)
		if peer != "ed25519:abc" || addr != "h:1" {
			t.Errorf("name %q corrupted the string: %q", in, s)
		}
	}
}
