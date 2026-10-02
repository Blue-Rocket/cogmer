# D-080 — There is no current room, and a room-scoped command gets its room from its session

**Date:** 2026-09-20 · **Status:** active · **Areas:** rooms, commands

**Decision.** There is no machine-level current room and no value, such as an environment
variable, that picks one. A room-scoped command gets its room from the session that
invoked it, and a terminal that wants one emulates a session. The one exception is
`conflicts`, which takes a room by name.

**Support.**
- No room-scoped operation has a terminal-only reason to exist, since each fulfills a slash
  command or is being tested, so the pointer had no user left once the view gave each room
  its own address. D-077 (every room has its own address in the view).
- Naming a room to read it is not an exercise of standing, and `conflicts` is a diagnostic
  wanted when a room is misbehaving and there may be no healthy session to ask.
  `cmd/cogmer/main.go`, `runConflicts`, D-079 (changing a room requires standing in it).
- A test fails if the command reads `COGMER_ROOM`. `cmd/cogmer/membership_test.go`,
  `TestNoAmbientRoomOverride`.
- Every command that resolves its room asks the session first. `cmd/cogmer/main.go`,
  `sessionRoom`.

**Rejected.**
- *A machine-level pointer for terminal commands.* It answered the wrong room three times,
  once in the view, once in `where` and once in `currentRoom`, each time as a convenience
  that behaved as a silent wrong answer.
- *A room named in an environment variable and ranked above the session's own.* An exported
  variable has the same value in every session a machine starts, so it would choose one room
  for all of them, and a command run in one room would act on another without an error.
  `84a0751:docs/room-choice-findings.md`, "An ambient variable ranked above the session
  answers for every session".

**Revisit when** something other than a session can hold membership.
