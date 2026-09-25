# cogmer

Peer-to-peer daemon replicating one Claude Code conversation between people working
separately, each on their own subscription, with no central server.

This file is operating doctrine, not documentation: how we work, what this project
has chosen, what it must not break, and how to run it. It is deliberately not an
encyclopedia — the last section says where to read instead.

## How we work

- **Read `docs/decisions.md` before proposing a change to how anything here works.**
  It records what was **rejected and why**, and several awkward-looking choices are
  load-bearing. Add an entry whenever a real alternative was weighed. Never
  renumber: a reversed decision becomes a tombstone, and the reason for reversing
  it goes in a finding.
- **Read `docs/writing.md` before writing any document or commit message**, and
  follow it. It holds the rules and a template for each kind of item.
- **Read the code before characterising it.** Paraphrasing a grep result produced a
  wrong account of two decisions in one session. Read the function, not the line
  that mentions it.
- **Check `git log` before concluding something was removed.** `git log -S` across
  all commits answers "was this ever written?" — which is a different question from
  "is it here now", and twice it was the one that mattered.
- **Cite nothing you have not confirmed exists.** Twelve citations pointed at three
  decisions that were never written, because a number was allocated while writing a
  commit and the entry never followed.
- Corrections are clean replacements: no superseded text, no narration of what the
  old text said.
- Give every `§`, `D-NNN` and `B-NN` a few words — "D-054 (verification gates
  sync)", not "D-054" — in conversation as well as in files.
- Say **user** for someone using cogmer, **colleague** for another user in their
  room, and **person** only for a human as distinct from the model or a program.
  Never "developer": a user may not write code. Never describe the tool by a
  count. Never call the project **local-first**: it claims
  more than we deliver, since rendezvous needs a relay somebody else operates and
  D-029 (losing a room database) makes recovery a refetch from peers. §3.1 (first,
  do no harm) is the principle it was standing in for.

**Where a thing gets written down.** Putting an answer in the wrong document is how
they rot.

| document | answers |
|---|---|
| `Shared Claude Sessions.md` | what must be true |
| `docs/decisions.md` | why, and what was rejected |
| `docs/*-findings.md` | what we observed when we tried it |
| `cmd/cogmer/behaviors.go` | what someone else's software does that we rely on |
| `docs/open.md` | actions not yet taken, decisions not yet made |
| `docs/phases.md` | the phases the work was planned in, and how each ended |
| `docs/work/` | material for one open item, deleted with it; nothing durable cites it |
| `README.md` | what somebody who has not installed it needs (D-122) |
| `plugin/README.md` | what somebody who has installed it needs |

- A finding never goes in the spec — put the requirement it justifies there.
- A finding about someone else's software goes in the behaviour registry, the only
  one of the five that tests itself.
- An open item is deleted when it is done, never marked done. Nothing records status
  except the phase table in `docs/phases.md`.
- A decision that changes what the system *is* updates the spec in the same pass
  (D-098).

## The local physics

```
cmd/cogmer/
  main.go        subcommands + hook client (fails open)
  daemon.go      localhost HTTP, §20 context formatting, §21 limits
  store.go       SQLite event store, §19 delivery watermark
  transcript.go  turn reassembly — read the comments before editing
  identity.go    §6 peer identity, §28 config split
```

State is `~/.cogmer/` (`identity.json`, `identity.key`, `config.json`,
`rooms/*.db`). Pure Go throughout — `modernc.org/sqlite`, no cgo — so every target
cross-compiles with `CGO_ENABLED=0`.

- Build `go build -o bin/cogmer ./cmd/cogmer`. `bin/` is untracked: 17MB
  per commit, and the build reproduces it.
- Test `go test ./...`. To drive a real session, `claude -p … --settings <file>`
  with a throwaway `COGMER_ROOM` per run, and `< /dev/null` or it waits on
  stdin.
- `cogmer doctor` checks the behaviours we rely on: ~5s, one Claude turn.
  `--deep` adds compaction, ~40s. It auto-runs on new room formation, cached by
  `claude --version`. `COGMER_PREFLIGHT=off` for CI.
- `docs/relied-on-behaviors.md` is **generated** (`cogmer behaviors
  --markdown`). Edit `behaviors.go`, never the doc.

**This repo has a codegraph index, and reaching for grep instead is the default
failure.** grep is the reflex for every lookup and wins by inertia unless the split
is stated:

- **Symbols, callers, callees, impact** — `codegraph_search`, `codegraph_callers`,
  `codegraph_impact`. `grep -rn "funcName"` is the wrong tool for a call graph, and
  running `codegraph_impact` before changing a signature is the whole point.
