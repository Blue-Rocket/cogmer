---
description: Stop the cogmer daemon on this machine
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" stop:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" stop`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" stop
```

If the section above shows cogmer's output, run nothing.

## Your task

Report what the output says, in a line or two.

If it stopped a daemon, say that nothing is captured or shared, in any room and any
session on this machine, until a daemon runs again. A new agent session with cogmer enabled starts
one. This command does not, because somebody who stops the daemon may want it stopped.

If it left a process alone, name the process and say it is not cogmer's, so the user
knows `stop` did not touch it.

If nothing was listening, say so.
