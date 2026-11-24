# D-003 — Reassemble turns by unioning two incomplete sources

**Date:** 2026-09-16 · **Status:** active · **Areas:** capture

**Decision.** A completed turn is reassembled in `ReassembleLastTurn` from the union
of the transcript, read when the turn completes, and the Stop hook's
`last_assistant_message`.

**Support.**
- `last_assistant_message` holds only the turn's final text block. B04.
- Together the two give the complete turn on either kind of version. B12, and §15
  (capturing Claude responses).

**Rejected.**
- *The transcript alone.* On a version that writes the final block after Stop fires it
  loses that block, which is usually most of the answer, and an installed Claude Code
  can be any version.
- *`last_assistant_message` alone.* It loses everything said before a tool call, which
  is most of a substantive turn.
- *Waiting and retrying until the transcript settles.* It adds latency to every turn
  and is still a race, only a longer one.

**Revisit when** the check for B04 fails, or when no Claude Code version that cogmer runs
against writes the final block after Stop fires, which would make `last_assistant_message`
redundant.
