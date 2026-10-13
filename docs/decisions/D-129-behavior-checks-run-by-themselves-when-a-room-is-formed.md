# D-129 — Behavior checks run by themselves when a room is formed

**Date:** 2026-09-16 · **Status:** active · **Areas:** behaviors

**Decision.** The session tier of the behavior checks runs by itself the first time a
room is formed under a Claude Code version this machine has not checked.
`COGMER_PREFLIGHT=off` turns that off.

**Support.**
- Several of the behaviors cogmer relies on fail silently, so a user never sees the
  problem a check would find. `cmd/cogmer/behaviors.go`.
- A check's result is recorded for the version it ran on, so running it at each new room
  costs nothing after the first. D-008 (behavior checks are keyed on the Claude Code
  version).

- A session that forms a room runs its own host's preflight, and an attempt that could not run is recorded and
  left alone for an hour, so a machine where the checks cannot run does not run two Claude turns at every room.
  `cmd/cogmer/host.go`, `claudeHost.Preflight`, `cmd/cogmer/doctor.go`, `EnsureVerified`,
  `cmd/cogmer/claudepreflight_test.go`, `TestAClaudePreflightThatCouldNotRunIsNotRepeatedWithinTheRetryWindow`.
- The probe session loads only the project's settings and its own, not the user's, where installed plugins
  are enabled, so the real plugin's hooks cannot fire inside it and write to the user's state directory.
  `cmd/cogmer/probe.go`, `claudeArgs`, `cmd/cogmer/claudepreflight_test.go`,
  `TestEveryClaudeProbeRunLoadsOnlyTheProjectSettings`.

**Rejected.**
- *Running the checks only by hand.* The failures are silent, and nobody runs a check
  for a problem they cannot see.

**Revisit when** running the checks makes forming a room noticeably slower.
