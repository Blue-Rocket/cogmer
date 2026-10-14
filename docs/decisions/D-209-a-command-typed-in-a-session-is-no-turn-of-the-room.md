# D-209 — A command typed in a session is no turn of the room

**Date:** 2026-10-14 · **Status:** active · **Areas:** capture, commands, hosts

**Decision.** A prompt that begins with its host's command prefix is not captured as a turn of the room. The daemon offers no colleague's turn with it, and does not store what the model says about it at the turn's stop call. The prefix is `/cogmer:` for Claude Code and Cowork and `$cogmer ` for Codex.

**Support.**
- A project command typed in `claude -p` reached the prompt hook with its text as typed, so the prompt would be captured as any other when its session is in a room. `a2f6dde`, B33 (a project or plugin command reaches UserPromptSubmit with the text as typed), B18 (built-in slash commands do not reach UserPromptSubmit).
- What a command prints and what the model says about it can hold an invitation, a pairing string or a room's guests, which the user typed the command to see and did not mean to share. `plugin/commands/room-invite.md`, `plugin/commands/self-status.md`.
- A command is an instruction to cogmer, and its text is of no use to a colleague's model. `cmd/cogmer/command_test.go`, `TestACommandTypedInAnotherHostIsHeldBackFromTheRoomAndNotRun`.

**Rejected.**
- *Publishing a command with the turn that carries it.* The room would hold every command a user typed and every account the model gave of one.

**Limits.** The daemon remembers which prompts were commands in its own memory, so a stop call that reaches a daemon started since the prompt, as after `/cogmer:self-restart` in Claude Code, stores the model's account. Codex's prompt for its skill was not observed, so its prefix is the text a user types and nothing more is assumed. A command of a plugin with another name is not held back.

**Revisit when** B33 fails, or a command should be seen by the room.
