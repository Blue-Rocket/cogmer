# D-217 — A cloud session's daemon carries its relay connection as a WebSocket

**Date:** 2026-10-18 · **Status:** active · **Areas:** transport

**Decision.** A daemon whose environment has `CLAUDE_CODE_REMOTE=true` sets Tailscale's
`TS_DEBUG_DERP_WS_CLIENT` before it makes any relay connection, and every release is
built with the `ts_debug_websockets` tag that compiles in the WebSocket relay client.
A Claude Code cloud session's proxy breaks the relay's own protocol upgrade and passes
the same connection carried as a WebSocket, which reached a daemon on another machine
(e415f1d). A value the user set for the variable is left as it is.

**Rejected.**
- *WebSocket everywhere.* A daemon outside a cloud session works with the relay's own
  protocol, and every user's daemon would then depend on a Tailscale debug setting,
  rather than only the daemons in cloud sessions.
- *Switching whenever `HTTPS_PROXY` is set.* A proxy on a user's own machine may pass the
  relay's upgrade, and nothing observed says which proxies break it.
- *Trying the relay's own protocol first and falling back.* Each connection in a cloud
  session would wait out a timeout before it worked.

**Limits.** `TS_DEBUG_DERP_WS_CLIENT` and the `ts_debug_websockets` tag are Tailscale
debug settings with no stability promise, and an upgrade of the module can rename or
remove them without failing the build. The WebSocket client compiles only for Linux and
macOS, and a build without the tag, such as `go build` for development, ignores the
variable.

**Revisit when** Tailscale renames or removes `TS_DEBUG_DERP_WS_CLIENT` or the
`ts_debug_websockets` tag, or a cloud session passes the relay's own upgrade.
