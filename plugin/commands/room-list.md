---
description: Rooms on this machine, and any waiting for you to accept
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms:*)
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms`

## Your task

List the rooms plainly. The one marked with an asterisk is the room this session
is in, if it is in one.

**If anything is waiting to be accepted, lead with it.** Those are rooms a
colleague has already admitted this person to, and the only thing left is for them
to accept. Give the room name and who offered it, and say that joining takes the
name alone — there is nothing to paste.

Do not describe an offer as an invitation they need to find. It has already
arrived; it is sitting there.
