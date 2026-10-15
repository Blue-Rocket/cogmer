# D-208 — The bridge keeps a daemon running for a Cowork user, and runs a restart and a stop itself

**Date:** 2026-10-14 · **Status:** active · **Areas:** daemon, commands, hosts

**Decision.** The bridge does for a Cowork user what the session-start hook does for the other hosts. When it starts, and when a call finds no daemon, it starts a daemon of its own binary, detached and logged, unless the user stopped it, and it starts one over an older daemon, which gives way (D-158). A call that is only a hook never waits for the start. A command the user typed ends the user's stop, starts a daemon, waits up to six seconds for it and runs. `cogmer stop` leaves a file that says the user's stop stands, and a daemon removes it when it starts. `/cogmer:self-restart` and `/cogmer:self-stop` run in the bridge, which outlives the daemon, and the model's account of either is not published.

**Support.**
- A Cowork user's machine has no session-start hook that reaches it, since a call at session start does not reach the bridge, and may have no plugin scripts. `a191489`, D-150 (the session-start hook starts the daemon, and never delays the session).
- A start has to be detached and logged, and a new session does not cost a daemon its GUI session. B23 (a detached daemon can open a window), `cmd/cogmer/daemonstart.go`, `startDaemon`.
- A command that acts on the daemon cannot run in it, since the daemon would wait on a process that waits for the daemon to exit. `cmd/cogmer/command.go`, `typedCommandsUnavailable`.
- With real programs, the bridge started a daemon when it started, stopped it, left it stopped through an ordinary prompt, started it for the next typed command and restarted it. `cmd/cogmer/bridge_daemon_test.go`, `TestTheBridgeKeepsARealDaemonForACoworkUser`, `TestStopLeavesTheMarkAndAStartingDaemonRemovesIt`.

**Rejected.**
- *Running the plugin's shell function from the bridge.* A Cowork user's machine need not have the plugin's scripts, which live where the plugin is installed. D-195 (a restart stops the daemon and starts it with the hook's own function).
- *Starting a daemon for every call that finds none.* A user who stopped the daemon would find it running at the next prompt, and D-193 (`stop` ends the daemon and starts nothing) gives a stop a meaning.

**Limits.** The two starts, the plugin's shell function and this one, are separate code. A restart waits five seconds for the new daemon and says when it has not answered, and a typed command that finds none waits six, so a slow machine can see either end first. A stop typed in Claude Code also ends the bridge's starts until a daemon runs or a command is typed in Cowork.

**Revisit when** the shell function's start moves into the binary, since both starts would then be one.
