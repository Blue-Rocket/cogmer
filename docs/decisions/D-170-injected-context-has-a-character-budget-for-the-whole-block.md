# D-170 — Injected context has a character budget for the whole block

**Date:** 2026-09-18 · **Status:** active · **Areas:** capture

**Decision.** After the cap on events and the cap on characters in one turn, a budget on
the characters of the whole rendered block applies, and it drops the oldest turns first.
Each of the three limits is read from `COGMER_MAX_EVENTS`, `COGMER_MAX_EVENT_CHARS` or
`COGMER_MAX_BLOCK_CHARS` each time context is injected, and a value that is zero or not a
number is ignored.

**Support.**
- The other two limits do not bound the block, since forty turns of ten thousand
  characters each pass both and the block holds four hundred thousand. `cmd/cogmer/delivery_test.go`,
  `TestTheInjectedBlockIsBounded`.
- The budget is measured on the rendered turns and not estimated from their inputs, so it
  bounds what reaches the context window. `cmd/cogmer/daemon.go`, `renderedSize`.
- The newest turns survive, because a referent is most likely to point at recent
  conversation, and a note says that earlier turns were left out. §21 (context window
  management), `cmd/cogmer/delivery_test.go`, `TestTheInjectedBlockIsBounded`.
- A limit of zero would inject nothing, which looks like a room where nobody is talking,
  so it is ignored along with a value that does not parse. `cmd/cogmer/delivery_test.go`,
  `TestLimitsAreConfigurable`.

**Rejected.**
- *The caps on events and on one turn alone.* Neither bounds what the block holds.

**Revisit when** a room routinely exceeds the block budget. A room is scoped to a session,
so hitting a limit means an unusually long pairing, and hitting one often means rooms are
not as session-scoped in practice as D-015 (rooms are scoped to sessions, not to
projects) assumes.
