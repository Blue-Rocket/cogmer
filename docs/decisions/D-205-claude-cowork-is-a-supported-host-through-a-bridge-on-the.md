# D-205 — Claude Cowork is a supported host through a bridge on the user's machine

**Date:** 2026-10-13 · **Status:** not built · **Areas:** hosts, daemon

**Decision.** cogmer supports Claude Cowork sessions in the same room as Claude Code and Codex sessions. The daemon, the identity and the rooms remain on the user's machine. A plugin installed in Cowork declares hooks of type `mcp_tool` that call the bridge, an MCP server that cogmer ships and the user adds to Claude Desktop's local servers, and the bridge relays each hook to the daemon with the host declared as `cowork`. A Cowork session's identifier begins with `cowork:`. A command on the user's machine writes the bridge's Desktop entry and the plugin's hook file together, so that the names the hooks call agree with the entry.

**Support.**
- A Cowork session's sandbox cannot run the daemon: its network reaches no relay, and nothing written there survives into a new session. `06749a0`. The daemon and its scripts run on the session's machine. D-201 (Claude Code and Codex are supported hosts).
- A plugin hook of type `mcp_tool` reaches a program on the user's machine through a server Cowork names `remote-devices`, and passes it the session, the prompt and the last assistant message, the fields the daemon's prompt and stop handlers read. `06749a0`, `cmd/cogmer/daemon.go`, `handlePrompt`, `handleStop`.
- A hook's call to a server that is absent or has exited returns nothing and raises no error, and the same session's calls succeed once the server is available again, which is the behavior of a hook that fails open. `9fe0021`, D-011 (hooks fail open: exit 0, empty stdout).
- The daemon resolves a host for each request and stores a session under its host's identifier. D-203 (a host is declared by its entry point and resolved per request), D-202 (host-specific hooks use the shared peer protocol).
- An MCP server and a plugin are extension points that a host loads unchanged. D-037 (Claude Code is launched and used unchanged), §3.8 (the host is launched and used unchanged).

**Rejected.**
- *The daemon in the Cowork sandbox.* It cannot reach a relay or a peer, and its state does not survive the session. `06749a0`.

**Limits.** The user adds the bridge to Desktop's configuration, because a server that a plugin bundles would start in the sandbox. A call at session start does not reach the bridge (`a191489`), so the daemon first hears of a session at its first prompt. A hung bridge held a session for the full 60 seconds, past the 15 seconds the hook declared, so the bridge answers within a deadline of its own and exits when a call runs long, and a bridge stopped from outside can still cost that wait. The names the hook calls, the server `remote-devices` and the tool `<server>__<tool>`, are Anthropic's and are not documented, and a changed name stops capture without a sign, so a room's status says how long ago a Cowork session last reached the machine (`cmd/cogmer/main.go`, `coworkLine`). A user with more than one computer was not tested, and neither was a renamed computer, which the documentation tells the user to set up again. Anthropic chooses the Cowork version and can change what a hook does.

**Revisit when** Anthropic documents the device bridge's tool names and the start of a session, or gives a plugin a way to reach the daemon directly.
