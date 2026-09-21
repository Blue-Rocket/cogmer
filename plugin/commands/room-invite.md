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

The output is one of three things and says which.

**Told them.** The invitation reached their machine over the connection the two
already share. Say so in a line, and that they accept it with the room name shown.
There is nothing to copy.

**Not told yet, because the two have not verified each other.** This is not a
refusal and must not be relayed as one: the admission is recorded and the
invitation goes the moment they verify. Relay the step offered, which opens the
two-word check in a browser. Their wanting to start a room is exactly why the check
is worth doing now, so present it as one step remaining rather than an obstacle.

**Could not reach them.** Then there is a line to send, and it should be given
exactly as printed, unaltered. Do not rewrite it or explain its parts unless asked
— it is a string to be copied.
