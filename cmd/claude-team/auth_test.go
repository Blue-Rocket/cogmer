package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"
)

func authDaemon(t *testing.T) (*Daemon, *Identity) {
	t.Helper()
	id := testIdentity(t)
	return &Daemon{room: "r", id: id}, id
}

func credentials(t *testing.T, id *Identity, room string) syncRequest {
	t.Helper()
	ts, nonce, sig, err := signRequest(id, room)
	if err != nil {
		t.Fatal(err)
	}
	return syncRequest{Protocol: wireVersion, Room: room, PeerID: id.PeerID,
		Timestamp: ts, Nonce: nonce, Signature: sig}
}

// A room's conversation is not public. Before this, any host that could reach the
// peer port could read one.
func TestUnauthenticatedRequestIsRefused(t *testing.T) {
	d, _ := authDaemon(t)
	if err := d.verifyRequest(syncRequest{Room: "r"}); err == nil {
		t.Error("a request with no credentials was accepted")
	}
}

func TestGenuineRequestIsAccepted(t *testing.T) {
	d, id := authDaemon(t)
	if err := d.verifyRequest(credentials(t, id, "r")); err != nil {
		t.Fatalf("a correctly signed request was refused: %v", err)
	}
}

// Possession of the key, not knowledge of the identifier. An identifier is public
// by design (D-042), so presenting one must prove nothing on its own.
func TestKnowingAnIdentifierIsNotEnough(t *testing.T) {
	d, victim := authDaemon(t)
	attacker := testIdentity(t)

	// The attacker knows the victim's identifier -- everyone does -- and signs
	// with the only key it has.
	req := credentials(t, attacker, "r")
	req.PeerID = victim.PeerID
	if err := d.verifyRequest(req); err == nil {
		t.Error("a peer authenticated as another by presenting its identifier")
	}
}

// A captured request must not work twice.
func TestReplayedRequestIsRefused(t *testing.T) {
	d, id := authDaemon(t)
	req := credentials(t, id, "r")
	if err := d.verifyRequest(req); err != nil {
		t.Fatalf("first use refused: %v", err)
	}
	if err := d.verifyRequest(req); err == nil {
		t.Error("the same request was accepted twice")
	}
}

// Old credentials must expire, or a capture is good forever.
func TestStaleRequestIsRefused(t *testing.T) {
	d, id := authDaemon(t)
	req := credentials(t, id, "r")
	req.Timestamp = time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339Nano)
	// Re-sign so the only defect is age, not the signature.
	req.Signature = base64.RawURLEncoding.EncodeToString(
		ed25519.Sign(id.private, requestBytes(req.PeerID, req.Room, req.Timestamp, req.Nonce)))
	if err := d.verifyRequest(req); err == nil {
		t.Error("a ten-minute-old request was accepted")
	}
}

// Credentials for one room must not open another.
func TestCredentialsAreBoundToTheRoom(t *testing.T) {
	d, id := authDaemon(t)
	req := credentials(t, id, "a-different-room")
	req.Room = "r"
	if err := d.verifyRequest(req); err == nil {
		t.Error("credentials signed for one room were accepted for another")
	}
}

// The tolerance must survive real clock skew: two NTP-synced machines measured
// 408ms apart, and a peer on a worse network should still sync.
func TestToleranceSurvivesRealClockSkew(t *testing.T) {
	d, id := authDaemon(t)
	for _, skew := range []time.Duration{-30 * time.Second, -2 * time.Second, 2 * time.Second, 30 * time.Second} {
		req := credentials(t, id, "r")
		req.Timestamp = time.Now().UTC().Add(skew).Format(time.RFC3339Nano)
		req.Signature = base64.RawURLEncoding.EncodeToString(
			ed25519.Sign(id.private, requestBytes(req.PeerID, req.Room, req.Timestamp, req.Nonce)))
		if err := d.verifyRequest(req); err != nil {
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
