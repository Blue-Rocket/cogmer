# Design: the user identity that sits across machines

This is working material for the open item "A user's machines are separate identities to
colleagues, and a cloud session has no identity of its own to sign with" in `docs/open.md`,
and is deleted with it. It builds D-219 (a user is a master key that certifies machines
through a shared key), D-220 (a removal names the last event of a machine that counts),
D-221 (an event whose identity version the receiver lacks is withheld from the assistant
until fetched), D-222 (a machine tells every machine it knows of each machine it adds),
D-223 (the recovery phrase is typed only into the daemon's browser page), D-224 (a conflict
between identity records goes to the higher level, then to a removal, then to the lowest
hash), D-225 (a machine sends each removal and new shared key to every colleague until each
acknowledges), D-226 (a cloud session takes the highest valid snapshot from any source),
D-227 (the daemon pushes a new snapshot to the user's repo once the user has opted in) and
D-216 (a cloud session signs with a key of its own that the user's shared key vouches for).

The first run builds section 3 and section 4 as pure functions with no daemon. Section 6
lists the slices of the second run in the order of their dependencies. Section 7 lists the
choices this design makes where no decision settles the question, and `docs/open.md` lists
each of them as a question.

## 1. Keys

A user has three kinds of key, and a machine has one more.

| key | made from | held by | signs |
|---|---|---|---|
| master | the recovery phrase | nothing; derived while the phrase is in the page, then dropped | shared-key certificates, removals |
| shared key, version n | random, made when the master certifies it | each machine the user lets hold it, in a file of its own | machine certificates, removals |
| machine key | random, made once on the machine (D-141, a machine's key never leaves it) | the machine, in `identity.key` | events, sync requests, admission proofs, TLS, a self-removal |
| cloud key | random, made by the cloud setup on a machine | the cloud environment, as its one secret | session certificates, a self-removal |
| session key | random, made when a cloud session starts | the session's container | events, sync requests, admission proofs, TLS |

The user's identifier is the master public key, written as a peer identifier is
(`PeerIDFromPublic`, `ed25519:` and base64url). Every key here is Ed25519, and every
identifier of a key uses that form.

## 2. The recovery phrase and the master key

The phrase is 18 words. The first 16 encode 16 random bytes, which are 128 bits of entropy,
and the last two encode the first two bytes of the SHA-256 of those 16 bytes, as a check.
Each byte at an even position is a word from `sasEven` and each byte at an odd position a
word from `sasOdd`, the PGP word list that `sas.go` holds. The two lists alternate, so a
swapped or missing word gives a phrase that does not parse, and the check bytes catch a
wrong word that happens to land on the right list, with 1 chance in 65,536 of missing it.

Parsing ignores case and any run of whitespace. An error from parsing says which position
is wrong and never repeats a word, so that no error message, log line or test failure
carries part of a phrase.

The master key's seed is `HKDF-SHA256(secret = the 16 bytes, salt = none, info =
"peer-room/master-key/v1", 32 bytes)`, and the key is `ed25519.NewKeyFromSeed(seed)`. The
16 bytes have full entropy, so a slow function such as Argon2 would add a delay and no
security, and HKDF is in the standard library. The info string carries the namespace and
a version, so a different derivation is a new version beside this one.

## 3. Records and their wire forms

Every record is signed with Ed25519 over a length-prefixed encoding of its fields with a
leading purpose tag, as events are (D-152, events are signed at origin over length-prefixed
fields). A string is a 4-byte big-endian length and its bytes, and a number is 8 bytes
big-endian. Each record carries the number of its signing scheme outside the signature, as
an event does (D-058, signature schemes are kept, never replaced), and a scheme this build
does not know is refused with an error that says to upgrade. Each scheme below is
version 1.

A record's hash is the SHA-256 of its signing bytes. Ed25519 signatures are deterministic,
so a record has one hash whoever signs it, and the hash is the record's identifier in
lists, fetches and acknowledgements.

On the wire and in storage a record is JSON, with keys and signatures in base64url and
numbers as JSON numbers.

| record | tag | signed fields, in order | signed by |
|---|---|---|---|
| `SharedKeyCert` | `peer-room/shared-key-cert/v1` | user, version, shared key | master |
| `MachineCert` | `peer-room/machine-cert/v1` | user, shared version, machine key, kind, name, serial | shared key of that version |
| `SessionCert` | `peer-room/session-cert/v1` | user, cloud key, session key, expires | cloud key |
| `Removal` | `peer-room/removal/v1` | user, machine key, signer level, signer key, signer's shared version, count of cutoffs, then each room and sequence in room order | master, a shared key, or the machine itself |

A machine certificate's kind is `machine` or `cloud`. Only a `cloud` key may certify a
session key, and no certificate of either kind lets its key certify a machine. A shared key
version is at least 1, and versions need not be contiguous. A serial is a number the
issuing machine gives the certificate, one more than the highest it knows for the user, and
it is shown beside the machine's name. Two machines given one serial at the same time are
both valid, since a machine is identified by its key and never by its serial.

A removal's cutoffs are pairs of a room identifier and a sequence number, sorted by room,
with no room twice. The sequence is the removed machine's own sequence in that room
(D-029, the sequence lives with the identity), and its last counting event is the one with
that sequence. A room the removal does not name has a cutoff of 0 in it, so a removal that
names no room marks every event of the machine.

A session certificate's expiry is RFC 3339 in UTC. It is checked against the clock for a
live act, which is a connection, a sync request or an admission proof, and never for a
stored event, because an event relayed after the expiry must still verify (D-058).

### 3.1 The chain

A key speaks for a user when a chain of certificates ends at the user's master key:

```
master --SharedKeyCert(v)--> shared key v --MachineCert--> machine key
master --SharedKeyCert(v)--> shared key v --MachineCert(kind cloud)--> cloud key --SessionCert--> session key
```

A chain carries the shared-key certificate, the machine certificate, and a session
certificate only when the machine certificate's kind is `cloud`. Verifying it checks each
signature, that every record names the same user, that the machine certificate's shared
version is the shared certificate's version, and that the key at the end is the key that
signed whatever the chain came with. A chain from a cloud key that carries no session
certificate is refused, since a cloud key never signs an event itself (D-216).

### 3.2 An event signed by a machine on behalf of a user

This is the design for the trust and propagation slices, and no foundation code writes it.
An event under scheme v4 has the user's identifier in `PeerID`, a new field `Signer` naming
the machine or session key that signed it, and a new field `Chain` that carries the chain
and is not signed. The v4 signing bytes are the v3 fields under the tag
`peer-room/event/v4`, followed by the signer. The identity version the sender holds and
the hashes of the removals it has seen go in `Metadata`, which v3 already signs, under the
keys `idv` and `idr`. `PeerSequence` is the signer's own sequence in the room, so an event
is identified by its signer, room and sequence, and a sequence conflict (D-027) is between
two events of one signer. A v3 event verifies as it does now.

## 4. Resolving a user's records

A receiver holds a set of records for each user it knows. `Resolve` turns that set into the
state every receiver holding the same records computes, with no clock and no server.

1. Each shared-key certificate is checked against the master key. Of two valid
   certificates for one version with different keys, the one with the lowest hash wins
   (D-224, both are at the master level). The current version is the highest version with
   a winning certificate.
2. Each machine certificate is checked against the winning shared key of its version, and
   one signed by a shared key that lost in step 1 is discarded. Of several valid
   certificates for one machine key, the one under the highest version is the machine's,
   and of several under one version, the one with the lowest hash.
3. Each removal is checked against its signer. The master may remove any machine. A shared
   key whose certificate won in step 1, of any version, may remove any machine. A machine or
   cloud key may remove only itself, and only when it has a valid certificate. A session key
   may remove nothing.
4. A machine with at least one valid removal is removed. In each room its cutoff is the
   lowest any valid removal gives it, counting 0 for a removal that does not name the room
   (D-220, a cutoff only moves earlier).
5. A machine whose certificate is under a version below the current one is not current. It
   is listed with its certificate, and the slices decide what it may still do (question 4 in
   section 7).

Every record that fails a check or loses a conflict is returned with the reason, so that
the room page in the browser can show it and a test can name it.

D-224 orders the claims that conflict. A claim's level is the level of the key that signed
it: master, then shared key, then machine. A higher level prevails, then a removal over an
addition, then the lowest hash. `Prevails` implements that order once and `Resolve` uses it
for every conflict above. A removal and the certificate of the machine it removes do not
conflict, because a removal presupposes the certificate it ends (question 1 in section 7).

A machine's event counts when the machine is not removed, or when the event's sequence in
its room is at most the machine's cutoff there. An unknown machine's event does not count.

## 5. Go interfaces between the slices

The foundation is three files in `cmd/cogmer`, in package `main` like the rest. It reads no
file, opens no database and starts nothing.

```go
// userphrase.go: the phrase and the master key.
type Phrase [16]byte
func NewPhrase() (Phrase, error)
func (p Phrase) Words() []string                 // 18 words
func ParsePhrase(s string) (Phrase, error)        // errors never quote a word
func (p Phrase) MasterKey() ed25519.PrivateKey
func (p Phrase) UserID() string                   // PeerIDFromPublic of the master key

// idrecords.go: the records and the chain.
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
type Chain struct {
	Shared  SharedKeyCert `json:"shared"`
	Machine MachineCert   `json:"machine"`
	Session *SessionCert  `json:"session,omitempty"`
}

func CertifySharedKey(master ed25519.PrivateKey, version uint64, shared ed25519.PublicKey) SharedKeyCert
func CertifyMachine(shared ed25519.PrivateKey, user string, version uint64,
	machine ed25519.PublicKey, kind, name string, serial uint64) MachineCert
func CertifySession(cloud ed25519.PrivateKey, user string, session ed25519.PublicKey, expires time.Time) SessionCert
func SignRemoval(signer ed25519.PrivateKey, level Level, signerVersion uint64,
	user, machine string, cutoffs []RoomCutoff) (Removal, error) // sorts cutoffs, refuses a room twice

func (c SharedKeyCert) Hash() [32]byte  // and the same for MachineCert, SessionCert and Removal
func (c SharedKeyCert) Verify() error   // against the master key c.User names
func (c MachineCert) VerifyBy(shared ed25519.PublicKey) error
func (c SessionCert) VerifyBy(cloud ed25519.PublicKey) error

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
func (c Chain) Verify(user, signer string) (ChainLink, error)
func (l ChainLink) LiveAt(now time.Time) error // refuses an expired session certificate

// idstate.go: precedence and cutoffs.
type Claim struct {
	Level   Level
	Removal bool
	Hash    [32]byte
}
func Prevails(a, b Claim) bool // true when a wins over b under D-224

type RecordSet struct {
	Shared   []SharedKeyCert `json:"shared"`
	Machines []MachineCert   `json:"machines"`
	Removals []Removal       `json:"removals"`
}
type MachineState struct {
	Cert    MachineCert
	Current bool             // certified under the user's current shared version
	Removed bool
	Cutoffs map[string]int64 // room to last counting sequence; a room absent counts nothing
}
type Discarded struct {
	Hash   [32]byte
	Kind   string // "shared", "machine" or "removal"
	Reason string
}
type UserState struct {
	User       string
	Current    uint64            // 0 when no shared-key certificate is valid
	SharedKeys map[uint64]string // the winning shared key of each version
	Machines   map[string]*MachineState
	Removals   map[[32]byte]Removal // every valid removal, by hash
	Discarded  []Discarded
}
func Resolve(user string, rs RecordSet) *UserState
func (s *UserState) Counts(machine, room string, sequence int64) bool
func (s *UserState) List() []MachineCert  // the machines not removed, sorted by serial then key (D-222)
func (s *UserState) Check(c Chain, signer string) (ChainLink, error) // verifies c and refuses
	// a chain whose shared certificate lost, or whose machine is removed
func (s *UserState) Lacks(version uint64, removals [][32]byte) bool // D-221's test
```

`Removal` carries its signer's level because a receiver has to know which key to check the
signature against before it can check it. A removal whose level does not match its signer
fails in step 3.

## 6. The slices of the second run

Each slice is one agent in its own worktree. The order is the order of the dependencies,
and the slices on one line can run at the same time.

1. Storage and migration. A table of identity records in `membership.db`, keyed on the
   record's hash, holding records of every user this machine knows; the machine's own
   certificate and chain in `identity.json`; the shared key's private half in
   `shared.key`, a file of its own (D-153, the private key is kept in a file of its own);
   a `signer` column on `events`, migrated with `addColumnIfMissing` and filled with
   `peer_id` for v3 events. Tests: a store made before the migration opens and reads, and
   `shared.key` never appears in `identity.json` or `whoami`.
2. Trust. Event scheme v4 (section 3.2) beside v3, a sync request scheme carrying the chain,
   the chain in the TLS certificate as an extension that `pinnedVerifier` reads (D-101, peer
   connections are TLS pinned), admission by chain (D-143, admission is proved by signing a
   fresh challenge), and pairing that records and verifies a user's master key rather than
   a machine's key. Adding a machine of one's own belongs here. Tests: frozen v4 vectors
   beside the v3 ones, a machine the user added later syncs with a colleague who verified
   only the first, and a chain to an unverified master is refused.
3. Propagation, after trust. `idv` and `idr` in event metadata, a fetch route for identity
   records like `/blob`, the withhold rule (D-221) under the existing notice of a turn not
   yet given to the assistant, the list sync between a user's machines (D-222) and the push
   with acknowledgement to colleagues (D-225). Tests: an event naming a version the
   receiver lacks is shown and withheld until the record arrives from a third peer.
4. Removal and recovery, after propagation, at the same time as 5. The browser pages that
   take the phrase (D-223), removing a machine with a cutoff, certifying a new shared key,
   and marking a removed machine's later events in the view and never giving them to an
   assistant. Tests: the phrase reaches no command, argument or log, and an event past a
   cutoff never reaches a hook's output.
5. Cloud, after trust and propagation, at the same time as 4. The cloud key and session
   key (D-216), the snapshot and its sources (D-226), the push to the user's repo (D-227),
   the setup command, and fail-open behavior under `CLAUDE_CODE_REMOTE`. The three probes
   the brief names run in this slice.

Every slice that serves a room copies the two-room tests in `roomscope_test.go`.

## 7. Choices this design makes where no decision settles the question

1. D-224 read literally makes D-220's self-removal void. A self-removal is signed at the
   machine level and the certificate it removes at the shared level, so "the higher level
   wins" would keep every machine that removed itself. This design treats a removal and the
   certificate it removes as not in conflict, so every valid removal takes effect, and
   applies D-224 to two shared-key certificates for one version and to several certificates
   for one machine. Under this reading a removal always beats an addition, which agrees with
   D-224 for every pair except a self-removal and the certificate it removes.
2. The phrase is 18 words from the PGP word list, and the master key is HKDF-SHA256 of its
   16 bytes (section 2). The two words of the pairing comparison come from the same list,
   which a user might confuse with the phrase. The alternative is the BIP-39 English list of
   2,048 words, which would make the phrase 12 words and has to be added to the repository.
3. Records are length-prefixed and signed as events are, and carried as JSON (section 3).
4. What a machine that is not certified under the current shared version may still do.
   Its stored events keep counting, since nothing names a cutoff for them. The design leans
   to refusing its new connections, sync requests and events, which is what D-222's Limits
   means by an unknown machine that stops counting.
5. A removal signed by a shared key of an older version counts. A thief holding the old
   shared key can then mark every event of the user's machines, and the mark cannot be
   undone (D-220). Counting only removals signed under the current version would let a
   thief's removal fall away when the user certifies a new key, and also a user's own
   removal made before it.
6. Which of a user's machines hold the shared key. The design gives it to the first machine
   and to every machine added from a machine that holds it, and never to a cloud key.
7. Whether an event's author must be a guest of its room. The lean in the brief is to
   refuse an author who never was one, which needs a record of past guests. It belongs to
   the trust slice.
