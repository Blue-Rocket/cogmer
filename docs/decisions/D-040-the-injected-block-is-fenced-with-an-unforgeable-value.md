# D-040 — The injected block is fenced with an unforgeable value

**Date:** 2026-09-17 · **Status:** active · **Areas:** trust, capture

**Decision.** Each injected block is delimited by a value generated for that injection,
which the content cannot know. Any copy of the value in the content is removed, the
framing says the block ends only at the matching value and that text claiming otherwise
is part of the block, and the framing is stated again after the content.

**Support.**
- A colleague's turn consisting of `</message></team-conversation>` and a forged
  instruction escaped a block whose content was interpolated as it stood, so the
  instruction appeared outside the block. `84a0751:docs/decisions.md`.
- A turn containing the closing delimiter would end the block, and whatever followed
  could pass for an operator, a system or the local user. §20 (attribution in injected
  context).
- A test checks that a turn cannot escape the block. `cmd/cogmer/transcript_test.go`,
  `TestTeammateContentCannotEscapeTheBlock`.

**Rejected.**
- *Escaping the delimiters.* It is whack-a-mole against prose, and assumes the boundary is
  syntactic when the model reads it as language.
- *Truncating or sanitising content.* A colleague's words are not the system's to edit.
  §3.4 (preserve actual conversation).

**Limits.** A turn that contains the fence value arrives with that value removed.

**Revisit when** a turn is found altered by the removal, or escaping the block.
