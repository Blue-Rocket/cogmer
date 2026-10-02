# D-014 — Derive delivery state from transcript evidence, not from recorded intent

**Date:** 2026-09-16 · **Status:** active · **Areas:** capture

**Decision.** A colleague's turn counts as delivered to a session only when the daemon
finds the injected block in that session's transcript, matched by a SHA-256 hash of
the exact text the hook emitted. Delivery is recorded as a set of events, not as a
position in the room's stream.

**Support.**
- Claude Code records a hook's stdout in the transcript as a `hook_success`
  attachment. B20.
- The daemon reads the transcript when each turn completes, to reassemble the turn.
  D-003 (reassemble turns from two sources).
- A colleague's turn counts as delivered only once the session has received it.
  §19 (incremental context injection).
- The attachments remain in the transcript across turns and through compaction, so an
  injection missed when its own turn completed is confirmed when a later one does.
  `cmd/cogmer/delivery_test.go`, `TestConfirmationIsSelfHealing`.
- A lost injection leaves a gap that a single position in the stream cannot
  represent. `cmd/cogmer/store.go`, `session_delivered`.

**Rejected.**
- *Counting a turn delivered when it is injected.* The hook has a 3s timeout, so a
  reply that expires or is lost would record delivery while Claude saw nothing.
- *Counting it provisionally at injection and confirming it when the turn completes.*
  That the turn completed says nothing about whether the context arrived.
- *Putting each event's identifier in the injected block.* At about 34 characters an
  event, it adds noise to every colleague's context window, and hashing the block
  gives the same mapping.

**Revisit when** offering undelivered turns again proves more disruptive than the loss
it prevents.
