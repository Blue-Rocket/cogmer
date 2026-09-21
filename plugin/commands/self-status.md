---
description: Show who you are, what you send colleagues, and whether they can reach you
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" whoami:*)
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" whoami`

## Your task

Lead with the pairing string. It is the thing they came for most of the time, and it
is what a colleague needs in order to pair. Give it exactly as printed, and say it is
safe to send by any means — it is a public key and an address, and holding it admits
nobody.

Then, briefly: the name other people see, and the room this session is in if it is in
one.

If the output says the name is this computer's username rather than one anybody
picked, mention it once and offer `/self-name`. Do not press the point — a username
is often exactly right.

If the output says nobody can reach the address, that is the important part and it
outranks everything else: a colleague given that string cannot complete a pairing.
Relay the reason it gives, which distinguishes a daemon that has not published an
address yet from one that could not find a route out.

Do not print the key on its own. It identifies them exactly and is useless to a
colleague without the address attached, so handing it over alone sends them into a
pairing that cannot finish.
