---
description: List the peers this machine knows, and whether each is verified
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" peers:*)
disable-model-invocation: true
---

## Known peers

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" peers`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" peers
```

If the section above shows cogmer's output, run nothing.

## Your task

List them plainly, and give each peer's three names as the output labels them: what the
user calls them, the name derived from their key, and what they call themselves. The last
is their own claim, so never present it as a name the user chose or as proof of who they
are, and never merge the three into one. Mark anyone unverified as unverified and say what it means:
**nothing synchronizes with an unverified peer in either direction**, so a room
they are in will look quiet rather than broken. Verifying is done in a terminal, on
a call with that person — run `/cogmer:peer-pair`, which prints the exact command
with the right path in it. Do not write the command out yourself; a bare `cogmer`
does not resolve.
