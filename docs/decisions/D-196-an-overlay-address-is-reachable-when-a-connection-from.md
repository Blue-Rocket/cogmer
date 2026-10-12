# D-196 — An overlay address is reachable when a connection from outside gets through it

**Date:** 2026-10-12 · **Status:** active · **Areas:** transport, commands, trust

**Decision.** `/cogmer:self-status` calls an overlay address reachable only when the running daemon is
attached to its relay and a connection made to that address through the relay, by a client of its own, gets
through. Otherwise it prints no string and says which of four things is the case: no daemon is running, the
daemon runs no overlay, it is not attached to a relay, or it is attached and the test connection did not get
through.

**Support.**
- The check called any overlay address reachable because it had the right form, so a daemon that never
  attached to its relay handed out an address that led nowhere and reported it reachable. On 2026-10-12 a
  colleague could not be reached through the relay while their own check said they could be. D-115 (a
  centralized component must trace to a disclosed tradeoff that benefits the person), `cmd/cogmer/main.go`,
  `checkReachable`.
- The overlay server reports the relay it is attached to once it is, which takes a few seconds, and the
  check waits for that as long as it is allowed to, since a daemon that was just started has not failed to
  attach. `cmd/cogmer/tailcat.go`, `newSelfCheck`.
- A connection from outside is what a colleague makes, and it was run by hand against the relay and
  connected in 294 ms for a live daemon and timed out for one that is not at the relay. The code that does it
  lives in the file that confines the overlay. D-068 (tailcat is the cross-network transport, behind our own
  dialer), `cmd/cogmer/selfcheck_test.go`, `TestTheRealOverlayCheckPassesForALiveServerAndFailsForADeadOne`.
- A pass is held for 20 seconds so that two commands in a row open one connection through the relay, and a
  failure is asked again, because the next ask usually follows a fix. `cmd/cogmer/ui.go`, `handleSelf`,
  `cmd/cogmer/selfcheck_test.go`, `TestTheDaemonKeepsAPassAndAsksAgainAfterAFailure`.
- Each answer is held by a test, and breaking the shape check, the kept failure, the dial and the record of
  inbound requests each failed one. `cmd/cogmer/selfcheck_test.go`.

**Rejected.**
- *Treating a well-formed overlay address as reachable.* That check cannot fail.
- *Asking the colleague to run the check.* Only the daemon whose address it is can try its own, and the
  person with the fault is the one least able to.

**Limits.** The check dials this machine's own address through the relay, so the relay sees one more
connection from this machine to itself, and a daemon behind a network that blocks hairpin traffic could fail
the check while colleagues connect. A pass shows that a connection got through at that moment and does not
show that one will.

**Revisit when** the check fails for a daemon that colleagues reach, which would mean the dial from this
machine does not stand for theirs.