- **Text** — grep. SQL inside a string literal, a citation like `§3.1`, anything in
  a `.md`. The graph holds the Go sources and knows nothing about prose,
  which is most of this repository.

**A stale index answers confidently.** `codegraph_search` on a function added an
hour ago returns "No results found", which reads as *that does not exist* rather
than *this index is old* — a failure grep cannot have. `.codegraph/` is gitignored
and per-machine; `codegraph sync` costs ~0.2s and `codegraph status` says what it
holds. Hooks keep it fresh, but they only fire for edits this session made.

**Releasing is three steps in one order.** Bump `plugin/VERSION`, run
`scripts/release.sh <that same version>`, then `scripts/publish.sh`.

- `release.sh` builds five targets and rewrites every file that names the version:
  `plugin/checksums.txt` **from the bytes it just built**, `plugin/VERSION`, and the
  `version` in `plugin.json`. `checksums.txt` is the only thing authorising a
  downloaded binary to run, so never hand-edit it — a hash typed rather than
  computed authorises something nobody has seen.
- The manifest `version` is what pins an installed plugin (D-120): a person receives
  a new plugin, and with it the checksums that authorise the new binary, only when
  that string moves. A test fails if it and `plugin/VERSION` disagree.
- Publishing without rerunning `release.sh` leaves the installer refusing the new
  asset. That is the correct failure, not a bug to work around.
- `publish.sh` reads `plugin/VERSION`, requires `dist/`, and ships to the host in
  `plugin/release-url.txt`.

## Choices this project has made

Not universal truths — the part of the possibility space this project selected.

- **Two people is the target, and nothing rules out more** (D-109). Spend no effort
  on a third. O(n) work is a cost and is fine; a design that *cannot* extend past
  two is a ceiling and must be argued for. Events are immutable, so a ceiling in an
  event format is permanent.
- **No adapter machinery for a second host** (D-043, D-113). Capture generalises and
  injection does not, so the abstraction that looks safe on the capture side is not
  safe on the injection side. Hosts are ordered — Claude Code, CoWork soon, ChatGPT
  Desktop much later (D-110) — and naming them licenses nothing.
- **A view is a separate program, never drawn inside the session** (D-038, D-039).
  A session is read closely and a room is glanced at. Do not propose a terminal
  pane; prefer an OS notification from the daemon for ambient awareness (D-037,
  Claude Code is launched and used unchanged).
- **As few centralized components as possible, and each one explicable** (D-115).
  A compromise to privacy or decentralization must trace to a recorded decision, be
  disclosed to the person affected, and benefit them rather than us. Failing a test
  does not forbid the thing; it names what is missing. Three exist: the relay
  (forced by NAT, two of three), the model provider (inherent, two of three), the
  release host (pre-GA, one of three, retired when a public one exists).
- **No network provider is required** (D-019). None belongs in room identity,
  membership or replication. Tailscale is one provider, never a prerequisite.
- **No join tokens, and no join-by-name on a trusted network** (D-024, D-026).
  Nothing a person can hold admits them, and names are guessable by design.
- **Migrate, do not orphan.** `CREATE TABLE IF NOT EXISTS` ignores an existing
  table, so a column added later is missing from every older room and fails at first
  query rather than at open. `migrate()` runs on open; adding it there is the job.
- **No CRDT** until testing proves it necessary (§11).
- A command prefix names its target: `peer-` somebody else, `room-` a room, `self-`
  you (D-095, D-096). **Claude Code prefixes every command with the plugin manifest
  name and offers no unprefixed form** (D-118), so a person types
  `/cogmer:room-create`. The manifest `name` is therefore not cosmetic: changing it
  renames every command at once. The `room-`/`peer-`/`self-` prefixes are kept on
  top of it because they name a target, not to avoid collisions — the namespace
  does that now.

## Boundaries

Violating one makes an otherwise reasonable change wrong, and none is obvious from
reading a few files.

- **First, do no harm** (§3.1) — the duty every other requirement is subordinate
  to. A session must be no worse for having installed this: not broken, not
  slowed, not noisy, not robbed of context budget, not made to take a turn. Every
  hook failure path exits 0 with empty stdout, and a dead daemon means no
  collaboration, never a broken session. It is a general duty, so an unlisted way
  to degrade a session is forbidden as well.
- **A remote event never drives an interactive session** (§3.7). It may be read at a
  turn the person started; it must never be the reason a turn ran. `sync.go`,
  `daemon.go`, `store.go` and `transcript.go` must not import `os/exec` or
  `syscall`, or call `RunProbe`/`EnsureVerified`/`runDoctor` — a test enforces it.
  An MCP server must not expose sampling, for the same reason.
