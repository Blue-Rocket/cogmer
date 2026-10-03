---
description: Stop the cogmer daemon on this machine
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" stop:*)
disable-model-invocation: true
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" stop`

## Your task

Report what the output says, in a line or two.

If it stopped a daemon, say that nothing is captured or shared, in any room and any
session on this machine, until a daemon runs again. A new Claude Code session starts
one. This command does not, because somebody who stops the daemon may want it stopped.

If it left a process alone, name the process and say it is not cogmer's, so the user
knows `stop` did not touch it.

If nothing was listening, say so.
