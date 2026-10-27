---
description: Change what you call a peer
argument-hint: <peer> <new name>
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" rename:*)
disable-model-invocation: true
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" rename $ARGUMENTS`

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
