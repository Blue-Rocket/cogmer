# claude-team

Peer-to-peer daemon replicating one Claude Code conversation between people
working separately, each on their own subscription, with no central server.
`Shared Claude Sessions.md` is the specification; its §-numbers appear throughout
the code.

```
cmd/claude-team/
  main.go        subcommands + hook client (fails open)
  daemon.go      localhost HTTP, §20 context formatting, §21 limits
  store.go       SQLite event store, §19 delivery watermark
  transcript.go  turn reassembly — read the comments before editing
  identity.go    §6 peer identity, §28 config split
```

State is `~/.claude-team/` (`identity.json`, `config.json`, `rooms/*.db`). Drive a
real session with `claude -p … --settings <file>`, a throwaway `CLAUDE_TEAM_ROOM`
per run, and `< /dev/null` or it waits on stdin.

## Two is the target and nothing rules out more (D-109)

- Build for two people in a room. Spend no effort on a third.
- A design that *cannot* extend past two is a defect to argue for, not a saving to
  take quietly. O(n) work is a cost and is fine; a ceiling is not.
- Never describe the tool by a count.

## Hosts (D-110)

- Claude Code, then Claude CoWork a short interval later, then ChatGPT Desktop much
  later.
- Naming them licenses nothing: no `source` column, no adapter interface, nothing
  designed against an extension model we have not seen (D-043, D-086).
- Host-independence earns its place only where today's host already justifies it —
  the daemon and the view, which are a local service and a web page.

## Documents

| document | answers |
|---|---|
| `Shared Claude Sessions.md` | what must be true |
| `docs/decisions.md` | why, and what was rejected |
| `docs/*-findings.md` | what we observed when we tried it |
| `cmd/claude-team/behaviors.go` | what someone else's software does that we rely on |
| `docs/open.md` | what is to do, what is undecided, where things stand |

- **This file is rules. The reasoning is in the decision each rule cites.**
- Read `decisions.md` before proposing a change to how any of this works. Add an
  entry whenever a real alternative was weighed. Never renumber; supersede.
- A finding never goes in the spec — put the requirement it justifies there.
- A finding about someone else's software goes in the behaviour registry.
- Current state goes in `open.md`, and is deleted when done rather than marked done.
- A decision that changes what the system *is* updates the spec in the same pass
  (D-098).
- Corrections are clean replacements: no superseded text, no narration of what the
  old text said.
- Say **person** or **colleague**, never "developer": Claude Code is not used only
  by programmers, and nothing here is about code.
- Never call the project **local-first** unqualified. It is peer-to-peer and
  serverless; rendezvous across NAT needs a third-party relay, and D-029 (losing a
  room database) makes recovery a refetch from peers. The part that holds is §3's:
  a session survives every peer disappearing.
- Give every `§`, `D-NNN` and `B-NN` a few words — "D-054 (verification gates sync)",
  not "D-054" — in conversation as well as in files.

## The host is launched and used unchanged (§3.8)

The single test to apply to any proposal. The *host* is the application a session
runs in — Claude Code today. Three ways a proposal fails it:

- it needs the host started differently — a replacement command, a wrapper, a
  terminal interposed (D-034);
- it needs installing an artifact of a type that is not among the host's
  demonstrated, documented extension mechanisms. In Claude Code: hooks, skills,
  MCP servers and the plugin that carries them. The type is constrained; what the
  artifact *does* is not (D-113);
- it needs knowing how the host renders.

A host offering no way in is out of reach, never a reason to wrap one.

- Every extension point delivers to the model; presentation is best effort (D-033).
- MCP is never a display channel (D-036). Nothing checks this.

## A remote event never causes inference in an interactive session (§3.7)

- Remote events are stored, displayed and queued; they enter context at the next
  locally initiated turn and never before.
- An MCP server must not expose sampling, which would let a peer cause a turn in
  somebody's session.
- `sync.go`, `daemon.go`, `store.go` and `transcript.go` must not import `os/exec`
  or `syscall`, or call `RunProbe`/`EnsureVerified`/`runDoctor`. A test enforces it.

## The view (D-038, D-039)

- The view is a separate program outside the session. Do not interleave it, and do
  not propose a terminal pane.
- Room content is untrusted: `ui.html` escapes before applying markup, and a test
  asserts that order.
- No control that invites a click it cannot honour.
- The derived name appears on other peers, never on your own turns (D-021). An
  `unverified` marker is a backstop; seeing one means a filter failed.
- Prefer an OS notification from the daemon for ambient awareness.

## Pairing (D-088, D-055)

- Two words from a live commit/reveal exchange, compared aloud on a call, by both
  people at once. No fallback.
- Every pairing gets its own URL.
- `/verify/*` take a `pairId`, never a peer identifier.
- The 90-second window starts at Start, not at page load.
- The terminal path is the automatic fallback where no browser opens, so
  `openInBrowser` returns an error rather than failing quietly.

## The local API (D-087)

- State-changing routes require `X-Claude-Team: 1`. Require, not refuse.
- `Origin` is a second layer, not the check.
- Never `Referer`.
- `/healthz`, `/events`, `/stream` and the page stay unguarded.

## Installation (D-041)

`claude plugin install claude-team`; the session-start hook starts the daemon.

