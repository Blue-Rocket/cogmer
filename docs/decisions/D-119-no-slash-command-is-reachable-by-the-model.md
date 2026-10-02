# D-119 — No slash command is reachable by the model

**Date:** 2026-09-22 · **Status:** active · **Areas:** commands, trust

**Decision.** Every file in `plugin/commands/` carries `disable-model-invocation: true`, and a test
requires it.

**Support.**
- A command file is loaded as a skill is, so every command would be a skill the model can
  invoke, and each one's name and description would occupy the context of every session with the
  plugin installed, which §3.1 names as a harm in its own right. §3.1 (first, do no harm), D-118
  (the plugin manifest name is the command namespace, and commands keep their prefixes under it).
- The model could run a command that changes something: `room-create` and `room-join` bind a
  session to a room, which cannot be undone because injected context cannot be withdrawn,
  `room-invite` and `room-revoke` reach another person's machine, and `peer-forget` discards
  admissions that a pairing ceremony established. D-056 (a session's room is fixed when its first
  hook fires, and never changes).
- The read-only commands are included, because the model gains nothing from a reachable
  `room-status`. A user who wants it types it, and every extension point delivers to the model and
  nothing displays to a person, so the most the model can do is say what it read, which the file
  instructs it to do after a user has run the command. D-033 (the room cannot be displayed
  inside Claude Code), D-036 (MCP logging notifications are not a display channel).
- Keeping them reachable would make the rule carry an exception, which makes people stop trusting a
  rule. D-096 (a command prefix names its target, and `/self-status` is where you are).
- No file was written to be model-invoked. Each is a `!` pre-execution followed by an instruction to
  report what the output says, which assumes a user has just typed the command.
  `plugin/commands/room-status.md`.
- The test requires the frontmatter on every file and fails if the directory is empty, because the
  defect it guards is the command added later by somebody who does not know the frontmatter matters.
  `cmd/cogmer/commandinvocation_test.go`, `TestNoCommandIsModelInvocable`.

**Rejected.**
- *Disabling invocation only on the commands that change something.* The others buy no capability
  and cost context in every session, and the rule would carry an exception.

**Limits.** No registry check covers it, for the reason D-118 gives, that it is a property of plugin
load time. Moving to the `skills/<name>/SKILL.md` layout is a separate change.

**Revisit when** a command appears that the model genuinely should reach without being asked. The
question to answer first is what it would do with the result, since it cannot show anybody anything.
