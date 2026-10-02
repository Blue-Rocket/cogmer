# D-033 — The room cannot be displayed inside Claude Code; asking is the free affordance

**Date:** 2026-09-17 · **Status:** active · **Areas:** view, capture

**Decision.** Nothing shows the room inside a Claude Code session. Inside a session, a user
learns what the room holds by asking their own Claude, which answers from the context
injected, and a view of the room is a separate program outside the session.

**Support.**
- A hook's standard output becomes context for the model and never appears on screen,
  and neither its standard error nor a write to `/dev/tty` is shown.
  `84a0751:docs/decisions.md`.
- Asked what the team was discussing, a session answered who said what, and that they
  were unverified, from injected context alone. `84a0751:docs/decisions.md`.
- Every Claude Code extension point delivers to the model, and nothing displays to a
  person. D-036 (MCP logging notifications are not a display channel).

**Rejected.**
- *Making the injected block readable, so that it doubles as the display.* The block is
  never shown.
- *Writing to the terminal from a hook.* Nothing written there is shown, and it would
  contend with Claude Code's own rendering if it were.

**Revisit when** Claude Code shows hook output to the user. Nothing checks that
automatically, since it needs a terminal and somebody watching.
