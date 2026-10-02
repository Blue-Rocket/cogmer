# D-102 — The reveal step tolerates a peer that has finished

**Date:** 2026-09-20 · **Status:** active · **Areas:** pairing

**Decision.** A reveal that finds the other side not expecting a verification any more goes back
to the loop, as a failed commit does, and the inbound check at the top of the loop looks for
the nonce the finished side has revealed.

**Support.**
- Nothing is not ready between one call and the next, but something can stop being ready: the
  other side can complete between our commit and our reveal, and its cleanup discards its
  session, so one side finished with two words while the other reported that its peer was not
  expecting a verification. `cmd/cogmer/verify.go`, `RunVerification`.
- By the time the other side has finished it has sent its own reveal, so its nonce is in this
  session and the inbound check finds it. D-059 (either side completes a verification from
  inbound, so one side can drive it).

**Revisit when** the exchange gains a third round trip, which would widen the same window.
