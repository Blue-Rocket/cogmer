# D-141 — A machine's key is created once and never leaves the machine

**Date:** 2026-09-16 · **Status:** active · **Areas:** identity

**Decision.** A machine's key pair is created once, on the machine's first use of cogmer, and
persists, outliving every room, session and invitation. The private key never leaves the machine
that made it, and a user's machines are tied together by certificates and not by a shared
private key (D-219, a user is a master key that certifies machines through a shared key).

**Support.**
- A key that exists before there is anything to join lets a peer be recognized on a later
  occasion rather than met afresh. §6 (identity).
- A private key never leaves the machine that made it, so losing one machine never loses the
  key another holds. `cmd/cogmer/identity.go`.

**Rejected.**
- *A key per room, session or invitation.* A colleague would be met afresh each time.

**Revisit when** a machine's key has to be moved to another machine.
