# D-064 — A session's room is the one somebody chose inside it, never a machine default

**Date:** 2026-09-18 · **Status:** active · **Areas:** rooms

**Decision.** `RoomForSession` reports the room a session was put in and never puts a
session in one. A session joins a room because a person ran a command inside it, and a
session in no room is an ordinary Claude Code session with nothing captured, injected or
shared.

**Support.**
- Otherwise a room created and forgotten would be joined weeks later by a session in an
  unrelated repository, which would capture and publish with nobody having acted. The
  failure has no error, and its first sign is a colleague reading turns from work they
  were never shown. `cmd/cogmer/membership.go`, `RoomForSession`, and §12a (room
  membership).
- Nothing derives a room from a directory, so nothing else would catch it. D-015 (rooms
  are scoped to sessions, not to projects).
- A room created and forgotten collects no sessions, and a session joins a room only when
  put in one. `cmd/cogmer/membership_test.go`, `TestAForgottenRoomDoesNotCollectSessions`
  and `TestASessionJoinsARoomOnlyWhenPutInOne`.

**Rejected.**
- *Deriving a session's room from its working directory or repository.* D-015 forbids it.
- *A machine-level default that applies only when a session has no room.* It is the same
  conflation in the one case where it does the damage.

**Limits.** A command at a terminal names a room for each invocation and stores none.
D-080 (there is no current room, and a room-scoped command gets its room from its session).

**Revisit when** something other than a command at a terminal needs a room and has no
session to ask.
