---
description: Rooms on this machine, and any waiting for you to accept
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms
```

If the section above shows cogmer's output, run nothing.

## Your task

List the rooms plainly. The one marked with an asterisk is the room this session
is in, if it is in one.

**If anything is waiting to be accepted, lead with it.** Those are rooms a
colleague has already admitted this person to, and the only thing left is for them
to accept. Give the room name and who offered it, and say that joining takes the
name alone — there is nothing to paste.

Do not describe an offer as an invitation they need to find. It has already
arrived; it is sitting there.
