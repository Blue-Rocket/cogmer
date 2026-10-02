# D-129 — Behaviour checks run by themselves when a room is formed

**Date:** 2026-09-16 · **Status:** active · **Areas:** behaviors

**Decision.** The session tier of the behaviour checks runs by itself the first time a
room is formed under a Claude Code version this machine has not checked.
`COGMER_PREFLIGHT=off` turns that off.

**Support.**
- Several of the behaviours cogmer relies on fail silently, so a user never sees the
  problem a check would find. `cmd/cogmer/behaviors.go`.
- A check's result is recorded for the version it ran on, so running it at each new room
  costs nothing after the first. D-008 (behaviour checks are keyed on the Claude Code
  version).

**Rejected.**
- *Running the checks only by hand.* The failures are silent, and nobody runs a check
  for a problem they cannot see.

**Revisit when** running the checks makes forming a room noticeably slower.
