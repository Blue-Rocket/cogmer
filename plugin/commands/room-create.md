---
description: Create a room and put this session in it
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" create:*)
disable-model-invocation: true
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" create`

## Your task

Tell the user the room's name, and that sessions they start from here join it while
this one is already in it.

If the output says they are not in a room or that something failed, say what it said
and stop. Do not run the command again: creating a room twice makes two rooms, and
the second is indistinguishable from the first to everyone but them.
