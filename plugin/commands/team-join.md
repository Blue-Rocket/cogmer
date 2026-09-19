---
description: Join a room you were invited to
argument-hint: <invitation>
allowed-tools: Bash(claude-team join:*)
---

## What happened

!`claude-team join $ARGUMENTS`

## Your task

Report what the output says, in one or two lines.

If it says the inviting peer is unverified, say so plainly and say that **nothing
will synchronize until both people run `claude-team verify` in a terminal, on a
call with each other**. That is not a warning to soften: an unverified peer is
refused, so the room will look empty and silent rather than broken.
