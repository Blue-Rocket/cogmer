# D-089 — The reachability warning asks about the advertised address, and travels with the pairing string

**Date:** 2026-09-20 · **Status:** active · **Areas:** transport, pairing

**Decision.** `printPairingInvitation` asks whether the address in a pairing string can be
reached, and when it cannot it prints no string and says why. It asks about `AdvertisedEndpoint()`
and not the address the daemon binds, and it says which of three causes holds, that no daemon has
published an address, that the daemon could not start, or that it ran and negotiated no route out,
so that the user is sent to the right remedy.

**Support.**
- The bind address is loopback by default and remains so when tailcat has negotiated a routable
  endpoint, which is the ordinary case, so a warning on it fired under every string including
  the working ones, and a warning that is always on trains somebody to ignore the real one.
  `cmd/cogmer/main.go`, `pairingReachable`, `cmd/cogmer/pairview_test.go`,
  `TestPairingReachabilityWarnsOnlyWhenItShould`.
- Every path that hands a user's identity to somebody else comes through one function, so a
  check kept there travels with the string. `cmd/cogmer/main.go`, `printPairingInvitation`.
- "No daemon has ever published an address, so this one is a guess" is fixed by starting a
  session, "a daemon ran and negotiated no route out" is fixed by setting `COGMER_PEER_ADDR`,
  and "the daemon could not start" is fixed by whatever holds its address, which the daemon
  wrote down. Sending a user to start a session that has started wastes their time.
  `cmd/cogmer/main.go`, `endpointRecorded` and `daemonBlockedReason`,
  `cmd/cogmer/pairview_test.go`, `TestPairingReachabilityDistinguishesItsTwoCauses` and
  `TestPairingReachabilityNamesABlockedDaemon`.
- A string with an unreachable address is useless to a colleague, and a note printed after it
  still leaves a string to copy and send, so none is printed.
  `cmd/cogmer/pairview_test.go`, `TestNoPairingStringIsPrintedWhenNobodyCanReachIt`.
- A bare TCP endpoint must parse as a host and a port, because `ParseEndpoint` accepts
  anything without a scheme and `isLoopback` answers false for what it cannot parse, which
  would read a malformed address as fine. `cmd/cogmer/main.go`, `pairingReachable`.

**Rejected.**
- *A warning on the address the daemon binds.* It fired on every string.
- *Printing the string with a note after it.* The string goes out looking sendable.

**Revisit when** tailcat stops being on by default. Advertising a location is D-091 (identity
is advertised, and location is discovered).
