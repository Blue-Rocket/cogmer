package main

import (
	"crypto/ed25519"
	"encoding/json"
	"strings"
	"testing"
)

func testIdentity(t *testing.T) *Identity {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	id := &Identity{PeerID: PeerIDFromPublic(pub), UserDisplayName: "Alice", private: priv}
	id.PeerName = PeerName(id.PeerID)
	return id
}

func signed(t *testing.T, id *Identity, content string) *Event {
	t.Helper()
	e := &Event{
		EventID: "evt-1", PeerID: id.PeerID, PeerSequence: 1, RoomID: "r",
		Timestamp: "2026-09-17T00:00:00Z", UserDisplayName: id.UserDisplayName,
		EventType: EventUserPrompt, Content: content,
	}
	e.Sign(id.private)
	return e
}

func TestIdentifierIsTheKey(t *testing.T) {
	id := testIdentity(t)
	pub, err := PublicFromPeerID(id.PeerID)
	if err != nil {
		t.Fatalf("a peer id must name a verifiable key: %v", err)
	}
	if !strings.HasPrefix(id.PeerID, "ed25519:") {
		t.Errorf("identifier does not say what it is: %q", id.PeerID)
	}
	e := signed(t, id, "hello")
	if err := e.Verify(); err != nil {
		t.Fatalf("a freshly signed event must verify: %v", err)
	}
	if !ed25519.Verify(pub, e.signingBytesV2(), mustDecode(t, e.Signature)) {
		t.Error("signature does not verify against the key its own id names")
	}
}

// §13 forbids a relayer rewriting an event's origin. Until signing, that was a
// rule with no enforcement. Each field below is one a relayer might alter.
func TestRelayerCannotAlterAnEvent(t *testing.T) {
	id := testIdentity(t)
	for name, tamper := range map[string]func(*Event){
		"content":        func(e *Event) { e.Content = "something else entirely" },
		"peer sequence":  func(e *Event) { e.PeerSequence = 99 },
		"event id":       func(e *Event) { e.EventID = "evt-other" },
		"room":           func(e *Event) { e.RoomID = "another-room" },
		"timestamp":      func(e *Event) { e.Timestamp = "2030-01-01T00:00:00Z" },
		"display name":   func(e *Event) { e.UserDisplayName = "David" },
		"event type":     func(e *Event) { e.EventType = EventAssistantMessage },
		"claude session": func(e *Event) { e.OriginSessionID = "someone-elses-session" },
		"metadata":       func(e *Event) { e.Metadata = json.RawMessage(`{"toolCalls":99}`) },
	} {
		e := signed(t, id, "the original turn")
		tamper(e)
		if err := e.Verify(); err == nil {
			t.Errorf("altering %s went undetected", name)
		}
	}
}

// The attack §13 exists to prevent: Alice relays to Carlos an event claiming to
// come from David. She cannot produce David's signature.
func TestOnePeerCannotSpeakAsAnother(t *testing.T) {
	alice, david := testIdentity(t), testIdentity(t)

	forged := &Event{
		EventID: "evt-forged", PeerID: david.PeerID, PeerSequence: 1, RoomID: "r",
		Timestamp: "2026-09-17T00:00:00Z", UserDisplayName: "David",
		EventType: EventUserPrompt, Content: "David would never say this",
	}
	forged.Sign(alice.private) // the best she can do

	if err := forged.Verify(); err == nil {
		t.Fatal("a peer signed an event attributed to another peer and it verified")
	}
}

func TestUnsignedEventIsRejected(t *testing.T) {
	id := testIdentity(t)
	e := signed(t, id, "hi")
	e.Signature = ""
	if err := e.Verify(); err == nil {
		t.Error("an unsigned event verified")
	}
}

// Length-prefixing exists so that no two different events produce identical bytes
// by moving a field boundary.
func TestFieldBoundariesCannotBeShifted(t *testing.T) {
	a := &Event{PeerID: "ed25519:x", EventID: "ab", Content: "cd"}
	b := &Event{PeerID: "ed25519:x", EventID: "a", Content: "bcd"}
	if string(a.signingBytesV2()) == string(b.signingBytesV2()) {
		t.Error("two different events share signing bytes; a boundary can be moved")
	}
}

// whoami prints the identity struct. A private key reachable through it would be
// published every time someone asks who they are.
func TestIdentityNeverMarshalsThePrivateKey(t *testing.T) {
	id := testIdentity(t)
	buf, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(buf), "private") || len(buf) > 400 {
		t.Errorf("identity JSON may carry key material: %s", buf)
	}
}

// Fingerprint is a display, not a verification method (§25). It must therefore
// lose nothing -- a person comparing two keys after an alarm is reading all of it.
func TestFingerprintIsLosslessAndDisplayOnly(t *testing.T) {
	id := testIdentity(t)
	fp := strings.ReplaceAll(Fingerprint(id.PeerID), " ", "")
	if want := strings.TrimPrefix(id.PeerID, keyPrefix); fp != want {
		t.Errorf("fingerprint is not the whole identifier:\n got %s\nwant %s", fp, want)
	}
}

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := b64url(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The upgrade path. An event is immutable (§7) so it can never be re-signed, and
// old events do not sit still: §13 relays them and D-029 makes refetching a room's
// history the designed recovery from local loss. An event signed before a field was
// added must therefore still verify, or that recovery fails.
func TestAnEventSignedUnderAnOlderSchemeStillVerifies(t *testing.T) {
	id := testIdentity(t)
	e := signed(t, id, "written before sigVersion existed")

	// Exactly what a row written by an earlier build looks like when read back:
	// signature intact, no scheme recorded.
	e.SigVersion = 0
	if err := e.Verify(); err != nil {
		t.Fatalf("an event stored before the version was recorded no longer verifies: %v", err)
	}

	// And the same event once the version is present.
	e.SigVersion = 2
	if err := e.Verify(); err != nil {
		t.Errorf("an event signed under v2 does not verify as v2: %v", err)
	}
}

// A scheme this build does not know must be refused as an UPGRADE problem, not as
// a forgery. The two require opposite responses from a person: one says install a
// newer build, the other says somebody is attacking you.
func TestAnUnknownSchemeIsRefusedAsAVersionProblem(t *testing.T) {
	id := testIdentity(t)
	e := signed(t, id, "from the future")
	e.SigVersion = 99

	err := e.Verify()
	if err == nil {
		t.Fatal("an event signed under an unknown scheme verified")
	}
	if strings.Contains(err.Error(), "does not match the peer id") {
		t.Errorf("an unknown scheme is reported as a forgery, which sends someone hunting an attacker: %v", err)
	}
	if !strings.Contains(err.Error(), "upgrade") {
		t.Errorf("the refusal does not say what to do about it: %v", err)
	}
}

// Signing stamps the scheme, so nothing relies on the zero value meaning v2 except
// rows written before the column existed.
func TestSigningRecordsItsScheme(t *testing.T) {
	e := signed(t, testIdentity(t), "hi")
	if e.SigVersion != currentSigVersion {
		t.Errorf("Sign recorded scheme %d, want %d", e.SigVersion, currentSigVersion)
	}
}
