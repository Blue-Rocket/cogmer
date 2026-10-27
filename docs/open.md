# Open

Actions not yet taken and decisions not yet made. Nothing here is durable: every item
either becomes a decision, a change to the specification, or code — and is then
**deleted**, not marked done. The record of why lives in `docs/decisions/` and the
record of what the system is lives in the specification, so an item that has earned a
permanent home does not need one here too. Nothing here records status.

Items are grouped by milestone: a set of related functionality, named in the sentence
that opens its section. A milestone's section is deleted with its last item, and
nothing records it afterwards. The sections are in the order the work is done, and those after "Somebody else uses it" wait for what that colleague does.

Being scratch is the point. Be untidy in it.

## Installing and the first commands

A new user installs the plugin, and the first commands they run answer instead of failing.

**Whether the download is slow enough to matter.** Deliberately not addressed until
people installing it say so. The darwin/arm64 binary is 29.3MB and took 16s on
09-22; release binaries are already stripped. Compressed with `gzip -9` it is
11.0MB, so compression is the obvious lever. It would add `gunzip` to what the
installer needs, which is confirmed only on macOS; minimal Linux images may lack
it, and Git Bash on Windows is unchecked. The version that adds no requirement
publishes both forms, pins both, and fetches the compressed one only where
`gunzip` exists, checking the binary it will actually run, not only the file it
downloaded. Shipping binaries inside the plugin removes the download entirely but
commits about 150MB per release into the history everybody clones, and is ruled
out.

## Before a wider audience

People the maintainer has never met install cogmer, on machines, networks and settings the
maintainer has not seen, and none of them is worse off for it.

**cogmer's Codex skill is loaded into Claude Code sessions, and gives the model a route to the
commands.** `plugin/skills/cogmer/SKILL.md` is written for Codex and only
`.codex-plugin/plugin.json` declares it, but Claude Code loads a plugin's `skills/` directory by
itself. On 2026-10-18 the model in a Claude Code cloud session followed it to `codex-cli.sh`,
which refused for want of a Codex session, then ran `cli.sh` through its shell. D-119 (no slash
command is reachable by the model) keeps the command files from the model, and the skill
reaches the same commands. Moving the skill to a directory only the Codex manifest names would
keep it out of Claude Code; whether Claude Code reads a skills field in its own manifest that
could exclude it is unchecked.

**Nothing checks that a program accepts the hooks and commands of an older plugin on the
same machine.** One program in `~/.cogmer/bin` serves every host, and the installer and the
daemon replace it only with a newer one (D-215, the installer and the daemon replace a program
only with a newer version), so a host whose plugin lags runs its older hook lines, handler
scripts and command files against a newer program, read from `common.sh` and `install.sh` on
2026-10-25. This happens whenever one host updates its plugin from a newer marketplace listing
than another host holds. A release that renames a subcommand, or changes what a hook sends,
makes the lagging host's hooks fail open: its sessions show no error and silently stop
capturing and receiving turns until its plugin updates, and its command files lack any new
command.

A test that runs the current program through the previous release's `hooks.json`,
`codex-hooks.json`, handler scripts and command files would catch it, with the previous
release's plugin directory read from git by its tag. A release that has to break that
interface would then say so in the plugin's version and in what a lagging host's session
reports.

**A Cowork user whose bridge is down may see an error on every prompt.** An interactive Claude Code session on 2026-10-14 showed "Stop hook error: MCP server
'remote-devices' not connected" for a Stop hook that names a server that is absent. B31 (a hook of type
mcp_tool that names a server that is not there shows nothing) says nothing shows, and its check runs
with `claude -p`. Find out whether an interactive session differs, and if it does, whether the Cowork
plugin's hooks show the error when the bridge has not started. A user whose bridge is down would then
see an error on every prompt.
Either the check is made and the Cowork plugin's hooks are shown to be silent without the bridge, or Cowork is labeled experimental until it is.

**A colleague's pasted image can be any pixel size, and nothing limits it before the browser decodes
it.** `maxImageBytes` is 5 MB (`cmd/cogmer/images.go`), and no Go code reads an image's width or
height, read on 2026-10-27, so a small file that holds a very large picture could stall the browser
of every colleague who opens the room. Reading the width and height from the file's header and
refusing an image past a limit costs a header parser for each format the hosts send. Whether images
are labeled experimental until D-213 (a pasted image is sent to peers and each receiver stores it as
a file) is seen working between two machines is not decided.

**Whether a cloud environment shows the cloud key to the people who can edit it is not known.** D-246 (a cloud environment is added from the machines page)
keeps the cloud key in an environment variable of the cloud environment. Whether a cloud
environment hides a variable's value from other people who can edit the environment is not
known, and matters only for an environment shared with others, since the maintainer creates
theirs at a personal level, said on 2026-10-24.
For an environment shared with a team, whoever can read the value speaks as the user. A check from a second account that can edit the environment would answer it, and until then the documentation can say that a cloud environment holding the key is not shared.

