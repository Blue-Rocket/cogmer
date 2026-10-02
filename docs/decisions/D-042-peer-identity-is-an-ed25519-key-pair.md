# D-042 — Peer identity is an Ed25519 key pair

**Date:** 2026-09-17 · **Status:** active · **Areas:** identity

**Decision.** A peer's identifier is its Ed25519 public key, written `ed25519:<base64url>`,
not a fingerprint of it.

**Support.**
- An identifier that is the key can be checked against a signature with nothing to look
  up and no key to distribute. `cmd/cogmer/keys.go`.
- Knowing an identifier grants nothing, so it can be put in every event, interface and
  exchange. D-023 (a peer identifier must be safe to know).

**Rejected.**
- *A fingerprint of the key as the identifier.* Checking a signature would then need the
  key itself from somewhere else.

**Limits.** An identifier gives integrity and attribution, and never decides who may
connect or read a room, which is admission (D-045).

**Revisit when** a peer's key has to change while its identity does not.
