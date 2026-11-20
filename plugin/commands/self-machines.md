---
description: Add, rename or remove your machines, or recover, in your browser
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" machines:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" machines`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" machines
```

If the section above shows cogmer's output, run nothing.

## Your task

The output is one of three things and says which.

**A page was opened.** Adding, renaming and removing machines, and recovering, happen
in that page in their browser. Say so in one line and stop; the instructions are on the
page and repeating them competes with it. If they want to add a machine, the one thing
worth adding is that the new machine's page shows a string to enter in this page, and
both pages then show two words to compare.

**The identity is not made yet.** The page that opened asks whether they have used
cogmer on another machine. Say so in one line, and that they come back and run the
command again once the page is done.

**The daemon is not running.** Relay what the output says, and offer
`/cogmer:self-restart` when it names it.

Never ask for, repeat or suggest typing the recovery phrase here. It is typed only in
the page, because anything typed into this session reaches the model and the service
that runs it, and whoever holds the phrase is the user.

Never summarize, invent, or repeat two words yourself. The comparison is between the
two pages; anything that looks like it came from you undermines the only check that
makes it mean anything.
