# D-036 — MCP logging notifications are not a display channel

**Date:** 2026-09-17 · **Status:** active

**Decision.** cogmer shows nothing to a user through MCP. MCP carries capability to the
model, and nothing to a person.

**Support.**
- A minimal MCP server that declared the logging capability and emitted
  `notifications/message`, while idle and again during a tool call, had its messages
  shown nowhere: not in `--output-format stream-json`, `--debug`, `--debug-file` or
  `~/.claude/debug`. `84a0751:docs/decisions.md`.
- Claude Code's record of what it negotiated with the server lists tools, prompts,
  resources and resource subscriptions, and no logging, although the server declared it.
  `84a0751:docs/decisions.md`.

**Rejected.**
- *An MCP server as the carrier for showing a user colleagues' turns.* It loads without
  changing how Claude Code starts and reaches every surface, and what it emits reaches
  nothing a person sees.

**Revisit when** Claude Code's record of negotiated capabilities includes logging, or it
shows an MCP server's notifications.
