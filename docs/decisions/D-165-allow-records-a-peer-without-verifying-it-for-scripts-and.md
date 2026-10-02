# D-165 — `allow` records a peer without verifying it, for scripts and tests

**Date:** 2026-09-18 · **Status:** active

**Decision.** `cogmer allow <identifier> [name]` records a known peer without running a
verification, and prints that the peer is UNVERIFIED and how to finish.

**Support.**
- A peer is verified only when two users compare words, which `allow` does not do.
  `cmd/cogmer/main.go`, `runAllow`.
- Scripts and tests cannot take part in a ceremony that needs a second person on a call.
  `cmd/cogmer/ui_test.go`, which records a peer with no label as a script would.

**Rejected.**
- *Removing `allow` once `pair` exists.* A script or test would then have no way to
  record a peer without a second person.
- *Printing a fingerprint with advice about a ceremony `allow` does not perform.* The
  output would suggest a comparison that D-055 (a peer is verified in one way) does not
  accept.

**Revisit when** a script or test can run a verification without a second person.
