# D-170 — Injected context has a character budget for the whole block

**Date:** 2026-09-18 · **Status:** active · **Areas:** capture

**Decision.** After the cap on events and the cap on characters in one turn, a budget on
the characters of the whole block as it is written applies, and it drops the oldest turns first,
and shortens a single turn that is still over. The budget is 9,000 characters and a turn is cut to
5,000, which keeps the block under the 10,000 characters at which Claude Code stops injecting a
hook's output whole.
Each of the three limits is read from `COGMER_MAX_EVENTS`, `COGMER_MAX_EVENT_CHARS` or
`COGMER_MAX_BLOCK_CHARS` each time context is injected, and a value that is zero or not a
number is ignored.

**Support.**
- The other two limits do not bound the block, since forty turns of ten thousand
  characters each pass both and the block holds four hundred thousand. `cmd/cogmer/delivery_test.go`,
  `TestTheInjectedBlockIsBounded`.
- The budget is measured on the block as written and not estimated from its inputs, since the
  framing and the escaping in JSON add to the turns, and so it bounds what reaches the context
  window. `cmd/cogmer/daemon.go`, `FormatTeamContext`, `cmd/cogmer/delivery_test.go`,
  `TestTheBlockIsKeptUnderWhatTheHostInjectsWholeWhateverTheTurnsHold`.
- Claude Code stores a hook's output of more than about 10,000 characters in a file and gives the
  model a preview of its first part, and the newest turns are at the end of a block. The transcript
  records the preview, which is not the block that was offered, so the block is never confirmed
  delivered and is offered again at every prompt. B34 (a hook's output as long as the daemon's block
  budget is recorded whole and not stored as a preview).
- The newest turns survive, because a referent is most likely to point at recent
  conversation, and a note says that earlier turns were left out. §21 (context window
  management), `cmd/cogmer/delivery_test.go`, `TestTheInjectedBlockIsBounded`.
- A limit of zero would inject nothing, which looks like a room where nobody is talking,
  so it is ignored along with a value that does not parse. `cmd/cogmer/delivery_test.go`,
  `TestLimitsAreConfigurable`.

**Rejected.**
- *The caps on events and on one turn alone.* Neither bounds what the block holds.
- *Deriving an estimate of the tokens from the character count.* Nothing reads the figure, and a
  tokenizer would have to follow whichever model the session uses, which cogmer does not choose,
  so the limits are counted in characters and nothing is derived from them.
  `655f0bf:docs/decisions/D-171-the-token-count-of-injected-context-is-estimated-from.md`.

**Revisit when** B34 fails, or a room routinely exceeds the block budget. A room is scoped to a session,
so hitting a limit means an unusually long pairing, and hitting one often means rooms are
not as session-scoped in practice as D-015 (rooms are scoped to sessions, not to
projects) assumes.