**NAT to NAT, with a real colleague on a real home router.** The last unknown in the
transport, and not testable alone.

**Re-pick the overlay relay when it cannot be reached.** D-187 pins it so the
address is stable. Nothing re-picks, so a machine that relocates past its pinned
relay is unreachable and nothing says so. Change driven by failure, never by
preference.

**Say when your own address changes.** Every pairing string and invitation already
handed out is then stale, and only the daemon can know.

**Windows has never been run, and Claude Code is reported to break `bash` hooks there.** Claude
Code issues #21878, #25832, #26389, #23295, #25399 and #68295 (June 2026, Claude Code 2.1.177) report
`${CLAUDE_PLUGIN_ROOT}` expanding to a backslash path that Git Bash reads as escapes, and #70150,
still open, reports the same for exec-form hook arguments. Their current status may differ. Our
hooks run `bash "${CLAUDE_PLUGIN_ROOT}/hooks-handlers/..."`, the form those reports describe, and
the installer builds and accepts a Windows binary (`install.sh`), but nobody has run either on
Windows and the README does not say so. One run on Windows 11 with Git Bash shows whether it
fails. If it does, the choices are converting the path in a first command, which costs a `cygpath`
dependency, or stating that Windows is unsupported.

**Nothing independent of the repository vouches for a release.** On 2026-10-27 the GitHub API
showed `main` without branch protection and secret scanning and push protection disabled.
`plugin/checksums.txt`, which authorizes the downloaded binary, sits in the same repository as the
hook scripts, so whoever can push can change both, and a plugin runs with the user's privileges
and updates without a review of what changed (Anthropic's plugin security page, and the
cc-plugin-audit project that hashes plugin directories for that reason). Protecting `main` against
force pushes and requiring status checks, two-factor on every account that can push, and
secret scanning with push protection cost nothing on a public repository. Signing
`checksums.txt` with a key kept off the repository and checked by `install.sh` costs a key to
keep, a verifier to ship and a signing step in `release.sh`.

**There is no way to report a vulnerability privately.** The repository has no `SECURITY.md` and no
`.github` directory, checked on 2026-10-27, and whether GitHub's private vulnerability reporting is on
was not checked. Somebody who finds a fault would open a public issue. A `SECURITY.md` naming a
channel, with private reporting enabled, fixes it at the cost of a channel the maintainer reads.

**A colleague's pairing string reaches a shell command line, and Claude Code's permission check is
the only thing in the way.** `/cogmer:peer-pair` takes a string a colleague sent, and Claude Code
substitutes it into the line `"${CLAUDE_PLUGIN_ROOT}/cli.sh" pair $ARGUMENTS` before bash runs it.
The name after `#` can hold `;`, `$` and backticks, because `sanitizeName` strips only `#`, `@`, control
and unprintable characters. A throwaway plugin of the same shape, run on Claude Code 2.1.312 in
default permission mode on 2026-10-27, had both `x;touch <file>` and `x$(touch <file>)` refused by the
permission check and no file created. Not tried: a user who has allowed other commands, a mode that
skips the check, other Claude Code versions, and the Codex skill. Cogmer cannot check the string,
since the substitution happens before `cli.sh` runs. What cogmer can do is limit the characters in the
name its own pairing string carries, which costs names with other characters and stops no
attacker. Running the same test with an allowed command, and with the Codex skill, would show how
much the check covers.

**The `user-identity` branch is 134 commits ahead of `main` and changes the wire and the data of
every installed machine.** On 2026-10-27 it merges into `main` with no conflict (`git merge-tree`),
`main` and the branch both carry plugin version 0.10.5, the wire version on the branch is 3, and the
state directory carries data version 1 where `main` reads none. A program that finds newer data
touches nothing (D-244, the data in the state directory carries one version, and a program that
finds a newer one touches nothing), so a plugin that lags on a user's other host stops working
against the migrated data, and the first start of the new program migrates the databases after
copying them (D-256, the databases are copied before the state directory is migrated, and a program
that cannot migrate it stops). The two beta users, the maintainer's own `~/.cogmer`, and each host's
plugin all move together, and nothing yet says in what order. A release with its own version number,
such as 0.11.0, a plan for each of the three machines, and the lagging-plugin test above, would
cover it.

**No Codex or Cowork session has run on the `user-identity` branch.** The container tests and the
cloud run drive Claude Code, and the Codex and Cowork hooks, commands and bridge are covered by
tests that call the daemon directly. A Codex session and a Cowork session each making an identity,
pairing with a colleague, joining a room, renaming the colleague and removing a machine would show
what the tests do not. The removal of a machine is the least likely to have been seen, since it ends
the daemon that a host's hook starts again.

