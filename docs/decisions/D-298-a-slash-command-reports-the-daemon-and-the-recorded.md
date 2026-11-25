# D-298 — A slash command reports the daemon and the recorded behavior checks at once, and its argument starts the whole check in the background

**Date:** 2026-11-25 · **Status:** active · **Areas:** commands, behaviors, daemon

**Decision.** `/cogmer:self-doctor` runs the program's `health` subcommand, which answers at once and spends no model turn. It prints the versions of the plugin, the program and the daemon with what resolves a disagreement, whether a daemon answers, and what the behavior check recorded for the version of the host the session runs in: not checked, could not run and why, failed and which behaviors, or passed. With the argument `extended` it starts `cogmer doctor --host` for that host detached, writes its output to `doctor-extended.log` in the state directory under a first line that says when it started, and answers at once; a later `/cogmer:self-doctor` reports what that run found, or that it is running, or that it did not finish within three minutes. The whole check drives real turns on the user's subscription, so only the argument starts it, and a cloud container never does (D-297, the behavior checks are not run in a cloud container). The messages of the Claude Code, Codex and Cowork preflights name `/cogmer:self-doctor`, or `$cogmer self-doctor` in Codex, which a user in a session can run.

**Rejected.**
- *No slash command for the doctor, since the plugin path would diagnose itself.* A user in a session has no terminal, so a command that only a terminal can run is out of their reach. The terminal `doctor` and `behaviors` remain for a broken plugin path (D-175, a terminal command exists for diagnostics, the daemon's lifecycle, a machine's identity and testing), and the slash command reads the results they record.
- *Running the whole check in the foreground.* The session is held about 18 seconds with no output, which D-191 (a command answers at once and says what to do when the install is not finished) rules out for a wait of that length.

**Limits.** The slash command cannot run when the plugin path is broken, which is when the terminal `doctor` is needed. A Cowork prompt that types it has the daemon run `health` as a child, with the 20-second limit a typed command has, and `extended` starts the whole check detached from that child. `extended` spends two real turns for Claude Code and one for Codex each time it runs.

**Revisit when** Claude Code can show progress from a command's `!` line, since the whole check could then run in the foreground without a silent wait.
