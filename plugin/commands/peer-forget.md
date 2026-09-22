---
description: Discard a peer and every room admission it held
argument-hint: <peer>
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" forget:*)
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" forget $ARGUMENTS`

## Your task

Report what the output says, and be clear that this is the wide act, not the narrow
one. Forgetting discards the identity **and every room admission it held**, so a
later meeting is a first meeting: they will be unknown, unverified, and admitted to
nothing until somebody invites them again.

To remove somebody from one room while still knowing them, that is
`/cogmer:room-revoke`.
