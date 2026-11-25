---
description: Pair with a colleague: the two-word check, in your browser
argument-hint: [their pairing string] [what you call them]
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" pair:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" pair $ARGUMENTS`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" pair $ARGUMENTS
```

If the section above shows cogmer's output, run nothing.

## Your task

The output is one of four things and says which.

**The two halves of pairing.** They ran it with nothing. Relay what it says and
point them at `/cogmer:self-status` for their own string — do not paste a string
here, since this command did not print one.

**A request for a name.** They pasted a string and gave no name. Do not treat this as
an error and do not pick one for them. Ask what they call this person — their first
name is the usual answer — and then run the command again with the string and that
name. The name is theirs, is used everywhere they see that person, and is written
only if the two words match.

**A page was opened.** The check is now in their browser. Say so in one line and
stop; the instructions are on the page and repeating them competes with it. Worth
adding only this: their colleague has to do it at the same time, on a call, so if
they have not called them yet that is the next move.

**A refusal.** Usually the name already means a different key. Relay it whole. That
message is the one place a substituted key becomes visible, so do not shorten it to
"name taken".

Never summarize, invent, or repeat two words yourself. The comparison is between two
people on a call; anything that looks like it came from you undermines the only check
that makes it mean anything.

Do not tell them to open a terminal. Pairing happens in the browser, and the command
above already did the terminal part.

If the output has a line that begins "Next step", give it as the last sentence of your answer,
in the words it uses. It is the one thing the user does next on the way to a first shared room,
and the command's output reaches them only through you.