- **Room content is untrusted.** It comes from peers. `ui.html` escapes before
  applying markup and a test asserts that order; the injected block carries a
  per-injection fence, strips that fence from content, and restates its framing
  *after* the turns (D-040). Frame by classification, never by asserting authority.
  Never a static delimiter.
- **The local API is reachable from this machine's browser**; loopback is not a
  boundary (D-087). State-changing routes require `X-Cogmer: 1` — require, not
  refuse, which is what makes it fail closed. `Origin` is a second layer. Never
  `Referer`. Reads stay unguarded, because CORS already withholds them.
- **The private key lives in `identity.key` and never in `identity.json`**, which
  `whoami` prints. A test asserts it never marshals.
- **Verification gates synchronization** (D-054). Authentication, admission and
  verification answer three different questions and must not be collapsed — every
  one but the last passes equally well for a key substituted in transit. An
  unverified peer's events are refused at their **origin**, so a verified relay
  cannot launder an unverified author, and are **held, never dropped**.
- **Events are immutable** (§7). Never rewrite `eventId`, `peerId` or
  `peerSequence`; transitive relay depends on it. Signature schemes are added,
  never edited (D-058) — an event cannot be re-signed and refetching history is a
  recovery path, so add `signingBytesV4` beside `signingBytesV3` and bump
  `currentSigVersion`.
- **Durability precedes publication** (§23), and **reserve a sequence before
  publishing the event that uses it** (D-029). The reverse publishes a number with
  no record of it.
- **Never publish `thinking` blocks**, exclude `isSidechain` records (§3.5), and do
  not summarize conversation (§3.4) — store the actual text. Injected context is
  attributed, never disguised as local (§20).
- **Key on `roomId`, never on `roomName`** — names collide by design (D-017).
  Nothing derives a room from a directory, repository or project. A session's room
  is fixed at its first prompt and never changes (D-016), because injected context
  cannot be withdrawn from a context window.
- **`mergeTail` is not an append.** It detects the superset case rather than
  assuming it; if `last_assistant_message` ever widens to the whole turn, blind
  appending duplicates every pre-tool block and silently corrupts the room.
- **Delivery is confirmed by observation**, never marked at injection time (D-014).
  The daemon's reply can be lost, and committing then discards context permanently
  and silently.

## Facts that are not obvious from the code

- **Every Claude Code extension point delivers to the model, and nothing displays to
  a person** (D-033, D-036). A person sees only what the model then says. MCP as a
  display channel was tested exhaustively and surfaces nowhere anybody looks — do
  not re-attempt it. **Nothing checks this**, which is why it is written here.
- **Whether injected context survives a compaction is the summarizer's judgement**,
  not a format guarantee, so no assertion can cover it. Re-run Test B from
  `docs/phase0a-findings.md` when the model or the Claude Code version changes.
- **Claude Code behaviours this project relies on are undocumented, and several fail
  silently** — the room keeps accepting events while recording the wrong thing. If
  you find a new reliance, add one to `behaviors.go` with a negative test, because a
  check that cannot fail reads as protection. Write its `Reliance` field for someone
  debugging at 2am: what breaks, not what the behaviour is.
- **Assistant records carry no `promptId`** (B09), which is why turn segmentation is
  positional rather than keyed. B04, B05 and B09 would report *improvements* as well
  as breakage; watch for those, since they would let us simplify.
- **Nothing cryptographic depends on the product name** (D-069). Signing namespaces
  use `protocolNamespace`, which is arbitrary on purpose and must never change.

## Where to read before changing something

Not a summary of these — a map to them. If the area is not listed, the spec is.

| touching | read first |
|---|---|
| pairing, verification, the two words | D-055, D-088, D-093, D-047, §25 |
| the local HTTP API or the view | D-087, D-038, D-039, D-090, D-099 |
| capture, reassembly, injection | D-014, D-040, D-043, and B04/B05/B09 |
| rooms, membership, session binding | D-015, D-016, D-046, D-064, D-071, D-077 |
| identity, keys, admission | D-042, D-044, D-053, D-058, D-073, D-054 |
| addresses, transport, reachability | D-019, D-091, D-101, D-103, D-104 |
| another host | D-110, D-111, D-113, §3.8 |
| the name, a slash command, the plugin manifest | D-117, D-118, D-095, D-096, D-085 |
| sequences, recovery from local loss | D-029, D-060 |
| what is unfinished or undecided | `docs/open.md` |
