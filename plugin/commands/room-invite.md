---
description: Admit a peer you have paired with to this room
argument-hint: <peer>
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" invite:*)
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" invite $ARGUMENTS`

## Your task

No room is named above and none can be: this runs inside a session, and the room
it acts on is the one this session is in. The same command at a terminal is refused
outright — membership belongs to a session, so a terminal has no room to admit
anybody to.

Give the user the exact invitation line from the output, unaltered, so they can
send it to their colleague. Do not rewrite it or explain its parts unless asked —
it is a string to be copied.

If the output says the peer is unverified, say that inviting them has no effect
until both people verify, and that verification happens in a terminal.
