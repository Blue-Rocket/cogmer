package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	authTag       = protocolNamespace + "/sync-request/v4"
	// authTagV5 is the scheme a machine signs under for its user (D-044, sync
	// requests are signed): the request names the user as its peer and the machine
	// key that signed it, and carries the chain that ties the one to the other beside
	// the signed fields. v4 stays as it was for a machine with no user identity.
	authTagV5 = protocolNamespace + "/sync-request/v5"
)

// requestCreds is what a request carries to say who is asking: the peer, and under
// v5 the machine key that signed for it and the chain from that key to the peer.
type requestCreds struct {
	PeerID, Timestamp, Nonce, Signature, Endpoint, Signer string
	Chain                                                 *Chain
}

// signRequestAs signs a request's credentials under v5 when this machine speaks for
// a user, and under v4 otherwise, so a machine with no user identity asks exactly as
// it always has.
func signRequestAs(id *Identity, roomID, endpoint string) (requestCreds, error) {
	if !id.HasUser() {
		ts, nonce, sig, err := signRequest(id, roomID, endpoint)
		return requestCreds{PeerID: id.PeerID, Timestamp: ts, Nonce: nonce, Signature: sig, Endpoint: endpoint}, err
	}
	if id.private == nil {
		return requestCreds{}, errors.New("no private key: this peer cannot prove who it is")
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return requestCreds{}, err
	}
	c := requestCreds{
		PeerID: id.User, Signer: id.PeerID, Endpoint: endpoint,
		Nonce:     base64.RawURLEncoding.EncodeToString(raw),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	chain := *id.Chain
	c.Chain = &chain
	c.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(id.private,
		requestBytesV5(c.PeerID, c.Signer, roomID, c.Timestamp, c.Nonce, endpoint)))
	return c, nil
}

// requestBytesV5 is what a v5 request's signature covers: v4's fields with the
// signing machine's key after the peer, so that a signature by one machine cannot be
// presented as another's.
func requestBytesV5(peerID, signer, roomID, ts, nonce, endpoint string) []byte {
	var b strings.Builder
	put := func(s string) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(s)))
		b.Write(n[:])
		b.WriteString(s)
	}
	put(authTagV5)
	put(peerID)
	put(signer)
	put(roomID)
	put(ts)
	put(nonce)
	put(endpoint)
	return []byte(b.String())
}

// requestMachine is the machine that made a request: the key that signed a v5
// request, and the peer itself for v4, where a peer is one machine.
func (r syncRequest) requestMachine() string {
	if r.Chain != nil {
		return r.Signer
	}
	return r.PeerID
}

// signRequest produces the credentials a peer presents when asking to sync.
func signRequest(id *Identity, roomID, endpoint string) (ts, nonce, sig string, err error) {
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
		ed25519.Sign(id.private, requestBytes(id.PeerID, roomID, ts, nonce, endpoint)))
	return ts, nonce, sig, nil
}

// requestBytes is what a request signature covers.
//
// The `have` map is deliberately not covered. Altering it gains an authenticated
// peer nothing -- it can ask for everything anyway -- and canonicalising a map for
// signing invites the kind of ambiguity length-prefixing exists to avoid.
func requestBytes(peerID, roomID, ts, nonce, endpoint string) []byte {
	var b strings.Builder
	put := func(s string) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(s)))
		b.Write(n[:])
		b.WriteString(s)
	}
	put(authTag)
	put(peerID)
	put(roomID)
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

// connectedAs refuses a request signed by a machine other than the one whose key the
// connection's client certificate holds: the key that signed a v5 request, and the
// peer for v4, where a peer is one machine. A signed request carries a timestamp and
// a nonce, and each machine remembers the nonces it has seen, so without this a
// request captured on its way to one machine could be presented to another within
// the tolerance (D-143, admission is proved by a signed request bound to the
// connection). A connection with no client certificate, which only a test makes,
// binds nothing.
func connectedAs(r *http.Request, machine string) error {
	if r == nil || r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return nil
	}
	if k, ok := r.TLS.PeerCertificates[0].PublicKey.(ed25519.PublicKey); !ok || PeerIDFromPublic(k) != machine {
		return errors.New("the request is signed by a machine other than the one connected")
	}
	return nil
}

