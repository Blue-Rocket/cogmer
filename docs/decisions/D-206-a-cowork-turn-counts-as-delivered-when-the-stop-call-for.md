# D-206 — A Cowork turn counts as delivered when the stop call for its prompt arrives

**Date:** 2026-10-13 · **Status:** active · **Areas:** capture, hosts, behaviors

**Decision.** The Cowork host reads no transcript. The bridge returns context to the prompt hook only in the JSON form with `additionalContext`, and the daemon counts the turns that reply carried as delivered when the stop call for the same prompt identifier arrives. `cogmer doctor --host cowork` runs a check on the user's machine that a Claude Code session still adds the JSON form of an `mcp_tool` hook's reply to the model's context, and forming a room from Cowork runs it once for each Claude Code version. While that check is recorded as failing, the Cowork host's turns remain pending and are offered at every prompt.

**Support.**
- A reply in the JSON form reached the model and was recorded as a `hook_additional_context` attachment, and a plain reply was recorded as a `hook_success` attachment and did not reach the model, in Cowork and in Claude Code 2.1.296 on the user's machine. `5ac6450`, B20 (injected hook output is recorded as a hook_success attachment).
- The transcript that records this is in the sandbox, and no field a hook supplies carries it to the host, and the daemon reads a transcript through a path on its own machine. `b18b6aa`, `cmd/cogmer/host.go`, `InjectedBlocks`.
- The daemon commits the turns pending for a prompt identifier when the stop call arrives, and a stop carries the prompt's identifier. `cmd/cogmer/store.go`, `CommitPending`, `cmd/cogmer/daemon.go`, `confirmDelivery`, B29 (Stop carries the same prompt_id as the prompt that began the turn).
- The check starts a small MCP server of this binary in one isolated Claude Code turn, and its result is recorded under a key of its own, so Claude Code's record for the same version is left as it is. B30 (a Claude Code session adds the JSON form of an mcp_tool hook's reply to the model's context), `cmd/cogmer/cowork_probe.go`, `cmd/cogmer/coworkprobe_test.go`, `TestTheCoworkChecksAreRecordedApartFromClaudeCodes`.
- The same property shows a loss as noise and not as silence, since pending turns are offered again. D-131 (delivery is trusted without evidence only once the evidence is known broken), D-202 (host-specific hooks use the shared peer protocol).

**Rejected.**
- *The rule for Codex, where missing evidence leaves turns pending.* With no transcript the evidence never exists, so every prompt would offer every earlier turn again. D-202.
- *Asking the model to acknowledge each injection with a tool call.* It depends on the model complying, it adds a call nobody asked for, and a manipulated model could acknowledge what it did not read.

**Limits.** The check runs only against a Claude Code on the user's machine. A machine with none has no check, and the rule there counts the stop call alone. The check protects against a change in that version and not in the version Cowork runs, which Anthropic chooses and which `coworkd.log` reported as v2.1.295 when the machine's own was 2.1.296. A turn whose reply the harness dropped, with the stop call arriving, is counted delivered and lost, and the check narrows that case without removing it.

**Revisit when** a hook can carry the transcript's evidence to the host, or Cowork documents how it treats a hook's reply.
