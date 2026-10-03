package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// Frozen records, produced once from a fixed key and checked in as
// testdata/vectors.json. Every other signing test signs a fresh record and then
// verifies it, so an edit to a signing function changes signer and verifier
// together and those tests keep passing. A record made before the edit, held on a
// colleague's machine, would then fail to verify with an error that reads as a
// forgery (D-058). These tests read records no edit of this code can have changed.
//
// Never regenerate the file to make a test pass. A failure here means the bytes a
// scheme signs changed, and the fix is a new scheme beside the old one.

type frozenVectors struct {
	SeedHex     string      `json:"seedHex"`
	PeerID      string      `json:"peerId"`
	Events      []wireEvent `json:"events"`
	SyncRequest struct {
		RoomID, Timestamp, Nonce, Endpoint, Signature string
	} `json:"syncRequest"`
	Offer struct {
		RoomID, RoomName, Endpoint, Timestamp, Nonce, Signature string
	} `json:"offer"`
	VerifyStep struct{ Step, Payload, Signature string } `json:"verifyStep"`
	SAS        struct{ NonceA, NonceB, Words string }    `json:"sas"`
	Commitment string                                    `json:"commitment"`
	// SyncResponses holds a whole sync response for each wire version this build
	// reads, as text, because decoding it is the thing under test.
	SyncResponses map[string]string `json:"syncResponses"`
}

func loadVectors(t *testing.T) frozenVectors {
	t.Helper()
	raw, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v frozenVectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestTheFrozenKeyIsTheKeyTheVectorsNameSo(t *testing.T) {
	v := loadVectors(t)
	seed, _ := hex.DecodeString(v.SeedHex)
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	if PeerIDFromPublic(pub) != v.PeerID {
		t.Fatalf("the identifier for the frozen seed changed to %s; the encoding of an identifier is part of every record", PeerIDFromPublic(pub))
	}
}

func TestAnEventSignedUnderV3BeforeAnyEditStillVerifies(t *testing.T) {
	v := loadVectors(t)
	if len(v.Events) < 2 {
		t.Fatal("the frozen events are missing")
	}
	for _, w := range v.Events {
		e := fromWire(w)
		if err := e.Verify(); err != nil {
			t.Errorf("frozen event %.8s (sequence %d) no longer verifies: %v. The bytes scheme v3 signs have changed; add a scheme and leave v3 as it was (D-058)",
				e.EventID, e.PeerSequence, err)
		}
	}
}

func TestAChangeToAFrozenEventFailsVerification(t *testing.T) {
	// The check above cannot fail if Verify accepts anything, so show it refusing.
	e := fromWire(loadVectors(t).Events[0])
	e.Content += "x"
	if e.Verify() == nil {
		t.Error("an altered frozen event verified")
	}
}

func verifyFrozen(t *testing.T, what, peerID, sig string, msg []byte) {
	t.Helper()
	pub, err := PublicFromPeerID(peerID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pub, msg, raw) {
		t.Errorf("the frozen %s signature no longer verifies, so the bytes it covers changed and an older peer's messages are refused", what)
	}
	msg = append([]byte{}, msg...)
	msg[len(msg)-1] ^= 1
	if ed25519.Verify(pub, msg, raw) {
		t.Errorf("the frozen %s signature verifies over altered bytes", what)
	}
}

func TestFrozenRequestOfferAndVerificationSignaturesStillVerify(t *testing.T) {
	v := loadVectors(t)
	r := v.SyncRequest
	verifyFrozen(t, "sync request", v.PeerID, r.Signature, requestBytes(v.PeerID, r.RoomID, r.Timestamp, r.Nonce, r.Endpoint))
	o := v.Offer
	verifyFrozen(t, "offer", v.PeerID, o.Signature, offerBytes(v.PeerID, o.RoomID, o.RoomName, o.Endpoint, o.Timestamp, o.Nonce))
	s := v.VerifyStep
	verifyFrozen(t, "verification step", v.PeerID, s.Signature, verifyBytes(s.Step, v.PeerID, []byte(s.Payload)))
}

func TestFrozenWordsAndCommitmentAreWhatTwoVersionsMustAgreeOn(t *testing.T) {
	v := loadVectors(t)
	na, _ := base64.RawURLEncoding.DecodeString(v.SAS.NonceA)
	nb, _ := base64.RawURLEncoding.DecodeString(v.SAS.NonceB)
	if got := SAS(v.PeerID, na, "ed25519:other", nb); got != v.SAS.Words {
		t.Errorf("the words for a frozen exchange are %q, were %q; two people on different versions would read each other different words and refuse a genuine pairing", got, v.SAS.Words)
	}
	if got := base64.RawURLEncoding.EncodeToString(sasCommitment(v.PeerID, na)); got != v.Commitment {
		t.Errorf("the frozen commitment changed to %s", got)
	}
}

// A sync response at each wire version this build reads must decode, be accepted by
// speaks, and carry events that verify. Raising wireVersion without adding its
// frozen response here fails, so a new version cannot ship unrecorded.
func TestASyncResponseAtEveryReadableWireVersionStillDecodes(t *testing.T) {
	v := loadVectors(t)
	for ver := minWireVersion; ver <= wireVersion; ver++ {
		text, ok := v.SyncResponses[strconv.Itoa(ver)]
		if !ok {
			t.Errorf("no frozen sync response for wire version %d, which this build reads", ver)
			continue
		}
		var out syncResponse
		if err := json.Unmarshal([]byte(text), &out); err != nil {
			t.Errorf("v%d response does not decode: %v", ver, err)
			continue
		}
		if out.Protocol != ver || !speaks(out.Protocol) {
			t.Errorf("v%d response reads as protocol %d, speaks=%v", ver, out.Protocol, speaks(out.Protocol))
		}
		if len(out.Events) != 2 {
			t.Errorf("v%d response decodes to %d events, want 2", ver, len(out.Events))
		}
		for _, w := range out.Events {
			e := fromWire(w)
			if err := e.Verify(); err != nil {
				t.Errorf("v%d response event %.8s: %v", ver, e.EventID, err)
			}
		}
	}
}
