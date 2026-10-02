# D-005 — `mergeTail` tolerates a widened `last_assistant_message`

**Date:** 2026-09-16 · **Status:** active · **Areas:** capture

**Decision.** When `last_assistant_message` contains what the transcript supplied,
`mergeTail` uses it alone rather than appending it.

**Support.**
- D-003 (reassemble turns from two sources) appends `last_assistant_message` to the
  text from the transcript, which is correct only while it holds just the final block.
  `cmd/cogmer/transcript.go`, `mergeTail`.
- The check for B04 fails if `last_assistant_message` comes to hold the whole turn.
  B04.

**Rejected.**
- *Appending it unconditionally.* If Claude Code widened the field, every block said
  before a tool call would be repeated, and the room would be corrupted silently by a
  fix upstream.

**Revisit when** the check for B04 fails, when reading the transcript may become
unnecessary.
