---
description: Take this session out of its room
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" leave:*)
disable-model-invocation: true
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" leave`

## Your task

Report what the output says.

Be precise about what leaving did and did not do, because the difference matters:
this session stops capturing and stops receiving, and **nothing about the room's
history changes**. It is still readable, still shown in the browser view, and
everything already published stays in the room.

Say also that this session may rejoin the same room later, but may not join a
different one — what it has been told cannot be taken back out of its context.
