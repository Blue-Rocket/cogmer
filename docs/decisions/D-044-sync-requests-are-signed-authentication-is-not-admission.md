# D-044 — Sync requests are signed; authentication is not admission

**Date:** 2026-09-17 · **Status:** active · **Areas:** sync, identity, admission

**Decision.** Every sync request carries the caller's identifier, a timestamp and a nonce,
signed with a purpose tag by the machine making it. The receiver checks the signature
against that machine's key, whose certificates must end at the master key the identifier
names (D-219, a user is a master key that certifies machines through a shared key), and
refuses a timestamp more than two minutes off or a nonce it has seen.

**Support.**
- Peers poll, so a handshake for each poll would cost two round trips a second to avoid
  holding one piece of state, and a timestamp and nonce prevent replay without either.
  §16 (propagation).
- Two machines synchronized by NTP were 408ms apart, so a window of two minutes works on
  worse networks while keeping the set of seen nonces small. `84a0751:docs/decisions.md`,
  and `cmd/cogmer/auth.go`, `authTolerance`.
- The list of what the caller holds is not signed, because altering it gains an
  authenticated peer nothing, since it may ask for everything. `cmd/cogmer/auth.go`.

**Rejected.**
- *A session established once and reused.* It needs state on both sides, and a
  server-issued challenge would add a round trip for the same reason.

**Limits.** Signing establishes who is asking and never whether they may: a stranger who
generated a key pair authenticated correctly and read a private room until admission
existed. Admission is D-045 (a sync request is refused unless its peer is a guest).

**Revisit when** clocks between peers are found to differ by more than two minutes.
