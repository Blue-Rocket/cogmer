# D-012 — The preflight probe uses the `cogmer` binary as its own hook

**Date:** 2026-09-16 · **Status:** active

**Decision.** The probe registers `cogmer probe-hook <name> <dir>` as its hook
command, rather than writing a shell script to a temporary directory.

**Support.**
- The `cogmer` binary runs on every platform a release builds for, including Windows.
  `scripts/release.sh`, and D-001 (Go, not TypeScript/Node or Python).

**Rejected.**
- *A generated shell script.* It would not run on Windows, so the check of cogmer's
  portability would be the least portable part of it.

**Revisit when** the probe needs a hook that the binary cannot serve.
