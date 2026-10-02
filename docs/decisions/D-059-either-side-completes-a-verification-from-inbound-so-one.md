# D-059 — Either side completes a verification from inbound, so one side can drive it

**Date:** 2026-09-18 · **Status:** active · **Areas:** pairing

**Decision.** A peer whose own session holds the other's revealed nonce has what
the two words need and stops driving. Only one side has to run the exchange, and both
learn the words. The loop checks for an inbound completion first and then attempts a
round outbound.

**Support.**
- A side that completes ends its session, so a peer that began a moment later finds
  nothing to answer and waits out the timeout, which happens whenever two people type a
  few seconds apart. `cmd/cogmer/verify.go`, `RunVerification`.
- The inbound path reads only a nonce that arrived through `handleVerify`, which refuses
  a reveal without a commitment and a reveal that does not open it, so commit, commit,
  reveal, reveal still holds. `cmd/cogmer/sas_test.go`, `TestRevealBeforeCommitIsRefused`
  and `TestARevealMustOpenItsCommitment`.
- One side reaches the words from an address on the other side alone, and the first to
  type reaches them when the other types later. `cmd/cogmer/sas_test.go`,
  `TestOnlyOneSideNeedsAnAddress` and `TestTheFirstToTypeWaitsForTheOther`.

**Rejected.**
- *Each side driving its own exchange.* It reads as symmetric and fails whenever the two
  people type seconds apart, because the side that finishes strands the other.

**Revisit when** a third peer verifies. Nothing here assumes two, and nothing has
exercised more.
