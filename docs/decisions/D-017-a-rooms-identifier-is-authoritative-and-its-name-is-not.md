# D-017 — A room's identifier is authoritative, and its name is not

**Date:** 2026-09-16 · **Status:** active · **Areas:** rooms, sync

**Decision.** Every room has a UUID, globally unique, never reused and never changed,
which every event carries and on which replication, deduplication and storage all key.
Its name is for people: nothing synchronizes, routes, deduplicates or stores by the
name, events do not carry it, and unrelated rooms may share it.

**Support.**
- A room's database is named by its identifier. §22 (persistence).
- A synchronization request addresses a room by its identifier. D-049 (the sync request
  addresses a room by id).
- Names are generated independently on each machine, so unrelated rooms can share one.
  D-135 (a room name is unique only among one peer's rooms).

**Rejected.**
- *A single identifier.* A UUID cannot be read aloud on a call, and a name cannot be a
  key for synchronization.
- *Carrying the name on every event.* The name belongs to the room, and a display
  string inside immutable events invites drift.

**Revisit when** a room's name has to be authoritative somewhere, such as a directory of
rooms across an organization.
