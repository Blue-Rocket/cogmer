---
description: Show or set the name other people see for you
argument-hint: [what people should call you]
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" name:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" name $ARGUMENTS`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" name $ARGUMENTS
```

If the section above shows cogmer's output, run nothing.

## Your task

Report what the output says, in one or two lines.

This name is seen only by other people — it travels with every turn they receive and
with the pairing string handed to a colleague. The user never sees it on their own
turns, which is why it can sit wrong for weeks without anybody noticing.

If the output says the name is the computer's username rather than one anybody
picked, say so plainly and offer to set it. Do not pick one for them. A username is
often right and often not: "David" needs no changing, "Ec2-user" reads to a colleague
as a claim about a person.

If they set one, be clear about what did not change: turns already sent keep the name
they were sent with, and a colleague who gave them a name of their own still sees
that one. Renaming yourself is not a way to change what somebody else calls you.
