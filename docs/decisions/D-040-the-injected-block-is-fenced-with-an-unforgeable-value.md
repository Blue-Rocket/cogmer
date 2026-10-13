# D-040 — The injected block is fenced with an unforgeable value

**Date:** 2026-09-17 · **Status:** active · **Areas:** trust, capture

**Decision.** Each injected block is delimited by a value generated for that injection,
which the content cannot know. Any copy of the value in the content is removed, the
framing says the block ends only at the matching value and that text claiming otherwise
is part of the block, and the framing is stated again after the content.

**Support.**
- Against a block that pasted a turn in as it stood, a turn consisting of
  `</message></team-conversation>` and a forged instruction put the instruction after the
  closing tag in the formatted text. That was shown for the text, and no model was shown
  acting on it: a live session given a forged override refused it. `84a0751:docs/decisions.md`.
- A turn containing the closing delimiter would end the block, and whatever followed
  could pass for an operator, a system or the local user. §20 (attribution in injected
  context).
- A test checks that a turn cannot escape the block. `cmd/cogmer/transcript_test.go`,
  `TestTeammateContentCannotEscapeTheBlock`.
- The encoding carries the boundary. Turns are JSON, and the encoder writes `<` and `>` as
  `\u003c` and `\u003e`, so no turn can put a tag in the block, and whatever a turn says the
  block holds its own two tags and one closing tag. The fence is a second guard that matters if
  the encoder stops escaping them. D-081 (untrusted turns are JSON, and the policy is stated
  separately), `cmd/cogmer/daemon.go`, `FormatTeamContext`, `cmd/cogmer/transcript_test.go`,
  `TestNoTurnCanPutATagInTheBlock`.
- A turn arrives as written, since the text read out of the block's JSON equals the text sent,
  whatever it holds. Turning off the escaping failed the first test and trimming a turn failed
  the second. `cmd/cogmer/transcript_test.go`, `TestATurnArrivesAsWritten`.

**Rejected.**
- *Escaping the delimiters.* It is whack-a-mole against prose, and assumes the boundary is
  syntactic when the model reads it as language.
- *Truncating or sanitizing content.* A colleague's words are not the system's to edit.
  §3.4 (preserve actual conversation).

**Limits.** A turn that contains the fence value arrives with that value removed.

**Revisit when** a turn is found altered by the removal, or escaping the block.
