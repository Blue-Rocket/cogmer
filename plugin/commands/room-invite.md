---
description: Admit a peer you have paired with to this room
argument-hint: <peer>
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" invite:*)
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" invite $ARGUMENTS`

## Your task

No room is named above and none is needed: this runs inside a session, and a
command from inside a session acts on that session's room. At a terminal the same
command requires `--room <name>`, because granting access is not a thing to guess
at.

Give the user the exact invitation line from the output, unaltered, so they can
send it to their colleague. Do not rewrite it or explain its parts unless asked —
it is a string to be copied.

If the output says the peer is unverified, say that inviting them has no effect
until both people verify, and that verification happens in a terminal.
