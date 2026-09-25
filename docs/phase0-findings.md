# Phase 0 — Claude Code Integration Spike: Findings

**Verdict: both required directions work.** Conversation capture and cross-session
context injection are reliable against the installed Claude Code. The spike's
instructions said to stop here and report, so peer networking is not built.

| | |
|---|---|
| Claude Code | **2.1.273** (native binary, `~/.local/share/claude/versions/`) |
| Platform tested | macOS 15 (darwin/arm64) |
| Spike commit | this repo, `cmd/cogmer` |
| Date | 2026-09-16 |

> Claude Code ships as a **native binary, not an npm package**. A teammate can have
> Claude Code with no Node runtime at all. This is why the daemon is Go: one
> dependency-free file per platform. All four targets cross-compile from one Mac
> with no cgo (`modernc.org/sqlite` is pure Go).

---

## 1. Hooks that fire, and what they carry

Probed empirically by registering dump-stdin hooks via `--settings` and running a
real session. Six of nine registered hooks fired on a normal prompt→tool→response turn.

| Hook | Fires | Key fields |
|---|---|---|
| `SessionStart` | session open | `session_id`, `source`, `transcript_path`, `cwd` |
| `UserPromptSubmit` | each submitted prompt | **`prompt`**, `prompt_id`, `session_id`, `transcript_path`, `permission_mode` |
| `PreToolUse` | before each tool | `tool_name`, `tool_input`, `tool_use_id`, `prompt_id` |
| `PostToolUse` | after each tool | `tool_name`, `tool_input`, **`tool_response`**, `duration_ms`, `tool_use_id` |
| `Stop` | turn completion | **`last_assistant_message`**, `prompt_id`, `stop_hook_active`, `background_tasks` |
| `SessionEnd` | session close | `reason` |

`PreCompact`, `Notification`, and `SubagentStop` did not fire in this scenario —
not evidence they are unavailable, only unexercised.

Every payload carries `session_id`, `cwd`, and `transcript_path`. This satisfies §6
directly: **`session_id` is the `claudeSessionId`** with no inference needed.

**`prompt_id` is the turn-correlation key.** It is identical across
`UserPromptSubmit`, `PreToolUse`, `PostToolUse`, and `Stop` for one turn — this is
what pairs a `USER_PROMPT` with its `ASSISTANT_MESSAGE` (§7).

## 2. Capturing prompts (§14) — clean

`UserPromptSubmit` delivers the prompt verbatim. No transcript parsing needed.

In the transcript, genuine human prompts are distinguished by
`promptSource: "typed"` (interactive) or `"sdk"` (`-p` mode) with
`origin: {"kind":"human"}`. Tool-result records are also `type: "user"` but carry
**no `promptSource`** — so the filter is exact, not heuristic.

## 3. Capturing responses (§15) — needs both sources unioned

§15 anticipated trouble here, correctly. **Neither available source is complete,
and they fail in opposite directions.**

**`Stop.last_assistant_message` carries only the FINAL text block.** A turn
instructed to say `ALPHA`, run a Bash command, then say `OMEGA` reported exactly
`OMEGA`. The pre-tool text is silently dropped. Since real turns routinely open
with narration before tool calls, this loses content on most substantive turns —
publishing it would violate §3.4/§15.

**The transcript at Stop time is missing exactly that final block.** Stop fires
*before* the closing assistant record is flushed to disk. Reading the transcript
alone captured a 110-char preamble of a 2,804-char answer. (The full text is
present moments later — this is a write-ordering race, not absent data.)

**Their union is the complete turn**, implemented in `ReassembleLastTurn`:
transcript supplies every block except the last, `last_assistant_message` supplies
the last, with a guard against double-counting when the transcript wins the race.

*Verified:* captured event vs. actual emitted response — **2,582 chars, exact
string match**, including 3 tool calls recorded as metadata.

### Turn segmentation is positional, not by pointer

Two things rule out the obvious approaches:

- **Assistant records carry no `promptId`.** Only user records do (including
  tool-result records, which carry the originating turn's id).
- **The `parentUuid` chain has gaps.** Observed an assistant record whose
  `parentUuid` matched no preceding `uuid` in the same file.

So the reassembler takes every `assistant` record following the last
`promptSource`-bearing user record. `thinking` blocks are never published;
`isSidechain: true` records (subagent traffic) are excluded per §15.

## 4. Context injection (§18–§20) — works, and this was the critical unknown

**`UserPromptSubmit` stdout is injected into the pending turn.** Confirmed with
the test the spike's instructions set: a simulated teammate exchange was seeded into the daemon, then
a prompt was submitted whose referent existed *only* in that injected text —

> "I don't think her explanation is right. Check the retry path instead."

Claude resolved "her explanation" to the teammate's idle-pool/stale-socket theory,
named Alice, fused it with the local code, and **disagreed with it on the merits**
— treating it as context rather than instruction, which is exactly the §20
requirement. Injection used the §20 attributed format (`<team-conversation>` with
`speaker=` per message).

**One caveat worth recording:** injected context is *not* privileged. In an early
probe where the hook script sat in the working directory, Claude read it and
correctly identified the teammate conversation as synthetic. Injected content is
treated as evidence to be scrutinized, not fact. This is the right behavior, but
it means the room is persuasion, not ground truth.

## 5. Latency

| Path | Result |
|---|---|
| `UserPromptSubmit` round trip (spawn + HTTP + SQLite commit + inject) | **median 16.7 ms**, p95 18.7 ms |
| Daemon **down** | 17.4 ms, exit 0, empty stdout |

Well inside the §16 sub-second target, and the cost is per-prompt, not per-token.

## 6. §3.1 resilience — verified

With the daemon killed, a session ran normally and returned `SURVIVED`. Every hook
failure path exits 0 with empty stdout, so a dead daemon degrades to "no
collaboration," never a broken session. Connection-refused returns immediately
rather than burning the 3 s timeout.

## 7. Limitations and open questions

1. **`--settings` was used to isolate probes.** Hooks registered mid-session were
   not tested for hot-reload; the daemon's install path should assume a restart.
2. **Compaction was not exercised**, although the spike's instructions asked for its
   interaction with the session to be documented. Claude Code's compaction summarises
   the earlier turns of a session when its context window fills, and in this spike
   `PreCompact` never fired. Unknown how compaction interacts with the §19 delivery
   watermark — a compacted session has lost injected context it is still marked as
   having received. **This is the most
   likely source of surprise in Phase 4.**
3. **Windows is unverified.** All findings are macOS. Hook shell invocation,
   quoting, and exit-code semantics differ on Windows and must be re-probed there
   before Phase 2. The Windows binary builds but has not been run.
4. **`SubagentStop` unexercised** — subagent turns are currently excluded wholesale
   via `isSidechain`.
5. **No streaming.** Capture is turn-granular by construction; a teammate sees a
   response only once it completes. §16's sub-second target applies to propagation
   after completion, not to live token streaming.
6. **Interactive mode unverified.** All runs used `claude -p`. `promptSource`
   differs (`typed` vs `sdk`); the reassembler accepts both, but the end-to-end
   flow has not been driven through an interactive TTY session.

## 8. What this means for the phases ahead

The two riskiest unknowns in the spec are now resolved: capture is complete (given
the union fix) and injection genuinely changes what Claude understands. Phases 1–3
are ordinary distributed-systems work against a stable local interface.

The spec's §30 question — whether synchronizing complete turns approximates a
shared conversation — is not yet answered, but nothing in the integration surface
prevents answering it. The recommended next probe is **compaction behavior**,
since it is the one mechanism that can silently invalidate the §19 watermark.
