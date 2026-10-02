# D-007 — Record `COMPACTION` events as observability, not remediation

**Date:** 2026-09-16 · **Status:** not built

**Decision.** The daemon records an event when a session is compacted, so that a room's
history shows when compaction happened. The event changes nothing about delivery.

**Support.**
- D-006 (compaction resets no delivery state) rests on the summarizer keeping a
  colleague's turns, and nothing guarantees it will. B19.
- Claude Code runs a hook before compacting, and the hook reports whether the
  compaction was manual or automatic. `84a0751:docs/phase0a-findings.md`.

**Rejected.**
- *Recording nothing.* If D-006 ever becomes wrong, nothing would show when a
  compaction happened, and nobody could connect a lost turn to one.

**Revisit when** a compaction is found to have lost a colleague's turn.
