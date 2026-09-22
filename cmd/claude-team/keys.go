package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Peer identity is a key pair. The identifier IS the public key, so knowing one
// grants nothing (§6, D-023): it may be printed, logged, listed and read aloud,
// because only the private key produces a signature.
//
// That property is what makes an identifier safe to put in every event, which the
// system does by design. The previous identifier -- ten random bytes verified by
// nothing -- was a shared secret the system broadcast, where knowing one was
// sufficient to claim it.

const keyPrefix = "ed25519:"

// PeerIDFromPublic renders a public key as the peer identifier.
func PeerIDFromPublic(pub ed25519.PublicKey) string {
	return keyPrefix + base64.RawURLEncoding.EncodeToString(pub)
}

// PublicFromPeerID recovers the key an identifier names. An identifier that does
// not parse cannot be verified against, which is itself the answer: reject it.
func PublicFromPeerID(id string) (ed25519.PublicKey, error) {
	if !strings.HasPrefix(id, keyPrefix) {
		return nil, fmt.Errorf("peer id %.16q is not a key (no %q prefix)", id, keyPrefix)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(id, keyPrefix))
	if err != nil {
		return nil, fmt.Errorf("peer id is not valid base64url: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("peer id is %d bytes, want %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

// b64url decodes the encoding used for signatures and identifiers.
func b64url(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

func keyPath() string { return filepath.Join(homeDir(), "identity.key") }

// loadOrCreateKey reads the private key, generating one on first use.
//
// The private key lives in its own file, never in identity.json. That file is
// printed by `whoami` and is meant to be shared -- an identity a person cannot
// hand to a colleague without checking what else is in it is not much of an
// identity.
func loadOrCreateKey() (ed25519.PrivateKey, error) {
	path := keyPath()
	if raw, err := os.ReadFile(path); err == nil {
		key, derr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw)))
		if derr != nil || len(key) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("%s is not a usable key; move it aside to generate a new identity", path)
		}
		return ed25519.PrivateKey(key), nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(homeDir(), 0o700); err != nil {
		return nil, err
	}
	enc := base64.RawStdEncoding.EncodeToString(priv)
	if err := os.WriteFile(path, []byte(enc+"\n"), 0o600); err != nil {
		return nil, err
	}
	return priv, nil
}

// signingBytes is the exact sequence an event's signature covers.
//
// Every field is length-prefixed so that no two different events can produce the
// same bytes by moving a boundary -- concatenating fields directly would let a
// content ending in one value and a session id beginning with another swap places
// undetected.
//
// The leading tag binds a signature to this purpose and version: a signature made
// here can never be replayed as one made over something else. The version moved to
// v2 when the session field was renamed -- a signature covers field values, so
// changing what a field means changes what was signed.
// Signature schemes are kept, never replaced.
//
// An event is immutable (§7) and therefore can never be re-signed, and old events
// do not merely sit in a database: §13 relays them between peers, and D-029 makes
// refetching a room's history the designed recovery from local loss. So an event
// signed years ago must still verify under the scheme that signed it, or the
// recovery path fails -- reporting "signature does not match the peer id", which
// reads like an attack rather than a version change.
//
// Adding a field to an event therefore means adding a scheme here and leaving the
// old one intact. It does not mean editing signingBytesV2.
// protocolNamespace separates these signatures from any other use of the same
// keys. It is a NAMESPACE, not a name: it says which protocol a signature was
// made for, and nothing about what the software is called.
//
// It was the product name until the product name turned out not to be settled,
// which made every signature ever produced hostage to a naming decision — and by
// the rule below, a rename would then mean carrying the old namespace forever for
// a name nobody uses. Domain separation needs stability and uniqueness; it does
// not need meaning.
//
// **Never change this.** It is arbitrary on purpose, so there is never a reason to.
const protocolNamespace = "peer-room"

const currentSigVersion = 3

// signingBytes produces the bytes for the scheme an event was signed under, or an
// error if this build does not know that scheme. Refusing is correct: a signature
// this code cannot check is not a signature it may accept.
func (e *Event) signingBytes(version int) ([]byte, error) {
	switch version {
	case 0, 2:
		// 0 means an event stored before the version was recorded. Every such
		// event was signed under v2, which was the only scheme that existed.
		return e.signingBytesV2(), nil
	case 3:
		return e.signingBytesV3(), nil
	default:
		return nil, fmt.Errorf("event %.12s is signed under scheme v%d, which this build does not know; upgrade rather than discard it",
			e.EventID, version)
	}
}

// signingBytesV3 is v2 with the namespace no longer carrying a product name.
// v2 is kept below, untouched, which is the rule this file states and the first
// occasion to follow it.
func (e *Event) signingBytesV3() []byte {
	return e.eventBytes(protocolNamespace + "/event/v3")
}

func (e *Event) signingBytesV2() []byte {
	return e.eventBytes("claude-team/event/v2")
}

func (e *Event) eventBytes(tag string) []byte {
	var b bytes.Buffer
	put := func(s string) {
		_ = binary.Write(&b, binary.BigEndian, uint32(len(s)))
		b.WriteString(s)
	}
	put(tag)
	put(e.EventID)
	put(e.PeerID)
	_ = binary.Write(&b, binary.BigEndian, uint64(e.PeerSequence))
	put(e.RoomID)
	put(e.Timestamp)
	put(e.UserID)
	put(e.UserDisplayName)
	put(e.MachineID)
	put(e.OriginSessionID)
	put(e.EventType)
	put(e.Content)
	put(string(e.Metadata))
	return b.Bytes()
}

// Sign attaches a signature made by the event's originating peer.
func (e *Event) Sign(priv ed25519.PrivateKey) {
	e.SigVersion = currentSigVersion
	e.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, e.signingBytesV3()))
}

var errUnsigned = errors.New("event carries no signature")

// Verify checks an event against the key its own peer id names.
//
// This is what makes transitive relay safe rather than merely well-behaved (§13):
// an event arriving from Alice claiming to originate with David is now
// distinguishable from one Alice composed, because Alice cannot produce David's
// signature.
func (e *Event) Verify() error {
	if e.Signature == "" {
		return errUnsigned
	}
	pub, err := PublicFromPeerID(e.PeerID)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(e.Signature)
	if err != nil {
		return fmt.Errorf("signature is not valid base64url: %w", err)
	}
	msg, err := e.signingBytes(e.SigVersion)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, msg, sig) {
		return errors.New("signature does not match the peer id that claims to have made it")
	}
	return nil
}

// Fingerprint groups an identifier so a person can read it. It is a DISPLAY, not
// a ceremony, and nothing about looking at one marks a peer verified.
//
// There is exactly one way to verify a peer (§25): the two-word comparison in
// sas.go, over a live exchange. A second method would be a weaker way to satisfy
// the same gate, and a gate is only as strong as the weakest ceremony that
// satisfies it. This exists for the one place a person genuinely reads an
// identifier -- when a key has changed and there are two of them side by side.
func Fingerprint(peerID string) string {
	body := strings.TrimPrefix(peerID, keyPrefix)
	var out []string
	for i := 0; i < len(body); i += 4 {
		end := i + 4
		if end > len(body) {
			end = len(body)
		}
		out = append(out, body[i:end])
	}
	return strings.Join(out, " ")
}
