---
description: Restart the cogmer daemon on this machine
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" restart:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" restart`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" restart
```

If the section above shows cogmer's output, run nothing.

## Your task

Report what the output says, in a line or two.

If the daemon is answering, say it was restarted, and that nothing already captured is
lost. Colleagues may take a few seconds to reconnect. The user does not need a new
session.

If it says the daemon is not answering, say so plainly and give the reason the output
records. Do not guess at a cause the output does not give. If the output says another
cogmer daemon holds the address, say that `/cogmer:self-stop` ends it.

If it left a process alone, name the process and say it is not cogmer's.
