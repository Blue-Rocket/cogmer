---
description: Join a room you were invited to
argument-hint: <invitation>
allowed-tools: Bash(claude-team join:*)
---

## What happened

!`claude-team join $ARGUMENTS`

## Your task

Report what the output says, in one or two lines.

**If it refused because this session has already been in a room, relay the whole
explanation rather than condensing it.** The person has done nothing wrong, cannot
undo what happened, and the useful parts are the reason and the way forward — that
starting a second session works, and that this one can still rejoin the room it was
in. A one-line "cannot join" leaves them stuck with a rule and no route.

If it says the inviting peer is unverified, say so plainly and say that **nothing
will synchronize until both people run `claude-team verify` in a terminal, on a
call with each other**. That is not a warning to soften: an unverified peer is
refused, so the room will look empty and silent rather than broken.