**A machine with no browser cannot make a user identity, and whether a Codex command can open the
page is not known.** The identity page is the only place the recovery phrase is shown or typed
(D-223, the recovery phrase is typed only into the daemon's browser page), and every pairing and
room command stops at it until an identity exists (D-230, a user identity is made in the daemon's
page the first time the user needs one). `openInBrowser` runs in the command's own process
(`cmd/cogmer/selfcommands.go:77` and `main.go:2144`, read on 2026-10-27), which for Codex is a `bash
codex-cli.sh` run in its shell tool, where a sandbox may refuse a window, and a Codex user on a
machine with no browser has no other route, since pairing's terminal fallback sits behind the same
gate. A Cowork command runs in the daemon on the user's own machine, so it is not affected. Printing
the page's address for the user to open themselves, which holds no secret, costs the user opening
it, and having the daemon open it costs a route on the daemon and the window of a daemon with a
display (B23, a detached daemon can open a window). The item on details left undecided under "Cloud
sessions in a room" already names a machine with no browser for adding a machine.

**The Cowork bridge says nothing when an upgrade failed.** A failed upgrade or a failed copy now
writes a `daemon-state` mark, and the session-start hook of Claude Code and of Codex tells the model
that the daemon could not start (D-256, the databases are copied before the state directory is
migrated, and a program that cannot migrate it stops). The Cowork bridge has no session-start call
(D-205, Claude Cowork is a supported host through a bridge on the user's machine), so a Cowork user
who runs no command sees nothing capture or arrive and no reason, while the bridge goes on trying to
start a daemon that stops at once (D-208, the bridge keeps a daemon running for a Cowork user, and
runs a restart and a stop itself). A note of its own, read from the same mark and given to Cowork's
model with the bridge's reply to a prompt, would cover it, at the cost of a line in every prompt's
reply while the fault lasts.

**A machine that was removed and erased starts again as a new unconnected machine when any host next
starts a daemon.** D-252 (a reachable machine is asked to erase its keys before it is removed)
erases the keys and ends the daemon, and the next Claude Code session, Codex hook or Cowork call
starts one that makes a fresh machine key and no user (read from `retire.go` and the start code, not
run). The user then meets the identity page again on that machine, which suits a machine given away
and may surprise a user who only removed it from their list. D-208's mark that a user stopped the
daemon does not apply, since nobody ran `stop`. Whether the machine should keep a mark that it was
retired is not decided.

**A removal is pushed again and again to machines that can never acknowledge it.** D-225 (a machine
sends each removal and new shared key to every colleague it knows until each acknowledges) sends
each record until its receiver acknowledges. On 2026-10-26 the maintainer's test machine held, in
`identity_push` in `~/.cogmer-test/membership.db`, one removal with 29 attempts and another with 6
to the key of a cloud session, and the second with 5 to a machine that had erased its keys, none
acknowledged. Whether the attempts back off, and whether they ever stop, has not been read from the
code. A limit on attempts or age, or a rule that a removed machine is never a target, would stop
them, at the cost of a removal that is never delivered to a machine that was only asleep.

**One colleague's machine flips between reachable and unreachable every 15 to 30 minutes.** Noticed
on the maintainer's machine during the cloud run of 2026-10-25 and 2026-10-26, for the peer
`drizzle-ravine`, and not investigated. A user who sees a colleague's status change on that rhythm
cannot tell a fault from a quiet colleague. The daemon log and the room view's status history would
show whether the cause is the relay, the address or the status check (D-194, a colleague this
machine cannot reach is named, with how long, in the room status and the view).

## Somebody else uses it

A colleague who did not write cogmer installs it, pairs, joins a room and works in it.

**Somebody who did not write cogmer uses it.** The
repository is public, carries the marketplace manifest beside the plugin, and serves
releases the installer verifies against its pins (D-121, the repository is both the
release host and the marketplace). What it needs is a colleague installing, pairing,
joining, working, and saying what they hit in the order they hit it. It is the only
remaining work that can fail in a way nothing else detects, and the failure looks like
somebody quietly not using it again.

Two things to watch for, because both have been argued about without evidence:
whether the two-word comparison is performed or skipped, and whether the browser view
is consulted or forgotten.

**What a person actually notices in the view.** D-090 left this open: the display
name and the derived name are separate elements but carry similar weight, and
nothing has been measured about whether a reader distinguishes them.

**Whether arrival wants announcing.** D-039 (the browser view is the settled
avenue) left this open. Do not build a notification on speculation — it has a
different answer for close pairing than for long solo stretches.

## Reaching a peer on another network

Two peers on different networks, each behind its own router, reach each other with nothing configured.

**Peers on one network meet through the relay.** Section 4 orders a direct connection on the
same network before the relay. A pairing string carries one endpoint, a `tc://` address whenever
the overlay is on, so two peers on one network dial the overlay and the relay introduces them
before the path becomes direct (read from `AdvertisedEndpoint` and the dialer on 2026-10-02, not
run). Advertising both endpoints and trying the direct one first would follow section 4, and it
changes the pairing string, whose format is permanent once strings circulate.

**Only one address is ever tried for a peer, and none is demoted, aged or
rehabilitated.** Section 4 of the specification, "The lifecycle of a peer address",
has an address used as one of several candidates tried together, demoted by
silence, invalidated by a wrong key, aged at a rate set by what it names, and
restored by evidence. `known_peers` holds one endpoint per peer with the time it was
recorded, replaced when the peer advertises another (read on 2026-09-28). D-103 (an
address belongs to a machine, and is stored in one place) gave each address the owner
the lifecycle needs, and nothing past that is built.

**Whether to rebuild the post-quantum hedge.** D-104 removed the pre-shared key,
which was the only quantum-resistant element in the stack. It could be rebuilt at
our own layer from material the pairing exchange already produces — per peer, which
is better than one secret shared with everybody. Nothing depends on deciding this.

## First contact without a paste

A host can admit somebody who asks to join, without a pairing string being sent first.

**Whether a host approves an unsolicited join request.** A person who is not on a
room's guest list cannot ask to be let in; §12a (room membership) and D-051 (stranger
pairing is not a supported case) leave open whether they should be able to, and
nothing is built for it.

As section 12 describes it, any peer that can reach the host's address and name the
room could make something appear on the host's screen, and room names are guessable
on purpose. Guessing grants nothing, but a prompt people learn to dismiss is a poor
place for a decision that matters. The alternative is to accept requests only while
the host has said they are expecting someone. That is not a token and does not reopen
D-026 (no join token): arriving in such a window admits nobody, it only earns the
right to ask, and the host still approves.

## Discovery on a local network

Peers on the same network find each other without an address being typed.

**Local network discovery** (D-138, local discovery locates a room and never admits anyone) is not built.

## Three peers

A room works with three members, including one whose turns reach another only through a third.

**Nobody has checked that a peer's events reach a third peer through a second.** A
room of three in which two members cannot reach each other directly depends on the
third relaying their events with each one's origin kept (§13, transitive
synchronization). Every run so far has had two peers. Blocking the direct path between
two of three peers, and checking that each one's events arrive attributed to their
origin, would answer it. Pairs are the case that exists, so this waits until somebody
wants a third member.

