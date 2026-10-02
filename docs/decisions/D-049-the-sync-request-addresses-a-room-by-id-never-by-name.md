# D-049 — The sync request addresses a room by id, never by name

**Date:** 2026-09-17 · **Status:** active

**Decision.** A sync request carries `roomId` in both directions, and an identifier a
peer supplies resolves through `RoomByID`. `FindRoom` accepts a name, which suits a
command line and not the wire.

**Support.**
- A room's identifier is authoritative and its name is not. D-017 (a room's
  identifier is authoritative, and its name is not).
- Names collide by design, so a name on the wire could match more than one room, and
  the local schema holds no uniqueness rule for a peer to rely on. D-050 (room names
  may collide locally).
- A request that is correct in every other respect, from a genuine guest with a
  genuine signature, is refused with 401 when it names its room, and accepted when it
  gives the id. `cmd/cogmer/auth_test.go`, `TestTheWireRefusesARoomName` and
  `TestTheWireAcceptsARoomID`.

**Rejected.**
- *Addressing a room by name and relying on the store to keep names unique.* The
  guarantee would live in a local schema that no peer can see and nothing obliges the
  next schema change to keep.

**Limits.** A person types a name, the person's own daemon resolves it, and only the
identifier travels.

**Revisit when** a room identifier needs to be typed by a person on a path that reaches
a peer.
