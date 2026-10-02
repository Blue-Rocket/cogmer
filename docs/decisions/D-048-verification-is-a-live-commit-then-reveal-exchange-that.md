# D-048 — Verification is a live commit-then-reveal exchange that derives two words

**Date:** 2026-09-17 · **Status:** active · **Areas:** pairing

**Decision.** Each side commits to a hash of its contribution, and then both reveal.
The string derives from both long-term identity keys and both fresh nonces, and each
side prints two words for the two users to compare aloud on the call they
are on. A mismatch refuses, says plainly that something intercepted the exchange, and is
never presented as a transient error worth repeating.

**Support.**
- A standing identifier can be ground against offline at a cost of the entropy shown,
  so it has to be compared in full. A committed, freshly randomised value cannot be
  aimed at in advance, so a short form is sound. `cmd/cogmer/sas.go`, and ZRTP's short
  authentication string, https://www.rfc-editor.org/rfc/rfc6189.
- The commitment makes each side fix what it presents before it sees the other's, so
  a relaying attacker is reduced to one blind guess, and the nonces leave nothing to
  precompute. `cmd/cogmer/sas_test.go`, `TestARevealMustOpenItsCommitment`.
- The two words carry 16 bits, so a blind guess succeeds once in 65,536. A short
  string with silent retries is weak, which is why a mismatch has to stop a person.
  `cmd/cogmer/sas.go`.
- Every other guarantee binds a key to itself and none binds a key to a person. An
  attacker who substituted her own identifier in transit was invited under the name
  `alice`, read a private room and replied into it, with no refusal anywhere.
  `249ddd0:docs/decisions.md`, the entry D-047 held there.
- A reader shown forty-three characters of base64 to check over a telephone compares
  the first group and the last and skims the middle, so the entropy verified is far
  less than the entropy shown. `249ddd0:docs/decisions.md`, the entry D-047 held there.
- The exchange authenticates whichever key arrived, so the identifier can be sent by any
  means. §12 (forming a room), `cmd/cogmer/sas.go`.

**Rejected.**
- *Comparing the whole key as well, for peers whose daemons cannot reach each other.*
  A second, harder ceremony beside an easier one is a downgrade path, and the gate is
  only as strong as the weakest ceremony that opens it. D-055 (one way to verify a
  peer).

**Limits.** Two people who have never met gain nothing, since recognizing a voice
presumes acquaintance. D-051 (stranger pairing is not a supported case). Nothing
rate-limits attempts, so protection against repetition rests on a mismatch stopping a
person, which is a claim about the interface and not the protocol. A commitment step
implemented wrongly degrades to a value that can be ground while still looking like a
ceremony, and the test above fails when the check is removed.

**Revisit when** verification is wanted between peers that cannot be connected at the
same moment, since the short form is unavailable there.
