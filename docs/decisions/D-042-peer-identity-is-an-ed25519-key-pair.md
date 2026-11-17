# D-042 — Peer identity is an Ed25519 key pair

**Date:** 2026-09-17 · **Status:** active · **Areas:** identity

**Decision.** A peer's identifier is the peer's master public key, an Ed25519 key written
`ed25519:<base64url>`, not a fingerprint of it (D-219, a user is a master key that
certifies machines through a shared key). A machine signs with an Ed25519 key of its own,
and the certificates it sends with what it signs tie that key to the identifier, so a
receiver checks the signature against the machine's key and the chain against the
identifier.

**Support.**
- An identifier that is the key can be checked against a signature or a certificate with
  nothing to look up and no key to distribute. `cmd/cogmer/keys.go`.
- Knowing an identifier grants nothing, so it can be put in every event, interface and
  exchange. D-023 (a peer identifier must be safe to know).

**Rejected.**
- *A fingerprint of the key as the identifier.* Checking a signature would then need the
  key itself from somewhere else.

**Limits.** An identifier gives integrity and attribution, and never decides who may
connect or read a room, which is admission (D-045, rooms are records with a guest list,
and admission is enforced).

**Revisit when** a user's master key has to change while colleagues go on knowing the user
as the same peer.
