---
description: Take this session out of its room
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" leave:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" leave`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" leave
```

If the section above shows cogmer's output, run nothing.

## Your task

Report what the output says.

Be precise about what leaving did and did not do, because the difference matters:
this session stops capturing and stops receiving, and **nothing about the room's
history changes**. It is still readable, still shown in the browser view, and
everything already published stays in the room.

Say also that this session may rejoin the same room later, but may not join a
different one — what it has been told cannot be taken back out of its context.
