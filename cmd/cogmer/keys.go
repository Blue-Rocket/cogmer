package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

func sharedKeyPath() string { return sharedKeyPathIn(homeDir()) }

func sharedKeyPathIn(dir string) string { return filepath.Join(dir, "shared.key") }

// sharedKeyFile is the form of shared.key: the private half of the user's shared
// key and the version the master key certified it under.
type sharedKeyFile struct {
	Version    uint64 `json:"version"`
	PrivateKey string `json:"privateKey"`
}

// loadSharedKey reads the user's shared key, reporting false when this machine
// holds none. The shared key certifies the user's machines (D-219, a user is a
// master key that certifies machines through a shared key), so it is kept in a
// file of its own, apart from identity.json, which whoami prints and a user may
// hand to a colleague (D-153, the private key is kept in a file of its own).
func loadSharedKey() (ed25519.PrivateKey, uint64, bool, error) {
	return loadSharedKeyIn(homeDir())
}

// loadSharedKeyIn is loadSharedKey for the state directory dir.
func loadSharedKeyIn(dir string) (ed25519.PrivateKey, uint64, bool, error) {
	path := sharedKeyPathIn(dir)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, 0, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	var f sharedKeyFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, 0, false, fmt.Errorf("%s is not a usable shared key", path)
	}
	key, err := base64.RawURLEncoding.DecodeString(f.PrivateKey)
	if err != nil || len(key) != ed25519.PrivateKeySize {
		return nil, 0, false, fmt.Errorf("%s is not a usable shared key", path)
	}
	if f.Version == 0 {
		return nil, 0, false, fmt.Errorf("%s names shared key version 0; versions start at 1", path)
	}
	return ed25519.PrivateKey(key), f.Version, true, nil
}

// saveSharedKey writes the user's shared key, readable by this account alone. It
// writes a temporary file and renames it over the old one, so that a crash leaves
// either the old key or the new one and never half of either.
func saveSharedKey(priv ed25519.PrivateKey, version uint64) error {
	return saveSharedKeyIn(homeDir(), priv, version)
}

