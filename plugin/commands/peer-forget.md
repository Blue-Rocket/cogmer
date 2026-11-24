---
description: Stop recognizing a colleague on all your machines; their place in rooms is unchanged
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

Report what the output says. Forgetting discards the colleague's identity on every one
of the user's machines, so the user's machines stop recognizing them and their turns
stop reaching the user, since a colleague the user does not know is unverified. It
leaves every room's guest list as it is, and the other guests keep working with them.
Pairing with them again makes them known and verified again, in every room they were
never revoked from.

To remove somebody from one room while still knowing them, that is
`/cogmer:room-revoke`.
