# cogmer

This is the reference for the cogmer plugin once it is installed. What cogmer is,
and how to install it, are in the repository's `README.md`.

## What it adds to a session

The plugin adds three hooks and the commands below. It changes nothing about how
Claude Code or Codex starts, and it replaces or wraps nothing.

| hook | what it does |
|---|---|
| `SessionStart` | starts the daemon if it is not running, replaces a running daemon of another version, and fetches the binary in the background if it has not been fetched |
| `UserPromptSubmit` | captures your prompt, and injects the colleagues' turns this session has not seen |
| `Stop` | captures the completed response |

## The first room under a new version

The first time you form a room under a Claude Code version this machine has not checked, cogmer
runs two isolated Claude turns, about 13 seconds, to check the behaviors it relies on. That happens
once for each version, and `COGMER_PREFLIGHT=off` skips it. If the check cannot run, cogmer says so
and tries again after an hour.

## Commands in Codex

Invoke the cogmer skill with a command name and its arguments: `$cogmer self-status`,
`$cogmer peer-pair <string> David`, or `$cogmer room-create`. In the desktop app,
select the cogmer skill from the skill picker. The names and arguments in the table
below apply to both hosts; Codex uses the skill in place of the `/cogmer:` prefix.

The skill runs only when invoked. It obtains this session's identity from the tool
process environment, so running a command from a terminal cannot choose a session
on your behalf. Review the hooks when Codex says they need trust; until trusted,
no prompts or responses are captured and no colleague's turns reach the model.

The first room command on a Codex version runs one isolated behavior-check turn.
Run `cogmer doctor --host codex` to repeat it. This checks the local Codex runtime, not
another machine's installation. Set `COGMER_CODEX_BIN` to an absolute executable
path to check a different runtime, such as the desktop app's bundled Codex binary.
Codex compaction checks are not available.

## Command reference

In Claude Code, invoke a command with `/cogmer:` before its name. In Codex, invoke
the cogmer skill with `$cogmer` followed by the name and arguments.

Claude Code puts a plugin's name in front of every command the plugin provides, so
its commands start with `cogmer:`. In both hosts, `room-` commands act on one room,
`peer-` commands act on this machine's relationships with other peers, which outlast
every room, and `self-` commands act on you.

| command | what it does |
|---|---|
| `room-create [--show-only]` | create a room and put this session in it; with `--show-only` none of it is given to your agent, and that cannot change for the room |
| `room-join <invitation> [--show-only]` | join a room you were invited to, optionally show-only |
| `room-leave` | take this session out of its room; it may rejoin the same room later |
| `room-invite <peer>` | admit a peer you have paired with to this room |
| `room-revoke <peer>` | withdraw a peer's admission to this room |
| `room-status` | the room this session is in, and who is in it |
| `room-log` | the room's conversation so far |
| `room-conflicts` | events set aside because a peer's sequence counter went backwards |
| `room-list` | the rooms on this machine, and any waiting for you to accept |
| `peer-list` | the peers this machine knows, and whether each is verified |
| `peer-forget <peer>` | discard a peer and every room admission it held |
| `peer-pair [their pairing string] [what you call them]` | pair with a colleague by comparing two words in your browser |
| `self-status` | who you are, what you send colleagues, whether they can reach you, and which versions are running |
| `self-name [what people should call you]` | show or set the name other people see for you |
| `self-stop` | stop the daemon on this machine; nothing is captured or shared until a new session starts one |
| `self-restart` | stop the daemon and start it again, without a new session |
| `self-cowork` | add the bridge to Claude Desktop and write the plugin to upload in Cowork |

## Pairing happens in your browser, not through the model

The `peer-pair` command does what a command can do, then opens a page the daemon serves.
On that page you and your colleague compare two words, on a call. The words reach you
without passing through the model, because the model reads room content from peers
that may not be verified, and two words it relayed would prove nothing. On a machine
that cannot open a browser, the comparison happens at the terminal instead.

## Nothing is exchanged with an unverified peer

Admitting a peer to a room says that its key may enter. Verifying the peer, by
comparing the two words, says that the key is your colleague's. A key substituted in
transit passes every check except the comparison, so a room exchanges turns only with
peers that are both admitted and verified. A room whose only other member is
unverified looks quiet rather than broken, and `room-status` marks that
member as unverified.

## Where the hooks look for the binary

The hooks run a `cogmer` binary, and look for it in this order: `$COGMER_BIN`, then
`~/.cogmer/bin/cogmer` (under `$COGMER_HOME` instead, if that is set), then the
plugin's own `bin/`, then `PATH`. If none is found, the prompt and response hooks
exit without output, and the session carries on as if the plugin were not installed:
you lose collaboration, and nothing else. The session-start hook tells the model why
there is no binary yet, such as a download still running or one that failed, so that
it can say so when a command does not work.

## What leaves your machine

- Your conversation goes to the colleagues in a room. Each turn reaches their machines and
  their agent sessions, and so their model provider under their account. Their turns reach your
  provider, under yours.
- An image you paste into a prompt goes to the colleagues in the room, with whatever it shows. An image
  over 5 MB is not sent, and a colleague's view says so. Each colleague's machine keeps a copy as a file in
  `~/.cogmer`, and cogmer gives their model the turn's text and not the image.
- A relay introduces two machines on different networks. cogmer uses the free relays that
  Tailscale runs for its tailcat library. The conversation is encrypted between the two machines
  and the relay cannot read it. The daemon holds a connection to the relay for as long as it
  runs, so the relay sees this machine's network address and key whenever the daemon is up,
  and which machines connect to it, when, and how much passes between them. `COGMER_TAILCAT=off`
  stops the daemon using the relay, and colleagues then reach you only at the address in
  `COGMER_PEER_ENDPOINT`.

## Updating

Manage plugin updates through the host where you installed cogmer. The marketplace
commands and desktop controls below apply to Claude Code. Codex users manage cogmer
through Codex’s plugin controls.

Claude Code learns that a new version exists only when it refreshes its copy of the
`blue-rocket` marketplace, and for a marketplace from outside Anthropic it does that
on its own only once auto-update is on. Until then, the plugin's update button stays
grayed out.

In the Claude Code desktop app, open Manage plugins, choose Manage marketplaces from the menu on
the Add button at the top right, and choose Check for updates from the `blue-rocket`
marketplace's menu. Then update cogmer from Manage plugins. In a terminal, the same
two steps are `claude plugin marketplace update blue-rocket` and
`claude plugin update cogmer@blue-rocket`. Asking Claude in a session to update the
cogmer plugin also works, because it can run those two commands, provided `claude`
is installed as a command on the machine.

To have new versions arrive by themselves, turn on auto-update for the marketplace.
The Claude Code desktop app has no switch for it. In a terminal, run `claude`, then `/plugin`,
open the Marketplaces tab, select `blue-rocket` and turn on auto-update. This is done
once on each machine.

The next session after an update fetches the new binary in the background and
replaces the daemon that is running, so there is nothing to stop or restart.

If cogmer is installed in two places on one machine, such as with the terminal commands above and from the
desktop app's Plugins page, the newer plugin's program stays whichever one starts a session, and
`/cogmer:self-status` says when a plugin is older than the program. Keep one copy, and remove the older one
if the status says so.
