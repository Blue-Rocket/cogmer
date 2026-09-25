# cogmer

Two people, working separately on their own machines and their own Claude Code
subscriptions, in one conversation. Your turns and theirs replicate directly between
you — no server holds the conversation, and no account exists anywhere.

Your session keeps working when the other one is offline. Their turns arrive at your
next prompt, never before it: nothing they do makes your session take a turn.

## Install

Two commands, inside a Claude Code session. Run them one at a time: pasted
together, they arrive as a single command and the first one fails.

```
/plugin marketplace add Blue-Rocket/cogmer
```

```
/plugin install cogmer@blue-rocket
```

Then start a new session. A hook fetches the binary for your platform when a session
starts and starts a daemon on your machine, so on a first install the session you installed from has
no binary, and its `/cogmer:` commands say so. The download is about 30MB and
usually takes seconds. Nothing about how Claude Code starts changes, no shell
profile is edited, and no service is registered with the operating system.

Each binary is authorised by a sha256 pinned in the plugin. A download that does not
match it is deleted rather than run, and a machine with a Go toolchain and no
matching asset builds from source instead.

## Pair, once, on a call

Send your colleague your pairing string:

```
/cogmer:self-status
```

It is a public key and an address. Holding it admits nobody, so send it any way you
like. If it says nobody can reach your address, that is the part that matters — a
colleague given that string cannot finish a pairing.

Then, both at once, each with the other's string:

```
/cogmer:peer-pair ed25519:GoR7…IWPU@203.0.113.9:4783 David
```

A page opens in each of your browsers showing two words. You read them to each other
on the call. That comparison is the whole of the security model, and it is why the
words never pass through the model: two words Claude told you would mean nothing.
§25 explains why two are enough and why a mismatch must never be retried. On a
machine with no browser — over SSH, in a container — the same ceremony happens in
the terminal.

Nothing synchronizes until both of you confirm the words matched. An unverified peer
is refused, so a room with one looks quiet rather than broken.

## Work in one room

One of you makes a room and admits the other:

```
/cogmer:room-create            → misty-canyon
/cogmer:room-invite David
```

The other accepts it by name:

```
/cogmer:room-join misty-canyon
```

Then work normally. A session's room is fixed at its first prompt and never changes,
so joining from a session that has already been in one is refused — start a new
session rather than looking for a way around it.

The room is also a page, live over server-sent events at `http://127.0.0.1:4782`:
who said what, when, with attribution anchored on each peer's derived name rather
than the display name it asserts. Served from the binary, with no assets, no build
step and no external requests.

[`plugin/README.md`](plugin/README.md) is the reference for every command.

## Whether it is worth it

Two independent Claude sessions synchronizing turns produced an exchange where the
second resolved a referent from the first, disagreed with it on the merits, and
found a defect the first had missed. The runs are recorded in
[`phase2-experiment.md`](https://github.com/Blue-Rocket/cogmer/blob/84a0751/docs/phase2-experiment.md), as of commit `84a0751`.
Repeated across two machines and two Claude Code versions: 0.44 s sync over the open
internet, identical event ordering.

That is the question this project exists to answer, and it is answered for two
sessions the same person was watching. Whether it holds when the second person is
somebody else is not yet known, because nobody but the author has ever been in a
room. That is the next thing to find out.

## What it does not do

- **Discover anybody on your local network.** Pairing needs a string exchanged out
  of band, by design: nothing a person can hold admits them to a room.
- **Approve a join request you did not invite.** Whether it should is undecided.
- **Work on Windows with confidence.** The binaries build and are published; hook
  shell semantics differ there and have not been re-probed.
- **Summarize anything.** Teammate turns are stored and injected as the text that was
  actually said, attributed, never disguised as local and never compressed.

## Working on it

```sh
go build -o bin/cogmer ./cmd/cogmer
bin/cogmer daemon                  # hooks and UI on :4782, peer sync on :4783
go test ./...
```

A plugin install puts the binary in `~/.cogmer/bin`, deliberately not on PATH, so
`cogmer` resolves at a shell only if you built it yourself. Every command the binary
prints for you to type carries its own full path; those are the ones to trust.

`COGMER_ADDR` sets the local address, `COGMER_PEER_ADDR` the peer address,
`COGMER_PEERS` extra peer addresses, `COGMER_PREFLIGHT=off` skips behavior checks on
new rooms, and `COGMER_MAX_EVENTS`, `COGMER_MAX_EVENT_CHARS` and
`COGMER_MAX_BLOCK_CHARS` bound injected context (§21).

Go with pure-Go SQLite throughout, so every target cross-compiles from one machine
with `CGO_ENABLED=0` and depends on nothing at runtime — which matters because
Claude Code ships as a native binary and a colleague may have no Node installed.

### Where to read

| | |
|---|---|
| [`Shared Claude Sessions.md`](Shared%20Claude%20Sessions.md) | the specification: the system as a user experiences it |
| [`docs/decisions.md`](docs/decisions.md) | why, and what was rejected — read before proposing a simplification, because some of the awkwardness is load-bearing |
| [`docs/open.md`](docs/open.md) | actions not yet taken, decisions not yet made |
| [`docs/phases.md`](docs/phases.md) | the phases the work was planned in, and how each ended |
| [`docs/relied-on-behaviors.md`](docs/relied-on-behaviors.md) | what Claude Code does that this depends on, generated from the registry |

Earlier runs the design rests on are kept in history as of commit `84a0751`: the
integration spike in [`phase0-findings.md`](https://github.com/Blue-Rocket/cogmer/blob/84a0751/docs/phase0-findings.md), the compaction
probe in [`phase0a-findings.md`](https://github.com/Blue-Rocket/cogmer/blob/84a0751/docs/phase0a-findings.md), and the partition between
two machines in [`phase5-findings.md`](https://github.com/Blue-Rocket/cogmer/blob/84a0751/docs/phase5-findings.md).

### Behavior checks

This depends on undocumented Claude Code behaviors — how a hook reports a turn, what
the transcript contains, what compaction preserves, whether a detached daemon can
still put a window in front of a person. None is contractual and several fail
silently: the room keeps accepting events while recording the wrong thing.

```sh
bin/cogmer doctor          # one Claude turn, ~5s
bin/cogmer doctor --deep   # adds compaction, drives a real one, ~40s
```

The session tier runs automatically the first time a room is formed on a Claude Code
version nobody has verified, cached on the version rather than the room, so forming a
tenth room costs nothing. A failure never blocks the room: it names the assumption
that changed, and each behavior's `Reliance` says what breaks as a result.

## License

Apache-2.0. Copyright 2026 Blue Rocket, Inc.