## Rooms and sessions

What a room, and the sessions in it, do as people create, leave, return and join from more than one session.

**A room records only prompts and responses, so nobody sees a session join or
leave.** The specification's event model listed session-joined, session-left and
status events as a minimum, and `cmd/cogmer/store.go` defines only a user's prompt
and Claude's response (read on 2026-09-28). The specification now states the two
that exist. Whether a room shows a colleague's session joining, leaving or working
is undecided, and "Whether arrival wants announcing" under "Somebody else uses it"
is part of the same question.

**A room never closes, so nothing is archived.** D-132 (a closed room's log is archived)
has a closed room kept as an archive that can be read but never rejoined, and the
specification freezes a room's sequences at that point. Membership ends per person
through `leave` and `revoke`; the room itself has no closed state.

**A notice when a second session from this machine joins a room — not thought
through.** `SessionsInRoom` already counts this machine's live sessions in a room
and `join` never consults it, so noticing is free. What to *say* is the open part.
The test it has to pass: the notice earns its place only if it is **actionable**, or
at least explains how the person got here. A bare "this machine already has a
session in misty-canyon" fails that — it is the same defect as the rejected
discriminator (D-112), one level up: it distinguishes without informing.

Three ways somebody arrives at this moment, and a good notice separates them:

- the other session exists and they forgot — another tab, another directory;
- the other session crashed, and its row survives with `left_at` null because exit
  makes a member absent rather than gone;
- they want two on purpose.

**The fact that separates them is when the other session last produced a turn.**
Two minutes ago is a live session they may have meant to go to; three days ago is a
corpse and they should proceed. That is `MAX(created_at)` over `origin_session_id`
in the *room* store, while the notice fires from *membership* — so the cheap notice
needs a cross-database read, and whether that coupling is worth a display string is
undecided.

**Making the other session findable is a separate and harder question.** The useful
answer is its working directory, which is not stored. D-015 forbids *deriving* a
room from a directory and says nothing about showing one, but storing a directory
is how somebody later keys on it. Not obviously worth it.

Probably not actionable beyond informing: letting session B end session A would be
a session acting on another session's membership, and D-016 already forbids the
move it resembles.

**Whether a peer event may cause a *separate* run.** §3.7 forbids one in an
interactive session and leaves this open. Addressing another person's Claude
would need it, and it raises its own questions — whose subscription, what tool
access, what was agreed to — none of them answered.

An item on **idempotency** was in `residual-concerns.md` and is gone: the file was
deleted on the strength of a reading from earlier in the session, and being
untracked there is no copy. `/cogmer:peer-pair` repeated on a completed pairing was
worked in D-107 and D-108, which may or may not be what it said.

## Claude Cowork as a host

A user in Claude Cowork shares a conversation with a colleague as users of Claude Code and
Codex do, through a bridge on the user's machine.

**Build the Cowork host that D-205 (Claude Cowork is a supported host through a bridge on the
user's machine), D-206 (a Cowork turn counts as delivered when the stop call for its prompt
arrives) and D-207 (the Cowork bridge offers no tool that runs a command) decide.** The probe
that found what they rest on, its runs and what was only read are in `docs/work/cowork.md` and
`docs/work/cowork-probe/`, and both are deleted with this item. What remains, in order:

The Cowork plugin and the setup command are built and checked by tests and a local run. What a Cowork
session does with them has not been seen: that the plugin installs from the zip, that a typed command's
output reaches the model, that the empty text block the bridge returns for a call with nothing to add
raises nothing, that the plugin's name does not collide with another of the same name, and that a
computer's rename needs only the command again.

When a Cowork session has been seen to work with them, D-205 and D-207 become active. The probe left an
entry named `cogmer-host-probe` in the user's Claude Desktop configuration, the probe plugin installed in
Cowork, and the files `~/.cogmer-host-probe.*`, to be removed by the user.

**The cogmer plugin starts a daemon in every Cowork sandbox, where it has nothing to do.** The second image
probe on 2026-10-15 found `/root/.cogmer` in the sandbox, with a membership database, a tailcat key and a
daemon log written two seconds after the session started. The sandbox is remote and its checks of the relays
all failed, so that daemon has an identity, no room and no route to a colleague, and a user gets nothing
from it. Which version ran there was not read. The organization's inventory on the same day listed two
member-created copies, David's at 0.10.3 and Arielle's at 0.7.4. The sandbox's environment holds
`CLAUDE_CODE_REMOTE`, whose value was not read. If it marks a remote sandbox, the plugin's session-start hook
could start nothing there, since the Cowork route is the bridge on the user's machine (D-205).

D-205's reason for rejecting a daemon in the sandbox, that it "cannot reach a relay or another
machine" (`06749a0`, dated 2026-10-13), predates D-217 (a cloud session's daemon carries its relay
connection as a WebSocket), which showed a Claude Code cloud container reaching the relay on
2026-10-18. A daemon of 0.10.4 or later switches to WebSocket by itself when `CLAUDE_CODE_REMOTE` is
`true` (`cmd/cogmer/tailcat.go`, read on 2026-10-27), so the sandbox's failed relay checks, made
before that, may no longer hold, and nobody has run one there since. Reading the variable in the
sandbox and running a relay ping from it would answer. If the sandbox does reach the relay, the
daemon there is either stopped from starting, or the sandbox is made a cloud session, which needs a
way to give it `COGMER_CLOUD_KEY` that Cowork may not offer. If it does not, the daemon there is
stopped from starting.

