# D-163 — Only a person's confirmation records a verification, and a mismatch records nothing

**Date:** 2026-09-17 · **Status:** active · **Areas:** pairing

**Decision.** `MarkVerified` is reached only from a user's confirmation that the other
user read out the same words, never from the protocol. A mismatch records nothing, and no
failed-verification state exists.

**Support.**
- The exchange proves that both sides hold the keys they named, and it cannot show that
  the voice on the call is the colleague and not somebody in their place. §25 (security),
  `cmd/cogmer/membership.go`, `MarkVerified`.
- A stored failure would invite an interface that offers to retry, and a mismatch must
  refuse and must not present itself as a transient error worth repeating.
  `cmd/cogmer/sas.go`.
- `known_peers.verified_at` is empty until two users compare words, and the unverified
  marker in injected text follows it. `cmd/cogmer/sas_test.go`,
  `TestTheUnverifiedMarkerTracksVerification`.

**Rejected.**
- *Recording a failed verification.* It invites a retry, which is the one thing a
  mismatch must not offer.

**Revisit when** an approach is found that is equally secure with fewer manual steps.
