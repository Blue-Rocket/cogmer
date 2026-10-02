# D-141 — An identity is created once, and belongs to a machine

**Date:** 2026-09-16 · **Status:** active · **Areas:** identity

**Decision.** A peer identity is created once, on a machine's first use of cogmer, and
persists, outliving every room, session and invitation. It belongs to a machine, not a
person, so a user with a laptop and a desktop is two peers, and appears as two wherever
peers are listed or admitted.

**Support.**
- An identity that exists before there is anything to join lets a peer be recognized on
  a later occasion rather than met afresh. §6 (identity).
- A private key never leaves the machine that made it, so losing one machine never loses
  the key another holds. `cmd/cogmer/identity.go`.

**Rejected.**
- *An identity per room, session or invitation.* A colleague would be met afresh each
  time.
- *One identity per person, carried across machines.* The private key would have to be
  copied between machines.

**Revisit when** users need their machines to appear as one peer.
