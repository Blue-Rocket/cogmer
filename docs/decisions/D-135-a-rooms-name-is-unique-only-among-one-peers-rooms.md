# D-135 — A room's name is unique only among one peer's rooms

**Date:** 2026-09-16 · **Status:** active · **Areas:** rooms

**Decision.** A peer keeps the names of its own live rooms distinct, generating another
on a collision, and no name is unique beyond one peer.

**Support.**
- Rooms are created on machines that do not coordinate, so names cannot be globally
  unique. §12 (forming a room).
- A name is only ever resolved against one peer. §12.
- The two lists hold 88 and 87 words, which give 7,656 combinations, and a peer hosts
  few live rooms at once. `cmd/cogmer/roomname.go`.

**Rejected.**
- *Globally unique names.* They cannot be had without coordination, and nothing needs
  them while a name is resolved against one peer.

**Revisit when** a name has to be resolved beyond one peer.
