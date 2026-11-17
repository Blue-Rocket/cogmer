# D-024 — Admission is a guest list

**Date:** 2026-09-16 · **Status:** active · **Areas:** admission

**Decision.** Admission to a room is an entry for a user on the room's one guest list,
which every guest holds (D-235, a room's one guest list decides whose events count), made
by a guest who knows that user, and proved by a machine whose key carries a chain of
certificates to the master key the list names for that user (D-219, a user is a master
key that certifies machines through a shared key). A guest is a user, so any machine of an admitted user proves the
admission. The room's name locates the room and the guest list admits the user, so
nothing secret is typed, spoken or sent, and guessing the name gains nothing.

**Support.**
- A host holds a guest's public identifier before inviting them, from a pairing
  string, which is exchanged once per colleague and is safe to send any way, because
  holding it grants nothing. D-023 (a peer identifier must be safe to know).
- A room's name is never a credential. D-137 (a room's name is never a credential).
- An intercepted invitation naming a guest reveals only that a room exists. §12 (forming
  a room).

**Rejected.**
- *A join secret, kept until identity is cryptographic.* A secret has to remain secret
  in transit, is spent on each first meeting, and enrolls whoever intercepts it, while a
  public identifier is sent once per colleague and reveals nothing if intercepted.
  Identity is cryptographic (D-042, peer identity is an Ed25519 key pair), and there is
  no join token (D-026, there is no join token at all).
- *Admitting on the room's name on a trusted network.* The name is guessable (D-134, a
  room's name is generated from two word lists, never chosen), and no network is safe
  for that.
- *A guest list of machines.* Each machine a user adds would need an entry on the guest
  list of every room the user is in, before any session on it could take part.

**Revisit when** admission is needed for a user the host has never paired with and
cannot approve.
