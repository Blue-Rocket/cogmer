# D-050 — Room names may collide locally, so the room table does not require a unique name

**Date:** 2026-09-17 · **Status:** active · **Areas:** rooms, storage

**Decision.** `rooms.room_name` has no uniqueness constraint. `CreateRoom` asks whether
a name is free before it mints one, and `migrateMembership` rebuilds a table created
with the constraint.

**Support.**
- Names collide by design, and the table holds rooms other peers named, so a
  uniqueness rule would make this machine refuse to record a room another peer named.
  D-017 (a room's identifier is authoritative, and its name is not),
  `cmd/cogmer/membership.go`, `RecordRoom`.
- The table holds every room ever recorded and not only those in use, and names are
  drawn from 7,656 combinations, so about a hundred rooms make a clash as likely as
  not. D-135 (a room's name is unique only among one peer's rooms).
- A name this peer mints can be minted differently, so `CreateRoom` can avoid a local
  clash at no cost, which a name arriving with a room cannot. `cmd/cogmer/membership.go`,
  `CreateRoom`.
- SQLite has no `DROP CONSTRAINT`, so a table created with the rule keeps it until it
  is rebuilt. `cmd/cogmer/membership_test.go`, `TestMigrationDropsTheNameConstraint`.
- Joining a second room whose name matches a recorded one succeeds.
  `cmd/cogmer/membership_test.go`, `TestJoiningTwoRoomsWithOneNameSucceeds`.

**Rejected.**
- *Renaming a joined room locally to keep names unique.* This machine would then
  disagree with the host about what the room is called, so the name one user says
  aloud is not the name the other sees, and a name that differs per peer is not a
  mnemonic.

**Revisit when** a user has to refer to a room by name across peers where no host
resolves it. §12 (forming a room) has a name resolved only against a specific peer.
