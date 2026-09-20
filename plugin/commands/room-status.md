---
description: Show the room this session is in, and who is in it
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms:*), Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" guests:*), Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" where:*)
---

## Rooms

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms`

## Guests of the current room

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" guests`

## Where to watch it

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" where`

## Your task

Summarise in a few lines: which room this session is in, and who may enter it.
Mark anyone unverified as unverified — their turns are not being exchanged.

Give them the watch address from above verbatim. It is a link they open in a
browser and leave open beside this session, not something to summarise — the room
is read there, and this session is read here.
