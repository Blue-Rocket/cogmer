# D-055 — A peer is verified in one way, the two-word comparison

**Date:** 2026-09-18 · **Status:** active · **Areas:** pairing

**Decision.** The live two-word comparison is the only way to verify a peer, with no
whole-key comparison beside it. `Fingerprint` renders an identifier for a person to
read, and marks nobody verified.

**Support.**
- Verification gates synchronization, so it matters only when collaboration is about to
  happen, and collaboration needs both daemons running and reachable, which is what the
  live exchange needs. D-054 (verification gates synchronization and injection).
- A gate is only as strong as the weakest ceremony that satisfies it, and a reader shown
  forty-three characters of base64 compares the first group and the last and skims the
  middle. `249ddd0:docs/decisions.md`, the entry D-047 held there.
- `verified_at` is set only by the user's confirmation of the two words.
  `cmd/cogmer/verify.go`, `handleVerifyConfirm`.
- `Fingerprint` serves the mismatch alarm, where two keys sit side by side, and its test
  requires that it carry the whole identifier. `cmd/cogmer/keys_test.go`,
  `TestFingerprintIsLosslessAndDisplayOnly`.
- A short string is sound where a static one is not, which is what makes the single
  method defensible. D-048 (verification is a live commit-then-reveal exchange that
  derives two words).
- The specification says there is one ceremony. §25 (security).

**Rejected.**
- *Rendering the fingerprint as words to compare, beside the two words.* Two ceremonies
  leave the weaker one to be performed, and a reader shown forty-three characters of
  base64 compares the first group and the last and skims the middle.
  `249ddd0:docs/decisions.md`, the entry D-047 held there.
- *Keeping the whole-key comparison as a fallback where no live exchange is possible.*
  No state exists in which a peer needs verifying and cannot be verified by the two
  words, and a fallback is the path people reach for when the other is inconvenient.

**Limits.** Looking at a fingerprint records nothing.

**Revisit when** two users who must verify cannot have their daemons reach each other.
