# D-160 — The address in a sync request is signed

**Date:** 2026-09-17 · **Status:** active · **Areas:** sync, transport, identity

**Decision.** A sync request carries the address its sender listens on, and the
signature covers that address along with the request's other signed fields.

**Support.**
- A peer acts on the address by polling it, so an unsigned one would let anybody
  redirect a peer's polling, which forges nothing and denies a great deal.
  `cmd/cogmer/auth.go`, `requestBytes`.
- The sender signs the address it advertises. `cmd/cogmer/sync.go`.
- The other signed fields are D-044 (sync requests are signed; authentication is not
  admission).

**Rejected.**
- *Leaving the address unsigned because it forges nothing.* It lets a third party send a
  peer's polling to an address of their choosing.

**Revisit when** a peer learns another's address by a path that does not carry a signed
request.
