# cogmer

A peer-to-peer daemon that replicates one Claude Code conversation between people
working separately, each on their own subscription, with no central server.

This file holds how we work, what this project has chosen, what it must not break,
and how to run it.

## How we work

- **Read `docs/decisions.md` before proposing a change to how anything here works.**
  It records why each choice was made, and the alternatives somebody would be likely
  to propose again, and several choices that look awkward have something depending
  on them. Add an entry whenever a real choice about the system is made. Never
  renumber: a reversed decision becomes a tombstone, and the reason for reversing it
  goes in the replacing decision's Rejected field.
- **Read `docs/writing.md` before writing any document or commit message**, and
  follow it. It holds the rules and a template for each kind of item.
- **Read the code before characterising it.** A grep result shows the line that
  mentions something, not what the function does, so read the function.
- **Check `git log` before concluding something was removed.** `git log -S` across
  all commits answers "was this ever written?", which is a different question from
  "is it here now?".
- **Cite nothing you have not confirmed exists.** `TestCitationsNameThingsThatExist`
  fails for a citation that resolves to nothing, but not for one that resolves to an
  entry holding something else, so check what the entry says.
- **Corrections are clean replacements.** Leave no superseded text, and do not narrate
  what the old text said.
- **Give every `§`, `D-NNN` and `B-NN` a few words**, such as "D-054 (verification
  gates sync)" rather than "D-054", in conversation as well as in files.
- **Use the words for people that `docs/writing.md` fixes (W-01).** Say user for
  someone using cogmer, colleague for another user in their room, and person only for
  a human as distinct from the model or a program. Never say developer, since a user
  may not write code.
- **Never describe cogmer by a number of participants**, such as a tool for two.
  Pairs are the case that exists, and nothing rules out more (D-109).
- **Never call the project local-first.** It claims more than cogmer delivers, since
  rendezvous needs a relay somebody else operates, and D-029 (losing a room database)
  makes recovery a fetch from peers. §3.1 (first, do no harm) is the principle it was
  standing in for.

The project's values and the patterns it has selected are imported below this file's
sections, so every session reads them whole.

**Where a thing gets written down.** Putting an answer in the wrong document is how
documents rot.

| document | answers |
|---|---|
| `Shared Claude Sessions.md` | the system as the user experiences it, the constraints on the system's boundaries, and the negative outcomes it avoids (W-84) |
| `docs/values.md` | what this project holds itself to |
| `docs/patterns.md` | general patterns selected because they suit this project |
| `docs/decisions.md` | smaller-grained detail too mundane or technical for the spec, necessary exceptions to it, and why (W-84) |
| `docs/writing.md` | how a document is written, where a thing goes, and how the documents are checked |
| a commit, cited as `<commit>:<path>` or `<commit>` (W-42 in `docs/writing.md`) | what we observed when we tried it, where no test can hold the evidence |
| `cmd/cogmer/behaviors.go` | what someone else's software does that we rely on |
| `docs/open.md` | actions not yet taken, decisions not yet made |
| `docs/work/` | material for one open item, deleted with it; nothing durable cites it |
| `docs/explanations/` | how something works, walked through for a reader who has forgotten; cites nothing, and no other writing rule applies |
| `README.md` | what somebody should know to use cogmer or work on it, before installing it (W-75) |
| `plugin/README.md` | what somebody who has installed it needs |

- **Evidence never goes in the spec.** Put the requirement it justifies there.
- **What someone else's software does goes in the behaviour registry**, which tests
  itself, and what our own code does is a test. Neither is retold anywhere else.
- **An open item is deleted when it is done, never marked done.** `open.md` groups
  items by milestone, and nothing records a milestone once its last item is gone
  (W-53).
- **A change to what the user experiences changes the spec in the same commit**
  (W-85).
- **`docs/decisions.md` holds only decisions about the system.** How documents are
  written, where a thing goes and how they are checked are rules in `docs/writing.md`
  (W-28).

## The local physics

```
cmd/cogmer/
  main.go        subcommands + hook client (fails open)
  daemon.go      localhost HTTP, §20 context formatting, §21 limits
  store.go       SQLite event store, §19 delivery state
  transcript.go  turn reassembly; read the comments before editing
  identity.go    §6 peer identity
```

State is in `~/.cogmer/`: `identity.json`, `identity.key`, `membership.db` and
`rooms/*.db`. The code is pure Go throughout, with `modernc.org/sqlite` and no cgo, so
every target cross-compiles with `CGO_ENABLED=0`.

