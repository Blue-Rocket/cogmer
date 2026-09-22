# cogmer

Work in a shared Claude Code conversation with a colleague. Your turns and theirs
replicate directly between your machines — peer-to-peer, no server in between and
no account anywhere. Claude Code talks only to a daemon on your own machine, and
your session keeps working when the other one is offline.

## What it installs

Hooks and slash commands, listed below. Nothing about how Claude Code starts changes,
and nothing here replaces or wraps it.

| | |
|---|---|
| `SessionStart` | starts the daemon if it is not already running |
| `UserPromptSubmit` | captures your prompt; injects teammate turns you have not seen |
| `Stop` | captures the completed response |

| Command | |
|---|---|
| `/cogmer:room-create` | create a room and put this session in it |
| `/cogmer:room-join <invitation>` | join a room you were invited to |
| `/cogmer:room-leave` | take this session out of its room; it may rejoin |
| `/cogmer:room-invite <peer>` | admit a peer you have paired with |
| `/cogmer:room-revoke <peer>` | withdraw a peer's admission to this room |
| `/cogmer:room-status` | which room this session is in, and who may enter |
| `/cogmer:room-log` | the room's conversation so far |
| `/cogmer:room-conflicts` | quarantined events, if a peer's sequence went backwards |
| `/cogmer:room-list` | rooms this machine holds, and which one this session is in |
| `/cogmer:peer-list` | peers this machine knows, and whether each is verified |
| `/cogmer:peer-forget <peer>` | discard a peer and every admission it held |
| `/cogmer:peer-pair [string] [name]` | pair with a colleague: the two-word check, in your browser |
| `/cogmer:self-status` | who you are, what you send colleagues, whether they can reach you |
| `/cogmer:self-name [name]` | show or set the name other people see |

Three prefixes because there are three targets: `room-` acts on one room, `peer-` on
this machine's relationships, which outlast every room, and `self-` on you.

The `cogmer:` in front of all of them is not ours. Claude Code namespaces a command
by the plugin manifest's name and offers no unprefixed form, so `cogmer:` is what you
type whatever we would have preferred. The `room-`/`peer-`/`self-` prefixes are kept
on top of it because they name a target, not because anything would collide.

## The binary

The hooks call a `cogmer` binary, looked for in this order: `$COGMER_BIN`,
`~/.cogmer/bin/cogmer`, this plugin's `bin/`, then `PATH`. If none is
found, every hook exits silently and Claude Code is unaffected — a missing binary
means no collaboration, never a broken session.

## Two things the model does not do for you

**The pairing ceremony does not pass through the model.** It is interactive, it
blocks on another person, and the two words you compare must reach your eyes without
passing through a model that is reading room content from unverified peers. So
`/cogmer:peer-pair` does the part a command can do and then opens a page the daemon
serves, where the two words are; a machine that cannot open a browser falls back to a
terminal. Nothing about the comparison is reported by the model, and two words it
told you would mean nothing.

**Nothing synchronizes with an unverified peer.** Admission says a key may enter;
verification says the key is your colleague's. Every check but the last passes
equally well for a key substituted in transit, so both are required. A room with an
unverified member looks quiet rather than broken, which is why the commands say so.
