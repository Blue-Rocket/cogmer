# D-089 — The reachability warning asks about the advertised address, and travels with the pairing string

**Date:** 2026-09-20 · **Status:** active · **Areas:** transport, pairing

**Decision.** The warning that a pairing string cannot be reached is in
`printPairingInvitation`, so it appears wherever a pairing string is printed. It asks about
`AdvertisedEndpoint()` and not the address the daemon binds, and it says which of two causes
holds, so that the user is sent to the right remedy.

**Support.**
- The bind address is loopback by default and remains so when tailcat has negotiated a routable
  endpoint, which is the ordinary case, so a warning on it fired under every string including
  the working ones, and a warning that is always on trains somebody to ignore the real one.
  `cmd/cogmer/main.go`, `pairingReachable`, `cmd/cogmer/pairview_test.go`,
  `TestPairingReachabilityWarnsOnlyWhenItShould`.
- Every path that hands a user's identity to somebody else comes through one function, so a
  check kept there travels with the string. `cmd/cogmer/main.go`, `printPairingInvitation`.
- "No daemon has ever published an address, so this one is a guess" is fixed by starting a
  session, and "a daemon ran and negotiated no route out" is fixed by setting
  `COGMER_PEER_ADDR`, and sending a user to the second when the first holds wastes their
  time. `cmd/cogmer/main.go`, `endpointRecorded`, `cmd/cogmer/pairview_test.go`,
  `TestPairingReachabilityDistinguishesItsTwoCauses`.
- A bare TCP endpoint must parse as a host and a port, because `ParseEndpoint` accepts
  anything without a scheme and `isLoopback` answers false for what it cannot parse, which
  would read a malformed address as fine. `cmd/cogmer/main.go`, `pairingReachable`.

**Rejected.**
- *A warning on the address the daemon binds.* It fired on every string.

**Revisit when** tailcat stops being on by default. Advertising a location is D-091 (identity
is advertised, and location is discovered).
