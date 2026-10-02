# D-064 — A session's room is the one somebody chose inside it, never a machine default

**Date:** 2026-09-18 · **Status:** active (implemented) · **reconstructed 2026-09-21**

**This entry was never written.** The number was allocated while writing `e4fd2d8`
("Phase 11: ship as a plugin, and bind sessions explicitly", 2026-09-18), which
cited it in six files and did not touch this document. What follows is
reconstructed from those citations, the code they annotate, and the tests that
enforce it. The rule and its hazard are recoverable exactly. **The alternatives
actually weighed are not**, so the rejected list below states only what the code
rules out, and should not be read as a record of deliberation.

**Context.** One pointer was answering two different questions.

- *Which room is this session in?* — `session_rooms`, written only when somebody
  runs `/room-create` or `/room-join` **inside that session**.
- *Which room does a command at a terminal act on?* — a machine-level
  `current_room`, for the case where there is no session to ask.

Answering the first with the second is the conflation this decision names.

**Decision.** `RoomForSession` reports the room a session was put in and never puts
it in one. A session joins a room because a person ran a command inside it, and a
session in no room is an ordinary Claude Code session: nothing captured, nothing
injected, nothing shared (§12a).

**The hazard is silent, which is why it needed a decision.** Otherwise a room
created and forgotten is joined weeks later by a session in an unrelated
repository, capturing and publishing with nobody having done anything. Nothing
derives a room from a directory (D-015), so nothing else would have caught it. The
failure has no error and no moment: its first sign is a colleague reading turns
from work they were never shown.

**Enforced by a test rather than by care.** `membership_test.go:424` names itself
"the hazard D-064 removes, stated as a test so it cannot come back."

**What became of the other pointer.** D-076 placed `COGMER_ROOM` above the
session's own room, reinstating this failure at higher precedence the same
afternoon. D-077 gave every room its own URL, removing the pointer's last consumer,
and D-080 deleted it. `current_room` no longer exists anywhere. Its schema comment
outlived it and described the removed behaviour as the design, which is most of why
this entry was hard to recover; the comment is gone as of 2026-09-21, along with the
`settings` table it annotated, which nothing had ever read or written.

**Rejected** — what the code rules out, not what was considered:

- *Derive a session's room from its working directory or repository.* Independently
  forbidden by D-015 (rooms are session-scoped).
- *Let a machine-level default apply only when a session has no room.* That is the
  conflation itself, in the one case where it does the damage.

**Revisit when** something needs a room and genuinely has no session to ask. That
case is a command at a terminal, and D-077 answers it by naming a room per
invocation rather than by storing one.
