---
description: Print where this command's line and the shell tool run, and what the host observer recorded
allowed-tools: Bash(bash:*)
disable-model-invocation: true
---

## From the command's own line

!`bash "${CLAUDE_PLUGIN_ROOT}/hooks/where.sh" bang`

## Your task

Run `bash "${CLAUDE_PLUGIN_ROOT}/hooks/where.sh" tool` with your shell tool. If
`CLAUDE_PLUGIN_ROOT` is empty in your shell, use the folder of this plugin that holds
`hooks/where.sh` instead, and say which path you used.

Then show the user both outputs, the one above and the one your shell printed, exactly
as printed, each in its own code block. Add nothing to them and draw no conclusions.
