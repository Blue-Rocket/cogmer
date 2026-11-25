---
description: Check that cogmer and the host's behaviors it relies on are working; `extended` runs the whole check in the background
argument-hint: [extended]
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" health:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" health $ARGUMENTS`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" health $ARGUMENTS
```

If the section above shows cogmer's output, run nothing.

## Your task

Report what the output says, briefly. Lead with whether anything is wrong: a failed behavior,
a daemon that is not running, or versions that disagree. Name each failed behavior as the output
does, and relay what the output says resolves a disagreement.

If the output says the whole check is running, say that it takes about 20 seconds and that the
user can run `/cogmer:self-doctor` again to read what it found. Do not run the check yourself,
and do not wait for it.

If the output says the whole check drives real turns on the user's subscription, say that once,
and offer `/cogmer:self-doctor extended` only if something above looks wrong or the user asked
for the whole check. Do not offer it when every check passed.

If the output has a line that begins "Next step", give it as the last sentence of your answer,
in the words it uses. It is the one thing the user does next on the way to a first shared room,
and the command's output reaches them only through you.
