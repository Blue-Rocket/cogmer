# D-085 — A line telling a user to run cogmer uses the path the binary reports for itself

**Date:** 2026-09-20 · **Status:** active · **Areas:** commands, release

**Decision.** Wherever the binary tells a user to type a command it prints its own invocation,
`invocation()`, resolved from `os.Executable()` with the home directory written as `~`, and
the command files relay that line without shortening it.

**Support.**
- The installer puts the binary in `~/.cogmer/bin`, which is not on the path, because
  installing edits no shell profile, so a line that says `cogmer pair` is `command not found`
  for every user who installed the plugin. §29 (installation), `plugin/cli.sh`.
- A path the binary reports about itself cannot go stale, and it is right for an install from
  source and for a plugin install alike. `cmd/cogmer/main.go`, `invocation`.
- `~` is what a user recognizes and it pastes into a shell unchanged. `cmd/cogmer/main.go`,
  `invocation`.

**Rejected.**
- *Printing the bare word `cogmer`.* It resolves only for a user who installed the binary
  themselves, and the people least equipped to diagnose `command not found` are the ones it
  fails.

**Revisit when** the install location changes.
