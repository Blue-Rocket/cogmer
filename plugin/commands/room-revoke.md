---
description: Withdraw a peer's admission to this room
argument-hint: <peer>
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" revoke:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" revoke $ARGUMENTS`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" revoke $ARGUMENTS
```

If the section above shows cogmer's output, run nothing.

## Your task

Report what the output says.

Be exact about the scope, because revoking is narrower than it sounds: it withdraws
admission to **this room only**, leaves the peer known to this machine, and does not
un-say anything they already read. If they should be forgotten entirely rather than
removed from one room, that is `/cogmer:peer-forget`.

No room is named above and none can be: this runs inside a session, and it acts on
the room that session is in.