**A Cowork turn shows the user's computer as its machine, and Desktop's name for the computer and
cogmer's name for the machine are two names.** Every turn carries the name of the machine that
signed it (D-233, every turn carries the name of the machine it came from, as a field the user
asserts), and a Cowork turn is signed by the machine that runs the bridge, so a colleague reads the
laptop's name for a turn whose model ran in a remote sandbox. D-205 says a renamed computer needs
the setup run again, and a machine's name is now a label the user chose and renames from the
machines page (D-231, a user's machines are managed from one command that opens the daemon's page).
Whether the two names can differ, and what a colleague sees when they do, has not been looked at.

## Cloud sessions in a room

A user in a Claude Code cloud session, such as one started from the iPhone app, shares a
conversation with colleagues without a daemon on their own computer being reachable.

**The decisions that make the user the peer leave details undecided.** Each is needed by a
slice of the second run, and none has a choice taken yet:

- Adding a machine or pairing on a machine with no browser, which the second run leaves out,
  since every machine a user adds has had a browser and a cloud session needs no phrase
  (D-223, the recovery phrase is typed only into the daemon's browser page).
- How a user who has lost every machine reaches colleagues, since the user could reach each
  colleague only by sending a new pairing string.
- How a cloud session's hooks load.
- Whether a colleague holding shares of the recovery phrase could recover a user who has lost
  it, which no decision has examined.

**Whether a Codex cloud task can be a cloud session is not known.** The cloud key, its session
certificates and the fetch of the user's lists depend only on `COGMER_CLOUD_KEY` being set
(`cmd/cogmer/cloud.go`, read on 2026-10-27), and so need nothing of a host. The WebSocket relay
switch tests `CLAUDE_CODE_REMOTE`, the plugin reaches the cloud through claude.ai's marketplace
sync, and the environment's variable is set in a Claude Code environment's settings, and each of
those is a Claude Code fact. D-201 (Claude Code and Codex are supported hosts) covers Codex CLI and
desktop sessions and not a cloud task, and nothing has shown whether a Codex cloud task loads the
plugin's hooks, accepts an environment variable and reaches the relay. A run in one, as
`docs/work/cloud-run.md` was run in a Claude Code environment, would answer it.

**Two checks of a cloud session in a room with a paired machine are not shown.**

`docs/work/cloud-run.md` is the run that shows a cloud session in a room with a paired machine,
on the maintainer's machine and in a cloud session. The end of the next paragraph cites its results
and names the two checks it has not shown.

`docs/work/cloud-network-probe.sh` tests, from a cloud session, whether cogmer's transport
opens a tunnel through the public relay and whether SSH over port 443 reaches GitHub. It writes
each run to a dated file beside it, and the script and its runs are deleted with this item.
Below the Full network level, on 2026-10-18, every check failed (`0139779`). At the Full level
the same day, the relay map and three relay hosts answered over HTTPS, but the tunnel test
timed out dialing and `ssh.github.com:443` returned no SSH banner (`b88f417`). The session
trusts its proxy's certificate authority, so the proxy may decrypt HTTPS and pass only HTTP,
which would break both the relay's protocol upgrade and SSH. That is inferred, not shown. A
later run at the Full level reached a daemon on another machine through the relay, as a
WebSocket: both relay pings answered, in 817ms and 774ms, and path discovery found the route
through relay region 1 in 132ms, with no change to that daemon (`e415f1d`). Without WebSocket
mode the same pings got no answer in 10s, so the failure there is at the relay. The tunnel
opened and dialed inside the session as a WebSocket, and timed out without it. A later run, on
2026-10-24, opened a TCP connection through the tunnel to the maintainer's daemon in 113ms and
completed a TLS handshake with it in 132ms, with the key its pairing string names, and the
daemon refused the run's unrecorded client certificate (`1d2b505`). Part 2 of
`docs/work/cloud-run.md` ran on 2026-10-25 and 2026-10-26 (`243df61`, `e85170d`), and the maintainer's
test machine recorded its half (`634abd7`). How that machine's room page marks the cloud session's
turns, and which of them it counts after the cloud key's removal, are not shown.

A cloud session with 0.10.4 reached the relay, observed on 2026-10-18. The user's claude.ai copy of
the plugin stayed at 0.10.3 after 0.10.4 reached `main`, until the user forced a sync in Manage
marketplaces, and how long automatic sync takes is not known. In the cloud session that followed,
`/cogmer:self-status` answered with plugin, binary and daemon at 0.10.4, and its test connection from
outside reached the daemon through the nyc relay in 712ms. The daemon has a key of its own, different
from the one on the user's computer, shows the container's username as its name, and is in no room. The
same plugin at 0.10.3, in a cloud session earlier that day, had the plugin synced into
`~/.claude/plugins/synced/` and its SessionStart hook ran, and the daemon could not reach the relay. D-217
(a cloud session's daemon carries its relay connection as a WebSocket) is in 0.10.4. Not shown: how the
0.10.4 daemon's `daemon.log` says the relay connection is carried, whether UserPromptSubmit and Stop fire
in a cloud session, since neither leaves a trace outside a room, and whether such a daemon pairs with a
colleague, which no run has tried.

## Tidying the commands

The plugin's commands sit in the layout Claude Code prefers, and each has a reason to exist.

**Move `plugin/commands/*.md` to the `skills/<name>/SKILL.md` layout.** The
documentation calls `commands/` legacy and says the two are loaded identically. D-119
deliberately did not do it in the same pass, because a directory restructure is a bad
thing to bury a §3.1 fix inside. `TestNoCommandIsModelInvocable` reads the old path
and would need to follow.

**Whether `verify` still earns its place.** `/cogmer:peer-pair <name> --again` now
does the same job, and `verify` has no slash command, so it may be a subcommand
nobody has a route to.

## Pasted images

An image a user pastes into a session is part of the conversation the user shares, so a colleague should
see the image and the colleague's model should be able to use the image.

**Pasted images are written and tested for the Claude Code terminal and desktop app and for Codex, and not yet seen between two machines.**
The daemon finds the image a prompt sent in the files its host saves, names it in the turn's signed metadata,
and serves it to colleagues, whose daemons fetch it by its hash and show it in the room view (D-213, a pasted
image is sent to peers and each receiver stores it as a file). The tests run two daemons in one process.

Two hosts were observed on 2026-10-15. The Claude Code terminal (2.1.300) saves an image the moment it is
pasted, even if the user then removes it from the prompt, and puts "[Image #N]" in the prompt text, so only an
image the text names is sent. The Claude Code desktop app (2.1.295) saves an image when the prompt is sent, in
the same second as the prompt hook, puts no placeholder in the text, and saves nothing for an image removed
before sending, so an image saved in the last 30 seconds that the session has not sent is sent. The hook's
environment names which one runs, as `CLAUDE_CODE_ENTRYPOINT` is `cli` or `claude-desktop`, and any other
value is read as the terminal. A `claude -p` stream-json image block produced neither the placeholder nor the
file, so the behavior registry, which checks each behavior with `claude -p`, cannot check any of this.

Codex 0.159.3 was observed on 2026-10-15. It puts "[Image #N]" in the prompt text a hook receives and gives
the hook no image and no path. The picture appears in the session's transcript after the hook returns, with
the path of a file named `codex-clipboard-` and random letters in the system temp folder, and nothing at
prompt time ties that file to the session. D-214 (an image a host shows after the prompt is sent in a second
event) is built for it: the daemon reads the transcript for the message whose text is the prompt, because the
turn holds other user messages, and publishes the image in a second event. It is checked against the file
Codex wrote during the paste, which gave the image and the hash of the saved file, and has not run in a live
Codex session with a colleague.

Cowork was observed twice on 2026-10-15, each time by pasting one image and sending "image test". The
hook received that text with no placeholder and no field for an attachment. A second probe found the image
saved in the sandbox at `/root/.claude/uploads/<session id>/<random name>-image.png`, there before the
prompt hook ran, with no `images` folder beside the scratchpad. The transcript at the end of the turn held
two image blocks and the text, with no placeholder. The session looks remote: its environment names a
remote session and a container, and its only mount that is not system code, `/mnt/attach/outputs`, is a
cloud file store. A marker written there reached neither `~/projects/probe`, the folder the user connected to
the session, nor the app's own folders, and the sandbox mounts no folder of the user's machine. So the image
is not on the user's machine, and a Cowork image has no route to the daemon that runs there. A turn from
Cowork reaches a colleague with its text and no picture, and a route would have to be built.
The Claude Code editor extensions are not observed and are
read as the terminal.

D-213 becomes active when a colleague on another machine sees a picture sent from each host that sends one.

Two questions are left. The relay speed for large transfers has not been measured, and a fetch may take
20 seconds plus one second for each 50 KB. Image files stay with a room's database, and nothing removes
either today, so D-132 (a closed room's log is archived, which is not built) has to say what becomes of a
closed room's images.

**A colleague's pasted image never reaches the model.** Context is injected as text in a hook's output,
under a budget of 9,000 characters (D-170, injected context has a character budget for the whole block),
so an image cannot be injected as an image unless Claude Code accepts an image in some form of hook
output, which has not been looked for.

Two ways remain:
- The text of the turn can say an image was attached. The colleague's model then knows there is an image
  it cannot see, and the text costs almost nothing.
- The injected text can name the path of the file the receiver stored (D-213), so that the model reads the
  file with its own tool. That puts a colleague's bytes where the model reads them, so this option needs a
  place the model is allowed to read, cleanup, and a decision on whether the model's permission prompt for
  a file outside the project is acceptable.

## Exporting the shared transcript

A user can keep a room's conversation as a reference after the room is abandoned.

**The export may not outlast its room, and says too little to be read later without it.** D-218 (the export is
the whole room as text-only Markdown) decides its form. Nothing closes a room today, since the closing in
D-015 (rooms are scoped to sessions, not to projects) and the archive in D-132 (a closed room's log is
archived) are not built, so a room's view and export stay on the machine after every session has left, and a
test says so. Whatever builds the closing has to leave both reachable.

The text is thin for a reader who was not there. As `exportMarkdown` in `cmd/cogmer/ui.go` writes it on
2026-10-16, the header holds the room's name, the time of the export and the number of turns. Who took part
appears only on the turns. Each turn's time is the exporter's local time with no zone, and nothing gives the
room's identifier, which two rooms with one name do not share, or the first and last turn. What a reader
needs in the header has not been decided.

## Reading a long conversation

A user can read what colleagues said in a room without their own turns in the way.

**The room page shows the user's own turns, which are the ones the user already has.** The page lists every
turn of every guest in order, the user's own included, and the header offers the connection, each
colleague's status and Export, with no control that narrows the turns (read from `cmd/cogmer/ui.html` and
the specification's list of what the view does, on 2026-10-22). A user who wrote most of a long
conversation scrolls past their own prompts and their assistant's responses to reach what a colleague
wrote, though both are in the user's own session.

A control in the header, beside the colleagues' status, would let the user show or hide the turns of each
person in the room, the user included. The user's own turns are their prompts, the responses of their
assistant, and the turns from their other sessions in the room. The page does the filtering, because the
daemon builds the export, and D-218 (the export is the whole room as text-only Markdown) rejects controls
that choose what the export holds, so a filter in the daemon's stream would put the export at risk of
following it. The page has to keep the unread count and the jump button right for the turns it hides. A
choice need not survive a reload that the user makes. The user's own turns start hidden, and the other
people's start shown.

## Beyond one conversation

What cogmer might do beyond sharing one conversation, once a room works for a pair.

**None of the capabilities the specification once listed as future is decided.** They
were: addressing another user's Claude ("@David-Claude what led you to that
conclusion?"); asking another user to investigate something; presence, showing whose
Claude is working, who is viewing the room and who has been offline and for how long;
searching history across archived rooms ("What did we discover about DLDV last
Tuesday?"); and a durable team memory of confirmed decisions, requirements,
discoveries, constraints and unresolved questions, derived from the complete
conversation rather than replacing it. Addressing another user's Claude depends on
"Whether a peer event may cause a separate run" under "Rooms and sessions".

**Whether cogmer needs a command that says what it is and lists the others.** Section 29
of the specification said there is none because the product's name was not settled, and
the name is now settled (section 37), so that reason no longer holds.

## The documents

Every document follows `docs/writing.md`.

**Nothing vets a proposed design against the documents it could conflict with.** The
only process is `CLAUDE.md`'s instruction to read the decisions that govern an area
before proposing a change, which points at a directory of about 190 files, and it names
none of the other documents. On 2026-09-25 and
2026-09-28 a session proposed moving implementation detail out of the specification,
restating decisions in it, and keeping the two-peer result because a decision said so,
and each was caught by the maintainer rather than by a process. A skill run before a
design is proposed, like `writing-review`, would take the areas the proposal touches
from the `decisions` skill's search and a search of the documents, and report what the proposal must satisfy and what it conflicts with in the
values, the specification, the decisions and their **Rejected.** fields, the behavior
registry, the patterns and `open.md`, and what records it would need. Each point would
quote its source, and a test would drop a quote that is not in the file. It would run on
any proposal that needs a decision entry or a change to the specification. It checks
consistency, not judgment: whether to proceed stays the maintainer's call.

**Nothing records which decisions and rules the maintainer ratified.** A Claude session
wrote most of `docs/decisions/` and `docs/writing.md`, and an entry the maintainer
approved reads the same as one a session made and logged by itself. Later sessions then
defend earlier sessions' choices as settled: on 2026-09-25 D-122 (the two READMEs) put
the two-peer result in `README.md`, W-75 carried the requirement over, and it was
defended twice before the maintainer removed it. A field on each decision and each rule
saying whether the maintainer ratified it would show which choices rest on a person's
judgment. What it costs is a pass over the existing entries to set the field, which
only the maintainer can do.

**No document defines the project's own terms.** W-01 fixes the words for people, and
W-08 makes a writer explain a term of our own where it appears, but room, peer, guest,
label, derived name, pairing, verification and delivery are defined nowhere once. Terms
drift as a result: on 2026-09-28 the specification used "watermark", "delivery state"
and "incorporated-event state" for one thing, and "person" for a user, a colleague and
a human. A glossary would give each term one meaning, and would need a place in
`CLAUDE.md`'s table of what each document answers and a rule in `docs/writing.md` that
every document uses the terms as defined.

**A requirement does not say how anybody would know it is met, and nothing links it to
the tests that check it.** Code comments cite § sections by habit, and tests do not do
so consistently, so a review cannot ask which tests check section 19 and get an answer.
Two options: each requirement states how it is verified, which grows the specification
and names tests in it; or each test that checks a requirement cites its § section, and
a check reports every section no test cites, which keeps the specification free of
verification detail.

**Facts about Claude Code that nothing checks are recorded only in `CLAUDE.md`.** A
constraint nobody here chose, such as what the host does and offers, differs from a
requirement or a decision, which the project chose and can revisit. What Claude Code
does that cogmer relies on is in the behavior registry, `cmd/cogmer/behaviors.go`, and
is checked. What Claude Code offers or forbids that nothing checks is in `CLAUDE.md`'s
"Facts that are not obvious from the code": that every plugin command carries the
plugin's name as a prefix, and that every extension point delivers to the model and
never to a person. `CLAUDE.md` is instructions to the model, not a record a reviewer
consults, so the facts kept there survive only because a session reads them.

The registry is the place for what someone else's software does that cogmer relies on,
but a fact about Claude Code that nothing checks, such as the command prefix or
extension points delivering only to the model, can go in the registry only if an entry
is permitted to omit the check. Permitting an entry to omit the check would put every
external reliance in one place: an entry without a check keeps its `Title` and
`Reliance`, says how a person can confirm the fact by hand, and appears in
`docs/relied-on-behaviors.md` in a group of its own, so it is never mistaken for
something `cogmer doctor` verifies. The code that relies on an entry cites it, and
something that routes a session to entries by area, as the `decisions` skill does for
decisions, would reach it when the work touches what it constrains rather than on every
turn (W-26).
