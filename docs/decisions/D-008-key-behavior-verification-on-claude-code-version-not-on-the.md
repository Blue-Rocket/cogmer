# D-008 — Key behavior verification on Claude Code version, not on the room

**Date:** 2026-09-16 · **Status:** active · **Areas:** behaviors

**Decision.** The result of the behavior checks is recorded against `claude --version`,
in `~/.cogmer/verified.json`, and a version that has been checked is not checked again.

**Support.**
- A behavior check costs a real turn on the user's subscription.
  `cmd/cogmer/probe.go`.
- What changes a behavior is a new Claude Code binary, and its version names the
  binary. `cmd/cogmer/doctor.go`.

**Rejected.**
- *Checking for every room.* It proves the same thing again and spends the user's
  quota, while the version is what varies.
- *Expiring the result after a time.* A time limit is a stand-in for whether the binary
  changed, which the version answers directly.

**Limits.** A behavior that changes without the version changing, such as one that
depends on the model a session uses, is not checked again.

**Revisit when** a behavior is found to have changed while `claude --version` did not.
