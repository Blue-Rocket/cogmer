# D-072 — A refused join is explained, not merely refused

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** A session that has been in one room is refused a second (§12a, D-071).
The refusal read:

> this session has been in gusty-cavern and cannot join another: what it has been
> told cannot be withdrawn

Delivered through `log.Fatalf`, so a person saw it prefixed with a timestamp. It
states a rule and a fragment of a reason, and offers no way forward.

**Why that is worse here than it would be elsewhere.** The person has done nothing
wrong: joining a second room is a reasonable thing to try, the refusal comes from an
invariant they have never read, and **nothing they can do will make it work** — the
session's context already holds the other room. A message that gives only the rule
leaves them stuck with no route, in the one situation where being stuck is
permanent.

**Decision.** Explain the mechanism and give the way out.

The mechanism is worth stating because it is not obvious and it is not about
copying. Joining a second room would carry the first room's conversation into it
**by way of what the session says next**, which may be shaped by everything it has
read — invisible to both rooms' members, and impossible to take back because a
context window cannot be un-read. Refusing is the only moment that prevents it.

The way out is that a second Claude Code session costs nothing, and that this
session may still rejoin the room it was in (D-071).

**A typed error, not a formatted string.** `BoundElsewhereError` carries both rooms,
because a caller that cannot name the room a person was trying to reach cannot tell
them what to do instead. The explanation lives where it is shown to a person rather
than inside an error value that tests compare and logs decorate.

**And the slash command is told not to condense it.** `/room-join` relays the whole
explanation rather than summarising, for the same reason `/cogmer:room-status` gives
the watch address verbatim (`plugin/commands/room-status.md`): a one-line "cannot
join" is exactly the unhelpful form this replaces, and a model paraphrasing
helpfully would reproduce it.

**Revisit when** anything else refuses a person for an invariant's sake. The pattern
generalises: state the mechanism, say what is impossible to undo, and give the route
— and if there is no route, say that too rather than implying one exists.
