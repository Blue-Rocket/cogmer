# D-138 — Local discovery locates a room, and never admits anyone

**Date:** 2026-09-16 · **Status:** not built · **Areas:** rooms, transport, trust

**Decision.** Peers on the same network can find a room by local service discovery,
without an address being typed. Discovery locates the room, and never admits anyone to
it.

**Support.**
- On a shared network any listener can list the advertised room names, and names can be
  guessed without listening. D-137 (a room's name is never a credential).
- Admission is the room's guest list. §12 (forming a room).

**Rejected.**
- *Joining by name alone on a trusted network.* Office and conference networks are
  neither small nor trusted.

**Revisit when** two users who share a network need to pair.
