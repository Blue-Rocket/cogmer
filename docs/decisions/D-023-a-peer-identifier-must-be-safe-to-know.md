# D-023 — A peer identifier must be safe to know

**Date:** 2026-09-16 · **Status:** active

**Decision.** A peer's identifier is its public key, so knowing it grants nothing. An
identifier is safe to know only because signatures are checked: every event is signed by
the peer that created it and rejected on receipt if it does not verify.

**Support.**
- An identifier appears in every event its peer creates, in every interface and in every
  exchange between peers, so an identifier that gave its holder any power could not be
  protected. §6 (identity).
- Peer identity is an Ed25519 key pair, and events are signed at origin. D-042 (peer
  identity is an Ed25519 key pair) and D-152 (events are signed at origin over
  length-prefixed fields).

**Rejected.**
- *Treating the identifier as sensitive.* It appears in every event, and a secret that
  must be broadcast is not a secret.
- *Introducing key pairs before signatures were checked.* A peer could still assert
  another's identifier and be believed, so it would look like the fix and deliver none
  of it.

**Limits.** An identifier proves possession of a key, and never which person holds it.
Verification establishes that. D-054 (verification gates synchronization and
injection).

**Revisit when** an identifier has to carry something other than the public key.
