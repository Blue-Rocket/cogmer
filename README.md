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

Outstanding: local network discovery, and a host approving an unsolicited join
request — the latter undecided rather than pending.

**Blocked on the name.** The plugin is built and the install is one line by design
(D-041), but a Go module path must match its repository URL, so until the name is
settled there is no repository and nowhere for a colleague to install from. That is
what stands between this and somebody else using it.

Earlier findings that the design still rests on:
[`docs/phase0-findings.md`](docs/phase0-findings.md) (integration spike, including
two non-obvious defects in the Claude Code surface) and
[`docs/phase0a-findings.md`](docs/phase0a-findings.md) (compaction — injected
teammate context survives it, so the delivery watermark is unchanged).

## Getting two people talking

Installed as a plugin, this is meant to be invisible: the session-start hook fetches
the binary and starts the daemon, and nothing below needs doing by hand. That path
is blocked on the name (see Status), so today it is built from source.

```sh
go build -o bin/claude-team ./cmd/claude-team
claude-team daemon                       # hooks and UI on 127.0.0.1:4782
```

**One thing to know before reading any command here.** A plugin install puts the
binary in `~/.claude-team/bin`, which is deliberately not on PATH — §29 forbids
editing a shell profile to put it there. So `claude-team pair` resolves only if you
installed it yourself. Every command below is written short for readability; the
binary prints its own path in the lines it asks you to type, and those are the ones
to trust:

```sh
~/.claude-team/bin/claude-team whoami
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

**Pair, once, on a call with them.** Each of you sends the other your pairing
string; then both run `/peer-pair` with the other's, at the same time. A page opens
in each of your browsers showing two words, which you read to each other.

```sh
/peer-pair                                   # prints your string — send it to them
/peer-pair ed25519:GoR7…IWPU@203.0.113.9:4783
#   Opened the pairing page:
#     http://127.0.0.1:4782/pair/mX_ky1Cyk1_TpgQjcTJHqA
```

Every pairing gets its own address, so a second one opens a new tab rather than
rewriting a page nobody is looking at — and a page from an earlier attempt can never
quietly become a different pairing (D-088).

Nothing synchronizes until both people confirm the words matched — verification is a
gate, not a label. They are worth understanding rather than clicking through: §25
explains why two words are enough, and why a mismatch must never be retried.

On a machine with no browser — over SSH, in a container — the same ceremony happens
in the terminal instead. `--terminal` forces it; otherwise it is automatic.

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
| `forget <peer>` | Discard a peer and every admission it held |

Rooms — per room, as often as you like:

| Command | Purpose |
|---|---|
| `create` | Create a room and make it current |
| `join <invitation>` | Enter a room; sessions started afterwards join it |
| `leave` | Take this session out of its room (it may rejoin) |
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

`CLAUDE_TEAM_ADDR` sets the local
address; `CLAUDE_TEAM_PEER_ADDR` the peer address; `CLAUDE_TEAM_PEERS` extra peer
addresses; `CLAUDE_TEAM_PREFLIGHT=off` skips behavior checks on new rooms;
`CLAUDE_TEAM_MAX_EVENTS`, `CLAUDE_TEAM_MAX_EVENT_CHARS` and
`CLAUDE_TEAM_MAX_BLOCK_CHARS` bound injected context (§21).

## Specification review

[`docs/spec-review.md`](docs/spec-review.md) — the spec evaluated against what the
implementation actually established. Each finding carries its own status, and the
open ones are collected at the end.

## Why things are the way they are

[`docs/decisions.md`](docs/decisions.md) — decisions numbered to D-100, with the alternatives
rejected and why, each tied to the check that would invalidate it. Read it before
proposing a simplification; some of the awkwardness is deliberate.

## Behavior checks

This project depends on 23 **undocumented** behaviors — how hooks report a turn,
what the transcript contains, what compaction preserves, and whether a detached
daemon can still put a window in front of a person. None are
contractual, and several fail silently: the room keeps accepting events while
recording the wrong thing.

[`docs/relied-on-behaviors.md`](docs/relied-on-behaviors.md) lists them, generated
from the registry in `cmd/claude-team/behaviors.go` so it cannot drift from what
is actually checked.

```sh
claude-team doctor          # 16 checks, one Claude turn, ~5s
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
