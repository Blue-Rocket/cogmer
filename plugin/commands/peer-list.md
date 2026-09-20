---
description: List the peers this machine knows, and whether each is verified
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" peers:*)
---

## Known peers

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" peers`

## Your task

List them plainly. Mark anyone unverified as unverified and say what it means:
**nothing synchronizes with an unverified peer in either direction**, so a room
they are in will look quiet rather than broken. Verifying is done in a terminal —
`/peer-pair` explains how.
