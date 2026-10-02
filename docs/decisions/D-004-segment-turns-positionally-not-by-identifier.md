# D-004 — Segment turns positionally, not by identifier

**Date:** 2026-09-16 · **Status:** active

**Decision.** A turn's assistant records are every assistant record after the last user
record that carries a `promptSource`.

**Support.**
- Assistant records carry no `promptId`, although user records do, including the
  records of tool results. B09.
- Human prompts carry a `promptSource`, and the records of tool results do not. B06.
- The `parentUuid` chain has gaps: an assistant record was seen whose `parentUuid`
  matched no `uuid` in the same file. `84a0751:docs/phase0-findings.md`.

**Rejected.**
- *Correlating on `promptId`.* Assistant records carry none.
- *Walking the `parentUuid` chain.* The chain has gaps.

**Revisit when** the check for B09 fails, since a `promptId` on assistant records would
allow exact correlation instead.
