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

It also builds the decisions that make the user, and not the machine, the peer: D-228 to
D-239, which section 7 names.

The first run built sections 2 to 4 as pure functions with no daemon. Section 6 lists the
slices of the second run in the order of their dependencies. Section 7 records the choices
the maintainer made. Section 8 proposes a choice for each detail the decisions leave open,
for the maintainer to confirm before the slice that needs it starts.

## 1. Keys

A user has three kinds of key, and a machine has one more.

| key | made from | held by | signs |
|---|---|---|---|
| master | the recovery phrase | nothing; derived while the phrase is in the page, then dropped | shared-key certificates, removals |
| shared key, version n | random, made when the master certifies it | every machine the user adds, never a cloud key (D-219), in a file of its own | machine certificates, removals |
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
`peer-room/event/v4`, followed by the signer, the shared-key version the sender holds and the
hashes of the removals it has seen, each a signed field of its own (D-239). `PeerSequence` is the signer's own sequence in the room, so an event
is identified by its signer, room and sequence, and a sequence conflict (D-027) is between
two events of one signer. A v3 event verifies as it does now.

### 3.3 Changes to a room's guest list

A room's guest list is the result of invitation and revocation events in the room itself
(D-235, a room's one guest list decides whose events count). They are v4 events like any
turn, signed by a guest's machine, sequenced and relayed with the room's other events, and
never given to an assistant.

| event type | content, as JSON |
|---|---|
| `GUEST_INVITE` | the invited user's identifier, and the hash of the revocation it lifts when it lifts one |
| `GUEST_REVOKE` | the revoked user's identifier, the event identifier of the invitation it ends, and a cutoff for each of the revoked user's machines: the machine's key and the last sequence that counts (D-236) |

The room's first event is the creator's invitation of themselves, so that the creator is on
the list by the same rule as everyone else and can be revoked like anyone.

### 3.4 Records a user's machines copy between them

A user's colleagues, verifications, labels and room list belong to the user (D-228, a
user's lists are copied between the user's machines). Each change is a `ListChange` record,
signed by the machine that made it under its chain and carried with the identity records
between the user's machines (D-222).

| field | meaning |
|---|---|
| user | the user whose list this is |
| entry | what the change is about: `colleague:<identifier>` or `room:<identifier>` |
| value | the entry as the change leaves it: a colleague's label, whether and when they were verified, or forgotten; a room's name and the user's machines that joined it |
| counter | one more than the highest counter the machine has seen for that entry |
| signer, chain, signature | as for any identity record (D-238) |

How two changes to one entry merge is detail 1 in section 8.

### 3.5 The two words when adding a machine

