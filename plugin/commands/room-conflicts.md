---
description: Show quarantined events for this room
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" conflicts:*)
disable-model-invocation: true
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" conflicts`

## Your task

If there are none, say so in one line.

If there are any, do not summarise them away. A conflict means a peer's sequence
counter went backwards — it lost its local state, or an event was forged — and its
events since then are **not stored and cannot be recovered by synchronization**.
Say which peer, and say that the gap is permanent unless that peer still holds the
events.