// saveSharedKeyIn is saveSharedKey for the state directory dir.
func saveSharedKeyIn(dir string, priv ed25519.PrivateKey, version uint64) error {
	if len(priv) != ed25519.PrivateKeySize {
		return errors.New("a shared key to save is not an Ed25519 private key")
	}
	if version == 0 {
		return errors.New("shared key version is 0; versions start at 1")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	buf, err := json.Marshal(sharedKeyFile{Version: version, PrivateKey: base64.RawURLEncoding.EncodeToString(priv)})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "shared.key.*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(buf, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), sharedKeyPathIn(dir))
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
// Signature schemes are added, never edited.
//
// An event is immutable (§7) and therefore can never be re-signed, and old events
// do not merely sit in a database: §13 relays them between peers, and D-029 makes
// refetching a room's history the designed recovery from local loss. So an event
// signed years ago must still verify under the scheme that signed it, or the
// recovery path fails -- reporting "signature does not match the peer id", which
// reads like an attack rather than a version change.
//
// Adding a field to an event therefore means adding a scheme beside the current
// one and leaving that one untouched. However few schemes the switch below lists,
// editing one is never the answer: the events it must still verify are on
// somebody else's machine, where this code cannot see them and cannot re-sign
// them.
// protocolNamespace separates these signatures from any other use of the same
// keys. It is a NAMESPACE, not a name: it says which protocol a signature was
// made for, and nothing about what the software is called.
//
// It carried the product name once, which made every signature ever produced
// hostage to a naming decision — and by the rule above, a rename would then mean
// carrying the superseded namespace forever. Domain separation needs stability
// and uniqueness; it does not need meaning.
//
// **Never change this.** It is arbitrary on purpose, so there is never a reason to.
const protocolNamespace = "peer-room"

const currentSigVersion = 3

// signingBytes produces the bytes for the scheme an event was signed under, or an
// error if this build does not know that scheme. Refusing is correct: a signature
// this code cannot check is not a signature it may accept.
func (e *Event) signingBytes(version int) ([]byte, error) {
	switch version {
	case 3:
		return e.signingBytesV3(), nil
	case 4:
		return e.signingBytesV4(), nil
	default:
		return nil, fmt.Errorf("event %.12s is signed under scheme v%d, which this build does not know; upgrade rather than discard it",
			e.EventID, version)
	}
}

func (e *Event) signingBytesV3() []byte {
	return e.eventBytes(protocolNamespace + "/event/v3")
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

// signingBytesV4 is v3's field list under a tag of its own, followed by the key
// that signed, the shared-key version that key's machine holds and the hashes of
// the removals it has seen, each a signed field (D-239, an event a machine signs for
// its user names the user, the signing key, the shared-key version and the removals
// seen, all as signed fields). A number is 8 bytes big-endian, as in the identity
// records (D-238, identity records are signed over length-prefixed fields), and the count of removals comes before them so that no boundary
// between the hashes can move.
func (e *Event) signingBytesV4() []byte {
	b := e.eventBytes(protocolNamespace + "/event/v4")
	put := func(s string) {
		b = binary.BigEndian.AppendUint32(b, uint32(len(s)))
		b = append(b, s...)
	}
	put(e.Signer)
	b = binary.BigEndian.AppendUint64(b, e.IdentityVersion)
	b = binary.BigEndian.AppendUint64(b, uint64(len(e.RemovalsSeen)))
	for _, h := range e.RemovalsSeen {
		put(h)
	}
	return b
}

// Sign attaches a signature made by the event's originating peer, under v3, where
// the peer's own key signs.
func (e *Event) Sign(priv ed25519.PrivateKey) {
	e.SigVersion = currentSigVersion
	e.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, e.signingBytesV3()))
}

// SignV4 signs the event under v4 with a machine's key on behalf of the user its
// chain ends at. PeerID must already name that user. The removals seen are sorted,
// so that two machines that have seen the same removals sign the same field.
func (e *Event) SignV4(priv ed25519.PrivateKey, chain Chain, removalsSeen []string) {
	e.SigVersion = 4
	e.Signer = PeerIDFromPublic(priv.Public().(ed25519.PublicKey))
	e.IdentityVersion = chain.Shared.Version
	e.RemovalsSeen = nil
	if len(removalsSeen) > 0 {
		e.RemovalsSeen = append([]string(nil), removalsSeen...)
		sort.Strings(e.RemovalsSeen)
	}
	c := chain
	e.Chain = &c
	e.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, e.signingBytesV4()))
}

// SignAs signs an event this machine writes: under v4 when the machine speaks for a
// user, and under v3 otherwise, so that a machine without a user identity signs
// as it always has.
func (e *Event) SignAs(id *Identity, removalsSeen []string) {
	if id.HasUser() {
		e.SignV4(id.private, *id.Chain, removalsSeen)
		return
	}
	e.Sign(id.private)
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
	if e.SigVersion == 4 {
		return e.verifyV4()
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

// verifyV4 checks the signature against the key the event names as its signer,
// then that the event's chain ties that key to the user its peer names (D-152,
// events are signed at origin over length-prefixed fields; D-219, a user is a master key that certifies machines through a shared key). Whether the
// user's other records remove or suspend the machine is the receiver's question,
// since it needs those records.
func (e *Event) verifyV4() error {
	pub, err := PublicFromPeerID(e.Signer)
	if err != nil {
		return fmt.Errorf("signer: %w", err)
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
		return errors.New("signature does not match the signer that claims to have made it")
	}
	if e.Chain == nil {
		return errors.New("an event signed for a user carries no chain to that user")
	}
	if _, err := e.Chain.Verify(e.PeerID, e.Signer); err != nil {
		return fmt.Errorf("chain: %w", err)
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
