# D-054 — Verification gates synchronization and injection, not just a marker

**Date:** 2026-09-18 · **Status:** active

**Decision.** An unverified peer's events are not served, not stored and not injected,
and they are held and not discarded. Serving refuses a sync request from an unverified
guest, accepting refuses an event whose origin peer is unverified, and the prompt hook
filters again before injecting.

**Support.**
- Admission decides whether a key may enter, and verification decides whether the key is
  the person's. Every other check passes for a key substituted in transit, because a
  substituted key is a real key held by whoever substituted it.
  `249ddd0:docs/decisions.md`, the entry D-047 held there.
- The serving gate runs after the guest check, so a caller that reaches it is a recorded
  guest and learns nothing it did not put there. `cmd/cogmer/auth.go`, `verifyRequest`.
- The accepting gate judges the origin and not the sender, so a verified relay cannot
  launder an unverified author. §13 (transitive synchronization),
  `cmd/cogmer/sync.go`.
- The injecting gate runs again because an event stored while its peer was verified
  outlives a later `forget`, and a context window has no delete. `cmd/cogmer/daemon.go`,
  `onlyVerified`.
- Refusing to store leaves an event on offer and does not advance the pull watermark, so
  verification brings the whole backlog on the next poll. `cmd/cogmer/sync.go`.
- The refusal names the peer and the command, so a quiet room is not mistaken for a room
  where nobody is talking. `cmd/cogmer/auth.go`, `verifyRequest`.
- A request correct in every other respect, from a real guest with a real signature, is
  refused until the peer is verified, and an unverified peer's events are not injected.
  `cmd/cogmer/auth_test.go`, `TestAnUnverifiedGuestIsRefused` and
  `TestUnverifiedEventsAreNotInjected`.

**Rejected.**
- *A marker on an unverified peer's turns, with the turns let through.* A marker is
  right to show once content is in front of a reader, but it is not a control, and the
  manual step the whole chain rests on is skipped if skipping it costs nothing.

**Limits.** Inviting an unverified peer is permitted and has no effect until the peer is
verified. D-051 (stranger pairing is not a supported case) leaves the choice of whom to
admit with the host.

**Revisit when** two users who cannot verify must collaborate. §25 (security) names that
case, and a real situation should decide it.
