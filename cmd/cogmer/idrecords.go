package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"time"
)

// The records that let a machine speak for a user (D-219, a user is a master key
// that certifies machines through a shared key). The master key certifies a
// shared key, the shared key certifies each machine or cloud key, and a cloud key
// certifies each session's key (D-216, a cloud session signs with a key of its
// own that the user's shared key vouches for).
//
// Each record is signed over length-prefixed fields behind a purpose tag, as an
// event is (D-152, events are signed at origin over length-prefixed fields), so
// no boundary can move and no record's signature can pass as another kind's.
// Each carries its scheme outside the signature and is verified under that
// scheme, never under the current one (D-058, signature schemes are kept, never
// replaced): these records travel with events that colleagues hold and relay, and
// none of them can be re-signed once the key that made it is gone.

// Level is the rank of the key that signed a claim. A conflict between claims is
// settled by it before anything else, so it is signed into a removal rather than
// inferred from the key.
type Level int

const (
	LevelMachine Level = 1 // a machine, cloud or session key
	LevelShared  Level = 2
	LevelMaster  Level = 3
)

const (
	KindMachine = "machine"
	KindCloud   = "cloud"
)

// idRecordScheme is the scheme every record below is signed under now. A change
// to the fields of any of them is a new scheme beside this one.
const idRecordScheme = 1

type SharedKeyCert struct {
	Scheme    int    `json:"scheme"`
	User      string `json:"user"`
	Version   uint64 `json:"version"`
	SharedKey string `json:"sharedKey"`
	Signature string `json:"signature"`
}

