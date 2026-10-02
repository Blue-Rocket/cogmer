# D-011 — Hooks fail open: exit 0, empty stdout

**Date:** 2026-09-16 · **Status:** active · **Areas:** daemon

**Decision.** Every path by which a hook fails exits 0 and writes nothing to stdout,
so a dead daemon means no collaboration and never a broken session.

**Support.**
- A session is never worse for having cogmer installed. §3.1 (first, do no harm).
- The prompt hook's stdout is injected into the user's turn. B02.
- A hook whose daemon is down exits 0 with empty output. B11, and
  `plugin/hooks-handlers/run.sh`.
- With the daemon down, the prompt hook took 17ms, because a refused connection
  returns at once rather than waiting out the timeout.
  `84a0751:docs/phase0-findings.md`.

**Rejected.**
- *Exiting non-zero on failure.* Claude Code would show an error on every prompt.
- *Explaining the failure on stdout.* It would be injected into the user's
  conversation, so diagnostics never go to stdout.

**Revisit when** Claude Code offers a hook a way to report a failure that reaches
neither the model nor the user's turn.