Adding a machine runs the commit-then-reveal exchange of a pairing (D-048) under the purpose
tags `peer-room/sas-machine/v1` and `peer-room/sas-machine-commit/v1`, over the two machine
keys and two fresh nonces (D-229, a machine is added from another of the user's machines).
The tags differ from a colleague pairing's, so neither exchange can pass for the other.

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
3. Each removal is checked against its signer. The master may remove any machine. The shared
   key of the current version may remove any machine, and a removal signed by an older shared
   key does not count, so the machine that certifies a new shared key signs again under it the
   removals the user confirms (D-220). A machine or
   cloud key may remove only itself, and only when it has a valid certificate. A session key
   may remove nothing.
4. A machine with at least one valid removal is removed. In each room its cutoff is the
   lowest any valid removal gives it, counting 0 for a removal that does not name the room
   (D-220, a cutoff only moves earlier).
5. A machine whose certificate is under a version below the current one is not current, and
   is suspended: none of its events count, and `Check` refuses its chain, until it is certified
   under the current version (D-234).

Every record that fails a check or loses a conflict is returned with the reason, so that
the room page in the browser can show it and a test can name it.

D-224 orders the claims that conflict. A claim's level is the level of the key that signed
it: master, then shared key, then machine. A higher level prevails, then a removal over an
addition, then the lowest hash. `Prevails` implements that order once and `Resolve` uses it
for every conflict above. A removal and the certificate of the machine it removes do not
conflict, because a removal presupposes the certificate it ends (question 1 in section 7).

A machine's event counts when the machine is current, and either not removed or the event's
sequence in its room is at most the machine's cutoff there. An unknown or suspended
machine's event does not count.

### 4.1 Resolving a room's guest list

`ResolveGuests` turns a room's `GUEST_INVITE` and `GUEST_REVOKE` events into the room's
list, with the same purity as `Resolve`: every machine holding the same events computes the
same list. An invitation admits its user unless a revocation names it; a later invitation
naming that revocation admits them again. A revoked user's machines carry the lowest cutoff
any revocation gave them (D-236). Whose changes count when their makers are revoked by
changes made at the same time is detail 10 in section 8.

An event reaches the store, the view and an assistant only when its author is on the list,
or is past it but within a cutoff, and its machine counts under `Resolve` (D-234, D-235).

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

// Added in the second run's first slice.
func (s *UserState) CheckEvent(c Chain, signer string) (ChainLink, error) // as Check, without
	// refusing a removed machine, whose events count up to its cutoff
func ResolveGuests(room string, events []Event) *GuestList
type GuestList struct {
	Admitted map[string]bool             // user identifier to whether the user is on the list
	Cutoffs  map[string]map[string]int64 // revoked user to machine key to last counting sequence
	Changes  []GuestChange               // who invited or revoked whom, for the view
}
func (g *GuestList) Counts(user, machine string, sequence int64) bool
func MergeListChanges(changes []ListChange) map[string]ListChange // detail 1 in section 8
```

`Removal` carries its signer's level because a receiver has to know which key to check the
signature against before it can check it. A removal whose level does not match its signer
fails in step 3.

## 6. The slices of the second run

Each slice is one agent in its own worktree, started from the last commit of
`user-identity` (`worktree.baseRef` is `"head"`). The order is the order of the
dependencies, and at most three slices run at once.

0. Foundation additions. `CheckEvent`, `ResolveGuests` and `GuestList`, and
   `MergeListChanges` once detail 1 is confirmed, as pure functions with tests, beside the
   foundation in `idstate.go`. Tests: two guests' machines holding one set of guest-list
   events in different orders compute one list; a removed machine's event within its cutoff
   passes `CheckEvent`.
1. Storage and migration, after 0. A table of identity records and `ListChange` records in
   `membership.db`, keyed on the record's hash; a machine table holding each machine's
   address, replacing the address on `known_peers` (D-103, an address belongs to a machine);
   the machine's own chain in `identity.json` and the shared key in `shared.key` (D-153);
   `signer` on `events`, filled with `peer_id` for v3 events; a cache of each room's guest
   list computed from its events. Tests: a store made before the migration opens and reads,
   and `shared.key` never appears in `identity.json` or `whoami`.
2. Trust, after 1. Event scheme v4 (D-239) beside v3; a sync request scheme carrying the
   chain; the chain in the TLS certificate as an extension that `pinnedVerifier` reads
   (D-101); admission by chain (D-143); pairing that records and verifies a colleague's
   master key, with the words over the two users' identifiers (D-048); the author check
   against the room's guest list (D-235) and revocation with its cutoff (D-236); forgetting
   that leaves rooms alone (D-073). Tests: frozen v4 vectors beside the v3 ones; a machine
   the user added later syncs with a colleague who verified only the first; a chain to an
   unverified master is refused; a revoked guest's turn past the cutoff is not stored.
3. A user's own machines, after 2, beside 4. The identity page (D-230) with the phrase and
   its confirmation; adding a machine with the machine words (D-229); `self-machines` and the
   machine list in `self-status` (D-231); copying `ListChange` records (D-228); joining a room
   by name from the shared room list; syncing a room between the user's machines that joined
   it (D-103, D-222). Tests: a colleague verified on one machine is verified on the other; a
   session on a second machine joins by name and receives the room; the two machines' turns
   reach each other with no colleague online.
4. Propagation, after 2, beside 3. The shared-key version and removals an event names
   (D-239); a fetch route for identity records like `/blob`; the withhold rule (D-221) under
   the existing notice of a turn not yet given to the assistant; the push with
   acknowledgement to colleagues (D-225). Tests: an event naming a version the receiver
   lacks is shown and withheld until the record arrives from a third machine.
5. Removal, recovery and what the user sees, after 3 and 4, beside 6. Removing a machine
   with a cutoff and certifying a new shared key with the user confirming each removal to
   carry (D-220); suspension (D-234); recovery from the phrase (D-223); the user's own turns
   marked as theirs (D-232); machine names (D-233); "mine" by user (D-198); reachability
   per colleague and per own machine (D-194). Tests: the phrase reaches no command, argument
   or log; a suspended machine's turn appears in no view, export or hook output; an event
   past a cutoff never reaches a hook's output.
6. Cloud, after 3 and 4, beside 5. The cloud key and session key (D-216), the snapshot and
   its sources (D-226), the push to the user's repo (D-227), the setup command, and
   fail-open behavior under `CLAUDE_CODE_REMOTE`. The three probes the brief names run in
   this slice.

Every slice that serves a room copies the two-room tests in `roomscope_test.go`. A slice's
decisions become active, and the specification already states them, when its behavior is
reachable.

## 7. Choices the maintainer made

The first run's seven questions were decided on 2026-10-24: a removal and the certificate it
removes do not conflict (D-224); the phrase is 18 words of the PGP word list (D-237); records
are signed over length-prefixed fields and carried as JSON, and an event names its signing
key, shared-key version and removals seen as signed fields (D-238, D-239); a machine not
certified under the current shared key is suspended (D-234); only the current shared key
removes, and a new key carries only the removals the user confirms (D-220); every machine the
user adds holds the shared key (D-219); and a room has one guest list that any guest changes
(D-235, D-236).

## 8. Details the decisions leave open, with a proposed choice

These are the details in `docs/open.md`, each with the choice this design proposes and the
slice that needs it. Each is put to the maintainer before that slice starts.

1. Merging changes to a user's copied lists (slice 0). The change with the highest counter
   for an entry wins, and the lowest hash between two with one counter, so every machine
   computes the same value with no clock.
2. Naming a machine and the display name (slice 3). The page asks for a machine's name when
   the machine is added, with no default, and asks for the display name with an empty field,
   since a guessed name is never presented as a choice.
3. Acknowledgement by a colleague (slice 4). One machine of the colleague acknowledging
   counts, since it copies the record to the colleague's other machines with the colleague's
   identity records.
4. Offers (slice 3). An offer goes to every machine of the guest the inviting machine knows,
   and an offered room enters the guest's shared room list as offered, so the guest can join
   from any machine.
5. The user's own turn that no session here holds (slice 5). It is marked as not given to an
   assistant, as a colleague's is, since it is injected like one.
6. A removal outside a theft (slice 5). The page proposes, for each room, the removed
   machine's last turn this machine has seen, and the user confirms it or chooses to mark all.
7. Recovery and machines the user no longer holds (slice 5). Certifying a new shared key
   suspends every machine not certified again (D-234), so no separate removal is needed.
8. Which machine a sync connection reaches (slice 2). Any machine of a guest is accepted, and
   the request's signature says which.
9. A machine with no browser (slice 3). Out of scope for the second run, since a cloud
   session needs no phrase (D-223) and every other machine a user adds has a browser.
10. Guests revoking each other at the same time (slice 0). Every revocation by a user who was
    ever invited counts, so two guests revoking each other are both revoked, and a revocation
    names one invitation, leaving another invitation of the same user standing.
11. The creator's admission (slice 0). The room's first event is the creator's invitation of
    themselves (section 3.3).
12. Turns past a lifted cutoff (slice 2). They are fetched again by ordinary sync, since a
    receiver never stored them and they are still offered by machines that hold them.
13. A forgotten guest on a room's list, and a cached list (slices 1 and 5). The list shows
    the guest by derived name, marked unverified, and `membership.db` caches each room's list
    computed from its events, rebuilt whenever they change.
