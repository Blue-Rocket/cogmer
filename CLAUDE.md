# claude-team

Prototype from `Shared Claude Sessions.md` — a local-first P2P daemon replicating
Claude Code conversations between developers. Read that spec before changing
architecture; section numbers (§7, §19, …) are referenced throughout the code.

## Rooms are session-scoped (D-015)

A room is a set of linked Claude Code sessions, entered by invitation, closed when
its members leave or it lies dormant. It has two identifiers: a `roomId` UUID that
everything keys on, and a generated `roomName` like `misty-canyon` for people.
**Never key on the name** — names collide, identities do not (D-017).

A room begins when someone is **invited**, never when a session starts — so solo
work before that is captured nowhere and is never handed over retroactively
(D-022). "From its beginning" always means the room's beginning.

A session belongs to **at most one room at a time**, and may not move to another
room once teammate context has been injected, because injected context cannot be
withdrawn from a context window (D-016). Process exit makes a member *absent*, not
gone — sessions resume under the same ID, so membership is durable and presence is
not. **Nothing derives a room
from a directory, repository, or project.** Membership ends; the event log is
archived and may be read, never rejoined.

**The implementation has not caught up with this yet.** `CLAUDE_TEAM_ROOM` selects
a single room per daemon process, there is no invite, no membership tracking, and
no archive. That is Phase 1 work; do not treat the current shape as the design.

## Where things stand

Phase 0 (integration spike) is complete — see `docs/phase0-findings.md`.
**Peer networking does not exist yet.** Per §36.10 the spike stopped deliberately;
do not start Phase 1+ without saying so explicitly.

Phase 0a (compaction probe) is complete — see `docs/phase0a-findings.md`.
Injected context survives compaction, so **the watermark is correct as written**.
Do not add a compaction rewind or a re-injection floor: both were evaluated and
rejected as duplicate injection for no benefit.

Phase 1 is unblocked but not started.

## Before proposing a change to how any of this works

Read `docs/decisions.md`. It records what was decided, and — more usefully — what
was **rejected and why**. Several awkward-looking choices are load-bearing:
`mergeTail` is not a plain append on purpose, turn segmentation is positional on
purpose, and the compaction watermark is deliberately left alone. Each entry has a
"Revisit when" naming the check that would invalidate it.

Add an entry whenever a real alternative was weighed. Never renumber; supersede.

## The behavior registry — read this before changing assumptions

`cmd/claude-team/behaviors.go` records every Claude Code behavior this project
relies on, with a check that verifies it against the installed version.
`docs/relied-on-behaviors.md` is **generated** from it
(`claude-team behaviors --markdown`) — edit the registry, not the doc.

**If you discover a new reliance, add a behavior.** That is the whole point: these
are undocumented behaviors of someone else's binary, and several fail silently —
the room keeps accepting events while recording the wrong thing.

- `claude-team doctor` — session tier, one Claude turn, ~5s
- `claude-team doctor --deep` — adds compaction, ~40s
- Auto-runs on **new room formation**, cached by `claude --version`
  (`~/.claude-team/verified.json`). Set `CLAUDE_TEAM_PREFLIGHT=off` for CI.

Two rules when adding one:

1. **Write a negative test.** `behaviors_test.go` asserts each check fails on the
   regression it claims to catch. A check that cannot fail is worse than none —
   it reads as protection.
2. **Write the `Reliance` field for someone debugging at 2am.** Say what breaks,
   not what the behavior is. The test enforces a minimum length; that is a floor,
   not a target.

Note B09 and B05 report *improvements* (assistant records gaining `promptId`, the
flush race disappearing), not just breakage. Those would let us simplify, and we
would otherwise never notice.

## Two findings the code depends on

These were established empirically against Claude Code 2.1.273 and are easy to
regress if the reassembly logic is "simplified":

1. **`Stop.last_assistant_message` holds only the turn's FINAL text block** — text
   emitted before a tool call is dropped. (Checked by B04.)
2. **The transcript at Stop time is missing exactly that final block** — Stop fires
   before it is flushed. (Checked by B05.)

`ReassembleLastTurn` unions both via `mergeTail`. Neither source alone is correct.
Verified by exact string match against a real 2,582-char response.

`mergeTail` deliberately tolerates #1 being *fixed* upstream: if
`last_assistant_message` ever widens to the whole turn, blind appending would
duplicate every pre-tool block and silently corrupt the room. It detects the
superset case instead of assuming. Do not "simplify" it back to an append.

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

## Delivery is confirmed, never assumed

Teammate events are marked delivered only when the injected block is **observed**
in the session transcript as a `hook_success` attachment (D-014). Do not
reintroduce marking at injection time: the daemon's reply can be lost, and
committing then discards the context permanently and silently.

When no evidence is found, events stay pending and are re-offered. Committing on
trust is gated on B20 being recorded as *failing* — absence of evidence is not
evidence that the format changed.

## No transport is required (D-019)

No network provider belongs in room identity, membership, or replication. The
system must work with **no VPN** when peers can already reach each other; local
discovery is the zero-configuration path and the first transport to build.
Tailscale is one provider among several, never a prerequisite.

Discovery locates a room. It never admits anyone to one — the **guest list** does
that (D-024): a host records a peer it already knows, the guest types only
`claude-team join misty-canyon`, and admission is proof of possession of a key the
host already holds. A join code is the fallback for people who have never exchanged
identifiers, and is explicitly weaker — it is a bearer credential. Do not add a "join by name on a trusted network" path: names are guessable
by design (D-017), and office and conference networks are neither small nor
trusted.

## Peer identity is self-asserted (D-020)

A `peerId` is ten random bytes generated locally and verified by nothing — so
**knowing one is enough to claim it.** It is a shared secret that the system
broadcasts in every event. The fix is to make the identifier the public key itself
(D-023), but note the ordering: it becomes safe to know when signatures are
*checked*, not when keys are introduced. Generating keypairs without verification
looks like the fix and delivers none of it.

Identities are created once per machine on first use and exchanged when a peer
joins — never obtained in advance. So
attribution is accurate among cooperating peers and not resistant to one that
lies, a guest list cannot replace the join secret, and §13's rule that a relayer
must not rewrite event origin has no enforcement.

Peer names (`quiet-otter`) are **derived** from `peerId`, never chosen or stored —
`PeerName()` hashes it, so it keeps working when the id becomes a key fingerprint.
A derived name is not unforgeable: 8,280 combinations are grindable, so a name is a
mnemonic for an identity already verified, never an introduction to a stranger.
Unverified speakers must be marked **inside injected text**, not just in a UI — the
model reasons about attribution.

**Build nothing that assumes otherwise.** Once identity is cryptographic, prefer
admitting known peers over holders of a secret — the secret is a first-contact
mechanism, not a standing requirement.

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
