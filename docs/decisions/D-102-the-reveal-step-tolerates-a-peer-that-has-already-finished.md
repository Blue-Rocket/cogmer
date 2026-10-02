# D-102 — The reveal step tolerates a peer that has already finished

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Making every peer connection TLS (D-101) turned a test that had passed
for weeks into one that failed about one run in six. The handshake did not break it;
it changed the timing enough to expose a race that was always there.

**The two halves of the exchange were not treated alike.** A commit that reached a
peer not yet expecting a verification set `lastErr` and retried, which is right: the
other person may not have run their side yet, and waiting is the whole point of the
ninety-second window. A reveal that got the same answer returned it.

Nothing is not-yet-ready between one call and the next, but something can be
*no longer* ready. The other side can complete between our commit and our reveal,
and its deferred cleanup discards the session, so the reveal lands on a daemon that
is not expecting one. One side then finished with two words while the other reported
that its peer was not expecting a verification.

**It recovers by looping rather than by retrying that call.** By the time the other
side has finished it has already sent its own reveal, so its nonce is in this
session, and the inbound check at the top of the loop finds it. The fix is to give
the reveal the same tolerance the commit has.

**A test that passes is not a test that holds.** This one ran green through the
whole of Phase 5 and the two-machine runs, and the defect was reachable the entire
time — it needed one side to finish inside the window between another's two calls.
Treat a timing change that breaks an old test as evidence about the test's coverage
before assuming it is evidence about the change.

**Revisit when:** the exchange gains a third round trip, which would widen the same
window.
