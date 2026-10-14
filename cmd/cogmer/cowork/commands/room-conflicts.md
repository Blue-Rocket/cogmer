---
description: Show quarantined events for this room
disable-model-invocation: true
---

## What happened

The output of this command is in your context, under the heading "Output of /cogmer:room-conflicts". Nothing else produced it.

If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.

## Your task

If there are none, say so in one line.

If there are any, do not summarize them away. A conflict means a peer's sequence
counter went backwards — it lost its local state, or an event was forged — and its
events since then are **not stored and cannot be recovered by synchronization**.
Say which peer, and say that the gap is permanent unless that peer still holds the
events.
