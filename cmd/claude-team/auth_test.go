package main

import (
	"crypto/ed25519"
	"encoding/base64"
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

func credentials(t *testing.T, id *Identity, room string) syncRequest {
	t.Helper()
	ts, nonce, sig, err := signRequest(id, room, "127.0.0.1:4783")
	if err != nil {
		t.Fatal(err)
	}
	return syncRequest{Protocol: wireVersion, Room: room, PeerID: id.PeerID,
		Timestamp: ts, Nonce: nonce, Signature: sig, Endpoint: "127.0.0.1:4783"}
}

// A room's conversation is not public. Before this, any host that could reach the
// peer port could read one.
func TestUnauthenticatedRequestIsRefused(t *testing.T) {
	d, _, room := authDaemon(t)
	if err := d.verifyRequest(syncRequest{Room: room.RoomName}, room.RoomID); err == nil {
		t.Error("a request with no credentials was accepted")
	}
}

func TestGenuineRequestIsAccepted(t *testing.T) {
	d, id, room := authDaemon(t)
	if err := d.verifyRequest(credentials(t, id, room.RoomName), room.RoomID); err != nil {
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
	req := credentials(t, attacker, room.RoomName)
	req.PeerID = victim.PeerID
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("a peer authenticated as another by presenting its identifier")
	}
}

// A captured request must not work twice.
func TestReplayedRequestIsRefused(t *testing.T) {
	d, id, room := authDaemon(t)
	req := credentials(t, id, room.RoomName)
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
	req := credentials(t, id, room.RoomName)
	req.Timestamp = time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339Nano)
	// Re-sign so the only defect is age, not the signature.
	req.Signature = base64.RawURLEncoding.EncodeToString(
		ed25519.Sign(id.private, requestBytes(req.PeerID, req.Room, req.Timestamp, req.Nonce, req.Endpoint)))
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("a ten-minute-old request was accepted")
	}
}

// Credentials for one room must not open another.
func TestCredentialsAreBoundToTheRoom(t *testing.T) {
	d, id, room := authDaemon(t)
	req := credentials(t, id, "a-different-room")
	req.Room = room.RoomName
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("credentials signed for one room were accepted for another")
	}
}

// The tolerance must survive real clock skew: two NTP-synced machines measured
// 408ms apart, and a peer on a worse network should still sync.
func TestToleranceSurvivesRealClockSkew(t *testing.T) {
	d, id, room := authDaemon(t)
	for _, skew := range []time.Duration{-30 * time.Second, -2 * time.Second, 2 * time.Second, 30 * time.Second} {
		req := credentials(t, id, room.RoomName)
		req.Timestamp = time.Now().UTC().Add(skew).Format(time.RFC3339Nano)
		req.Signature = base64.RawURLEncoding.EncodeToString(
			ed25519.Sign(id.private, requestBytes(req.PeerID, req.Room, req.Timestamp, req.Nonce, req.Endpoint)))
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
	req := credentials(t, id, room.RoomName)
	req.Endpoint = "127.0.0.1:6666" // an attacker's address
	if err := d.verifyRequest(req, room.RoomID); err == nil {
		t.Error("a request's endpoint was altered without invalidating its signature")
	}
}
