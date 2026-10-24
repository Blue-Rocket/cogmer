# D-202 — Host-specific hooks use the shared peer protocol

**Date:** 2026-10-12 · **Status:** active · **Areas:** hosts, capture, sync

**Decision.** Host-specific hook configuration declares the host, and the daemon resolves it for each request to select capture and delivery evidence on the loopback API (D-203). Replicated events carry an opaque origin session and no host field. A Codex session's identifier begins with `codex:`, which distinguishes it from a Claude Code session on the same machine, and which another machine can read and none relies on.

**Support.**
- Capture and delivery depend on different transcript records for each host. `cmd/cogmer/transcript.go`, `cmd/cogmer/codex.go`.
- Sessions on one machine can exchange turns through a room without another machine. `cmd/cogmer/codex_test.go`, `TestMixedHostRoomCapturesAndDeliversBothDirections`.
- Origin session identity and signatures belong to the shared event format. `cmd/cogmer/store.go`, `cmd/cogmer/protocol.go`.

**Rejected.**
- *Preparing for a second host through naming alone.* The second host has its own implemented capture and evidence paths. `cmd/cogmer/codex.go`, `ReassembleCodexTurn`, `CodexInjectedBlocks`.
- *A host discriminator on every replicated event.* The receiver needs the origin session and the text, and does not interpret the sender's transcript. `cmd/cogmer/daemon.go`, `FormatTeamContext`.

**Limits.** Codex delivery has no trust fallback. Missing evidence leaves turns pending, independently of Claude Code behavior results.
