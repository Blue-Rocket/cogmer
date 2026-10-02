# D-191 — A command answers at once and says what to do when the install is not finished

**Date:** 2026-10-02 · **Status:** active · **Areas:** commands, release

**Decision.** A slash command that finds no binary does not wait for one. `cli.sh` answers at once
and says what is happening and what to do, and it exits 0 with everything on stdout. When nothing
has been fetched it starts the download itself, detached, and when an earlier install stalled it
clears the lock that install left and starts again.

**Support.**
- A command whose `!` line exits non-zero is abandoned, no model turn runs, and the explanation
  never reaches the model or the user, so the script exits 0 and prints the binary's stderr on
  stdout. `cmd/cogmer/cli_test.go`, `TestACommandWithABinaryPrintsEverythingAndExitsZero`.
- A command waiting for a download would hold the session with no output until it ended, and a
  session is never made worse. §3.1 (first, do no harm), D-114 (section 3.1 is a duty not to harm
  the session, and says nothing about where data lives).
- A plugin installed inside a running session never runs its session-start hook, so nothing fetches
  the binary, and the first command a user types is the first chance to start the download.
  `plugin/cli.sh`.
- A stalled install means the process died, and it may have left its lock, which every later
  installer finds and leaves, so a new session cannot help. D-075 (the install has a state, and
  absence is not an inference), `cmd/cogmer/cli_test.go`,
  `TestACommandAfterAStalledInstallClearsTheLockAndStartsAgain`.
- After a failure the installer waits an hour, so the message says to remove the state file and run
  the command again. `cmd/cogmer/cli_test.go`, `TestACommandAfterAFailedInstallSaysWhyAndHowToTryAgain`.

**Rejected.**
- *Waiting for the install within a bound of about 30 seconds.* The download takes about 15 to 20
  seconds, and a wait that long inside a session is a pause with no output, which §3.1 does not
  permit however briefly and for whatever reason the command was typed.
- *Telling the user to start a new session.* It cannot help after a stalled install, and for an
  install made mid-session it costs the session the user was working in.

**Revisit when** Claude Code can show progress from a command's `!` line, since a wait would then
not be silent.
