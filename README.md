# claude-team

Local-first peer-to-peer shared Claude Code conversations. Several developers,
each on their own Claude Code subscription and session, collaborate in one
replicated conversation with no central server.

Full specification: [`Shared Claude Sessions.md`](Shared%20Claude%20Sessions.md)

## Status

**§30's question is answered affirmatively.** Two independent Claude sessions
synchronizing turns produced an exchange where the second resolved a referent from
the first, disagreed with it on the merits, and found a defect the first had missed
— see [`docs/phase2-experiment.md`](docs/phase2-experiment.md). Repeated across two
machines and two Claude Code versions: 0.44 s sync over the open internet, identical
event ordering, all behavior checks passing on Linux.

Working: hooks, storage, turn reassembly, two-peer synchronization over the
internet, signed events, signed sync requests, peer identity, pairing with two-word
verification, rooms with per-room guest lists, and a live browser view.

Offline and reconnection is verified across two machines — see
[`docs/phase5-findings.md`](docs/phase5-findings.md) — and Phase 7's hardening is
done, including recovery from a lost room database. **Every numbered phase in §31 is
now complete or deliberately dissolved.**

Outstanding: plugin packaging, local network discovery, and a host approving an
unsolicited join request (undecided rather than pending). A host approving an
unsolicited join request is undecided rather than pending.

Earlier findings that the design still rests on:
[`docs/phase0-findings.md`](docs/phase0-findings.md) (integration spike, including
two non-obvious defects in the Claude Code surface) and
[`docs/phase0a-findings.md`](docs/phase0a-findings.md) (compaction — injected
teammate context survives it, so the delivery watermark is unchanged).

## Getting two people talking

```sh
go build -o bin/claude-team ./cmd/claude-team
claude-team daemon                       # hooks and UI on 127.0.0.1:4782
```

Register the hooks (`--settings` keeps this out of your real config):

```json
{"hooks":{
 "UserPromptSubmit":[{"hooks":[{"type":"command","command":"/abs/path/bin/claude-team hook prompt"}]}],
 "Stop":[{"hooks":[{"type":"command","command":"/abs/path/bin/claude-team hook stop"}]}]
}}
```

Set `CLAUDE_TEAM_PEER_ADDR` to an address your colleague can reach — it defaults to
loopback, so nobody can reach you until you do.

**Pair, once, at a terminal, on a call with them.** Each of you runs `whoami` and
sends the other the string it prints; then both run `pair` at the same time and
compare two words out loud.

```sh
claude-team whoami
#   ed25519:M7Kd…4Fq2@198.51.100.7:4783

claude-team pair ed25519:GoR7…IWPU@203.0.113.9:4783 david
#         ribcage tambourine
#   did they say the same two words? [y/N] y
```

Nothing synchronizes until that says yes — verification is a gate, not a label.
The words are worth understanding rather than clicking through: §25 explains why
two of them are enough, and why a mismatch must never be retried.

**Then make a room and invite them.**

```sh
claude-team create                       # → misty-canyon
claude-team invite david
#   give them: claude-team join misty-canyon/0f7a…@198.51.100.7:4783#ed25519:GoR7…
```

Start Claude Code normally on both machines. Turns replicate; teammate turns are
injected at each session's next locally initiated prompt, never before.

## The room, in a browser

```sh
claude-team daemon        # then open http://127.0.0.1:4782
```

Live-updating over server-sent events, with attribution anchored on each peer's
derived name rather than the display name it asserts, timestamps, code and
Markdown rendering, connection status, and peer reachability. Served from the
binary — no assets, no build step, no external requests.

## Commands

Peers — durable, above any room, done once with each colleague:

| Command | Purpose |
|---|---|
| `whoami` | Your identity, pairing string, and active room |
| `pair <string> [name]` | Record a peer **and** verify it, on a call with them |
| `peers` | Known peers, and whether each is verified |
| `verify <peer>` | Re-run just the two-word comparison |
| `allow <id> [name]` | Record a peer **without** verifying (scripts and tests) |
| `forget <peer>` | Discard a peer entirely |

Rooms — per room, as often as you like:

| Command | Purpose |
|---|---|
| `create` | Create a room and make it current |
| `join <invitation>` | Enter a room; sessions started afterwards join it |
| `leave` | Leave the current room |
| `invite <peer>` | Admit a known peer, and print their invitation |
| `revoke <peer>` | Withdraw admission to this room only |
| `guests` | Who may enter the current room |
| `rooms` | Rooms this machine belongs to |
| `log` | Print the room transcript |
| `conflicts` | Quarantined events (sequence conflicts) |

Running it:

| Command | Purpose |
|---|---|
| `daemon` | Local daemon: hooks and UI on `:4782`, peer sync on `:4783` |
| `hook prompt` | `UserPromptSubmit` — captures the prompt, injects unseen teammate turns |
| `hook stop` | `Stop` — reassembles and stores the completed response |
| `seed` | Insert a simulated teammate conversation |
| `doctor [--deep]` | Verify relied-on Claude Code behaviors against the installed version |
| `behaviors` | List those behaviors (`--markdown` regenerates the doc) |

`CLAUDE_TEAM_ROOM` overrides the current room; `CLAUDE_TEAM_ADDR` the local
address; `CLAUDE_TEAM_PEER_ADDR` the peer address; `CLAUDE_TEAM_PEERS` extra peer
addresses; `CLAUDE_TEAM_PREFLIGHT=off` skips behavior checks on new rooms;
`CLAUDE_TEAM_MAX_EVENTS`, `CLAUDE_TEAM_MAX_EVENT_CHARS` and
`CLAUDE_TEAM_MAX_BLOCK_CHARS` bound injected context (§21).

## Specification review

[`docs/spec-review.md`](docs/spec-review.md) — the spec evaluated against what the
implementation actually established. Each finding carries its own status, and the
open ones are collected at the end.

## Why things are the way they are

[`docs/decisions.md`](docs/decisions.md) — 67 decisions with the alternatives
rejected and why, each tied to the check that would invalidate it. Read it before
proposing a simplification; some of the awkwardness is deliberate.

## Behavior checks

This project depends on 21 **undocumented** Claude Code behaviors — how hooks
report a turn, what the transcript contains, what compaction preserves. None are
contractual, and several fail silently: the room keeps accepting events while
recording the wrong thing.

[`docs/relied-on-behaviors.md`](docs/relied-on-behaviors.md) lists them, generated
from the registry in `cmd/claude-team/behaviors.go` so it cannot drift from what
is actually checked.

```sh
claude-team doctor          # 14 session checks, one Claude turn, ~5s
claude-team doctor --deep   # adds 7 compaction checks, drives a real compaction, ~40s
```

The session tier runs automatically the first time a room is formed on a Claude
Code version that has not been verified, cached in `~/.claude-team/verified.json`.
Keyed on version rather than room, so forming a tenth room costs nothing.

A failure never blocks the room — it reports which assumption changed, and
`Reliance` on each behavior says what breaks as a result.

## Cross-platform

Go with pure-Go SQLite, so every target builds from one machine with no cgo and
no runtime dependency — which matters because Claude Code now ships as a native
binary and teammates may have no Node installed.

```sh
GOOS=windows GOARCH=amd64 go build -o bin/claude-team.exe ./cmd/claude-team
```

Windows binaries build but are **unverified at runtime** — hook shell semantics
differ and have not been re-probed.
