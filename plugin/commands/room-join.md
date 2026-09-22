---
description: Join a room you were invited to
argument-hint: <invitation>
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" join:*)
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" join $ARGUMENTS`

## Your task

Report what the output says, in one or two lines.

A room can be joined by name alone when its host's invitation already reached this
machine. If the output instead lists rooms waiting to be accepted, relay the list
and the names — the person likely typed a name they were told verbally rather than
one that was delivered.

**If it refused because this session has already been in a room, relay the whole
explanation rather than condensing it.** The person has done nothing wrong, cannot
undo what happened, and the useful parts are the reason and the way forward — that
starting a second session works, and that this one can still rejoin the room it was
in. A one-line "cannot join" leaves them stuck with a rule and no route.

If it says the inviting peer is unverified, say so plainly and say that **nothing
will synchronize until both people verify each other, in a terminal, on a call
with each other**. That is not a warning to soften: an unverified peer is refused,
so the room will look empty and silent rather than broken.

Do not invent the command for that. `/cogmer:peer-pair` prints it with the correct path
filled in; point them there rather than writing `cogmer verify`, which will not
resolve for anybody who installed this as a plugin.
