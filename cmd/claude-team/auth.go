package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Proving possession on connection.
//
// Signing events gives integrity: nothing can be forged or falsely attributed.
// It does not say who may ask, so the peer API still admitted any host that could
// reach it to read a room. This closes that.
//
// Each request is signed rather than a session being established, because the
// protocol polls: a handshake per poll would cost two round trips a second to
// avoid holding one piece of state. A signed request needs neither.
//
// Replay is prevented by a timestamp and a nonce rather than by a server-issued
// challenge, which would reintroduce the extra round trip. The window is wide
// enough to survive real clock skew -- two NTP-synced machines measured 408ms
// apart -- and narrow enough that the seen-nonce set stays small.

const (
	authTolerance = 2 * time.Minute
	authTag       = "claude-team/sync-request/v2"
)

// signRequest produces the credentials a peer presents when asking to sync.
func signRequest(id *Identity, room, endpoint string) (ts, nonce, sig string, err error) {
	if id.private == nil {
		return "", "", "", errors.New("no private key: this peer cannot prove who it is")
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", err
	}
	nonce = base64.RawURLEncoding.EncodeToString(raw)
	ts = time.Now().UTC().Format(time.RFC3339Nano)
	sig = base64.RawURLEncoding.EncodeToString(
		ed25519.Sign(id.private, requestBytes(id.PeerID, room, ts, nonce, endpoint)))
	return ts, nonce, sig, nil
}

// requestBytes is what a request signature covers.
//
// The `have` map is deliberately not covered. Altering it gains an authenticated
// peer nothing -- it can ask for everything anyway -- and canonicalising a map for
// signing invites the kind of ambiguity length-prefixing exists to avoid.
func requestBytes(peerID, room, ts, nonce, endpoint string) []byte {
	var b strings.Builder
	put := func(s string) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(s)))
		b.Write(n[:])
		b.WriteString(s)
	}
	put(authTag)
	put(peerID)
	put(room)
	put(ts)
	put(nonce)
	// The caller's endpoint is signed because a peer acts on it -- it polls there.
	// An unsigned one could redirect a peer's polling, which forges nothing but
	// denies plenty.
	put(endpoint)
	return []byte(b.String())
}

// replayGuard remembers nonces for as long as a timestamp could still be accepted.
type replayGuard struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

func (g *replayGuard) admit(nonce string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.seen == nil {
		g.seen = map[string]time.Time{}
	}
	for k, t := range g.seen {
		if now.Sub(t) > authTolerance*2 {
			delete(g.seen, k)
		}
	}
	if _, used := g.seen[nonce]; used {
		return false
	}
	g.seen[nonce] = now
	return true
}

// verifyRequest establishes WHO is asking. It does not decide whether they may:
// that is admission, and it belongs with the guest list.
func (d *Daemon) verifyRequest(req syncRequest, roomID string) error {
	if req.PeerID == "" || req.Signature == "" {
		return errors.New("unauthenticated request")
	}
	pub, err := PublicFromPeerID(req.PeerID)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(req.Signature)
	if err != nil {
		return fmt.Errorf("signature is not valid base64url: %w", err)
	}
	if !ed25519.Verify(pub, requestBytes(req.PeerID, req.Room, req.Timestamp, req.Nonce, req.Endpoint), sig) {
		return errors.New("signature does not match the peer id presenting it")
	}

	t, err := time.Parse(time.RFC3339Nano, req.Timestamp)
	if err != nil {
		return fmt.Errorf("unreadable timestamp: %w", err)
	}
	now := time.Now().UTC()
	if d := now.Sub(t); d > authTolerance || d < -authTolerance {
		return fmt.Errorf("timestamp is %s away from now, tolerance is %s", d.Round(time.Second), authTolerance)
	}
	if !d.replay.admit(req.Nonce, now) {
		return errors.New("nonce already used; this is a replayed request")
	}

	// Who is established. Whether they may is a separate question, and answering
	// only the first is what let a stranger with a freshly generated key read a
	// private room (D-044).
	if roomID != "" && !d.members.IsGuest(roomID, req.PeerID) {
		return fmt.Errorf("%s (%s) is authenticated but is not a guest of this room",
			PeerName(req.PeerID), req.PeerID[:24])
	}
	return nil
}
