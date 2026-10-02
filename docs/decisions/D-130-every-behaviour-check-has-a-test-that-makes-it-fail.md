# D-130 — Every behaviour check has a test that makes it fail

**Date:** 2026-09-16 · **Status:** active · **Areas:** behaviors

**Decision.** Each behaviour check has a test showing that it fails on the change it is
meant to catch.

**Support.**
- A check can report the wrong result for a reason of its own: the first `--deep` run
  reported B05 and B12 failing because of a bug in how the probe handled its evidence,
  not because either behaviour changed. `84a0751:docs/decisions.md`.
- The tests that make checks fail are in `cmd/cogmer/behaviors_test.go`.

**Rejected.**
- *A check with no such test.* It reads as protection while giving none.

**Limits.** Nothing checks that every behaviour has such a test.

**Revisit when** a behaviour is added whose change cannot be simulated in a test.
