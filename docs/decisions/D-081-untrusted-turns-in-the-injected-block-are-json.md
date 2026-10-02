# D-081 — Untrusted turns in the injected block are JSON

**Date:** 2026-09-20 · **Status:** active · **Areas:** trust, capture

**Decision.** The turns in the injected block are JSON-encoded, with the per-injection fence
kept beside the encoding.

**Support.**
- A value cannot leave a JSON string without an unescaped quote, and the encoder guarantees
  there is not one, so delimiting is structural and not a matter of care. `cmd/cogmer/daemon.go`,
  `FormatTeamContext`, and `cmd/cogmer/transcript_test.go`, `TestTeammateContentCannotEscapeTheBlock`.
- The fence is tested, costs nothing, and does one thing JSON does not: it lets the framing
  say where the block ends in a way the content cannot imitate. `cmd/cogmer/daemon.go`,
  `FormatTeamContext`, D-040 (the injected block is fenced with an unforgeable value).
- A registry check fails if the block stops being JSON. B22 (injected hook output is
  positioned as data, not as the session's own instruction), `cmd/cogmer/behaviors.go`.

**Rejected.**
- *Interpolating turns into markup and escaping by hand.* Escaping by hand is escaping by
  hand.
- *Screening each injection through a classifier.* It needs an inference call for every
  injection and spends the user's own quota on every colleague's turn.
