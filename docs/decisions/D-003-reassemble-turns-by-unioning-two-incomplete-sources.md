# D-003 — Reassemble turns by unioning two incomplete sources

**Date:** 2026-09-16 · **Status:** active · **Areas:** capture

**Decision.** A completed turn is reassembled in `ReassembleLastTurn` from the union
of the transcript, read when the turn completes, and the Stop hook's
`last_assistant_message`.

**Support.**
- `last_assistant_message` holds only the turn's final text block. B04.
- The transcript, read when Stop fires, lacks that final block, because Stop fires
  before it is written. B05.
- Together the two give the complete turn. B12, and §15 (capturing Claude responses).

**Rejected.**
- *The transcript alone.* It loses the final block, which is usually most of the
  answer.
- *`last_assistant_message` alone.* It loses everything said before a tool call, which
  is most of a substantive turn.
- *Waiting and retrying until the transcript settles.* It adds latency to every turn
  and is still a race, only a longer one.

**Revisit when** the check for B04 or B05 fails.
