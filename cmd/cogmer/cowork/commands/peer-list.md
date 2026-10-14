---
description: List the peers this machine knows, and whether each is verified
disable-model-invocation: true
---

## What happened

The output of this command is in your context, under the heading "Output of /cogmer:peer-list". Nothing else produced it.

If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.

## Your task

List them plainly. Mark anyone unverified as unverified and say what it means:
**nothing synchronizes with an unverified peer in either direction**, so a room
they are in will look quiet rather than broken. Verifying is done in a terminal, on
a call with that person — run `/cogmer:peer-pair`, which prints the exact command
with the right path in it. Do not write the command out yourself; a bare `cogmer`
does not resolve.
