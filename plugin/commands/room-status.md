---
description: Show the room this session is in, and who is in it
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms:*), Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" guests:*), Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" where:*)
disable-model-invocation: true
---

## Rooms

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms`

## Guests of this session's room

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" guests`

## Where to watch it

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" where`

## Your task

Summarize in a few lines: which room this session is in, and who may enter it.
Mark anyone unverified as unverified — their turns are not being exchanged.

If the output says a colleague is NOT REACHABLE, say that first, before anything else
about the room. A room whose colleague cannot be reached looks quiet and is not, and
nobody would otherwise know. Say for how long, and that none of their turns reach this
machine until it can reach them. If the output says they reached this machine
recently, say that their daemon is running and the fault is in reaching it. If it says
they have not, say that this machine has also heard nothing from them. Suggest asking the colleague to run
`/cogmer:self-status` and say whether it tells them colleagues can reach them. Do not
suggest the fault is in this session: the colleague's daemon is the likely place.

Give them the watch address from above verbatim. It is a link they open in a
browser and leave open beside this session, not something to summarize — the room
is read there, and this session is read here.
