---
description: Withdraw a peer's admission to this room
argument-hint: <peer>
disable-model-invocation: true
---

## What happened

The output of this command is in your context, under the heading "Output of /cogmer:room-revoke". Nothing else produced it.

If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.

## Your task

Report what the output says.

Be exact about the scope, because revoking is narrower than it sounds: it withdraws
admission to **this room only**, leaves the peer known to this machine, and does not
un-say anything they already read. If they should be forgotten entirely rather than
removed from one room, that is `/cogmer:peer-forget`.

No room is named above and none can be: this runs inside a session, and it acts on
the room that session is in.
