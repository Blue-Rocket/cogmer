# D-162 — A daemon takes part in a verification only while its own user has asked for one

**Date:** 2026-09-17 · **Status:** active · **Areas:** pairing, trust

**Decision.** A daemon answers a peer's verification step only when it holds a session its
own user opened, and otherwise answers 409 and displays nothing to anyone. The caller
must also be a peer this machine knows.

**Support.**
- Only the user's own request creates a session, so no inbound request creates anything,
  no prompt exists that a user can learn to dismiss, and no stranger can cause anything
  to appear on a host's screen. `cmd/cogmer/verify.go`, `handleVerify`, and
  `cmd/cogmer/sas_test.go`, `TestVerificationCannotBeStartedByAPeer`.
- Verification confirms a key that is on file, and there is no key to confirm for a
  stranger. `cmd/cogmer/membership.go`, `Knows`, and `cmd/cogmer/sas_test.go`,
  `TestAnUnknownPeerCannotVerify`.

**Rejected.**
- *An inbound verification request that prompts the host.* A prompt people learn to
  dismiss is a poor place for a decision that matters, and it lets any peer that can
  reach the address put something on the host's screen.

**Limits.** Which side drives the exchange is D-059 (either side completes a verification
from inbound), and the answers that mean wait and the answer that means give up are D-168
(409 for what waiting can fix, 401 for what it cannot).

**Revisit when** a verification has to begin from the other side before its user has
asked.
