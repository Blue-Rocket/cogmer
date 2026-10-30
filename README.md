# cogmer

cogmer is public so that it can be tested. It is not ready for general use.

cogmer puts people working separately, on their own machines and their own Claude Code or Codex subscriptions, in one conversation. Your turns and theirs replicate directly between
your machines. No server holds the conversation, and no account exists anywhere.

Your session keeps working when your colleague's is offline. Their turns reach your
session at your next prompt, never before it, so nothing they do makes your session
take a turn.

On Windows, Claude Code and Codex run cogmer's hooks and commands through `bash`, which Git for
Windows provides, so install Git for Windows first. Without it the hooks do not run and cogmer
does nothing. Claude Cowork does not use `bash`.

## Install in Claude Code

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

## Install in Codex

Codex CLI and the desktop app use the same plugin. Add the repository marketplace:

```sh
codex plugin marketplace add Blue-Rocket/cogmer
codex plugin add cogmer@blue-rocket
```

In the desktop app, the Blue Rocket marketplace also appears in the plugin browser.
Review and trust the plugin's hooks before starting a new session. The CLI's `/hooks`
command shows hooks waiting for review. Installing a plugin does not trust its hooks.

Use the cogmer skill in your session, such as `$cogmer self-status` or
`$cogmer room-create`. In the desktop app, select the cogmer skill from the skill
picker and give it the same command. Room names and pairing strings work across
Claude Code and Codex, and both hosts can share the same daemon on one machine.

When testing an unpublished build, run `go build -o bin/cogmer ./cmd/cogmer`, set
`COGMER_BIN` to that binary's absolute path before launching Codex, and add this
checkout with `codex plugin marketplace add .`. The release download is available
only after that version has been published.

## Use it from Claude Cowork

Claude Cowork runs its session in a sandbox that cannot reach other machines, so cogmer runs on your
own computer and a small bridge connects the two. cogmer has to be installed on that computer first,
through Claude Code or Codex as above, since that fetches and verifies the program. Then, in a Claude
Code session, run `/cogmer:self-cowork`, or in a Codex session `$cogmer self-cowork`. It adds the
bridge to Claude Desktop's configuration, keeping a copy of the file as it was, and writes a plugin
for Cowork. Quit Claude Desktop completely and open it again, upload that plugin in Cowork under
Customize, then Plugins, and start a new Cowork session. Typing `/cogmer:self-status` there shows
whether it works.

Run `/cogmer:self-cowork` again if you rename your computer or move cogmer, and upload the new plugin.
Cowork's own limits apply: its hooks are Anthropic's to change, a Cowork turn counts as delivered when
its reply is complete, and `cogmer doctor --host cowork` checks what that rests on. A Cowork session
cannot tell the daemon about the tool operations in a turn, so its turns show none.

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
model: two words the model told you would prove nothing. §25 (security) explains why two words are enough, and why a mismatch
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

In Codex, use `$cogmer room-create`, `$cogmer room-invite David`, and
`$cogmer room-join misty-canyon` for the same steps.

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
machine with `CGO_ENABLED=0` and needs nothing else at run time. A colleague may have no Node installed, so cogmer requires no JavaScript runtime.

### Where to read

| | |
|---|---|
| [Specification](Shared%20Claude%20Sessions.md) | the specification: the system as a user experiences it |
| [`docs/decisions/`](docs/decisions/) | the detail beneath the specification, and why; read it before proposing a simplification, because some awkward-looking choices are there for a reason |
| [`docs/writing.md`](docs/writing.md) | how every document here is written |
| [`docs/open.md`](docs/open.md) | actions not yet taken, and decisions not yet made |
| [`docs/relied-on-behaviors.md`](docs/relied-on-behaviors.md) | what the supported hosts do that cogmer depends on, generated from the registry |

### Behavior checks

cogmer checks the host behaviors it depends on: how hooks identify a turn, what the
transcript contains, and whether injected context reaches the model. Claude Code
checks also cover compaction and whether a background daemon can open a window.
Host behavior can change, and several failures are silent: the room keeps accepting events while it
records the wrong thing.

```sh
bin/cogmer doctor          # each installed host: two Claude turns (about 18s), one Codex turn
bin/cogmer doctor --host claude   # one host only; --host codex likewise
bin/cogmer doctor --deep   # also drives a real Claude compaction, about 40s
```

The session checks run by themselves the first time a room is formed under a host
version this machine has not checked, and the result is kept for that version,
so forming another room costs nothing. A failed check never blocks the room. It names
the behavior that changed, and each behavior's `Reliance` says what breaks as a
result.

Apache-2.0. Copyright 2026 Blue Rocket, Inc.