- **Build with `go build -o bin/cogmer ./cmd/cogmer`.** `bin/` is untracked, because
  the binary is 17MB and the build reproduces it.
- **Test with `go test ./...`.** To drive a real session, run `claude -p …
  --settings <file>` with a throwaway `COGMER_HOME` per run, and `< /dev/null`, or it
  waits on stdin.
- **`cogmer doctor` checks the behaviours we rely on**, in about 5s and one Claude
  turn. `--deep` adds a compaction, in about 40s. It runs by itself when a room is
  formed under a Claude Code version this machine has not checked, and
  `COGMER_PREFLIGHT=off` turns that off for CI.
- **`docs/relied-on-behaviors.md` is generated** by `cogmer behaviors --markdown`.
  Edit `behaviors.go`, never the document.

**This repository has a codegraph index, and reaching for grep instead is the default
failure.** grep is the reflex for every lookup, and wins by inertia unless the split
is stated:

- **Use codegraph for symbols, callers, callees and impact:** `codegraph_search`,
  `codegraph_callers` and `codegraph_impact`. `grep -rn "funcName"` is the wrong tool
  for a call graph, and running `codegraph_impact` before changing a signature is what
  the index is for.
- **Use grep for text:** SQL inside a string literal, a citation like `§3.1`, or
  anything in a `.md`. The graph holds the Go sources and knows nothing about prose,
  which is most of this repository.

**A stale index answers confidently.** `codegraph_search` on a function added an hour
ago returns "No results found", which reads as "that does not exist" rather than "this
index is old", a failure grep cannot have. `.codegraph/` is gitignored and belongs to
one machine. `codegraph sync` takes about 0.2s, and `codegraph status` says what the
index holds. Hooks keep it fresh, but only for edits the current session made.

**Releasing is three steps in one order.** Bump `plugin/VERSION`, run
`scripts/release.sh <that same version>`, then run `scripts/publish.sh`.

- **`release.sh` builds five targets and rewrites every file that names the
  version:** `plugin/checksums.txt`, computed from the bytes it just built,
  `plugin/VERSION`, and the `version` in `plugin.json`. `checksums.txt` is the only
  thing that authorises a downloaded binary to run, so never edit it by hand: a hash
  typed rather than computed authorises something nobody has seen.
- **The manifest `version` is what pins an installed plugin** (D-120). A user receives
  a new plugin, and with it the checksums that authorise the new binary, only when
  that string moves. A test fails if it and `plugin/VERSION` disagree.
- **Publishing without rerunning `release.sh` leaves the installer refusing the new
  asset.** That is the correct failure, not a bug to work around.
- **`publish.sh` reads `plugin/VERSION`, requires `dist/`**, and ships to the host
  named in `plugin/release-url.txt`.

## Facts that are not obvious from the code

- **Claude Code prefixes every plugin command with the plugin manifest's `name`, and
  offers no unprefixed form** (D-118). A user types `/cogmer:room-create`, so changing
  the manifest name renames every command at once.
- **Every Claude Code extension point delivers to the model, and nothing displays to a
  person** (D-033, D-036). A person sees only what the model then says. MCP as a
  display channel was tested exhaustively and surfaces nowhere anybody looks, so do
  not try it again. Nothing checks this, which is why it is written here.
- **Whether injected context survives a compaction is the summarizer's judgement**,
  not a format guarantee, so no assertion can cover it. Re-run Test B from
  `84a0751:docs/phase0a-findings.md` when the model or the Claude Code version
  changes.
- **The Claude Code behaviours this project relies on are undocumented, and several
  fail silently:** the room keeps accepting events while recording the wrong thing. If
  you find a new reliance, add it to `behaviors.go` with a negative test, because a
  check that cannot fail reads as protection. Write its `Reliance` for someone
  debugging at 2am: what breaks, not what the behaviour is.
- **Assistant records carry no `promptId`** (B09), which is why turns are segmented by
  position rather than by key. B04, B05 and B09 would report improvements as well as
  breakage, so watch for those, since they would let us simplify.
- **Nothing cryptographic depends on the product name** (D-069). Signing uses the
  `protocolNamespace` constant, which is arbitrary on purpose and never changes.

## Where to read before changing something

This table maps an area to what to read first, and summarises none of it. If the area
is not listed, read the spec.

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

@docs/values.md

@docs/patterns.md
