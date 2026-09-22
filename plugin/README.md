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
| `/cogmer:peer-list` | peers this machine knows, and whether each is verified |
| `/cogmer:peer-forget <peer>` | discard a peer and every admission it held |
| `/cogmer:peer-pair` | your pairing string, and how to pair (done in a terminal) |

Two prefixes because there are two scopes (§12): `room-` acts on one room,
`peer-` on this machine's relationships, which outlast every room.

Commands are prefixed at all because a slash command's **invocation** is not
namespaced by the plugin that supplies it — a subdirectory changes how a command is
displayed, not what you type. Two plugins in the official marketplace already both
define `/help`.

Commands are prefixed because a slash command's **invocation** is not namespaced by
the plugin that supplies it — a subdirectory changes how a command is displayed,
not what you type. Two plugins in the official marketplace already both define
`/help`. The prefix is the only thing that keeps these apart.

## The binary

The hooks call a `cogmer` binary, looked for in this order: `$COGMER_BIN`,
`~/.cogmer/bin/cogmer`, this plugin's `bin/`, then `PATH`. If none is
found, every hook exits silently and Claude Code is unaffected — a missing binary
means no collaboration, never a broken session.

## Two things are deliberately not commands here

**Pairing and verification happen in a terminal.** Both are interactive, both block
on another person, and the two words you compare must reach your eyes without
passing through a model that is reading room content from unverified peers.
`/cogmer:peer-pair` prints your pairing string and tells you what to run; it does not
attempt the ceremony.

**Nothing synchronizes with an unverified peer.** Admission says a key may enter;
verification says the key is your colleague's. Every check but the last passes
equally well for a key substituted in transit, so both are required. A room with an
unverified member looks quiet rather than broken, which is why the commands say so.