type MachineCert struct {
	Scheme        int    `json:"scheme"`
	User          string `json:"user"`
	SharedVersion uint64 `json:"sharedVersion"`
	Machine       string `json:"machine"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Serial        uint64 `json:"serial"`
	Signature     string `json:"signature"`
}

type SessionCert struct {
	Scheme    int    `json:"scheme"`
	User      string `json:"user"`
	CloudKey  string `json:"cloudKey"`
	Session   string `json:"session"`
	Expires   string `json:"expires"`
	Signature string `json:"signature"`
}

type RoomCutoff struct {
	Room     string `json:"room"`
	Sequence int64  `json:"sequence"`
}

type Removal struct {
	Scheme        int          `json:"scheme"`
	User          string       `json:"user"`
	Machine       string       `json:"machine"`
	SignerLevel   Level        `json:"signerLevel"`
	Signer        string       `json:"signer"`
	SignerVersion uint64       `json:"signerVersion"` // the shared version, when a shared key signed
	Cutoffs       []RoomCutoff `json:"cutoffs"`
	Signature     string       `json:"signature"`
}

// Chain is what a key sends to show that it speaks for a user. A session
// certificate is present only under a cloud key, since a cloud key never signs
// an event itself, and each session numbers its own events (D-216, a cloud
// session signs with a key of its own).
type Chain struct {
	Shared  SharedKeyCert `json:"shared"`
	Machine MachineCert   `json:"machine"`
	Session *SessionCert  `json:"session,omitempty"`
}

// ChainLink is what a verified chain says about the key at its end.
type ChainLink struct {
	User          string
	SharedVersion uint64
	Machine       string // the machine or cloud key certified by the shared key
	Kind          string
	Name          string
	Serial        uint64
	Signer        string // the key that signs: the machine key, or the session key
	Expires       string // a session certificate's expiry, empty otherwise
}

// idBytes builds signing bytes in the encoding events use: a string is its
// 4-byte big-endian length and its bytes, a number 8 bytes big-endian.
type idBytes []byte

func (b *idBytes) str(s string) {
	*b = binary.BigEndian.AppendUint32(*b, uint32(len(s)))
	*b = append(*b, s...)
}

func (b *idBytes) num(n uint64) { *b = binary.BigEndian.AppendUint64(*b, n) }

// unknownScheme words a scheme this build cannot check as a version problem. The
// same refusal worded as a bad signature sends a person hunting an attacker when
// the answer is to install a newer build (D-058, signature schemes are kept,
// never replaced).
func unknownScheme(what string, scheme int) error {
	return fmt.Errorf("%s is signed under scheme v%d, which this build does not know; upgrade rather than discard it",
		what, scheme)
}

func signID(priv ed25519.PrivateKey, msg []byte) string {
	return base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, msg))
}

func checkID(pub ed25519.PublicKey, msg []byte, sig, whose string) error {
	raw, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return fmt.Errorf("signature is not valid base64url: %w", err)
	}
	if !ed25519.Verify(pub, msg, raw) {
		return fmt.Errorf("signature does not match %s", whose)
	}
	return nil
}

// recordHash is the identifier of a record in lists, fetches and acknowledgements.
// A record under a scheme this build does not know has no signing bytes here,
// and so the zero hash. Nothing may use a hash before the record has verified,
// since the zero hash is the lowest and would win every tie.
func recordHash(msg []byte, err error) [32]byte {
	if err != nil {
		return [32]byte{}
	}
	return sha256.Sum256(msg)
}

// CertifySharedKey has the master key vouch for a shared key of one version.
func CertifySharedKey(master ed25519.PrivateKey, version uint64, shared ed25519.PublicKey) SharedKeyCert {
	c := SharedKeyCert{
		Scheme:    idRecordScheme,
		User:      PeerIDFromPublic(master.Public().(ed25519.PublicKey)),
		Version:   version,
		SharedKey: PeerIDFromPublic(shared),
	}
	msg, _ := c.signingBytes()
	c.Signature = signID(master, msg)
	return c
}

func (c SharedKeyCert) signingBytes() ([]byte, error) {
	switch c.Scheme {
	case 1:
		var b idBytes
		b.str(protocolNamespace + "/shared-key-cert/v1")
		b.str(c.User)
		b.num(c.Version)
		b.str(c.SharedKey)
		return b, nil
	default:
		return nil, unknownScheme(fmt.Sprintf("shared-key certificate v%d", c.Version), c.Scheme)
	}
}

func (c SharedKeyCert) Hash() [32]byte { return recordHash(c.signingBytes()) }

// Verify checks the certificate against the master key its user names, which is
// the only key a shared-key certificate can be signed by.
func (c SharedKeyCert) Verify() error {
	msg, err := c.signingBytes()
	if err != nil {
		return err
	}
	if c.Version == 0 {
		return errors.New("shared key version is 0; versions start at 1")
	}
	if _, err := PublicFromPeerID(c.SharedKey); err != nil {
		return fmt.Errorf("shared key: %w", err)
	}
	master, err := PublicFromPeerID(c.User)
	if err != nil {
		return fmt.Errorf("user: %w", err)
	}
	return checkID(master, msg, c.Signature, "the master key the certificate names")
}

// CertifyMachine has a shared key vouch for a machine or cloud key. The version
// is the shared key's own, which the certificate names because it does not name
// the shared key: a receiver finds the key by its version.
func CertifyMachine(shared ed25519.PrivateKey, user string, version uint64,
	machine ed25519.PublicKey, kind, name string, serial uint64) MachineCert {
	c := MachineCert{
		Scheme:        idRecordScheme,
		User:          user,
		SharedVersion: version,
		Machine:       PeerIDFromPublic(machine),
		Kind:          kind,
		Name:          name,
		Serial:        serial,
	}
	msg, _ := c.signingBytes()
	c.Signature = signID(shared, msg)
	return c
}

func (c MachineCert) signingBytes() ([]byte, error) {
	switch c.Scheme {
	case 1:
		var b idBytes
		b.str(protocolNamespace + "/machine-cert/v1")
		b.str(c.User)
		b.num(c.SharedVersion)
		b.str(c.Machine)
		b.str(c.Kind)
		b.str(c.Name)
		b.num(c.Serial)
		return b, nil
	default:
		return nil, unknownScheme(fmt.Sprintf("machine certificate %q", c.Name), c.Scheme)
	}
}

func (c MachineCert) Hash() [32]byte { return recordHash(c.signingBytes()) }

func (c MachineCert) VerifyBy(shared ed25519.PublicKey) error {
	msg, err := c.signingBytes()
	if err != nil {
		return err
	}
	if c.SharedVersion == 0 {
		return errors.New("shared key version is 0; versions start at 1")
	}
	if c.Kind != KindMachine && c.Kind != KindCloud {
		return fmt.Errorf("kind %.16q is neither %q nor %q", c.Kind, KindMachine, KindCloud)
	}
	if _, err := PublicFromPeerID(c.Machine); err != nil {
		return fmt.Errorf("machine key: %w", err)
	}
	return checkID(shared, msg, c.Signature, fmt.Sprintf("the shared key of version %d", c.SharedVersion))
}

// CertifySession has a cloud key vouch for one session's key until expires. The
// expiry is written in UTC to the second, so that every receiver reads the same
// instant from the same bytes.
func CertifySession(cloud ed25519.PrivateKey, user string, session ed25519.PublicKey, expires time.Time) SessionCert {
	c := SessionCert{
		Scheme:   idRecordScheme,
		User:     user,
		CloudKey: PeerIDFromPublic(cloud.Public().(ed25519.PublicKey)),
		Session:  PeerIDFromPublic(session),
		Expires:  expires.UTC().Format(time.RFC3339),
	}
	msg, _ := c.signingBytes()
	c.Signature = signID(cloud, msg)
	return c
}

func (c SessionCert) signingBytes() ([]byte, error) {
	switch c.Scheme {
	case 1:
		var b idBytes
		b.str(protocolNamespace + "/session-cert/v1")
		b.str(c.User)
		b.str(c.CloudKey)
		b.str(c.Session)
		b.str(c.Expires)
		return b, nil
	default:
		return nil, unknownScheme("session certificate", c.Scheme)
	}
}

func (c SessionCert) Hash() [32]byte { return recordHash(c.signingBytes()) }

// VerifyBy checks the signature and not the expiry. An event a session signed
// before its certificate expired is still that session's event when a peer
// relays it afterwards, so the clock is consulted only for a live act (LiveAt).
func (c SessionCert) VerifyBy(cloud ed25519.PublicKey) error {
	msg, err := c.signingBytes()
	if err != nil {
		return err
	}
	if c.CloudKey != PeerIDFromPublic(cloud) {
		return errors.New("certificate names a cloud key other than the one checking it")
	}
	if _, err := PublicFromPeerID(c.Session); err != nil {
		return fmt.Errorf("session key: %w", err)
	}
	return checkID(cloud, msg, c.Signature, "the cloud key the certificate names")
}

// SignRemoval signs a removal of machine at the signer's level. The cutoffs are
// sorted by room so that one set of cutoffs has one encoding and so one hash,
// whoever signs it and in whatever order they were given.
func SignRemoval(signer ed25519.PrivateKey, level Level, signerVersion uint64,
	user, machine string, cutoffs []RoomCutoff) (Removal, error) {
	sorted := append([]RoomCutoff{}, cutoffs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Room < sorted[j].Room })
	r := Removal{
		Scheme:        idRecordScheme,
		User:          user,
		Machine:       machine,
		SignerLevel:   level,
		Signer:        PeerIDFromPublic(signer.Public().(ed25519.PublicKey)),
		SignerVersion: signerVersion,
		Cutoffs:       sorted,
	}
	if err := r.checkShape(); err != nil {
		return Removal{}, err
	}
	msg, _ := r.signingBytes()
	r.Signature = signID(signer, msg)
	return r, nil
}

// checkShape refuses a removal whose meaning is ambiguous before its signature
// is worth checking. Strict order by room is what makes a room given twice, with
// two cutoffs to choose between, impossible to sign or accept.
func (r Removal) checkShape() error {
	switch r.SignerLevel {
	case LevelShared:
		if r.SignerVersion == 0 {
			return errors.New("removal signed by a shared key names shared version 0; versions start at 1")
		}
	case LevelMaster, LevelMachine:
		if r.SignerVersion != 0 {
			return fmt.Errorf("removal at level %d names shared version %d, which only a shared key's removal has",
				r.SignerLevel, r.SignerVersion)
		}
	default:
		return fmt.Errorf("removal signer level %d is not a level", r.SignerLevel)
	}
	if _, err := PublicFromPeerID(r.Machine); err != nil {
		return fmt.Errorf("removed machine: %w", err)
	}
	for i, c := range r.Cutoffs {
		if c.Room == "" {
			return fmt.Errorf("removal cutoff %d names no room", i)
		}
		if c.Sequence < 0 {
			return fmt.Errorf("removal cutoff for room %.12s is negative", c.Room)
		}
		if i > 0 && r.Cutoffs[i-1].Room >= c.Room {
			if r.Cutoffs[i-1].Room == c.Room {
				return fmt.Errorf("removal names room %.12s twice", c.Room)
			}
			return errors.New("removal cutoffs are not sorted by room")
		}
	}
	return nil
}

func (r Removal) signingBytes() ([]byte, error) {
	switch r.Scheme {
	case 1:
		var b idBytes
		b.str(protocolNamespace + "/removal/v1")
		b.str(r.User)
		b.str(r.Machine)
		b.num(uint64(r.SignerLevel))
		b.str(r.Signer)
		b.num(r.SignerVersion)
		b.num(uint64(len(r.Cutoffs)))
		for _, c := range r.Cutoffs {
			b.str(c.Room)
			b.num(uint64(c.Sequence))
		}
		return b, nil
	default:
		return nil, unknownScheme("removal", r.Scheme)
	}
}

func (r Removal) Hash() [32]byte { return recordHash(r.signingBytes()) }

// VerifyBy checks the removal's shape and that pub, the key it names as its
// signer, made it. Whether that key may remove this machine at the level it
// claims depends on the user's other records, and is decided where they are.
func (r Removal) VerifyBy(pub ed25519.PublicKey) error {
	msg, err := r.signingBytes()
	if err != nil {
		return err
	}
	if err := r.checkShape(); err != nil {
		return err
	}
	if r.Signer != PeerIDFromPublic(pub) {
		return errors.New("removal names a signer other than the key checking it")
	}
	return checkID(pub, msg, r.Signature, "the signer the removal names")
}

// Verify checks that the chain ends at user's master key and that its last key
// is signer, the key that signed whatever the chain came with. It does not read
// the clock, so that a relayed event verifies after its session expired; a live
// act also calls LiveAt.
func (c Chain) Verify(user, signer string) (ChainLink, error) {
	s := c.Shared
	if s.User != user {
		return ChainLink{}, errors.New("shared-key certificate is for another user")
	}
	if err := s.Verify(); err != nil {
		return ChainLink{}, fmt.Errorf("shared-key certificate: %w", err)
	}
	sharedPub, _ := PublicFromPeerID(s.SharedKey)

	m := c.Machine
	if m.User != user {
		return ChainLink{}, errors.New("machine certificate is for another user")
	}
	if m.SharedVersion != s.Version {
		return ChainLink{}, fmt.Errorf("machine certificate is under shared version %d and the chain carries version %d",
			m.SharedVersion, s.Version)
	}
	if err := m.VerifyBy(sharedPub); err != nil {
		return ChainLink{}, fmt.Errorf("machine certificate: %w", err)
	}

	link := ChainLink{
		User:          user,
		SharedVersion: s.Version,
		Machine:       m.Machine,
		Kind:          m.Kind,
		Name:          m.Name,
		Serial:        m.Serial,
		Signer:        m.Machine,
	}
	switch m.Kind {
	case KindMachine:
		if c.Session != nil {
			return ChainLink{}, errors.New("session certificate under a machine key, which only a cloud key may issue")
		}
	case KindCloud:
		ss := c.Session
		if ss == nil {
			return ChainLink{}, errors.New("session certificate missing: a cloud key never signs for itself")
		}
		if ss.User != user {
			return ChainLink{}, errors.New("session certificate is for another user")
		}
		if ss.CloudKey != m.Machine {
			return ChainLink{}, errors.New("session certificate is from a cloud key other than the one the machine certificate names")
		}
		cloudPub, _ := PublicFromPeerID(m.Machine)
		if err := ss.VerifyBy(cloudPub); err != nil {
			return ChainLink{}, fmt.Errorf("session certificate: %w", err)
		}
		link.Signer = ss.Session
		link.Expires = ss.Expires
	}
	if link.Signer != signer {
		return ChainLink{}, fmt.Errorf("chain ends at %.24s, not at the key that signed", link.Signer)
	}
	return link, nil
}

// LiveAt refuses a session whose certificate has expired, for a connection, a
// sync request or an admission proof. An expiry that does not parse is refused
// rather than read as never expiring.
func (l ChainLink) LiveAt(now time.Time) error {
	if l.Kind != KindCloud {
		return nil
	}
	exp, err := time.Parse(time.RFC3339, l.Expires)
	if err != nil {
		return fmt.Errorf("session certificate expiry %.32q is not RFC 3339", l.Expires)
	}
	if !now.Before(exp) {
		return fmt.Errorf("session certificate expired at %s", l.Expires)
	}
	return nil
}

// ListChange is one change to an entry of a user's copied lists: a colleague's
// label or verification, or a room on the user's room list (D-228, a user's
// colleagues, verifications, labels and rooms belong to the user and are copied
// between the user's machines). Value is opaque here, since what an entry holds
// is the caller's to define and only its bytes need to be signed. The machine
// that made the change signs it, and the caller checks that machine's chain:
// this record says which key signed it, not whether that key speaks for the user.
type ListChange struct {
	Scheme    int    `json:"scheme"`
	User      string `json:"user"`
	Entry     string `json:"entry"`
	Value     string `json:"value"`
	Counter   uint64 `json:"counter"`
	Signer    string `json:"signer"`
	Signature string `json:"signature"`
}

// SignListChange signs a change to entry. The counter is one more than the
// highest the machine has seen for the entry, which the caller knows and this
// does not.
func SignListChange(machine ed25519.PrivateKey, user, entry, value string, counter uint64) ListChange {
	c := ListChange{
		Scheme:  idRecordScheme,
		User:    user,
		Entry:   entry,
		Value:   value,
		Counter: counter,
		Signer:  PeerIDFromPublic(machine.Public().(ed25519.PublicKey)),
	}
	msg, _ := c.signingBytes()
	c.Signature = signID(machine, msg)
	return c
}

func (c ListChange) signingBytes() ([]byte, error) {
	switch c.Scheme {
	case 1:
		var b idBytes
		b.str(protocolNamespace + "/list-change/v1")
		b.str(c.User)
		b.str(c.Entry)
		b.str(c.Value)
		b.num(c.Counter)
		b.str(c.Signer)
		return b, nil
	default:
		return nil, unknownScheme("list change", c.Scheme)
	}
}

func (c ListChange) Hash() [32]byte { return recordHash(c.signingBytes()) }

// VerifyBy checks that pub, the key the change names as its signer, made it. A
// counter of 0 is refused because every counter is one more than one already
// seen, so 0 comes only from a change that skipped that rule.
func (c ListChange) VerifyBy(pub ed25519.PublicKey) error {
	msg, err := c.signingBytes()
	if err != nil {
		return err
	}
	if c.Entry == "" {
		return errors.New("list change names no entry")
	}
	if c.Counter == 0 {
		return errors.New("list change has counter 0; counters start at 1")
	}
	if c.Signer != PeerIDFromPublic(pub) {
		return errors.New("list change names a signer other than the key checking it")
	}
	return checkID(pub, msg, c.Signature, "the signer the list change names")
}

// RoomIDFor is the identifier of the room creator founds with nonce. It is a
// hash of the two, so only the creator's self-invitation carrying that nonce
// hashes to it, and nobody who learns the identifier can admit themselves by
// inviting themselves (D-240, a room's identifier is the hash of its creator
// and a random value). The fields are length-prefixed behind a tag, as a
// record's are, so no other creator and nonce run together into the same bytes.
// The tag is peer-room/room-id/v1, written in two pieces only because the check
// for slash commands named without the plugin's namespace would read it whole as
// one.
func RoomIDFor(creator string, nonce []byte) string {
	var b idBytes
	b.str(protocolNamespace + "/" + "room-id/v1")
	b.str(creator)
	b.str(string(nonce))
	h := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(h[:])
}
