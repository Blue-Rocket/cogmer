# D-079 — Changing a room requires standing in it, which only a session has

**Date:** 2026-09-20 · **Status:** active · **Areas:** admission, rooms, commands

**Decision.** `invite` and `revoke` require the invoking session to be in the room, with
no flag, no fallback and no pointer. At a terminal they are refused, and the refusal says
to run the slash command inside the Claude Code session that is in the room.

**Support.**
- Membership is held by a session, so a terminal is a member of nothing, and naming a room
  there reaches into a room that belongs to a session the user is not in. §3.6
  (session-scoped rooms).
- Granting access in the wrong room lets a person read a conversation nobody invited them
  to, and `revoke` stops future reading and does not un-read, so the command with the worst
  failure leans on no weaker answer to which room. `cmd/cogmer/main.go`, `roomToChange`.
- A command run without a session exits, and one run in a session that is in no room says
  how to enter one. `cmd/cogmer/membership_test.go`, `TestOnlyASessionInTheRoomCanChangeIt`.

**Rejected.**
- *A `--room <name>` flag so that a terminal can say which room.* It answers which room
  and not whether a terminal has any standing to change one, and its care makes reaching
  into somebody else's room look like choosing among one's own.
- *A stored pointer at a terminal, or a room named in the environment.* Each answers
  from something that was set at some other time, and none asks whether the caller has any
  business acting on the room. D-080 (there is no current room).

**Limits.** Showing a room is not an exercise of standing, so a command that only shows
one may name it. D-080 holds the one command that does, `conflicts`.

**Revisit when** something other than a session can hold membership.
