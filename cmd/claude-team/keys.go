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
// printed by `whoami` and is meant to be shared -- an identity a developer cannot
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
// here can never be replayed as one made over something else.
func (e *Event) signingBytes() []byte {
	var b bytes.Buffer
	put := func(s string) {
		_ = binary.Write(&b, binary.BigEndian, uint32(len(s)))
		b.WriteString(s)
	}
	put("claude-team/event/v1")
	put(e.EventID)
	put(e.PeerID)
	_ = binary.Write(&b, binary.BigEndian, uint64(e.PeerSequence))
	put(e.RoomID)
	put(e.Timestamp)
	put(e.UserID)
	put(e.UserDisplayName)
	put(e.MachineID)
	put(e.ClaudeSessionID)
	put(e.EventType)
	put(e.Content)
	put(string(e.Metadata))
	return b.Bytes()
}

// Sign attaches a signature made by the event's originating peer.
func (e *Event) Sign(priv ed25519.PrivateKey) {
	e.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, e.signingBytes()))
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
	if !ed25519.Verify(pub, e.signingBytes(), sig) {
		return errors.New("signature does not match the peer id that claims to have made it")
	}
	return nil
}

// Fingerprint renders a whole identifier for comparison by people (§25).
//
// Grouped for reading aloud. It covers the entire key: a short mnemonic catches an
// accident and not an adversary, because the bits it omits are free to differ.
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
