---
description: Admit a peer you have paired with to this room
argument-hint: <peer>
allowed-tools: Bash(claude-team invite:*)
---

## What happened

!`claude-team invite $ARGUMENTS`

## Your task

Give the user the exact invitation line from the output, unaltered, so they can
send it to their colleague. Do not rewrite it or explain its parts unless asked —
it is a string to be copied.

If the output says the peer is unverified, say that inviting them has no effect
until both people verify, and that verification happens in a terminal.
