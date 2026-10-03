# D-193 — `stop` ends the daemon and starts nothing, and `/cogmer:self-stop` runs it

**Date:** 2026-10-03 · **Status:** active · **Areas:** daemon, commands

**Decision.** `cogmer stop` ends the cogmer daemons on this installation's addresses and starts
none, and says that nothing is captured or shared until a daemon runs and that a new session
starts one. `/cogmer:self-stop` runs it.

**Support.**
- A person who stops the daemon may want it stopped, and the next session's start hook starts one
  anyway, so a stop that also started one would take the choice away. D-150 (the session-start hook
  starts the daemon, and never delays the session).
- The person is in a session and the binary is not on PATH, so reaching `stop` from a terminal
  means typing a path under `~/.cogmer/bin`, and a profile is never edited to change that. §29
  (installation), `plugin/cli.sh`.
- Nothing runs a daemon in the session after a stop, so the output says so, and the command tells
  the user in terms of what they will notice, a room that is silent. §3.1 (first, do no harm),
  `cmd/cogmer/main.go`, `runStop`, and `plugin/commands/self-stop.md`.
- The command changes this machine's installation and no room, so it takes the `self-` prefix, and
  like every command it is unreachable by the model. D-096 (a command prefix names its target, and
  `/self-status` is where you are), D-119 (no slash command is reachable by the model).

**Rejected.**
- *`stop` starting the daemon afterwards.* Somebody who wants it stopped would have to stop it
  twice.
- *No command, only the terminal form.* Typing the installed binary's path is the cost this
  decision removes.

**Revisit when** a session can start a daemon without being a new one, since stopping and starting
are then one idea that somebody may want as a restart.
