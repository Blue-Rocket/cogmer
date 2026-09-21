---
description: Pair with a colleague — shows your string, or starts the two-word check
argument-hint: [their pairing string] [what you call them]
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" pair:*)
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" pair $ARGUMENTS`

## Your task

The output is one of four things and says which.

**A pairing string.** They ran it with nothing, so they are at the first half. Give
them the string exactly as printed and say it is safe to send by any means — it is a
public key and an address, and holding it admits nobody. Then say what comes back:
their colleague sends one too, and it takes a name as well as a string.

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

Never summarise, invent, or repeat two words yourself. The comparison is between two
people on a call; anything that looks like it came from you undermines the only check
that makes it mean anything.

Do not tell them to open a terminal. Pairing happens in the browser, and the command
above already did the terminal part.
