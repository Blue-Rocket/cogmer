# D-176 — The standing policy for room content is also stated once at session start

**Date:** 2026-09-20 · **Status:** active · **Areas:** trust, capture, daemon

**Decision.** The policy for reading room content is stated once at session start through
the session-start hook's `additionalContext`, in addition to the framing inside each block.

**Support.**
- A position the content cannot occupy is not one it can imitate, and framing that travels
  in the same text as the content sits where a model may discount instructions.
  `plugin/hooks-handlers/session-start.sh`.
- The framing inside the block is tested and survives compaction in a way one statement may
  not, so the standing policy reinforces it and does not replace it. `cmd/cogmer/daemon.go`,
  `FormatTeamContext`.
- That hook output is positioned as data and not as the session's instruction is a reliance
  on Claude Code that a registry check records. B22 (injected hook output is positioned as
  data, not as the session's own instruction).

**Rejected.**
- *The framing inside the block alone.* It shares a position with the content it governs.

**Limits.** The policy could not be shown to help. The same hostile turn was refused with
and without it, and the in-block framing alone produced the same refusal and the same report
to the user, so it is defense in depth against a position problem that is real in principle
and was not observed. `f288913:docs/decisions/D-081-untrusted-turns-are-json-the-policy-is-stated-separately.md`.
