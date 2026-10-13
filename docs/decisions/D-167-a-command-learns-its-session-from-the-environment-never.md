# D-167 — A command learns its session from the environment, never from an argument

**Date:** 2026-09-18 · **Status:** active · **Areas:** commands, rooms

**Decision.** A session-scoped command reads its session from the variable of its host, `CLAUDE_CODE_SESSION_ID`
for Claude Code or `CODEX_THREAD_ID` for Codex, the host being the one its entry point declares (D-203), and a
command passes no session id. Run at a terminal, where the variable is absent, a session-scoped command has no
session to act on and refuses, and no machine-level setting stands in for it.

**Support.**
- Claude Code exports `CLAUDE_CODE_SESSION_ID` into every tool call's environment, and it
  equals the id the hooks report. B21 (CLAUDE_CODE_SESSION_ID is exported into a tool
  call's environment).
- A command with no session says so and names the slash command to run inside a session.
  `cmd/cogmer/main.go`, `sessionRoom`.
- Codex exports the thread identifier into a shell tool environment, matching its hook session. B28 (CODEX_THREAD_ID equals the hook session identifier).

**Rejected.**
- *Passing the session id as an argument.* Binding a room to the session that asked for it
  would need a name for a session that nothing at a terminal can supply.
- *A machine-level current room for commands run outside a session.* A room created and
  forgotten would be joined weeks later by a session in an unrelated repository, which
  would begin publishing without anyone acting. D-080 (there is no current room).

**Revisit when** B21 or B28 fails. A session-scoped command then cannot know its session, and
the answer is to refuse and not to restore a machine-level current room.
