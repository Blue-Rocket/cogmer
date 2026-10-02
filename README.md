# cogmer

cogmer is public so that it can be tested. It is not ready for general use.

cogmer puts people working separately, on their own machines and their own Claude
Code subscriptions, in one conversation. Your turns and theirs replicate directly between
your machines. No server holds the conversation, and no account exists anywhere.

Your session keeps working when your colleague's is offline. Their turns reach your
session at your next prompt, never before it, so nothing they do makes your session
take a turn.

## Install

Run these two commands inside a Claude Code session, one at a time. Pasted together,
they arrive as a single command, and the first one fails.

```
/plugin marketplace add Blue-Rocket/cogmer
```

```
/plugin install cogmer@blue-rocket
```

The first `/cogmer:` command you run starts the download and tells you to run it again
in about 15 to 20 seconds, so the session you installed from needs no restart. A session
started afterwards fetches the binary for your platform with a hook and starts a daemon on
your machine. The binary is about 30MB, and the download took 13 to 16 seconds when
measured on 2026-09-22.
Installing changes nothing about how Claude Code starts, edits no shell profile, and
registers no service with the operating system.

Each binary is authorized by a SHA-256 hash pinned in the plugin. A download whose
hash does not match is deleted without being run. If there is no verified download
and the machine has Go installed, the binary is built from source instead.

## Pair, once, on a call

Send your colleague your pairing string, which this command prints:

```
/cogmer:self-status
```

The string is a public key and an address. Holding it admits nobody, so you can send
it any way you like. If the command says nobody can reach your address, a colleague
given the string cannot finish pairing with you.

Then, both at the same time, each run this with the other's string:

```
/cogmer:peer-pair ed25519:GoR7…IWPU@203.0.113.9:4783 David
```

A page opens in each of your browsers showing two words, and you read them to each
other on the call. That comparison is what shows the key each of you received is the
other's, and not one substituted on the way, so the words never pass through the
model: two words Claude told you would prove nothing. §25 (security) explains why two words are enough, and why a mismatch
must never be retried. On a machine with no browser, such as over SSH or in a
container, the comparison happens in the terminal instead.

Nothing is exchanged with a colleague until both of you confirm the words matched. A
room whose other member is unverified looks quiet rather than broken.

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

Then work as you normally would. A session that has been in one room can never join
a second, so start a new session to join a different room.

The room is also a page in your browser, at `http://127.0.0.1:4782`, which updates as
turns arrive. It shows who said what, and when. Each colleague is labeled with a name
derived from their key as well as the name they chose, so two people who choose the
same name still read as two. The page is served by the binary and makes no requests
to anywhere else.

[`plugin/README.md`](plugin/README.md) is the reference for every command.

## What it does not do

It does not find anybody on your local network. Pairing needs a string exchanged by
some other means, because nothing a stranger could obtain admits them to a room.

It does not let somebody you have not invited ask to join a room.

It does not summarize. A colleague's turns are stored and injected as the text that
was actually written, attributed to them, never presented as your own and never
compressed.

## Working on it

```sh
go build -o bin/cogmer ./cmd/cogmer
bin/cogmer daemon                  # hooks and UI on :4782, peer sync on :4783
go test ./...
```

A plugin install puts the binary in `~/.cogmer/bin`, which is not on `PATH`, so
`cogmer` at a shell finds only a binary you built and installed yourself. Every
command the binary prints for you to type carries its full path.

`COGMER_ADDR` sets the local address, `COGMER_PEER_ADDR` the peer address, and
`COGMER_PEERS` extra peer addresses. `COGMER_PREFLIGHT=off` skips the behavior checks
when a room is formed. `COGMER_MAX_EVENTS`, `COGMER_MAX_EVENT_CHARS` and
`COGMER_MAX_BLOCK_CHARS` bound the context injected into a session (§21, context
window management).

The code is Go, with a SQLite library written in Go, so every target builds from one
machine with `CGO_ENABLED=0` and needs nothing else at run time. Claude Code ships as
a native binary, so a colleague may have no Node installed.

### Where to read

| | |
|---|---|
| [`Shared Claude Sessions.md`](Shared%20Claude%20Sessions.md) | the specification: the system as a user experiences it |
| [`docs/decisions/`](docs/decisions/) | the detail beneath the specification, and why; read it before proposing a simplification, because some awkward-looking choices are there for a reason |
| [`docs/writing.md`](docs/writing.md) | how every document here is written |
| [`docs/open.md`](docs/open.md) | actions not yet taken, and decisions not yet made |
| [`docs/relied-on-behaviors.md`](docs/relied-on-behaviors.md) | what Claude Code does that cogmer depends on, generated from the registry |

### Behavior checks

cogmer depends on Claude Code behaviors that are not documented: how a hook reports a
turn, what the transcript contains, what compaction keeps, and whether a daemon
running in the background can still open a window in front of a person. None of them
is promised, and several fail silently: the room keeps accepting events while it
records the wrong thing.

```sh
bin/cogmer doctor          # two Claude turns, about 13s
bin/cogmer doctor --deep   # also drives a real compaction, about 40s
```

The session checks run by themselves the first time a room is formed under a Claude
Code version this machine has not checked, and the result is kept for that version,
so forming another room costs nothing. A failed check never blocks the room. It names
the behavior that changed, and each behavior's `Reliance` says what breaks as a
result.

Apache-2.0. Copyright 2026 Blue Rocket, Inc.
