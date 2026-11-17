# D-199 — No event records a compaction

**Date:** 2026-10-12 · **Status:** active · **Areas:** capture, behaviors

**Decision.** The daemon records nothing when a session is compacted, and no event type exists for one.

**Support.**
- Injected teammate context survived a compaction in every test, with attribution intact, including a
  turn the conversation never discussed, so a compaction changes nothing the room holds or delivers.
  `84a0751:docs/phase0a-findings.md`, D-006 (compaction resets no delivery state and re-injects
  nothing).
- An event is signed, relayed to every machine and never edited, so an event type added for one
  purpose is part of what every machine holds from then on. D-058 (an event is verified under
  the scheme it was signed with), §7 (event model).

**Rejected.**
- *Recording an event when a session is compacted, so that a room's history shows when it happened.*
  The event would have changed nothing about delivery, and its use would have been to connect a lost
  turn to a compaction. Injected context survived every compaction tried, and the event would be
  permanent for every machine. `84a0751:docs/phase0a-findings.md`.
