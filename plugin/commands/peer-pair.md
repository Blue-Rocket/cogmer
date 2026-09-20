---
description: Pair with a colleague — shows your string, or starts the two-word check
argument-hint: [their pairing string]
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" pair:*)
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" pair $ARGUMENTS`

## Your task

This command does one of two things and the output says which.

**If it printed a pairing string**, they ran it with nothing. Give them the string
exactly as it appears, and say it is safe to send by any means — it is a public key
and an address, and holding it admits nobody. Say what comes back: their colleague
sends one too, and `/peer-pair <that string>` is the next step.

**If it says a page was opened**, the two-word check is now in their browser. Say so
in one line and stop — the instructions are on the page and repeating them here just
competes with it. The one thing worth adding: their colleague has to do this at the
same time, on a call, so if they have not called them yet that is the next move.

Do not tell them to open a terminal. Pairing happens in the browser, and the command
above already did the terminal part.

**If it says the daemon is not answering, or fell back to the terminal**, relay that
plainly. A fallback is not a failure — it is what happens on a machine with no
browser — but they need to know the words will appear in their terminal instead.

Never summarise, invent, or repeat two words yourself. The comparison is between two
people on a call; anything that looks like it came from you undermines the only check
that makes it mean anything.
