# D-182 — A display name a peer asserts for itself has no uniqueness constraint

**Date:** 2026-09-20 · **Status:** active · **Areas:** identity

**Decision.** The name a peer gives itself carries no uniqueness constraint, and two peers may
assert the same one.

**Support.**
- It is the peer's name for itself, and two colleagues may both be David. The first two-peer
  run had both daemons assert one because both ran under the same operating system user.
  `cmd/cogmer/identity.go`.
- Nothing keys on a display name, and the derived name, which comes from the key, is the
  authoritative one. D-017 (a room's identifier is authoritative, and its name is not),
  D-021 (peer names are word pairs derived from the identity, never chosen).

**Rejected.**
- *A constraint that refuses a colliding display name.* A colleague would be refused for
  having the same name as another, and the name proves nothing about who they are.
