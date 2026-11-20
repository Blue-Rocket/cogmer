---
description: Set this machine up for Claude Cowork
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" cowork-setup:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" cowork-setup`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" cowork-setup
```

If the section above shows cogmer's output, run nothing.

## Your task

Tell the user what the output says changed, in a few lines: that Claude Desktop now starts
cogmer's bridge, and where the plugin for Cowork was written.

Then give the steps it lists, in order, as the user's to do: quit Claude Desktop completely and
open it again, upload the file in Cowork's plugin settings, and start a new Cowork session. Say
that typing `/cogmer:self-status` there shows whether it works. Do not do any of it for them.

If the output says nothing was changed because a file could not be read or written, say what it
said and stop. Do not try again, and do not edit any file yourself.
