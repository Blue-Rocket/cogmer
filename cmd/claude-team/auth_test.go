package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A daemon, a room, and an identity admitted to it: authentication and admission
// are separate checks and both must pass.
func authDaemon(t *testing.T) (*Daemon, *Identity, Room) {
	t.Helper()
	d, room := testDaemon(t)
	guest := testIdentity(t)
	if err := d.members.Invite(room.RoomID, guest.PeerID); err != nil {
		t.Fatal(err)
	}
	return d, guest, room
}

func credentials(t *testing.T, id *Identity, roomID string) syncRequest {
	t.Helper()
	ts, nonce, sig, err := signRequest(id, roomID, "127.0.0.1:4783")
	if err != nil {
		t.Fatal(err)
	}
	return syncRequest{Protocol: wireVersion, RoomID: roomID, PeerID: id.PeerID,
		Timestamp: ts, Nonce: nonce, Signature: sig, Endpoint: "127.0.0.1:4783"}
}

// A room's conversation is not public. Before this, any host that could reach the
// peer port could read one.
func TestUnauthenticatedRequestIsRefused(t *testing.T) {
	d, _, room := authDaemon(t)
	if err := d.verifyRequest(syncRequest{RoomID: room.RoomID}, room.RoomID); err == nil {
		t.Error("a request with no credentials was accepted")
	}
}

func TestGenuineRequestIsAccepted(t *testing.T) {
	d, id, room := authDaemon(t)
	if err := d.verifyRequest(credentials(t, id, room.RoomID), room.RoomID); err != nil {
		t.Fatalf("a correctly signed request was refused: %v", err)
	}
}

// Possession of the key, not knowledge of the identifier. An identifier is public
// by design (D-042), so presenting one must prove nothing on its own.
func TestKnowingAnIdentifierIsNotEnough(t *testing.T) {
	d, victim, room := authDaemon(t)
	attacker := testIdentity(t)

	// The attacker knows the victim's identifier -- everyone does -- and signs
	// with the only key it has.
	req := credentials(t, attacker, room.RoomID)
	req.PeerID = victim.PeerID
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("a peer authenticated as another by presenting its identifier")
	}
}

// A captured request must not work twice.
func TestReplayedRequestIsRefused(t *testing.T) {
	d, id, room := authDaemon(t)
	req := credentials(t, id, room.RoomID)
	if err := d.verifyRequest(req, room.RoomID); err != nil {
		t.Fatalf("first use refused: %v", err)
	}
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("the same request was accepted twice")
	}
}

// Old credentials must expire, or a capture is good forever.
func TestStaleRequestIsRefused(t *testing.T) {
	d, id, room := authDaemon(t)
	req := credentials(t, id, room.RoomID)
	req.Timestamp = time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339Nano)
	// Re-sign so the only defect is age, not the signature.
	req.Signature = base64.RawURLEncoding.EncodeToString(
		ed25519.Sign(id.private, requestBytes(req.PeerID, req.RoomID, req.Timestamp, req.Nonce, req.Endpoint)))
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("a ten-minute-old request was accepted")
	}
}

// Credentials for one room must not open another.
func TestCredentialsAreBoundToTheRoom(t *testing.T) {
	d, id, room := authDaemon(t)
	req := credentials(t, id, "a-different-room")
	req.RoomID = room.RoomID
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("credentials signed for one room were accepted for another")
	}
}

// The tolerance must survive real clock skew: two NTP-synced machines measured
// 408ms apart, and a peer on a worse network should still sync.
func TestToleranceSurvivesRealClockSkew(t *testing.T) {
	d, id, room := authDaemon(t)
	for _, skew := range []time.Duration{-30 * time.Second, -2 * time.Second, 2 * time.Second, 30 * time.Second} {
		req := credentials(t, id, room.RoomID)
		req.Timestamp = time.Now().UTC().Add(skew).Format(time.RFC3339Nano)
		req.Signature = base64.RawURLEncoding.EncodeToString(
			ed25519.Sign(id.private, requestBytes(req.PeerID, req.RoomID, req.Timestamp, req.Nonce, req.Endpoint)))
		if err := d.verifyRequest(req, room.RoomID); err != nil {
			t.Errorf("a peer %s out of step was refused: %v", skew, err)
		}
	}
}

// The nonce set must not grow without bound on a daemon polled every second.
func TestReplayGuardForgetsOldNonces(t *testing.T) {
	var g replayGuard
	now := time.Now()
	for i := 0; i < 50; i++ {
		g.admit(string(rune('a'+i%26))+string(rune('0'+i/26)), now.Add(-time.Hour))
	}
	g.admit("recent", now)
	if len(g.seen) > 2 {
		t.Errorf("guard retained %d nonces long past their tolerance", len(g.seen))
	}
}

// A peer acts on the endpoint it is told: it polls there. An unsigned one could
// redirect another peer's polling, which forges nothing and denies plenty.
func TestEndpointIsCoveredBySignature(t *testing.T) {
	d, id, room := authDaemon(t)
	req := credentials(t, id, room.RoomID)
	req.Endpoint = "127.0.0.1:6666" // an attacker's address
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("a request's endpoint was altered without invalidating its signature")
	}
}

// The wire addresses a room by identity, never by name (D-017). A name is a
// mnemonic: it is generated from a small space, it is only unique among the rooms
// one peer happens to hold, and nothing about it is guaranteed by the sender.
// Accepting one here would let a request address a room its sender did not mean.
func TestTheWireRefusesARoomName(t *testing.T) {
	d, id, room := authDaemon(t)

	// Correct in every other respect: a genuine guest, signing genuinely, naming
	// the room it really is a guest of -- by its name.
	req := credentials(t, id, room.RoomName)
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	d.handleSync(w, httptest.NewRequest("POST", "/sync", bytes.NewReader(body)))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("a request addressing the room by name was answered with %d, want %d",
			w.Code, http.StatusUnauthorized)
	}
}

// And the same request, addressed by identity, is answered -- so the test above
// is failing on the identifier rather than on some unrelated defect in the fixture.
func TestTheWireAcceptsARoomID(t *testing.T) {
	d, id, room := authDaemon(t)

	body, _ := json.Marshal(credentials(t, id, room.RoomID))
	w := httptest.NewRecorder()
	d.handleSync(w, httptest.NewRequest("POST", "/sync", bytes.NewReader(body)))

	if w.Code != http.StatusOK {
		t.Fatalf("a correctly addressed request was answered with %d: %s", w.Code, w.Body.String())
	}
	var out syncResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.RoomID != room.RoomID {
		t.Errorf("response named room %q, want %q", out.RoomID, room.RoomID)
	}
}
