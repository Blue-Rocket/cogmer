---
description: Discard a peer and every room admission it held
argument-hint: <peer>
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" forget:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" forget $ARGUMENTS`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" forget $ARGUMENTS
```

If the section above shows cogmer's output, run nothing.

## Your task

Report what the output says, and be clear that this is the wide act, not the narrow
one. Forgetting discards the identity **and every room admission it held**, so a
later meeting is a first meeting: they will be unknown, unverified, and admitted to
nothing until somebody invites them again.

To remove somebody from one room while still knowing them, that is
`/cogmer:room-revoke`.
