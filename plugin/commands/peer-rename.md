---
description: Change what you call a peer
argument-hint: <peer> <new name>
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rename:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rename $ARGUMENTS`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rename $ARGUMENTS
```

If the section above shows cogmer's output, run nothing.

## Your task

Report what the output says. Renaming changes only the name this user gives the
peer: whether they are verified, and every room they are in, are as they were, and
the user's other machines are told the new name.

If the output asks for a peer and a name, they gave one without the other. Ask what
they want to call that person, and do not choose a name for them. Then run the command
again with the peer and that name.

If the output says the user already calls a different peer by that name, relay it
whole. That message is the one place two keys under one name becomes visible, so do
not shorten it to "name taken".
