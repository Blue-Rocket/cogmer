# D-062 — Tailcat's own client allowlist is not used, so admission has one authority

**Date:** 2026-09-18 · **Status:** active · **Areas:** admission, transport, identity

**Decision.** cogmer does not set tailcat's `AllowedClients`. Whether anyone may enter is
decided by the guest list and by verification, and by nothing in the transport.

**Support.**
- A client allowlist would be a third list, keyed on a WireGuard key and not on a peer's
  identifier, that has to agree with `known_peers` and `room_guests`, and lists that can
  disagree give a room that works in one direction only. D-159 (an invitation carries the
  room and the inviter, and joining admits the inviter).
- Tailcat's WireGuard keypair is a transport identity. A peer's identifier is an Ed25519
  master key that a user verifies, and it certifies the machine keys that sign events, so
  the WireGuard key and the Ed25519 keys have separate jobs. D-042 (peer identity is an
  Ed25519 key pair), D-219 (a user is a master key that certifies machines through a shared
  key).
- A `tc` address reaches the door and admits nobody, since the guest list and the
  verification decide. D-026 (there is no join token at all), D-054 (verification gates
  synchronization and injection).

**Rejected.**
- *Restricting connections by key in tailcat as well.* It is a second authority that
  can disagree with the first.

**Limits.** The evaluation that led to adopting tailcat, with its measured sizes, its
dependency count and its risks, is
`f288913:docs/decisions/D-062-tailcat-evaluated-for-phase-15-a-good-fit-adopted-behind-an.md`.
D-068 (tailcat is the cross-network transport, behind our own dialer) holds the adoption.
