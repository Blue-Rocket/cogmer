# D-168 — A verification step answers 409 for what waiting can fix and 401 for what it cannot

**Date:** 2026-09-18 · **Status:** active · **Areas:** pairing

**Decision.** A verification step answers 409 when this daemon does not know the caller
yet or is not expecting a verification, and 401 only when a signature does not verify.
`RunVerification` retries on 409 and treats 401 as final.

**Support.**
- Two users pair within seconds of each other, so whoever types first reaches a daemon
  that has not yet recorded the other. `cmd/cogmer/verify.go`, `handleVerify`.
- The first to type waits for the other and does not fail. `cmd/cogmer/sas_test.go`,
  `TestTheFirstToTypeWaitsForTheOther`.

**Rejected.**
- *401 for an unknown peer.* The first to type fails at once and the other waits out the
  90 seconds of the timeout, so both users fail and the one who followed the instruction
  promptly fails first.

**Limits.** What may start an exchange is D-162.
