# D-175 — A terminal command exists for diagnostics, the daemon's lifecycle, a machine's identity and testing

**Date:** 2026-09-20 · **Status:** active · **Areas:** commands, daemon

**Decision.** A command is available at a terminal only for one of four reasons. It must
work when the plugin path is broken, as `doctor`, `behaviors`, `version` and reading
`install.log` do. It is the daemon's own lifecycle, as `daemon` and `stop` are. It concerns
this machine and no room, as `whoami`, `peers`, `forget` and `allow` do. Or it is a test
fixture, such as `seed`. Every other command has a slash command.

**Support.**
- A diagnostic that works only when things work is not a diagnostic, so `doctor` and
  `behaviors` have no slash command, which would suggest the plugin path is a reasonable
  way to diagnose the plugin path. `cmd/cogmer/main.go`.
- The daemon runs when no session exists, and the user whose machine it runs on has to be
  able to find and stop it. §29 (starting the daemon).
- Every slash command names a subcommand the binary has, and every room-scoped command
  fulfills a slash command. `cmd/cogmer/membership_test.go`,
  `TestEverySlashCommandNamesARealSubcommand`.
- The two words of a pairing must reach a person without passing through the model, and the
  view shows them without a terminal. D-088 (the pairing ceremony lives in the view, and
  every pairing gets its own URL).

**Rejected.**
- *A slash command for `doctor` or `behaviors`.* The plugin path would diagnose itself.

**Limits.** `daemon`, `hook`, `probe-hook` and `seed` are machinery or fixtures that no user
types, and `allow` is the unverified form for scripts. D-165 (`allow` records a peer
without verifying it, for scripts and tests).
