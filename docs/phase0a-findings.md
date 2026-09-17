# Phase 0a — Compaction Probe: Findings

**Verdict: the feared failure does not occur. No remediation is required, and the
delivery watermark needs no change.** Injected teammate context survived
compaction in every test, with attribution intact. Two secondary findings —
compaction never interrupts a turn, and slash commands never reach the room — are
each worth more than the original hypothesis.

One caveat governs all of it: **survival is summarizer behavior, not a contract.**
See §9.

| | |
|---|---|
| Claude Code | 2.1.273 |
| Platform | macOS 15 (darwin/arm64) |
| Sessions driven | 3, session IDs pinned via `--session-id` |
| Compactions observed | 2 manual; 0 automatic (see §8) |

---

## 1. `PreCompact` fires, and carries a usable payload

It did not fire during Phase 0 only because compaction never occurred. It fires
reliably, **before** the compaction runs:

```json
{
  "hook_event_name": "PreCompact",
  "trigger": "manual",
  "custom_instructions": null,
  "session_id": "8a69b9b9-...",
  "prompt_id": "2d3ee53f-...",
  "transcript_path": "/Users/david/.claude/projects/.../8a69b9b9-....jsonl",
  "cwd": "/private/tmp/ct0a"
}
```

`trigger` distinguishes `"manual"` from automatic, so a daemon can tell the two
apart without inference.

`/compact` is accepted in `-p` mode with `--resume`, which is what made this probe
cheap to run — no interactive TTY needed.

## 2. Identity and storage survive compaction

Every anchor the daemon depends on holds:

| Property | Result |
|---|---|
| `claudeSessionId` | **Unchanged** — stable across 5 turns and a compaction |
| `transcript_path` | **Unchanged** — same file |
| Transcript file | **Appended to**, never rewritten or truncated (27 → 42 records) |
| New transcript file | **None created** |

This matters because the delivery watermark is keyed on session ID. Had
compaction minted a new one, every compaction would have silently re-injected the
entire room. It does not.

## 3. The boundary is explicitly marked

Compaction writes a marker record, so it is detectable by reading the transcript
alone, without relying on having caught the hook:

```
type: "system", subtype: "compact_boundary"
  compactMetadata: {trigger, preTokens, postTokens, durationMs,
                    cumulativeDroppedTokens, preservedSegment, preservedMessages}
```

followed by a `type: "user"` record flagged `isCompactSummary: true` carrying the
summary itself.

Observed: `preTokens 21211 → postTokens 2384` (~8.9× reduction, 42s);
`19998 → 2660` (~7.5×, 47s). `preservedMessages` lists verbatim-retained records
by UUID.

## 4. The critical question: injected context survives

Tested twice, with the delivery watermark guaranteeing **zero re-injection**
(`injected=0` in the daemon log both times), so any correct answer had to come
from surviving context.

**Test A — teammate context was the topic.** Injected, discussed, compacted, then
asked post-compaction who owned the theory:

> That was Claude-Alice's theory (relayed from Alice's session), and the number
> given was 300 seconds for how long the OkHttp pool holds idle sockets.

**Test B — teammate context was incidental.** This is the realistic hazard: the
same context was injected, but the conversation was four unrelated turns of
haiku and colors, giving the summarizer no reason to retain it. After compaction:

> Alice raised the theory — they asked whether idle connection-pool expiration
> explained the SessionLambda timeout. Claude-Alice confirmed it and supplied the
> number: OkHttp's connection pool holds idle sockets for **300s**, while the
> upstream load balancer silently drops them at 60s.

The summary retained the teammate exchange *and its attribution* even when it was
irrelevant to the conversation's subject. The predicted silent-loss failure did
not reproduce.

## 5. Chosen remediation: none, plus observability

Phase 0a required a remediation only if context was lost. It was not, so:

- **Do not rewind the watermark on compaction.** It would inject duplicate
  teammate turns on every compaction for no benefit.
- **Do not add a re-injection floor.** Same reason.
- **Do record a `COMPACTION` event** when the daemon is built — the event model
  already reserves the type. This is cheap insurance: it makes compaction visible
  in the room's history, so if the summarizer's behavior ever changes, the
  correlation is already recorded rather than needing reconstruction.

Not implemented here; Phase 0a is a probe, and this belongs to Phase 1.

## 6. Turn reassembly survives the boundary

Reassembly anchors on the last user record bearing a `promptSource`. The compaction
summary is a user record with **no `promptSource`**, so it is correctly skipped and
never mistaken for a prompt. The post-compaction turn was captured intact.

The append-only transcript is what makes this safe: the anchor is never rewritten.

## 7. Slash commands do not pollute the room

`/compact` **never reached `UserPromptSubmit`** — only the two genuine prompts did.
The room contained no `/compact` event and no summary text.

Compaction also runs as a **subagent** (`SubagentStop` fired during it). The
existing `isSidechain` filter already excludes its output, so the summarizer's
own reasoning cannot leak into the shared conversation. Both behaviors are
load-bearing and neither was designed for deliberately — worth a regression test
if the hook wiring is ever changed.

## 8. Compaction never interrupts a turn

The spec asked whether compaction can strike mid-turn, between prompt submission
and completion — which would break turn reassembly.

A single turn was driven to **378,916 tokens** against a configured
`--autocompact 100k` threshold, via 20 Read calls over 960 KB of generated text.
**No compaction fired.** The turn ran to completion nearly 4× past the threshold.

Compaction is evaluated at turn boundaries, not during a turn. Turn reassembly is
therefore safe from mid-turn compaction.

## 9. Limitations — read these before trusting §4

1. **Automatic compaction was never triggered.** `--autocompact` rejects anything
   below `100k`, and even a 378k-token session with a 100k threshold did not
   compact at the next turn boundary. Auto-trigger mechanics remain **unverified**.
   All findings here describe manual compaction. The `trigger` field implies a
   shared code path, but that is inference, not evidence.
2. **Survival is model behavior, not a guarantee.** The summarizer chose to retain
   the teammate exchange. Nothing in the format *requires* it. A different model,
   a longer conversation, or a heavier compression ratio could drop it. §4 says
   this failure does not reproduce today — not that it cannot occur.
3. **Single compaction cycle only.** Repeated compaction may erode teammate
   context progressively, each pass summarizing the previous summary. Untested.
4. **Small injected volume.** Two teammate turns. A room with hundreds of unseen
   turns compresses differently.
5. **`custom_instructions` was always null.** `/compact <instructions>` may alter
   what is retained; not tested.
6. **macOS only**, and `-p` mode only.

## 10. What this means for the phases ahead

Phase 0a set out to find a silent failure and did not find one. The watermark
design in the specification is sound as written, and Phase 1 is unblocked.

The residual risk is entirely in §9.2 — survival depends on a summarizer's
judgment, and the system has no way to detect it changing. That argues for
recording `COMPACTION` events early (§5) so the signal exists before it is needed,
and for re-running Test B whenever the model or Claude Code version changes.

The remaining genuine unknown is automatic compaction under real context pressure,
which will first appear naturally during Phase 4 rather than in a probe.
