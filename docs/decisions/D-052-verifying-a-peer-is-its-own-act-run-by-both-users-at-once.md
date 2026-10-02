# D-052 — Verifying a peer is its own act, run by both users at once

**Date:** 2026-09-17 · **Status:** active

**Decision.** Verification is separate from joining a room and from host approval. Both
users run it at the same time on a call. Each daemon runs the exchange with the other,
both show the same two words, and each user says whether the other read out the same
ones.

**Support.**
- Verification confirms a key that is on file, whichever way the key arrived, so it
  needs nothing from the path by which a key is approved. `cmd/cogmer/verify.go`,
  `handleVerify`, and `cmd/cogmer/sas_test.go`, `TestAnUnknownPeerCannotVerify`.
- It is interactive and blocks on another person. `cmd/cogmer/main.go`, `runPair`.
- Each daemon tries every address this machine knows for the peer and keeps the one that
  answers signed by the identity asked for, and a reply from a different peer is a wrong
  peer and not a wrong address. `cmd/cogmer/verify.go`, `RunVerification`, and
  `cmd/cogmer/verify_test.go`, `TestVerifyDialsOnlyTheNamedPeer`.

**Rejected.**
- *Running the exchange as a step of joining a room, where both daemons are connected.*
  Joining is a room operation and verifying a key is not, and the exchange is interactive
  and blocks on another person.

**Limits.** The surface that shows the words is D-088 (the pairing ceremony lives in the
view). What may start an exchange is D-162, what records a verification is D-163, and
which words are shown is D-164.
