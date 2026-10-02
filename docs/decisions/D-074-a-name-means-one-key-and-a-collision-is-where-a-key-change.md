# D-074 — A name means one key, and a collision is where a key change surfaces

**Date:** 2026-09-19 · **Status:** active · **Areas:** identity, pairing

**Decision.** A name a user gives a peer belongs to one key. `Allow` refuses a name held
by a different key and returns a `NameTakenError` that carries both keys. Its explanation
says that a changed key is indistinguishable from somebody else's key sent in their name,
so the user should check on a call before recording it, and that the way to record a new
key is to `forget` the old one first.

**Support.**
- A peer's identifier is a key, so a person who changes keys appears to this machine as a
  peer it has never seen and is refused as a stranger, and nothing connects the new key to
  the name on the old one. D-042 (peer identity is an Ed25519 key pair).
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

**Revisit when** identity rotation is wanted. An identity is created once and lives until
its key is lost, and rotation would need a way for the old key to vouch for a new one.
