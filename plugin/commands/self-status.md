---
description: Show who you are, what you send colleagues, whether they can reach you, and which versions are running
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" whoami:*)
disable-model-invocation: true
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" whoami`

## Your task

Lead with the pairing string, when the output has one. It is the thing they came for
most of the time, and it is what a colleague needs in order to pair. Give it exactly as
printed, and say it is safe to send by any means — it is a public key and an address,
and holding it admits nobody.

Then, briefly: the name other people see, and the room this session is in if it is in
one.

If the output says the name is this computer's username rather than one anybody
picked, mention it once and offer `/cogmer:self-name`. Do not press the point — a
username is often exactly right.

If the output says there is no string to send yet, that is the important part and it
outranks everything else: a colleague given a string with no address in it cannot
complete a pairing, so none was printed. Relay the reason it gives, which distinguishes
a daemon that has not published an address yet, one that could not start, and one that
could not find a route out.

The output ends with the versions of the plugin, the binary and the running daemon.
Give them in one line. If the output says they disagree, say which, and relay what it
says resolves it.

Do not print the key on its own. It identifies them exactly and is useless to a
colleague without the address attached, so handing it over alone sends them into a
pairing that cannot finish.