// verifyRequest establishes WHO is asking. It does not decide whether they may:
// that is admission, and it belongs with the guest list.
func (d *Daemon) verifyRequest(req syncRequest, roomID string) error {
	if req.PeerID == "" || req.Signature == "" {
		return errors.New("unauthenticated request")
	}
	sig, err := base64.RawURLEncoding.DecodeString(req.Signature)
	if err != nil {
		return fmt.Errorf("signature is not valid base64url: %w", err)
	}
	if req.Chain == nil {
		pub, err := PublicFromPeerID(req.PeerID)
		if err != nil {
			return err
		}
		if !ed25519.Verify(pub, requestBytes(req.PeerID, req.RoomID, req.Timestamp, req.Nonce, req.Endpoint), sig) {
			return errors.New("signature does not match the peer id presenting it")
		}
	} else {
		// v5: the machine that signed speaks for the user the request names, and
		// only while it is neither removed nor suspended (D-044, sync requests are signed; D-220, a removal
		// names the last event of a machine that counts; D-234, a suspended
		// machine's sync requests are refused).
		pub, err := PublicFromPeerID(req.Signer)
		if err != nil {
			return fmt.Errorf("signer: %w", err)
		}
		if !ed25519.Verify(pub, requestBytesV5(req.PeerID, req.Signer, req.RoomID, req.Timestamp, req.Nonce, req.Endpoint), sig) {
			return errors.New("signature does not match the machine presenting it")
		}
		if _, err := d.members.checkLiveChain(req.PeerID, req.Signer, *req.Chain, time.Now()); err != nil {
			return fmt.Errorf("%s does not speak for %s: %w", PeerName(req.Signer), PeerName(req.PeerID), err)
		}
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
	if req.Chain != nil {
		// Kept once the request has passed every check of who is asking, so that
		// this machine knows the machine before any event of it arrives.
		if err := d.members.StoreChain(*req.Chain); err != nil {
			log.Printf("sync: could not keep the chain of %s: %v", PeerName(req.Signer), err)
		}
		// A key known here as a colleague's that now signs for a user carries what it
		// had to that user, or, linked to another user first, is refused (D-242).
		if err := d.carryMachineKey(req.Signer, req.PeerID, true); err != nil {
			return err
		}
	}

	// Who is established. Whether they may is a separate question, and answering
	// only the first is what let a stranger with a freshly generated key read a
	// private room (D-044, sync requests are signed).
	if roomID != "" && !d.members.IsGuest(roomID, req.PeerID) {
		return &notGuestError{peer: req.PeerID}
	}

	// Authentication says WHO holds the key. Admission says WHETHER that key may
	// enter. Neither says the key is the person's -- only a verification does, and
	// without one every check above passes just as well for whoever substituted it
	// in transit (D-054, verification gates synchronization). So no transcript crosses to an unverified peer.
	//
	// Not a disclosure: a caller reaching this line is already a recorded guest,
	// so it learns nothing it did not put there itself, and the message has to be
	// actionable or the refusal looks like a fault.
	//
	// By user: a colleague verified once is verified for every machine of theirs,
	// and this machine's own user is its own (D-219, a user is a master key that certifies machines through a shared key).
	if roomID != "" && !d.authorVerified(req.PeerID) {
		return fmt.Errorf("%s is a guest but is UNVERIFIED — nothing has confirmed this key is theirs; "+
			"both of you run `"+invocation()+" verify`", PeerName(req.PeerID))
	}
	return nil
}

// notGuestError is a request whose asker is established and is not on the room's
// list. It is a type of its own because the one refusal that says more than
// "unauthenticated" is to a user the room revoked, who is sent the revocations that
// expelled them (D-241, an expelled user is told, through the refusal that carries the
// revocation), and that is known only once the asker is.
type notGuestError struct{ peer string }

func (e *notGuestError) Error() string {
	return fmt.Sprintf("%s (%s) is authenticated but is not a guest of this room",
		PeerName(e.peer), e.peer[:24])
}