- Starting must not delay the session.
- Already running is the ordinary outcome, not an error.
- Failure is silent to the person at the keyboard.
- The daemon outlives its session, so it stays discoverable and stoppable.

## Command names (D-095, D-096)

- `peer-` acts on somebody else, `room-` on a room, `self-` on you. The prefix names
  the target, not the activity.
- No prefix for the tool itself, and no placeholder product name.

## Rooms (D-015, D-046)

- Key on `roomId`, never on `roomName` (D-017).
- Nothing derives a room from a directory, repository or project.
- A room begins at the first invitation, never at session start (D-022). Nothing
  before it is handed over.
- A session's room is fixed at its first prompt and never changes (D-016, D-056).
- Membership is durable, presence is not. A log is archived and read, never rejoined.
- The daemon is the machine's local service, not a room. A session in no room
  captures and injects nothing.
- Guest lists are per-peer, and an invitation carries the inviting peer's identifier.

## Transport and admission (D-019, D-024)

- No network provider in room identity, membership or replication. It must work with
  no VPN. Tailscale is one provider, never a prerequisite.
- Discovery locates a room and never admits anyone to one.
- Two scopes: known peers, per machine and durable; a room's guests, per room.
- Admission is a fresh signed challenge, never replayable.
- A refusal names the peer to the host (D-025).
- No join tokens (D-026).
- No join-by-name on a trusted network; names are guessable by design (D-017).

## Identity and verification (D-042, D-054)

Three checks answering three questions. Do not collapse them:

- **Authentication** — who holds the key (signature).
- **Admission** — whether that key may enter this room (D-045).
- **Verification** — whether the key is the person's (D-055).

- A `peerId` is an Ed25519 public key; knowing one grants nothing.
- Events are signed at origin and rejected on receipt if they do not verify (§13).
- Sync requests are signed, with replay and staleness rejected (D-044).
- The private key lives in `identity.key` (0600), never in `identity.json`. A test
  asserts it never marshals.
- An unverified peer is refused sync, its events are refused at their origin, and
  injection filters again.
- Events from an unverified peer are held, never dropped.
- Every refusal names the peer and the command that clears it.
- One way to verify and no fallback (D-055). `Fingerprint` marks nothing.
- Pairing is machine scope, inviting is room scope (D-053).
- `revoke` withdraws one room's admission; `forget` discards the identity and every
  admission it held (D-073); `allow` records without verifying, for scripts and tests.
- Peer names are derived from `peerId`, never chosen or stored, and are grindable —
  a mnemonic for an identity already verified, never an introduction.

## Capture and injection (D-014, D-040, D-043)

- Never publish `thinking` blocks, and exclude `isSidechain` records (§3.5).
- Do not summarize conversation (§3.4) — store the actual text.
- Injected context is attributed, never disguised as local (§20).
- Each injected block carries a per-injection fence, strips that fence from content,
  and restates the framing after the turns, framing by classification rather than by
  authority (D-040). Never a static delimiter.
- `ReassembleLastTurn` unions `Stop.last_assistant_message` with the transcript
  (B04, B05). Turn segmentation is positional (B09).
- `mergeTail` is not an append. Do not simplify it to one.
- No compaction rewind and no re-injection floor.
- Delivery is marked only on observing `hook_success` in the transcript (D-014),
  never at injection time. Unobserved events stay pending and are re-offered.
  Committing on trust requires B20 recorded as failing.
- Re-run Test B (`docs/phase0a-findings.md`) when the model or the Claude Code
  version changes. Whether injected context survives compaction is unassertable.
- `protocol.go` and `store.go` stay separate types with explicit conversion (D-043).
  Sync carries a version and refuses a peer speaking a different one.
- The session field is `originSessionId`. No `source`/adapter machinery.
- `migrate()` runs on open; add columns there. `CREATE TABLE IF NOT EXISTS` orphans
  every older room.

## The behavior registry

`cmd/claude-team/behaviors.go`, from which `docs/relied-on-behaviors.md` is
generated (`claude-team behaviors --markdown`) — edit the registry, not the doc.

- A new reliance gets a behavior. These fail silently.
- Every check gets a negative test in `behaviors_test.go`. A check that cannot fail
  reads as protection.
- Write `Reliance` for someone debugging at 2am: what breaks, not what the behavior
  is. The length minimum is a floor, not a target.
- Watch B04, B05 and B09 for improvements, not only breakage; they would let us
  simplify.
- `doctor` is ~5s, `doctor --deep` adds compaction at ~40s. Auto-runs on new room
  formation, cached by `claude --version`. `CLAUDE_TEAM_PREFLIGHT=off` for CI.

## Invariants

- Hooks never break Claude Code (§3.1). Every failure path exits 0 with empty stdout.
- Durability precedes publication (§23).
- Events are immutable (§7). Never rewrite `eventId`, `peerId` or `peerSequence`.
- Signature schemes are added, never edited (D-058). Add `signingBytesV3` and bump
  `currentSigVersion`.
- Reserve a sequence before publishing the event that uses it (D-029). The sequence
  position lives outside the room database.
- No CRDT until testing proves it necessary (§11).
- The name is not settled and Phase 13 is blocked on it (§31). `protocolNamespace`
  is arbitrary on purpose and must never change (D-069).
