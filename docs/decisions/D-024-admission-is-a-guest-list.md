# D-024 — Admission is a guest list

**Date:** 2026-09-16 · **Status:** active

**Decision.** Admission to a room is an entry on the host's guest list for a peer the
host knows, proved by possession of the key the host holds for it. The room's name
locates the room and the guest list admits the peer, so nothing secret is typed, spoken
or sent, and guessing the name gains nothing.

**Support.**
- A host holds a guest's public identifier before inviting them, from a pairing
  string, which is exchanged once per colleague and is safe to send any way, because
  holding it grants nothing. D-023 (a peer identifier must be safe to know).
- A room's name is never a credential. D-137 (a room's name is never a credential).
- An intercepted invitation naming a guest reveals only that a room exists. §12 (forming
  a room).

**Rejected.**
- *A join secret, kept until identity is cryptographic.* A secret has to remain secret
  in transit, is spent on each first meeting, and enrols whoever intercepts it, while a
  public identifier is sent once per colleague and reveals nothing if intercepted.
  Identity is cryptographic (D-042), and there is no join token (D-026).
- *Admitting on the room's name on a trusted network.* The name is guessable (D-134),
  and no network is safe for that.

**Revisit when** admission is needed for a peer the host has never paired with and
cannot approve.
