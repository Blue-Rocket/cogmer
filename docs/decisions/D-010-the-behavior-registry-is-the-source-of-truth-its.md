# D-010 — The behavior registry is the source of truth; its documentation is generated

**Date:** 2026-09-16 · **Status:** active · **Areas:** behaviors

**Decision.** Each behavior and its check live together in
`cmd/cogmer/behaviors.go`, and `docs/relied-on-behaviors.md` is generated from them by
`cogmer behaviors --markdown`.

**Support.**
- The generated document is produced from the registry, so it lists what is checked.
  `cmd/cogmer/behaviors.go`, `MarkdownReport`.

**Rejected.**
- *A document written by hand beside the checks.* It drifts from them, and a stale list
  of safety properties is worse than none, because it is believed.

**Revisit when** the registry needs to hold a behavior that no check can confirm.
