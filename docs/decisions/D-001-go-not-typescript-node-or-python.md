# D-001 — Go, not TypeScript/Node or Python

**Date:** 2026-09-16 · **Status:** active · **Areas:** project, release

**Decision.** cogmer is written in Go, with `modernc.org/sqlite`, a SQLite written in
Go, so that nothing needs cgo.

**Support.**
- Users run macOS, Linux and Windows, and a release builds a binary for each.
  `scripts/release.sh`.
- Claude Code ships as a native binary, so a colleague can have Claude Code and no
  Node runtime at all. `84a0751:docs/phase0-findings.md`.
- Go builds one binary per platform with no runtime dependencies, and every target
  cross-compiles from one machine with `CGO_ENABLED=0`. `scripts/release.sh`.

**Rejected.**
- *TypeScript on Node.* It would need Node on every colleague's machine, and
  `node:sqlite` in Node 22.12 throws without `--experimental-sqlite`, so it would mean
  `better-sqlite3`, a native dependency with its own set of prebuilt binaries.
- *Python.* Its environments fragment on Windows, and the view is hand-written
  JavaScript whatever the daemon is written in.

**Revisit when** the users standardize on a Node toolchain, or the view outgrows plain
JavaScript and server-sent events.
