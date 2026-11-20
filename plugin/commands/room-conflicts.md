---
description: Show quarantined events for this room
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" conflicts:*)
disable-model-invocation: true
---

## What happened

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" conflicts`

## If the output above is missing

If the section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run it yourself with the Bash tool, once, and take what it prints as that section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" conflicts
```

If the section above shows cogmer's output, run nothing.

## Your task

If there are none, say so in one line.

If there are any, do not summarize them away. A conflict means a peer's sequence
counter went backwards — it lost its local state, or an event was forged — and its
events since then are **not stored and cannot be recovered by synchronization**.
Say which peer, and say that the gap is permanent unless that peer still holds the
events.
