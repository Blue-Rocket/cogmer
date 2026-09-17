# claude-team

Prototype from `Shared Claude Sessions.md` — a local-first P2P daemon replicating
Claude Code conversations between developers. Read that spec before changing
architecture; section numbers (§7, §19, …) are referenced throughout the code.

## Where things stand

Phase 0 (integration spike) is complete — see `docs/phase0-findings.md`.
**Peer networking does not exist yet.** Per §36.10 the spike stopped deliberately;
do not start Phase 1+ without saying so explicitly.

Phase 0a (compaction probe) is complete — see `docs/phase0a-findings.md`.
Injected context survives compaction, so **the watermark is correct as written**.
Do not add a compaction rewind or a re-injection floor: both were evaluated and
rejected as duplicate injection for no benefit.

Phase 1 is unblocked but not started.

## Two findings the code depends on

These were established empirically against Claude Code 2.1.273 and are easy to
regress if the reassembly logic is "simplified":

1. **`Stop.last_assistant_message` holds only the turn's FINAL text block** — text
   emitted before a tool call is dropped.
2. **The transcript at Stop time is missing exactly that final block** — Stop fires
   before it is flushed.

`ReassembleLastTurn` unions both. Neither source alone is correct. Verified by
exact string match against a real 2,582-char response.

Also: assistant records carry no `promptId` and the `parentUuid` chain has gaps,
so turn segmentation is **positional** — assistant records following the last
`promptSource`-bearing user record.

## Compaction behavior these rely on

Verified in Phase 0a, and load-bearing:

- `claudeSessionId` and `transcript_path` **survive compaction**; the transcript
  is append-only and is never rewritten. The watermark is keyed on session ID, so
  a changed ID would silently re-inject the entire room.
- The compaction summary is a user record with **no `promptSource`**, so the
  reassembly anchor correctly skips it.
- Compaction runs as a **subagent**; the `isSidechain` filter is what keeps the
  summarizer's output out of the room.
- Slash commands do **not** reach `UserPromptSubmit`, so `/compact` never becomes
  a room event.
- Compaction never fires mid-turn (verified to 378k tokens against a 100k
  threshold), so reassembly cannot be split across a boundary.

Residual risk: context survival is summarizer judgment, not a format guarantee.
Re-run Test B from the findings when the model or Claude Code version changes.

## Invariants

- **Hooks must never break Claude Code** (§3.1). Every failure path exits 0 with
  empty stdout. A dead daemon means no collaboration, never a broken session.
- **Events are immutable** (§7). Never rewrite `eventId`, `peerId`, or
  `peerSequence` — transitive relay (§13) depends on it.
- **Durability precedes publication** (§23): commit locally, then transmit.
- **Never publish `thinking` blocks**, and exclude `isSidechain` records (§3.5).
- **Injected context is attributed, never disguised as local** (§20).
- **Do not summarize conversation** (§3.4) — store the actual text.
- **No CRDT** until testing proves it necessary (§11).

## Layout

```
cmd/claude-team/
  main.go        subcommands + hook client (fails open)
  daemon.go      localhost HTTP, §20 context formatting, §21 limits
  store.go       SQLite event store, §19 delivery watermark
  transcript.go  turn reassembly — read the comments before editing
  identity.go    §6 peer identity, §28 config split
```

State lives in `~/.claude-team/` (`identity.json`, `config.json`, `rooms/*.db`).

## Testing

`claude -p ... --settings <file>` drives a real session without touching your own
config. Use a throwaway `CLAUDE_TEAM_ROOM` per run. Note that `claude -p` needs
`< /dev/null` or it waits on stdin.
