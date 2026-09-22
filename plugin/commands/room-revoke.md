---
description: Withdraw a peer's admission to this room
argument-hint: <peer>
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" revoke:*)
disable-model-invocation: true
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" revoke $ARGUMENTS`

## Your task

Report what the output says.

Be exact about the scope, because revoking is narrower than it sounds: it withdraws
admission to **this room only**, leaves the peer known to this machine, and does not
un-say anything they already read. If they should be forgotten entirely rather than
removed from one room, that is `/cogmer:peer-forget`.

No room is named above and none can be: this runs inside a session, and it acts on
the room that session is in.
