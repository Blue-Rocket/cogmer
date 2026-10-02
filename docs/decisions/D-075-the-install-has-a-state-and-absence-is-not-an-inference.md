# D-075 — The install has a state, and absence is not an inference

**Date:** 2026-09-20 · **Status:** active · **Areas:** release, daemon

**Decision.** The installer writes one state file that everything reads. It holds
`installing` when the work begins and `failed` with the reason when it does not, and it is
removed on success. An `installing` mark older than five minutes is reported as `stalled`.
The command wrapper names the state and what to do, and the session-start hook gives it to
the model as `additionalContext` while the binary is absent and at no other time.

**Support.**
- A missing binary meant three things, that nothing had started, that a download was in
  progress, or that an install had failed and was waiting out its cooldown, and one
  hopeful answer for all three sent a person to wait for something that would not happen.
  `plugin/hooks-handlers/common.sh`, `install_state`.
- The cooldown reads the same mark, so a wait is explained and not mysterious. D-172 (the
  installer runs detached, one at a time, outside the plugin directory, and waits an hour
  after a failure).
- A crash between setting the mark and clearing it would leave a download in progress for
  ever, which is the one answer worse than silence, so an old `installing` reads as
  stalled. `plugin/hooks-handlers/common.sh`, `install_state`.
- Everything a person learns arrives by way of what the model says, so the hook gives the
  state to the model, and a session asked why nothing works answers from the fact and does
  not speculate. D-033 (the room cannot be displayed inside Claude Code), D-036 (MCP
  logging notifications are not a display channel), `plugin/hooks-handlers/session-start.sh`.
- The hook emits nothing when the binary is present, so an ordinary session carries
  nothing. `plugin/hooks-handlers/session-start.sh`.

**Rejected.**
- *Telling a session that the download may still be arriving.* It is false for a failed
  install.
- *Notifying, interrupting or polling about progress.* A hook must not break a session,
  and a progress message nobody asked for is a small way of breaking one. §3.1 (first, do
  no harm).
