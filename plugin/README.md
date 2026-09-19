# claude-team

Work in a shared Claude Code conversation with a colleague. Your turns and theirs
replicate directly between your machines — local-first, peer-to-peer, no server in
between and no account anywhere.

## What it installs

Three hooks and six slash commands. Nothing about how Claude Code starts changes,
and nothing here replaces or wraps it.

| | |
|---|---|
| `SessionStart` | starts the daemon if it is not already running |
| `UserPromptSubmit` | captures your prompt; injects teammate turns you have not seen |
| `Stop` | captures the completed response |

| Command | |
|---|---|
| `/room-create` | create a room and put this session in it |
| `/room-join <invitation>` | join a room you were invited to |
| `/room-invite <peer>` | admit a peer you have paired with |
| `/room-status` | which room this session is in, and who may enter |
| `/room-log` | the room's conversation so far |
| `/room-pair` | your pairing string, and how to pair (done in a terminal) |

Commands are prefixed because a slash command's **invocation** is not namespaced by
the plugin that supplies it — a subdirectory changes how a command is displayed,
not what you type. Two plugins in the official marketplace already both define
`/help`. The prefix is the only thing that keeps these apart.

## The binary

The hooks call a `claude-team` binary, looked for in this order: `$CLAUDE_TEAM_BIN`,
`~/.claude-team/bin/claude-team`, this plugin's `bin/`, then `PATH`. If none is
found, every hook exits silently and Claude Code is unaffected — a missing binary
means no collaboration, never a broken session.

## Two things are deliberately not commands here

**Pairing and verification happen in a terminal.** Both are interactive, both block
on another person, and the two words you compare must reach your eyes without
passing through a model that is reading room content from unverified peers.
`/room-pair` prints your pairing string and tells you what to run; it does not
attempt the ceremony.

**Nothing synchronizes with an unverified peer.** Admission says a key may enter;
verification says the key is your colleague's. Every check but the last passes
equally well for a key substituted in transit, so both are required. A room with an
unverified member looks quiet rather than broken, which is why the commands say so.
