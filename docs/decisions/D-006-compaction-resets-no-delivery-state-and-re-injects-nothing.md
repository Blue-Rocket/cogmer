# D-006 — Compaction resets no delivery state and re-injects nothing

**Date:** 2026-09-16 · **Status:** active · **Areas:** capture

**Decision.** A session's delivery state is left as it is at a compaction, and nothing
is injected again afterwards.

**Support.**
- Injected turns survived a compaction, with their attribution, both when they were
  the conversation's subject and when four unrelated turns followed them, with nothing
  injected again in either run. `84a0751:docs/phase0a-findings.md`, "Injected context
  survives".
- The check for B19 fails if a colleague's turn is lost at a compaction. B19.

**Rejected.**
- *Resetting delivery state before a compaction.* Every compaction would inject the
  same turns again, for no benefit.
- *Injecting a bounded number of recent turns after every compaction.* The same cost,
  and the same absence of benefit.

**Limits.** Survival is the summarizer's judgment, not a guarantee of the format, and
could change with a different model, a longer conversation or repeated compactions.
Only one compaction per session was tested.

**Revisit when** the check for B19 fails.
