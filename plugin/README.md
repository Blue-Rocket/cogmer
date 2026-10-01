# cogmer

This is the reference for the cogmer plugin once it is installed. What cogmer is,
and how to install it, are in the repository's `README.md`.

## What it adds to a session

The plugin adds three hooks and the commands below. It changes nothing about how
Claude Code starts, and it replaces or wraps nothing.

| hook | what it does |
|---|---|
| `SessionStart` | starts the daemon if it is not running, replaces a running daemon of another version, and fetches the binary in the background if it has not been fetched |
| `UserPromptSubmit` | captures your prompt, and injects the colleagues' turns this session has not seen |
| `Stop` | captures the completed response |

## Commands

Claude Code puts a plugin's name in front of every command the plugin provides, so
each command starts with `cogmer:`. After that, `room-` commands act on one room,
`peer-` commands act on this machine's relationships with other peers, which outlast
every room, and `self-` commands act on you.

| command | what it does |
|---|---|
| `/cogmer:room-create` | create a room and put this session in it |
| `/cogmer:room-join <invitation>` | join a room you were invited to |
| `/cogmer:room-leave` | take this session out of its room; it may rejoin the same room later |
| `/cogmer:room-invite <peer>` | admit a peer you have paired with to this room |
| `/cogmer:room-revoke <peer>` | withdraw a peer's admission to this room |
| `/cogmer:room-status` | the room this session is in, and who is in it |
| `/cogmer:room-log` | the room's conversation so far |
| `/cogmer:room-conflicts` | events set aside because a peer's sequence counter went backwards |
| `/cogmer:room-list` | the rooms on this machine, and any waiting for you to accept |
| `/cogmer:peer-list` | the peers this machine knows, and whether each is verified |
| `/cogmer:peer-forget <peer>` | discard a peer and every room admission it held |
| `/cogmer:peer-pair [their pairing string] [what you call them]` | pair with a colleague by comparing two words in your browser |
| `/cogmer:self-status` | who you are, what you send colleagues, and whether they can reach you |
| `/cogmer:self-name [what people should call you]` | show or set the name other people see for you |

## Pairing happens in your browser, not through the model

`/cogmer:peer-pair` does what a command can do, then opens a page the daemon serves.
On that page you and your colleague compare two words, on a call. The words reach you
without passing through the model, because the model reads room content from peers
that may not be verified, and two words it relayed would prove nothing. On a machine
that cannot open a browser, the comparison happens at the terminal instead.

## Nothing is exchanged with an unverified peer

Admitting a peer to a room says that its key may enter. Verifying the peer, by
comparing the two words, says that the key is your colleague's. A key substituted in
transit passes every check except the comparison, so a room exchanges turns only with
peers that are both admitted and verified. A room whose only other member is
unverified looks quiet rather than broken, and `/cogmer:room-status` marks that
member as unverified.

## Where the hooks look for the binary

The hooks run a `cogmer` binary, and look for it in this order: `$COGMER_BIN`, then
`~/.cogmer/bin/cogmer` (under `$COGMER_HOME` instead, if that is set), then the
plugin's own `bin/`, then `PATH`. If none is found, the prompt and response hooks
exit without output, and the session carries on as if the plugin were not installed:
you lose collaboration, and nothing else. The session-start hook tells the model why
there is no binary yet, such as a download still running or one that failed, so that
it can say so when a command does not work.

## Updating

Claude Code learns that a new version exists only when it refreshes its copy of the
`blue-rocket` marketplace, and for a marketplace from outside Anthropic it does that
on its own only once auto-update is on. Until then, the plugin's update button stays
greyed out.

In the desktop app, open Manage plugins, choose Manage marketplaces from the menu on
the Add button at the top right, and choose Check for updates from the `blue-rocket`
marketplace's menu. Then update cogmer from Manage plugins. In a terminal, the same
two steps are `claude plugin marketplace update blue-rocket` and
`claude plugin update cogmer@blue-rocket`. Asking Claude in a session to update the
cogmer plugin also works, because it can run those two commands, provided `claude`
is installed as a command on the machine.

To have new versions arrive by themselves, turn on auto-update for the marketplace.
The desktop app has no switch for it. In a terminal, run `claude`, then `/plugin`,
open the Marketplaces tab, select `blue-rocket` and turn on auto-update. This is done
once on each machine.

The next session after an update fetches the new binary in the background and
replaces the daemon that is running, so there is nothing to stop or restart.
