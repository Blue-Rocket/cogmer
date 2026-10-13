# D-201 — Claude Code and Codex are supported hosts

**Date:** 2026-10-12 · **Status:** active · **Areas:** hosts, project

**Decision.** cogmer supports Claude Code and Codex CLI and desktop sessions in the same room. Users run each host through its own launch and extension mechanisms.

**Support.**
- Hosts are used unchanged, and a room belongs to the session that joins it. §3.8 (the host is launched and used unchanged), §3.6 (session-scoped rooms).
- Codex documents plugins and the prompt and completion hooks needed to capture and inject turns. https://learn.chatgpt.com/docs/hooks, https://developers.openai.com/plugins/build/plugins.
- Host capture and session identity are checked against a real Codex turn. `cmd/cogmer/codex_probe.go`, `RunCodexChecks`.

**Rejected.**
- *Ordering CoWork before Codex and deferring OpenAI hosts.* Codex provides the extension mechanisms this integration uses. `plugin/.codex-plugin/plugin.json`, `plugin/hooks/codex-hooks.json`.

**Limits.** Remote cloud sessions are outside this integration. The plugin scripts and daemon must run on the session's machine.
