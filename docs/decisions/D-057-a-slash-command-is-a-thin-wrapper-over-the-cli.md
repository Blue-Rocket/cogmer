# D-057 — A slash command is a thin wrapper over the CLI

**Date:** 2026-09-18 · **Status:** active

**Decision.** A slash command shells out to the corresponding CLI command. There is one
implementation with two entry points, and the CLI is the surface that can be tested
without a Claude session.

**Support.**
- A command file runs the binary through `cli.sh` and carries no logic of its own.
  `plugin/commands/room-status.md`.
- Every slash command names a subcommand the binary has. `cmd/cogmer/membership_test.go`,
  `TestEverySlashCommandNamesARealSubcommand`.

**Rejected.**
- *A second implementation for the commands that run inside a session.* Two
  implementations of one act drift apart, and the session-side one could not be tested
  without a session.

**Limits.** How a command learns which session ran it is D-167.
