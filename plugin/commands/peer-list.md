---
description: List the peers this machine knows, and whether each is verified
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" peers:*)
disable-model-invocation: true
---

## Known peers

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" peers`

## Your task

List them plainly. Mark anyone unverified as unverified and say what it means:
**nothing synchronizes with an unverified peer in either direction**, so a room
they are in will look quiet rather than broken. Verifying is done in a terminal, on
a call with that person — run `/cogmer:peer-pair`, which prints the exact command
with the right path in it. Do not write the command out yourself; a bare `cogmer`
does not resolve.
