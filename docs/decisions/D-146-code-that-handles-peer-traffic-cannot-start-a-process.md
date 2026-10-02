# D-146 — Code that handles peer traffic cannot start a process

**Date:** 2026-09-17 · **Status:** active

**Decision.** A test fails if a file that handles peer traffic imports `os/exec` or
`syscall`, and another fails if `sync.go` or `daemon.go` calls `RunProbe`,
`EnsureVerified` or `runDoctor`, the functions that start a Claude run. The guard is
stricter than D-035 (a remote event never makes an interactive
session take a turn), since it forbids starting any process from those files.

**Support.**
- The only path that starts a Claude run is `RunProbe`, reached from `runDoctor`, a typed
  command, and from `EnsureVerified`, the daemon's check at start. `cmd/cogmer/probe.go`,
  and `cmd/cogmer/invariant_test.go`, `TestExecutionLivesOnlyInTheProbe`.
- Checking imports fails the moment the capability is added to the wrong file, which is
  when someone should be asked to justify it. `cmd/cogmer/invariant_test.go`,
  `TestPeerFacingCodeCannotExecute`.

**Rejected.**
- *Relying on review.* A rule that nothing checks is broken by the first change nobody
  reviews with it in mind.
- *Checking the call graph.* It is more precise, and harder to keep correct than a list
  of imports.

**Revisit when** a peer event needs to start a process, which D-035 leaves undecided.
