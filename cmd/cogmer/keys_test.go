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
	if !ed25519.Verify(pub, e.signingBytesV3(), mustDecode(t, e.Signature)) {
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
	if string(a.signingBytesV3()) == string(b.signingBytesV3()) {
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

// The zero value must be refused rather than resolved to whichever scheme happens
// to be oldest. An event arriving with no recorded scheme is a build this one does
// not know or a field somebody dropped, and guessing on its behalf verifies bytes
// whose provenance nobody established -- which is the one thing a signature check
// exists to prevent.
func TestAnEventWithNoRecordedSchemeIsRefused(t *testing.T) {
	id := testIdentity(t)
	e := signed(t, id, "correctly signed, but the version is missing")
	e.SigVersion = 0

	err := e.Verify()
	if err == nil {
		t.Fatal("an event with no recorded scheme verified; the zero value is being guessed at")
	}
	if strings.Contains(err.Error(), "does not match the peer id") {
		t.Errorf("a missing version is reported as a forgery, which sends somebody hunting an attacker: %v", err)
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

// Signing stamps the scheme, so nothing anywhere relies on the zero value meaning
// a particular one.
func TestSigningRecordsItsScheme(t *testing.T) {
	e := signed(t, testIdentity(t), "hi")
	if e.SigVersion != currentSigVersion {
		t.Errorf("Sign recorded scheme %d, want %d", e.SigVersion, currentSigVersion)
	}
}

// The state directory is where the product name reaches the filesystem, and the
// installer already honoured this variable while the binary ignored it -- so a
// person who set it got a binary in one place and its state in another, silently.
func TestStateDirectoryIsOverridable(t *testing.T) {
	t.Setenv("COGMER_HOME", "/tmp/somewhere-else")
	if got := homeDir(); got != "/tmp/somewhere-else" {
		t.Errorf("homeDir() = %q, want the override", got)
	}
	t.Setenv("COGMER_HOME", "   ")
	if got := homeDir(); !strings.HasSuffix(got, stateDirName) {
		t.Errorf("a blank override was honoured, giving %q", got)
	}
}

// Domain separation must not depend on what the product is called. A rename that
// changed these would, by the rule in keys.go, mean carrying the old namespace
// forever for a name nobody uses.
func TestSigningNamespacesCarryNoProductName(t *testing.T) {
	// The tags that are hashed rather than signed in the clear are asserted as
	// constants, because a digest cannot be inspected for the string that made it.
	for name, tag := range map[string]string{
		"sync-request": authTag,
		"sas":          sasTag,
		"sas-commit":   sasCommit,
	} {
		if strings.Contains(tag, "cogmer") {
			t.Errorf("the %s tag still carries the product name: %s", name, tag)
		}
		if !strings.HasPrefix(tag, protocolNamespace+"/") {
			t.Errorf("the %s tag is not in the protocol namespace: %s", name, tag)
		}
	}

	// And the ones signed in the clear are asserted on the bytes themselves.
	e := &Event{EventID: "e", PeerID: "ed25519:x", Content: "c"}
	if got := string(e.signingBytesV3()); strings.Contains(got, "cogmer") {
		t.Error("the event namespace still carries the product name")
	} else if !strings.Contains(got, protocolNamespace) {
		t.Error("the event namespace is not separated at all")
	}

}

// The name is seen only by OTHER people, so there is no moment where its owner
// notices it is wrong. That is why the difference between a name somebody picked
// and one $USER supplied has to be recorded rather than guessed at (D-095).
func TestAChosenNameIsDistinguishedFromAGuessedOne(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())

	id, err := LoadIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if id.NameChosen {
		t.Fatal("a freshly invented name reports itself as chosen")
	}
	if id.UserDisplayName == "" {
		t.Fatal("no name at all; other people would see nothing")
	}
	if !strings.Contains(nameLine(id), "username") {
		t.Errorf("a guessed name does not say so: %q", nameLine(id))
	}

	// Keeping the guessed name IS a choice, and must stop the offer. Comparing
	// against the guess instead of recording the act would pester the person whose
	// username really is their name, for ever.
	same := id.UserDisplayName
	got, err := SetDisplayName(same)
	if err != nil {
		t.Fatal(err)
	}
	if !got.NameChosen || got.UserDisplayName != same {
		t.Fatalf("keeping the guessed name did not count as choosing it: %+v", got)
	}
	if strings.Contains(nameLine(got), "username") {
		t.Errorf("the offer survives a deliberate choice: %q", nameLine(got))
	}

	// It survives a reload, or the offer returns every session.
	again, err := LoadIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if !again.NameChosen || again.UserDisplayName != same {
		t.Errorf("the choice was not persisted: %+v", again)
	}

	// And a new name replaces it.
	renamed, err := SetDisplayName("Alice")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.UserDisplayName != "Alice" {
		t.Errorf("name is %q, want Alice", renamed.UserDisplayName)
	}
}
