# D-090 — Attribution in the injected block is structured, and nothing of ours is joined to a peer's text

**Date:** 2026-09-20 · **Status:** active · **Areas:** trust, capture, identity

**Decision.** In the injected block the derived name, the verified state, the kind of turn,
whether the turn is this user's own from another session, and the name of the machine the
turn came from are separate fields, and nothing cogmer states is concatenated with text a
peer wrote.

**Support.**
- A display name of `Alice (quiet-otter)` joined to a parenthetical that is meant to read as
  the derived identity produced a speaker that imitates it, and the test of escaping passed,
  because escaping stops a value leaving its field and does nothing about a value imitating
  the field beside it. `cmd/cogmer/transcript_test.go`,
  `TestACraftedDisplayNameCannotImitateTheDerivedName`.
- The derived name is its own field, and so are the verified state, a boolean, and whether the
  turn came from a person or from their Claude. `cmd/cogmer/daemon.go`, `FormatTeamContext`.
- A machine's name is chosen by its user, so it is a field of its own, marked as asserted, and
  never joined to a name. D-233 (every turn carries the name of the machine it came from, as a
  field the user asserts).
- Whether a turn is this user's own is set by the receiving machine and never by the turn's
  text. D-232 (a user's own turn from another session is injected as a report marked as the
  user's).
- It applies one level further in what D-040 (the injected block is fenced with an
  unforgeable value) and D-081 (untrusted turns in the injected block are JSON) do, which is
  that content cannot close a block or leave a string.

**Rejected.**
- *One speaker string made of a display name and a derived name.* A peer chooses the first
  part.

**Limits.** The view shows the two names in separate elements with similar visual weight, and
what a person skimming a room actually notices has not been measured.

**Revisit when** another field is added that a peer can influence.
