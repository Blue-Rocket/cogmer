# D-207 — The Cowork bridge offers no tool that runs a command, and a command runs from the typed prompt

**Date:** 2026-10-13 · **Status:** not built · **Areas:** commands, trust, hosts

**Decision.** The bridge offers two tools, `prompt` and `stop`, which take what a hook supplies. A `/cogmer:` command typed in a Cowork session reaches the daemon as the text of the prompt, and the daemon runs it there for the session that sent it and returns its output as context for the model to relay. A test fails if the bridge offers any other tool. The daemon runs a command as the subcommands of its own binary that the plugin's command files call, without a shell, in a child process whose environment names the host and the session and holds no other host's session. `self-stop` and `self-restart` are not run by the daemon, since each acts on it, and the bridge runs them (D-208). A prompt that is a command is no turn of the room, offers no colleague's turn, and needs no room, and the model's account of it is not published. The bridge waits longer for a call whose prompt is a command.

**Support.**
- The model's tool list holds the host program's tool, and a call by the model reached the program as a hook's call does, so no tool the bridge offers is hidden from the model, and the user remembers no approval prompt for such a call. `f82fa47`, `da39258`, D-119 (no slash command is reachable by the model).
- The prompt hook supplies the text as the user typed it, which the daemon's prompt handler reads. `da39258`, `cmd/cogmer/daemon.go`, `promptReq`.
- The plugin's commands call these subcommands of the binary, and a test compares the plugin's files with the daemon's table. `plugin/cli.sh`, `cmd/cogmer/command.go`, `typedCommands`, `cmd/cogmer/command_test.go`, `TestTheTypedCommandsAreTheOnesThePluginHas`.
- A command's output can hold a pairing string, an invitation or the guests of a room, which the user typed a command to see. `plugin/commands/self-status.md`, `cmd/cogmer/command_test.go`, `TestACommandTypedInCoworkIsNotATurnOfTheRoom`.
- A session's room is fixed when its first hook fires, so a command that binds a room changes only a session that is not yet bound. D-056 (a session's room is fixed when its first hook fires, and never changes).
- A command file lets the shell call run only while a command runs, and the command script is an ordinary file, so a model that ran it through its shell tool would be asked by the user's permission prompt, read in the files and not tried. `plugin/cli.sh`, `plugin/commands/room-invite.md`.

**Rejected.**
- *Running the subcommands inside the daemon's own process.* They print and exit as programs do, and one that exits would stop the daemon.
- *Publishing the model's account of a command as a turn.* It can hold what the output held, and the room's members did not ask to read it.
- *Confirmation in the browser for each command that changes something.* It moves the command surface out of the chat, for a risk that is small beside the shell, the connected folders and the connectors that a Cowork model holds.
- *An allowlist that leaves out `room-invite`, `room-revoke` and `peer-forget`.* A user who works only in Cowork could then not admit or withdraw a colleague.
- *A rule that refuses a prompt call while a turn is open.* It depends on how the harness treats a prompt typed during a turn and a turn that is interrupted, which no run has observed.

**Limits.** A model that room content manipulates could call `prompt` with text that records a user turn or runs a command, and the bridge cannot tell that call from a hook's. The room holds peers the user verified. A forged command can admit a verified peer to the room's history, withdraw or forget a peer, or bind a session that is not yet bound, and a user can undo each. A Cowork model also holds a shell, connected folders and any connectors, which reach further than this. D-119 guards the command files in `plugin/commands/` and does not cover this route.

**Revisit when** a forged turn or command is reported, or Cowork asks approval for tool calls in a way a plugin can rely on.
