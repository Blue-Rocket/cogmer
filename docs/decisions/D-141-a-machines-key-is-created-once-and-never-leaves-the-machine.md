# D-141 — A machine's key is created once and never leaves the machine

**Date:** 2026-09-16 · **Status:** active · **Areas:** identity

**Decision.** A machine's key pair is created once, on the machine's first use of cogmer, and
persists, outliving every room, session and invitation, so that the machine's sequence
numbers and its certificate run unbroken from one use to the next. Colleagues recognize the
user by the master key, and never by a machine's key. The private key never leaves the
machine that made it, and a user's machines are tied together by certificates and not by a
shared private key (D-219, a user is a master key that certifies machines through a shared
key). A cloud session's key is made for each session (D-216, a cloud session signs with a
key of its own that the user's shared key vouches for).

**Support.**
- A key that exists before there is anything to join, with a certificate that ends at the
  master key, lets a user be recognized on a later occasion rather than met afresh. §6
  (identity), D-219.
- A machine signs its events with its own key and its own sequence. D-219.
- A private key never leaves the machine that made it, so losing one machine never loses the
  key another holds. `cmd/cogmer/identity.go`.

**Rejected.**
- *A key per room, session or invitation on a machine the user keeps.* Each new key would
  start a sequence of its own and would need a new certificate from a machine that holds
  the shared key before it could speak. A cloud session has one because sessions that run
  at the same time would otherwise claim the same sequence numbers (D-216).

**Revisit when** a machine's key has to be moved to another machine.
