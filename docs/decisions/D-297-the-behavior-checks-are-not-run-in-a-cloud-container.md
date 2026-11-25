# D-297 — The behavior checks are not run in a cloud container

**Date:** 2026-11-25 · **Status:** active · **Areas:** behaviors, hosts

**Decision.** The behavior check that runs when a room is formed (D-129, behavior checks run by themselves when a room is formed) does not run where `CLAUDE_CODE_REMOTE` is `true` or the cloud key is set, which is a Cowork cloud task or another Claude Code cloud session. It records no result and prints nothing there. The check drives `claude -p` in a probe session, which fails in a cloud container with an authentication error, and the behaviors the check watches are those of the Claude Code on the user's own computer. This is an exception to D-129 for a cloud container, and `COGMER_PREFLIGHT=off` still skips the check anywhere.

**Rejected.**
- *Running the check and leaving its failure out of the output.* A probe that cannot succeed would run at each unchecked version and record a failure that says nothing about the host. In a Cowork cloud task on 2026-11-24 the room-join output said the probe session hit an authentication error (`e48ef29`).
- *Giving the container a login for the probe.* cogmer never holds a user's login, and a cloud task has none to offer.

**Limits.** Nothing checks the Claude Code that Anthropic runs in a cloud container, so a change in what cogmer relies on there shows only as the failure it causes. A Codex cloud task is not covered, since the Codex preflight does not look for a cloud container.

**Revisit when** a cloud container holds a login that a probe session can use, or Anthropic documents a way to check the Claude Code that runs there.
