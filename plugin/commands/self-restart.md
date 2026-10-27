---
description: Restart the cogmer daemon on this machine
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" restart:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" restart`

## Your task

Report what the output says, in a line or two.

If the daemon is answering, say it was restarted, and that nothing already captured is
lost. Colleagues may take a few seconds to reconnect. The user does not need a new
session.

If it says the daemon is not answering, say so plainly and give the reason the output
records. Do not guess at a cause the output does not give. If the output says another
cogmer daemon holds the address, say that `/cogmer:self-stop` ends it.

If it left a process alone, name the process and say it is not cogmer's.
