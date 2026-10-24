# D-074 — A name means one master key, and a collision is where a new master key surfaces

**Date:** 2026-09-19 · **Status:** active · **Areas:** identity, pairing

**Decision.** A name a user gives a peer belongs to one master key, which is the peer's
identifier (D-042, peer identity is an Ed25519 key pair). `Allow` refuses a name held by a
different key and returns a `NameTakenError` that carries both keys. Its explanation says
that a changed key is indistinguishable from somebody else's key sent in their name, so the
user should check on a call before recording it, and that the way to record a new key is to
`forget` the old one first. A colleague's new machine or new shared key is no key change,
since its certificates end at the master key the name holds (D-219, a user is a master key
that certifies machines through a shared key), so only a new master key meets this rule.

**Support.**
- A peer's identifier is the user's master key, so a user who makes a new master key
  appears to this user as a peer never seen and is refused as a stranger, and nothing
  connects the new key to the name on the old one. D-042 (peer identity is an Ed25519 key
  pair).
- The one moment the two can be connected is when somebody records the new key under a name that is in use, which is also what a substitution looks like from the host's side, and
  without the rule it passed silently. `cmd/cogmer/membership_test.go`, `TestANameMeansOneKey`.
- Forgetting the old key discards its admissions, so the user invites the new one again on purpose and inherits no rooms. D-073 (forgetting a peer discards their admissions
  with them).

**Rejected.**
- *Two keys under one name.* A name would then admit whichever key was listed first,
  which is the failure verification exists to prevent.

**Limits.** The rule does not detect a key change by itself. It catches only a key filed
under a familiar name, and a peer that presents a new key is an unknown peer, refused
without ceremony, which is the limit of self-certifying identifiers.

**Revisit when** a user needs a new master key while colleagues keep the user's name and
admissions. A user identity is created once and lives until its recovery phrase is lost or
stolen (D-219), and a new master key would need a way for the old one to vouch for it.
