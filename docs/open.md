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

**A command names a next step only for its own case, and none walks a user from a fresh install to a room.** Several commands name what comes next in their case: the output of `/cogmer:self-status` ends with the command to pair, `/cogmer:peer-list` points to `/cogmer:peer-pair`, and `/cogmer:room-create` says that a second session joins with `/cogmer:room-join` (read from `plugin/commands` and the output of 0.10.2 on 2026-11-02). Which next step a user is told therefore depends on which command they ran, and a user who has run only `/cogmer:self-status` is told how to pair and not that a room comes after. Each command's output could end with the next step chosen from the user's state: make the identity, send the string, pair, create a room. That costs a decision, code and a test for each state, and a line that the model has to relay, since a command's output reaches a person only through the model.

**The message that says to run `cogmer doctor` names a command that a user who installed the plugin has no way to run.** The messages "Run `cogmer doctor`" and "or run cogmer doctor --host claude" are in `cmd/cogmer/doctor.go` (read on 2026-11-24). The plugin has 19 commands, `self-`, `peer-` and `room-`, and none runs `doctor`, and `git log -S doctor -- plugin/commands` finds that none ever did. `README.md` gives `bin/cogmer doctor`, which is the path in a built checkout, and a user who installed through the plugin has the program in `~/.cogmer/bin`. No decision or item says why it is not a command. A command would let the user run it from the session, at the cost of two Claude turns driven from inside a session and, in a Cowork session on the user's computer, a shell line that Cowork replaces with a placeholder (`e7ddab7`). The message could name the program's path, or `plugin/README.md` could say how to run it, which costs no new command.

## Before a wider audience

People the maintainer has never met install cogmer, on machines, networks and settings the
maintainer has not seen, and none of them is worse off for it.

**Another account on the same machine can read and drive a user's daemon, because the daemon's local
address is open to every account.** On 2026-10-28, on a droplet (Ubuntu 24.04) with two accounts,
the `codex` account fetched the transcript of a room belonging to the `claude` account, with no
credential, from `/export?room=<name>` and `/events?room=<name>` on the daemon's loopback address,
and `/healthz` and the identity page answered 200 as well. The room held fabricated turns in a
throwaway home. The daemon leaves those reads unguarded because only a web page was considered, and
a cross-origin page cannot read their answers (the comment before `LocalRoutes` returns,
`cmd/cogmer/daemon.go`, read on 2026-10-28). The routes that change something check a method, an
Origin and a header whose value is a constant, so any account that sends it passes (D-087, the local
API requires a header that a web page cannot send, read from `cmd/cogmer/localguard.go`; not tried
across accounts). A second account's cogmer also treats the first's daemon as its own: with the
default addresses, its daemon logged "a cogmer daemon is already serving 127.0.0.1:4782; leaving it
to it" and exited 0, its commands then reached the other account's daemon, and `cogmer stop` as the
second account said nothing was listening, since it cannot see another account's process. This
affects any machine with more than one account, such as a shared workstation or a build server. A
secret in a file of mode 0600 in the state directory, which hooks and commands send and which the
page receives in the address `cogmer machines` opens, as Jupyter does, closes the reading and the
driving, at the cost of a token in an address and a read of the file by each caller. A Unix socket
for commands and hooks, with the TCP listener serving only the page, costs a second listener and
does not exist on Windows. A port chosen from the account stops the sharing and not the reading.
Which to choose is a decision, and D-087 does not cover it.

**A release that has to break the interface an older plugin uses has no way to tell a lagging host's
session.** One program in `~/.cogmer/bin` serves every host, so a host whose plugin has not updated
runs its older hook lines, handler scripts and command files against a newer program (D-269, the
program serves the plugin of each of the last three releases). A test now runs the program through
each of the last three plugins and fails when a release renames a subcommand they call or changes
something their scripts read, but a release that must break one has nowhere to say so. The lagging
host's hooks then fail open, its sessions show no error, and they silently stop capturing and
receiving turns until its plugin updates. Raising the plugin's version, which the host reads when it
syncs, and a line in the session-start note that tells the model its plugin is older than the
program's, would say it, at the cost of a check of the plugin's version against the program's in the
hook that runs at every session start.

**A Cowork user whose bridge is down may see an error on every prompt.** An interactive Claude Code session on 2026-10-14 showed "Stop hook error: MCP server
'remote-devices' not connected" for a Stop hook that names a server that is absent. B31 (a hook of type
mcp_tool that names a server that is not there shows nothing) says nothing shows, and its check runs
with `claude -p`. Find out whether an interactive session differs, and if it does, whether the Cowork
plugin's hooks show the error when the bridge has not started. A user whose bridge is down would then
see an error on every prompt. A `claude -p --include-hook-events` run on Windows on 2026-10-28 reported two such errors from hooks the test did not define, probably the account's Cowork plugin, which was not identified (`5baeaea`).
Either the check is made and the Cowork plugin's hooks are shown to be silent without the bridge, or Cowork is labeled experimental until it is.

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
skips the check, other Claude Code versions, and the Codex skill. `cli.sh` cannot check the string,
since the substitution happens before it runs. Two things can limit it. Cogmer can limit the characters
in the name its own pairing string carries, which costs names with other characters and stops no
attacker. The prompt hook the daemon already runs on every prompt sees a command as typed (B33, a
project or plugin command reaches UserPromptSubmit with the text as typed) and could exit 2 for a
`/cogmer:` command whose text holds a character outside an alphabet. On 2026-10-28 on a Mac, such a hook
blocked `x;touch <file>` before the command file's shell line ran, showed its message as the result, and
no file was created (`4b70323`). That check would also protect a user who has allowed other commands,
while the hook runs. It costs an entry in `behaviors.go` with a negative test, a block the user sees,
and no protection where the hook is disabled or fails open because the daemon is down. Not tried: an
interactive session, Windows, Codex and Cowork. Running the same test with an allowed command, and
with the Codex skill, would show how much the permission check covers.

**The next release moves every installed machine to user identities, and nothing yet says in what order the three installs move.** On 2026-11-20 `host-probes`, which holds `user-identity`, is 281 commits ahead of `origin/main` and merges into it with no conflict (`git merge-tree`). The wire version is 3 and the state directory carries data version 1, where 0.10.5 has neither. The first start of the new program copies the databases and then migrates them (D-256, the databases are copied before the state directory is migrated, and a program that cannot migrate it stops). One program serves every host, and it serves the plugin of each of the last three releases, so a host whose plugin has not updated keeps working against the migrated data (D-269, the program serves the plugin of each of the last three releases). A 0.10.5 machine answered by an upgraded one is told to upgrade, and turns either side sends meanwhile reach the other once both have upgraded and made their user identities, with nobody pairing again (`TestContainerUpgradeKeepsAPairedRoom`, which passed on this droplet on 2026-11-20 with Docker 29.1.3 and `main` at `357af6b`, the 0.10.5 code, along with the other six container tests). The installs are the maintainer's and the two beta users'. A release would:

1. On a branch from `host-probes`, which merges into `main` as a fast-forward, set the version to 0.11.0 and run `scripts/release.sh`.
2. Run the container tests again on that commit, with `go test -tags 'docker ts_debug_websockets' -run Container ./cmd/cogmer`, through `sg docker -c` on this droplet, and then run `scripts/stage.sh`.
3. Follow `README.md` alone on a Mac with cogmer's state saved and every other trace removed, through Claude Code and then a Cowork cloud task, with `Blue-Rocket/cogmer-staging` in place of `Blue-Rocket/cogmer`. In the task, `/cogmer:room-join` reaching the room shows that a command's repeated lines ran in the sandbox and bound that session (D-291, a command gives the model its shell lines to run when the host did not run them), which no run has shown, and whether the task asked permission before the model ran `cli.sh` says what stands between a model and the script (D-291's limits). A Claude Code session on the Mac shows whether the account's copy of cogmer loads beside the one its command line installed.
4. Push the release commit to `main` and run `scripts/publish.sh` on it, and on the Mac replace the staging marketplace with `Blue-Rocket/cogmer`, in Claude Code and on the Claude account.
5. Tell both beta users before the maintainer upgrades, since at least one shares a room with the maintainer, and turns between an upgraded machine and a 0.10.5 one wait until both sides have upgraded and made their user identities. A user whose marketplace does not update by itself gets the release only by updating as `plugin/README.md` says under "Updating".
6. Upgrade everyone in a shared room within the same short window: the maintainer first, then each beta user, each updating the plugin in every host on every computer and making their user identity, then checking `/cogmer:self-status` and a turn in a room.

**Codex's shell tool cannot start on Ubuntu 24.04 with its default kernel settings, so `doctor`
reports B28 failed and a Codex user's commands fail there.** On 2026-10-28, on a droplet with Ubuntu
24.04 and codex-cli 0.160.0, `kernel.apparmor_restrict_unprivileged_userns` is 1, and a shell tool
run with `-s workspace-write` answered `bwrap: loopback: Failed RTM_NEWADDR: Operation not
permitted` and did not run. `cogmer doctor --host codex` there passed B25, B26, B27 and B36 and
failed B28 with `CODEX_THREAD_ID ""`, because the probe's tool wrote no file. Run with the sandbox
off, the same Codex exported `CODEX_THREAD_ID` to the tool, equal to the thread it reported at the
start of the turn, and the shipped `codex-cli.sh` accepted the session and ran `whoami` and `rooms`
with the Linux binary, so the failure is the sandbox and no change in what Codex exports. A Codex
user on such a machine gets capture and delivery from the hooks, which run outside the tool (B25 and
B27 passed), and no `$cogmer` command until the tool can run, which Codex's own shell use needs as
well. Not tried: a command in a session of the installed plugin, and the probe with network allowed
(`-c sandbox_workspace_write.network_access=true`), whose one attempt ran no tool. The message for
B28 could name this cause when the tool wrote nothing, at the cost of a sentence and a test, and the
README could tell a Codex user on such a machine that the sandbox needs that setting changed or
turned off, which is their own decision.

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

On the droplet on 2026-10-28, `cogmer machines` on a machine with no screen printed "could not open
a browser (exit status 3). Open this address in a browser on this machine", though `xdg-open` is
installed, and the address is of no use from another computer. An SSH tunnel from the user's
computer reached the page only when its local port was the daemon's own: a tunnel on the same port
answered 200 to the page and to a guarded POST, and a tunnel on another local port answered 403,
"this address is not served here", to both, because the Host check names the daemon's own port
(`isOwnHost`, `cmd/cogmer/localguard.go`). A user whose own computer already runs a daemon on that
port cannot use the same one. Telling the user to forward the daemon's own port, in the message,
costs a sentence. Accepting any loopback port in the Host check and the Origin check would make a
page of any local web server acceptable to the daemon, so it is not offered here; an explicit
setting that names the one port a tunnel uses, read at the daemon's start, would not.

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

**One colleague's machine goes unreachable about 45 times a day, and the cause is probably that
their computer sleeps and wakes.** The maintainer's `~/.cogmer/daemon.log` on 2026-10-28 holds 216
outages of the peer `drizzle-ravine` between 2026-10-22 and 2026-10-27, a median of 17.6 minutes
between the starts of two (10th percentile 9.7, 90th 52.1), each lasting a median of 10 minutes and
at most 150 (D-210, the daemon says where a connection to a peer fails, and discards a connection
that is stuck, logs the outage). Fifty-five requests from the same peer were refused for a timestamp
20 to 48 minutes from now, against a tolerance of two minutes, which is what a request signed before
a computer slept and sent after it woke would carry. That fits a laptop that wakes for a few minutes
at a time, and does not fit a fault in the relay or in the address, which would not delay a signed
request by half an hour. Nothing from the colleague's side has been read. Asking the colleague
whether the machine sleeps, and reading their power log for wake times, would settle it. If it does,
no change is needed in the code, and a user who sees the status change on that rhythm is told by the
room status how long the colleague has been out of reach (D-194, a colleague this machine cannot
reach is named, with how long, in the room status and the view).

**A relay change would break addresses users already hold, which §4 (networking) says a change to cogmer's servers never does.** A machine's overlay address names the relay region it pinned at its first start, including the relay's host names, and colleagues hold that address (D-187, the overlay's relay region is chosen once and stored with the node key; `loadTailcatRegion` and `StartTailcat` in `cmd/cogmer/tailcat.go`, read on 2026-11-20). D-192 (the public relay is an accepted dependency, and `plugin/README.md` discloses it) has the overlay, which is on by default, meet through the free relays on tailcat's map, whose documentation promises no availability and says they are rate-limited. A Cowork cloud session's daemon reaches those relays only after an organization owner allows five hosts, the map at tailcat.dev and four relay servers (`ce962eb`). No daemon in a sandbox has been seen to reach a colleague directly, and a daemon that cannot sends all of its traffic through the relay. The cost of moving grows with each user, since every address handed out names the present relay.

The options, each with its cost:

- An address of keys only, on one relay region that cogmer runs at cogmer.ai. Nothing a user holds then names a server, so cogmer can move, add or replace relay servers without any user noticing; a region can be several meshed servers in several places. Every user depends on the one relay network cogmer runs unless the address can also name another relay. §4 (networking) says no network provider is required, so an address that names no relay would mean cogmer's region, and an optional relay in the address would serve a user who runs their own relay or would rather not use cogmer's. The address format changes, which D-091 (identity is advertised and location is discovered) and the description of an overlay address in §4 would follow.
- A region number and a map that cogmer serves, which tailcat supports through its `DERPMapURL` option. Host names leave the address and a relay can move by changing the map, but each address still pins a region number, so retiring a region still breaks the addresses that name it. This option is the least new code and a step towards the first option, and every daemon fetches the map at start, from a host an owner then has to allow.
- Keys only and a lookup of which relay a key is on now, from DNS records or a service at cogmer.ai. This option is the most flexible, and the service is a central component that learns who looks for whom, which D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person) asks to be justified to the user.

Any relay at cogmer.ai makes cogmer the operator that sees each machine's network address and key, which machines connect, when and how much passes, as D-192 says the present relay does. The user has to be told, and only a benefit to the user can justify it. The relayed traffic of every cloud session is cogmer's bandwidth to pay for, and the relay must stay up and limit abuse. It would cut what an owner allows to one or two cogmer names, and it would run beside the servers cogmer would run for D-285 (a user with no computer of their own has a machine hosted for them).

A machine that starts advertising a new address moves the colleagues it reaches, since a stored address is replaced when a peer advertises another (`known_peers`, read on 2026-09-28), and an endpoint matters only until the first contact (D-018, identity and reachability are separate). Pairing strings and invitations not yet used when the old relay stops would fail.

A relay at cogmer.ai does not help where a sandbox's allowlist cannot be edited. Whether a Pro or Max user can add domains to the allowlist for a Cowork cloud session is not known, and running cogmer in a Cowork cloud session on a Pro account with the relay's hosts added would answer it. Whether a wildcard such as `*.cogmer.ai` works in the allowlist is not known either.

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

**A colleague reaches a user only through a pairing string of random characters, where §6 (identity) says what a user gives a colleague reads as a name and stays the same.** A pairing string carries an identifier, an address and a name. D-142 (identities are exchanged through pairing strings, with no directory) rejected a directory because needing one before two people could work together would defeat an invitation, and because it is a central component. D-142's revisit condition is users needing to find a colleague's identity without the colleague sending it.

An email address the user has proved they receive mail at would be the handle a colleague knows already, and a Cowork user's Claude account has one. The handle would name the user's key and never be the identity: the key stays the identity, and the recovery phrase stays the only way back to it once every machine is gone (§6, identity). A user can attach more than one address and detach one that is lost or reassigned without the identity changing. A handle says where to find a key and not whose it is, so the two words are still checked at pairing (D-054, verification gates synchronization and injection, not just a marker). An address never restores an identity, because whoever controls the mailbox would then hold the identity.

Who vouches that an address belongs to a key, each with its cost:

- The organization's domain, through a file or a DNS record on the domain that lists its members' keys. No directory of cogmer's is involved. The organization's admin sets the file or record up once, and this form reaches only users whose organization has set one up.
- A one-time code that a service at cogmer.ai sends to the address, for every other user. It is a directory cogmer runs, which reverses D-142 and needs the benefit to the user that D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person) asks for, here reaching a colleague by an address the colleague knows. Anyone can ask it whether an address uses cogmer. Answering only an exact address and limiting how often one caller asks narrows that and does not remove it. The service holds a list of addresses.
- A name at cogmer.ai that is not an email address, for a user who wants a handle not tied to an employer. cogmer decides who may hold a name and what happens to a name given up.

Every form leaves pairing strings working, so no directory is needed before two people can work together. An address changes when its user changes jobs, and a former employer can give it to somebody else, so how long a vouching lasts and how a colleague learns it was withdrawn is not decided. How a handle relates to the display name, which is not unique, and to the label a colleague gives, is not decided either.

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

**The session start hook adds cogmer's text to the context of every session, in a room
or not.** It starts the daemon, starts the binary fetch, and injects the standing policy
for room content into every session on the machine (read from
`plugin/hooks-handlers/session-start.sh` on 2026-11-16). D-284 (a session in no room
reaches cogmer only through the commands its user types) governs the prompt and stop
hooks and leaves this hook out. D-176 (the standing policy is also stated once at session
start) places the policy there so it reaches the model where room content cannot, and a
session not yet in a room may join one later in the same session.

The options, each with its cost:
- Leave it. Every session carries the policy, 106 words, whether or not it ever joins a
  room.
- State the policy when a session joins, in the command's output. It then arrives in a
  tool result, beside text the command printed, rather than in a position of its own,
  which reverses D-176.
- Drop the policy and rely on the framing inside each block. D-176's Limits record that
  the same hostile turn was refused with and without the policy.
- Stop starting the daemon at session start and start it from the commands. Every command
  that needs the daemon would have to start it and wait for it to answer, and colleagues
  could not fetch this machine's events while no session on it had started one.

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

**`cowork-setup` writes Claude Desktop's configuration to a file that Desktop on Windows did not read, and reports success.** On 2026-10-28 on Windows 11 ARM64, the setup wrote `%APPDATA%\Claude\claude_desktop_config.json`, and Desktop's Developer settings listed no servers. Desktop on one account read `AppData\Local\Claude-Data\claude_<id>\claude_desktop_config.json`. On the other, a fresh Microsoft Store (MSIX) install, Desktop's log said at its first launch to create the file under `AppData\Local\Packages\Claude_<id>\LocalCache\Roaming\Claude\`, and from a later launch Desktop used the `Claude-Data` folder, carrying the `cogmer` entry across and rewriting the file. The setup has no way to find the file Desktop reads, so a user who runs it sees success and then no `cogmer` server. With the right file named in `COGMER_DESKTOP_CONFIG`, Desktop started the bridge, the bridge started the daemon, and Desktop's device-bridge line changed from `+0 local-mcp` to `+2 local-mcp`, whose entries the log does not name. The setup could look for each of these locations and use the one that exists, which fails again on a layout nobody has seen. It could also say which file it wrote and how to check that Desktop lists `cogmer`, which costs the user a step but is true on every layout.

**Cowork has not been shown to run on Windows: Claude Desktop reported the workspace VM unsupported in the Parallels guest, and no hook reached the bridge.** On 2026-10-28 Desktop's `cowork_vm_node.log` said `workspace VM not supported (status=unsupported)` and listed the VM's disk, kernel and initrd files as missing. A Cowork session there answered `/cogmer:self-status` with "cogmer did not answer", and the bridge's log shows no call after `tools/list`, so the session's hook never reached the bridge. The cause was not established. Claude Code issue #49997 reports Cowork working on Windows 11 ARM64 in Parallels until a Desktop update, and issue #51598 reports that Windows Server 2022 needed the full Hyper-V role, both read only as search-result summaries on 2026-10-28. Cowork on a machine whose virtualization works, such as a Mac, would show whether the hook reaches the bridge. A Windows machine that supports nested or hardware virtualization would show the same on Windows. Three `cogmer` processes ran after Desktop started the bridge, and one held the daemon's ports. The other two were not identified, so whether Desktop starts the bridge twice is not known. Anthropic's support articles say new Cowork sessions on Pro, Max and Ultra plans run in the cloud, and a setting on a Team plan chooses where they run (read on 2026-10-31), so the unsupported workspace VM may not have decided that run, and which kind it was is not known.

**Three tests of stopping a daemon skip on Windows over ssh, one of them the Cowork bridge's, and no test there checks that a file holding a key is private.** On 2026-10-30 `go test ./...` on Windows 11 ARM64 over ssh passed 716 tests, skipped 21 and failed none (`3a340e1`). Three skip on every system unless asked, such as the one that needs the public relay. Fifteen skip because Windows has no group or other mode bits or lets an ordinary account make no link: eleven read mode bits, three make a migration fail with a read-only file, and one makes a symbolic link. Three skip because `tasklist` answers "Access denied" for every process in an ssh session, so `stop` cannot name the process that holds a port: `TestTheBridgeKeepsARealDaemonForACoworkUser`, `TestARestartReplacesARunningDaemonWithOneThatAnswers` and `TestStopLeavesTheMarkAndAStartingDaemonRemovesIt`. They failed there before the skip, and nobody has run them in an interactive session. What keeps `shared.key` and the databases private on Windows is the access list the account's profile folder passes down, which no test reads. Running the suite as a scheduled task in the account's interactive session would show whether the three pass, and costs a task on the test machine. Naming the process with the Windows API instead of `tasklist` would make `stop` work where `tasklist` is refused, and costs code that calls the API and a change to D-123 (stop finds a daemon by the addresses it holds, and signals only cogmer), which names `tasklist`, and to the accounting of outbound code. A check of the access list could call `icacls`, and what it should require is not decided.

**A daemon in a Cowork cloud sandbox has to be started again at every session and resume, and takes its value only from a plugin file the model's shell can read.** A Cowork cloud session that holds a cloud value starts the daemon as its user (D-288, a Cowork cloud session that holds a cloud value starts the daemon as its user), and one with no cloud value fetches and starts nothing (D-286, a Cowork cloud session with no cloud value fetches and starts nothing), where the released 0.10.5 plugin started a daemon with a new identity named "Someone", no shared key and no room (`9c88af4:docs/work/host-probe-cloud-sessions-20261118T180600Z.txt`). The session's environment holds `CLAUDE_CODE_ENTRYPOINT` set to `remote_cowork`, as well as `CLAUDE_CODE_REMOTE` set to `true` (`9c88af4:docs/work/host-probe-cloud-sessions-20261118T180600Z.txt`, `78477ee`), and D-279 (a cloud session's daemon reads its cloud value from the plugin files it finds) uses only the second, which a Claude Code cloud session also sets. What `CLAUDE_CODE_ENTRYPOINT` holds in a Claude Code cloud session was not read.

D-205's reason for rejecting a daemon in the sandbox, that it "cannot reach a relay or another
machine" (`06749a0`, dated 2026-10-13), predates D-217 (a cloud session's daemon carries its relay
connection as a WebSocket), which showed a Claude Code cloud container at the Full network level reaching the relay on
2026-10-18. A 0.10.5 daemon run in a Cowork cloud session's sandbox on 2026-10-31, with `CLAUDE_CODE_REMOTE`
set to `true`, could not fetch the relay's map: tailcat.dev answered Forbidden, so the daemon was reachable only at 127.0.0.1 and logged no WebSocket (`78477ee`). The organization's network policy decided that refusal. Three runs of a probe in cloud sessions on 2026-10-31 show the proxy refusing the relay's map, its four servers and a WebSocket upgrade under the policy as it stood, which was package managers only plus the organization's own extra domains, and show the proxy passing all of them, with the upgrade accepted, once all domains were allowed or only those five hosts were added (`ce962eb`). Under the policy as it stood the daemon there has no route to a colleague. Where an owner has allowed the five hosts, a daemon there has a route. A 0.10.5 daemon that a session-start hook started connected to the relay as a WebSocket, and its `whoami` reported a connection from outside through the relay, in a first session and again after a resume (`7adc75c`, `c72ef75`).

A cloud session could be made without giving it any secret, by having one of the user's machines approve it. The daemon in the sandbox would make its own key pair, as a cloud session does under D-216 (a cloud session signs with a key of its own that the user's shared key vouches for), and show a string holding its public key and relay address. The string holds no secret, and could reach the user through the model like a pairing string. The user would approve the string on the machines page of one of their machines. That machine would certify the sandbox's key and send it the chain, as D-229 (a machine is added from another of the user's machines) does when it certifies a new machine, and the sandbox would fetch the user's lists from that machine over the relay, as D-245 (a cloud session takes its user's lists only from one of the user's machines) has a session do. The design would need no `COGMER_CLOUD_KEY` and no file, and it would change how a cloud session gets its chain under D-246 (a cloud environment is added from the machines page) and D-216. The design costs an approval for each new session, since a new sandbox makes a new key, and a machine awake to approve. Two things stop it today. A sandbox reaches the relay only where an organization owner has allowed it (`ce962eb`), although a daemon there connects once it is allowed (`7adc75c`). And the string would pass through the model, so room content could manipulate the model into showing an attacker's string, where pairing guards against a swapped string with two words read aloud (D-054, verification gates synchronization and injection). What the machines page could show for the user to check against the sandbox, without trusting the model, is not known.

Both cloud routes have reached the relay only at a wide enough network level. A Claude Code cloud environment failed every check below the Full level on 2026-10-18 (`0139779`) and reached a daemon on another machine at the Full level (`e415f1d`), which D-217 (a cloud session's daemon carries its relay connection as a WebSocket) records. The Cowork sandbox follows the organization's setting under Organization settings > Capabilities, which the screen tags Chat and Cowork and says does not apply to web search, web fetch or MCP connectors (`ce962eb`). Only an owner of a Team or Enterprise organization can change it, and for individual plans Anthropic's article on creating and editing files describes only an on or off toggle (read on 2026-10-31). Because the setting is the organization's, allowing the relay for a Cowork sandbox allows those hosts for every member and for the other uses the setting covers, which an owner has to accept. A Claude Code environment has a network access setting of its own, which read Full for the maintainer's personal environment on 2026-10-31, and whether the organization's setting also governs Claude Code cloud sessions was not seen. Not known: whether a Pro or Max user can choose any level for either route, and whether a wildcard such as `*.ipn.dev` works.

The session's hooks give a daemon in a Cowork cloud sandbox what capture and delivery need, but not a long life. On 2026-10-31 the command hooks ran in the sandbox as root. The transcript path was readable at every event, the Stop payload carried the last assistant message, and the model repeated the `additionalContext` that SessionStart and UserPromptSubmit returned (`7adc75c`). A detached process outlived its hook and ran between prompts. A loop started in the first session wrote about ten of the hundred and twenty lines an hour would have given, which ends it near two and a half minutes after the last response, while a loop started on the resume ran through a gap of about two minutes. Files in `/tmp` survived 4h40m with the laptop off, and the resume's hook had a fresh process id (`c72ef75`). So a daemon there has to be started by the session-start hook for each session and each resume, and has to ask its colleagues for what it missed, as D-031 (peer synchronization polls) already supports for a machine that was absent. No sandbox has had an identity, so none can yet be a member of a room. Not known: how long Cowork keeps a sandbox's processes, whether a resume on another day or with the desktop app closed behaves the same, and which program starts the session's Claude Code in the sandbox.

Other ways a value could reach the sandbox without passing through the model are also not shown to work. A `SessionStart` command hook could read a file from a folder the user attaches, which in a cloud session needs the desktop app open, leaves a permanent file the model's shell can read, and would need D-246 to accept a file as an equivalent of the variable. The cloud session in the probe showed no mounts, and which folder it had connected was not recorded (`78477ee`). A command hook could fetch the value over the network, which needs a credential for the store it fetches from and so moves the secret problem. An `mcp_tool` hook's plain-text reply would put the key in a transcript, and D-246 never shows the value to a log, and the server answering would have to hold the key.

**No released cogmer reads the cloud value, so a value a user uploads reaches no daemon in a Cowork cloud session.** D-278 (the machines page offers the cloud value as a plugin for the user to upload once), D-279 (a cloud session's daemon reads its cloud value from the plugin files it finds, as well as from the variable), D-280 (a user has one cloud key at a time) and D-281 (a command in a cloud session with no value says the user must first set cogmer up on one of their own computers) are built and tested, and 0.10.5 has none of them. On 2026-11-18 a build from `ffe2f33`, which a throwaway plugin carried into a Cowork cloud session and started, found the uploaded value at the session's first hook, spoke for a throwaway user made for the run with a session key certified for 24 hours, and took that user's lists from that user's machine through the relay as a WebSocket (`9c88af4:docs/work/host-probe-cloud-sessions-20261118T180600Z.txt`). That machine answered the first request for the lists with a 500, "the lists could not be read", and the session had the lists 16s later. Four TLS handshakes on the session daemon's peer port timed out in the next 4 minutes, and one came from an address the daemon's own relay client had registered 21s before (`9c88af4:docs/work/host-probe-cloud-sessions-20261118T180600Z.txt`). A daemon built from `63d0b04` with `CLAUDE_CODE_REMOTE` true read the zip the page made, unpacked into a folder laid out as a synced plugin, spoke for the user and took the user's lists from the user's machine (`e180ea5`). The same zip, uploaded as a personal plugin, reached a Cowork cloud session with the length, mode, owner and digest prefix of the page's file, as a hook of another plugin reported (`5fd9fa4`). The daemon looks for `cloud.value` in plugin folders named as the page names its plugin, one folder below an account folder under the home directory's `.claude/plugins/synced`, the layout that the Cowork sessions of `7a362f6`, `f6d721a` and `5fd9fa4` showed, and the daemon of the run on 2026-11-18 found the value there (`9c88af4:docs/work/host-probe-cloud-sessions-20261118T180600Z.txt`). No behavior check covers the layout, since the folder is in the sandbox and `cogmer doctor` runs on the user's machine. A Cowork session declined to run a prompt that asked it to list the files named `cloud.value` and reply with only the output, and said that the instructions looked shaped to extract credential material (`5fd9fa4`), so a check that asks a session to inspect the file can fail on the model's refusal, and the hook that reported the file in that run did not.

The runs behind the decisions showed some facts that none of them holds. Uploading a plugin of the same name asked "Replace existing plugin?", and a new session then saw only the new file (`dab6794`). A session that was running did not run the plugin's hook at its next prompt after the plugin was removed, and its synced folder no longer held the plugin's folder or its file (`d65dd22`, `2f35641`). Remove cleared every visible trace of the plugin from the member's settings and the organization's inventory (`0f35f77`). Desktop refused to upload a plugin with a `bin/` folder, because "claude.ai-hosted plugins may not ship bin/ executables because they are added to PATH on the CLI but are not shown on the admin approval surface" (`9c88af4:docs/work/host-probe-cloud-sessions-20261118T180600Z.txt`). A plugin that fetches its program, as cogmer's does under D-066 (the binary is fetched and verified, never shipped in the plugin), never meets that refusal. The desktop app kept listing a removed plugin until it was restarted (`d65dd22`), but a later one was gone from it 5 minutes after its removal with no restart, and the app may never have listed that one (`8897336`). The organization's inventory listed a personal plugin as a member's, with audience and sharing both "Only creator" and no publish request, and kept the replaced version in a version history with no control there to download, view or delete a version (`e282ceb`, `8808f16`, `ece3d10`). An owner can turn off plugins for everyone with the Skills switch, or stop Claude Code syncing account plugins with `syncClaudeAiPlugins` in its managed settings (Anthropic's page on managing plugins for an organization, read on 2026-11-01). A model that looked through a plugin printed its file (`cac2d7c`, `9477067`), and the same holds for the variable, which a routine `env` shows, since no model has been asked to print it. Anthropic's plugins article says a plugin added in Desktop reaches the user's other Claude Code installs (read on 2026-10-28), so a daemon on the user's own machine could find the file, and D-279 has the daemon read the files only where `CLAUDE_CODE_REMOTE` is `true`. A credential limited to the rooms the user chooses for cloud sessions would limit what a leak gives, and would need a new version of the value.

Not known: why the throwaway user's machine answered the first request for the lists with a 500, and what dialed the session daemon's peer port and timed out; whether a Claude Code cloud session syncs a plugin of only a manifest and a file, since `f6d721a` records one session, whose client was not stated; what the plugin's page lists for such a plugin; what the `.meta.json` files and `manifest.json` beside the synced plugins hold; whether the plugin loads in a second Claude Code environment, or in one below the Full network level; whether a plugin switched off with `"<name>@synced": false` is still synced with its file; whether an administrator can read a member's plugin files by any route, since Anthropic's articles on the compliance API and on managing plugins do not say (read on 2026-11-01), and the latter says a member-created item's page shows no version history, which the screens contradict; whether an organization with security scanning on, which Blue Rocket has off, scans a personal upload and how it treats a key-shaped file; what another member sees; how long a session waits for a replacement, with one observation of 8s; whether a daemon running between prompts ever sees the new file; and how many plugins can hold a `cloud.value` before a session fails to start.

**A Cowork cloud task runs the plugin version the account's copy of the marketplace last synced, which did not follow main by itself.** On 2026-11-20 the account's marketplace `cogmer` read synced at be0e3bb, the v0.10.4 commit, with "Sync automatically" on, a month after 357af6b put 0.10.5 on main, and it moved to 357af6b only when the user chose "Check for updates" (`db8eced`). The cloud task's session-start hook fetches the binary its plugin's version names, so every cloud task of such a user runs the older release, and the older release may not read records a colleague's newer release writes. D-290 (cogmer reaches a Cowork cloud session as the marketplace plugin installed on the user's Claude account) relies on this copy. `plugin/README.md` tells the user to check for updates after a release, a step the user repeats at every release.

Whether the automatic sync runs at all, and how often, is not known; watching the synced commit for some days after 0.11.0 is published would answer it. `/cogmer:self-status` in a cloud task could say when its plugin is older than the newest release, which costs a request to the release host from the sandbox. Anthropic could be asked how the automatic sync is scheduled.

**A Cowork turn shows the user's computer as its machine, and Desktop's name for the computer and
cogmer's name for the machine are two names.** Every turn carries the name of the machine that
signed it (D-233, every turn carries the name of the machine it came from, as a field the user
asserts), and a Cowork turn is signed by the machine that runs the bridge, so a colleague reads the
laptop's name for a turn whose model ran in a remote sandbox. D-205 says a renamed computer needs
the setup run again, and a machine's name is now a label the user chose and renames from the
machines page (D-231, a user's machines are managed from one command that opens the daemon's page).
Whether the two names can differ, and what a colleague sees when they do, has not been looked at.

**The Cowork plugin and the marketplace plugin share the name `cogmer`, and the bridge a Cowork plugin needs exists only on a machine whose Desktop was set up.** On 2026-10-28 Claude Desktop on a Mac listed both plugins and offered `self-status` and `self-stop` twice each, and `/self-status` in a Cowork-mode conversation answered that cogmer did not answer (`56aa259`). That Mac's Desktop configuration had no `cogmer` entry (`d306243`). Anthropic's plugins article says a plugin added in Desktop is saved to the account and reaches Claude Code from v2.1.273 (read on 2026-10-28), so the zip's hooks run on every machine of the account and give a non-blocking error where no bridge runs, as the item about a bridge that is down describes. Naming the zip's plugin differently, such as `cogmer-cowork`, ends the duplicate commands and adds a second command prefix (D-118, the plugin manifest name is the command namespace). One plugin for every surface, which the next item asks about, would end both problems. On 2026-11-20 a personal plugin named `cogmer` uploaded on the account left the marketplace plugin of the same name listed as disabled, and removing the upload enabled it again (`db8eced`), so the bridge's plugin under that name would disable the plugin a Cowork cloud task needs.

**A hook reaches a server on the user's computer in a local Cowork session and not in a cloud one, so the Cowork plugin's hooks neither capture nor inject in a cloud session.** The plugin that `cogmer cowork-setup` writes calls the bridge through the server `remote-devices` (`coworkDeviceServer` in `cmd/cogmer/cowork_setup.go`). On 2026-11-19, in a cloud session of Claude Code 2.1.329-ccs, a hook that named `remote-devices` with the tool `cogmer-host-probe__ping` reached nothing at `UserPromptSubmit`, `PostToolUse` or `Stop`, while the model's call of the same tool reached the Mac, and Claude Code 2.1.328 refuses such a hook wherever `CLAUDE_CODE_REMOTE` is true (read in the code on 2026-11-19, `263fd33:docs/work/cowork-connector-probe-20261119T134000Z.txt`). On 2026-10-31 a plugin's bundled MCP server ran on the user's computer in both kinds of session. The line in D-205 (Claude Cowork is a supported host through a bridge on the user's machine) that "a server that a plugin bundles would start in the sandbox" is contradicted by a probe on 2026-10-31 on a Mac (`78477ee`). In the cloud session the app started the server at about the time the plugin was enabled, before the session, and the hook that names `remote-devices`, the server through which a cloud session reaches the computer's servers (a name seen, not documented), with the tool `plugin_<plugin>_<server>__<tool>` reached it, while the hook that names `plugin:<plugin>:<server>` did not. In the local session a Claude Code process started it anew for each prompt, the plugin-named hook reached it, and the session recorded a non-blocking error on every prompt, which fits the `remote-devices` hook. Whether the other name errors in a cloud session was not seen. Anthropic's support articles say the same (read on 2026-10-31): a cloud session reaches "Local MCP servers, including plugins that bundle them" through the desktop app, which has to be open, and on Pro, Max and Ultra plans every new Cowork session runs in the cloud, where on this Team plan a setting chooses, which the article calls "Run new tasks in the cloud" and this Mac's Settings > Cowork shows as "Only on this computer".

A plugin that carried both hooks would leave an error on every prompt in at least the local session, and the hook that names `remote-devices` now serves no cloud session. If one plugin carried both, it could start the daemon with the server and need no Desktop configuration, and typed commands could use the path Cowork uses (B33, a project or plugin command reaches UserPromptSubmit with the text as typed). Whether a hook can be limited to one kind of session is not known. Not known either: whether Cowork asks for a plugin's `userConfig`. The probe also showed where command hooks run: in the local session on the Mac, where a cogmer plugin's `session-start.sh` ran too, and in the cloud session on Linux x86_64 with `CLAUDE_CODE_REMOTE` true, shown by the file that `relay.sh` wrote there (`78477ee`).

**A person with Claude Desktop and neither Claude Code nor Codex cannot install cogmer for Cowork.** The README says cogmer "has to be installed on that computer first, through Claude Code or Codex", since that fetches and verifies the program, and that setup is a command in one of them (read on 2026-10-29). `cogmer cowork-setup` writes the Desktop entry and a zip whose files depend only on the entry name, the constant `cogmer`, and the version, so by `coworkPluginFiles` and `writeZip` in `cmd/cogmer/cowork_setup.go` it is the same bytes on every machine for a release (read on 2026-10-29). Why D-205 (Claude Cowork is a supported host through a bridge on the user's machine) says a renamed computer needs the setup again is not clear from that code. Cowork on Windows has not been shown to run (`4dfaa00`), and the item above says why Cowork testing waits.

There are four ways to reach such a person, each with a cost. Requiring Claude Code or Codex costs no work and serves no Cowork-only user. A Cowork plugin built by `release.sh`, attached to the release and pinned in `checksums.txt` beside the binaries, could be named differently, such as `cogmer-cowork`, which ends the duplicate commands. It costs a command prefix that differs from Claude Code's (D-118, the plugin manifest name is the command namespace), a compatibility rule between the zip and the program, since one command no longer writes both, and a second route beside `self-cowork`. An MCP Bundle, a zip that Claude for macOS and Windows installs with one click and whose manifest takes a platform-specific binary (the `anthropics/mcpb` manifest and README, read on 2026-10-29), could carry the bridge with no Git, Claude Code or Codex. It costs a program that lives in the bundle's folder, not `~/.cogmer/bin`, which D-066 (the binary is fetched and verified, never shipped in the plugin) and D-215 (the installer and the daemon replace a program only with a newer version) do not cover, and nobody has shown that Cowork reaches a server installed that way. A setup program the user downloads would be a new Windows installer written in Go, and costs an unsigned executable that Windows warns about, since signing is deferred. A throwaway bundle with a stand-in server, installed on a Mac and called from a Cowork conversation, would answer the bundle question, and running `cowork-setup` on two machines with different names and comparing the zips would answer the one about the name.

A cloud session is no way around this. Its value comes from the machines page of one of the user's machines (D-246, a cloud environment is added from the machines page), and it takes the user's lists only from one of the user's machines (D-245), so a person with Claude Desktop alone needs the hosted machine of D-285 (a user with no computer of their own has a machine hosted for them), which nobody has built. What it leaves open is the next item.

**A hosted machine (D-285, a user with no computer of their own has a machine hosted for them) has no way yet to hold the user's master key, no operator certificate, and no sign-in, and every user keeps the master key as a recovery phrase.** The master key certifies a new shared key, which is how a user shuts out a machine that holds the old one (D-219, a user is a master key that certifies machines through a shared key; D-220, a removal names the last event of a machine that counts), so for a hosted user it is the only recourse against the operator. It is used only at recovery and after a removal. A phrase typed or shown in a page the operator serves is the operator's too, and a user of any kind keeps a phrase poorly or not at all. The ways to hold it, each with its cost:
- A recovery phrase, as today.
- A passkey whose PRF extension returns a secret that only that passkey produces, from which the page derives the Ed25519 master key, so records, verifiers and colleagues' checks stay as they are. Safari 18 with iCloud Keychain, Chrome, Firefox on macOS and Windows, and Windows Hello return the secret, and Firefox on Android does not, by sources that disagree on versions (https://developers.yubico.com/WebAuthn/Concepts/PRF_Extension/Developers_Guide_to_PRF.html and https://blog.hakhub.net/en/blog/webauthn-prf-passkeys/, read on 2026-11-18). The master key exists in the page's memory while it signs, so on a hosted machine's page it is the operator's, and a WebAuthn editor advises against a PRF secret as the only key, since losing the passkey loses the key (https://lilting.ch/en/articles/passkeys-prf-extension-encryption-risk, read on 2026-11-18).
- A passkey that signs a record's hash in place of a random challenge, as wallets that hold funds under a passkey do (https://docs.candide.dev/wallet/plugins/passkeys, read on 2026-11-18). The key never reaches the page. The prompt names the site and the account and never what is signed, and the page that asks chooses the record, so on a hosted machine a user can sign a record the operator chose. Passkeys sign with P-256, which only a new version of the records could carry.
- The operator holds the master key, and the user signs in to the operator. The user then has no recourse against the operator but cogmer revoking it, which reaches no machine the operator certified as an ordinary one.

A passkey belongs to one site for life, a site is a domain name and never an IP address, and a page may name any parent of its own name as the site (https://web.dev/articles/webauthn-rp-id, read on 2026-11-18). The daemon's page at `127.0.0.1` can therefore use no passkey. At `localhost` it could, but every page on `localhost`, whatever its port, could then ask for the same passkey, Safari failed every passkey prompt on `*.localhost` names in one project (https://github.com/znasllc-io/memql/issues/6233, read on 2026-11-18), and a hosted page could not share it. A name under the project's domain `cogmer.ai`, such as `local.cogmer.ai`, mapped to `127.0.0.1` by an entry in the hosts file that the installer asks the user's permission to write, would let the self-hosted page and a hosted page at a sibling name share one passkey under the parent domain, with no other local page able to ask for it. That page needs a certificate the browser trusts. A certificate authority made on the user's machine and trusted in the same step, limited by a name constraint to that one name, needs no service, and adds a root to the machine's trust store, which Firefox keeps separately and macOS asks the user to approve. A certificate cogmer issues for each user under a name of the user's own needs no root, and makes a cogmer service part of every self-hoster's setup and renewal, which D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person) would have to record. The project registered `cogmer.ai` on 2026-11-19, and releases are still served from GitHub (`plugin/release-url.txt`). Losing the domain, such as by letting its registration lapse, would end every passkey made under it.

A phone takes part only as the authenticator for a page on the computer, through the browser's offer to use a phone, which scans a code and checks that the two are near over Bluetooth, or not at all, when the passkey syncs to the computer through the user's Apple or Google account. Recovery on a computer outside that account, such as a Linux laptop, needs the phone's route or the phrase. The phrase would therefore stay the root that generates the master key, with a passkey's secret unlocking a copy for daily use.

A record the master key signs could count only after a delay, such as 48 hours, during which the user's cloud session and colleagues' views show it and the user can cancel it, so that a record signed by mistake is seen before it takes effect. A removal would take effect at once, since it only shuts machines out, so that a user who loses a machine cuts it off without waiting. A page at such a name on a Mac with a name-constrained local root would answer whether Safari, Chrome and Firefox return a PRF secret there, whether an iPhone answers it through the phone route, and whether each browser honors the name constraint. Not decided either: who holds cogmer's root key, how a revocation reaches every daemon, which could ship with each release beside `checksums.txt`, how a user signs in to the hosted machine's page, who operates the first host, and whether the hosted machine serves the room to a phone, which the item about a user on a phone asks.

**Nothing decides how hosted machines run or how each one's page gets its address.** A hosted machine (D-285, a user with no computer of their own has a machine hosted for them; D-287, a user with a computer of their own can also have a machine hosted) answers for its user while every other machine of theirs is off and keeps every turn of the user's rooms, by §6 (identity). The daemon serves one user, whose state is in one directory that `COGMER_HOME` names (`homeDir` in `cmd/cogmer/identity.go`, read on 2026-11-20). A hosted machine does not let a Cowork sandbox skip the allowlist, since the sandbox's daemon still reaches it through the relay. The listener for other machines' browsers and the remote sign-in it needs would be off on a user's own computer, whose page answers only on that computer, so that a user who runs cogmer themselves configures nothing new. The choices, each with its cost:

- One daemon process for each user, with its own state directory, or one process serving many users. One process for each user needs little new code, and a fault reaches one user's keys and turns, at the cost of each user's memory all the time. One process for many users is cheaper for each user, needs the daemon rewritten for many users, and lets one fault reach every user's keys and turns.
- What isolates one user's process from another's: a container for each user on servers cogmer runs, which share a kernel; a microVM for each user, such as Fly Machines or Firecracker on cogmer's servers, which isolates more and gives each user a volume that can be stopped and started, at the cost of a provider or more tooling; or Kubernetes, which is the most moving parts for a service this size.
- Always running, or started when traffic arrives. A machine that is off cannot answer for its user or keep a turn, and tailcat's relays cannot wake one, so starting on demand needs a relay cogmer runs (the item "A relay change would break addresses users already hold").
- Storage: each user's SQLite files on a volume of the user's own, with encrypted copies in object storage. A shared database would need the store rewritten.
- Running hosted machines in the same region as a relay cogmer runs, which shortens the path from a sandbox's daemon through the relay to the hosted machine.
- An operator other than cogmer runs the same image on servers of its own with its own certificate, so the packaging cannot depend on one provider.

A hosted page is best given its own name under one domain, such as a random identifier under `u.cogmer.ai`, rather than a handle or an email address. A passkey belongs to one site for life (https://web.dev/articles/webauthn-rp-id, read on 2026-11-18), so a page whose own name is its passkey's site can never be renamed without the user's passkey being lost. A name derived from the user's identifier lets anyone who knows the identifier find out whether the user is hosted. Let's Encrypt submits every certificate it issues to Certificate Transparency logs (https://letsencrypt.org/docs/ct-logs/, read on 2026-11-20), so a certificate for each page would publish a list of hosted users, and one wildcard certificate publishes only the domain. A wildcard needs Let's Encrypt's DNS challenge (https://letsencrypt.org/docs/challenge-types/, read on 2026-11-20), so whatever renews it holds a token for cogmer's DNS provider. Provisioning would then be: the user proves an email address at cogmer.ai, the control plane picks an identifier, starts the user's machine and adds a route to it, and one wildcard DNS record points every name at a proxy that holds the certificate and forwards each request by its name. No DNS record and no certificate is made for each user. Two choices follow:

- The passkey's site is the user's own page name or `cogmer.ai`. The page name confines each passkey to one user's page, so a fault in one page cannot ask for another user's passkey, and the name can then never change. A page-name site also keeps the hosted page from sharing a passkey with a page at `local.cogmer.ai` on the user's own computer, which the item on the master key proposes, so a user with both kinds of machine would need two passkeys. `cogmer.ai` lets the two share one passkey, lets sign-in happen in one place and survive a new name, and lets any page under `cogmer.ai` ask for any user's passkey.
- TLS ends at the proxy or in each machine. At the proxy, one certificate serves every page and the proxy reads the page's traffic, which costs little where cogmer also runs the machine behind it. In each machine, each machine needs a certificate of its own, which the logs publish, or a copy of the wildcard key.

**A hosted machine's cost is almost all polling, which puts the ceiling of a 4-vCPU server, with every vCPU busy, at about 76 users, and its bytes on the network, its growth and its isolation are not measured.** A hosted machine (D-285, a user with no computer of their own has a machine hosted for them) is a daemon for each user. On 2026-11-25 on the droplet, a user with 3 rooms, a different colleague in each and one turn a minute in each room, cost the box 189 CPU seconds an hour and 29MB of memory, and the user's daemon sent 67.5MB an hour (`f98fa3d:docs/work/hosting-fitness-runs-2026-11-25.md`). With no turns, polls took 156 of the daemon's 160 CPU seconds an hour: each daemon polls each colleague address every second, and each poll opens a new connection with a TLS handshake (`clientFor` in `cmd/cogmer/transport.go`), at about 15ms of CPU and 6.4KB each way. With the overlay on, each colleague address added 42MB, so memory limits a server before CPU does. The cost of one user did not change between 4 and 20 users on the box. The relays cogmer uses are free and rate-limited (D-192, the public relay is an accepted dependency). The hook and page address must be loopback (`cmd/cogmer/main.go`, read on 2026-11-24), so each hosted page also needs a proxy or a change to that rule.

A longer interval for a peer that has sent nothing, a long poll, a connection kept between polls, or one connection for many daemons would each cost a change to sync and a decision, and the measured terms price each. Still to measure: 6 hours of turns, for disk and memory growth; a daemon under an OS user of its own and in a container; the overlay with turns; bytes to a colleague on another machine; and an ARM server. The terms, the users each server holds, and the plan are in `docs/work/hosting-fitness.md`.

**A user in Chat mode captures and shares nothing, and the README does not say so.** The README's Cowork section describes Cowork alone. Anthropic's plugins article says hooks are grayed out in Chat, and one message in Chat mode reached no program on a Mac (`d306243`). A README sentence would name Chat mode, and say that a merged conversation shares nothing before its first turn that needs Claude Code, as the item on merged conversations below says.

**No behavior check records that a `Stop` hook with an empty reply ends the turn, and a stop reply that carried context restarted a turn eight times.** In a Cowork-mode conversation on a Mac, a probe's `Stop` hook returned the JSON form of `additionalContext`, and nine `Stop` calls followed in 25 seconds (`d306243`). The hooks reference caps consecutive continuations by stop hooks at eight and does not say that a reply's `additionalContext` continues a turn. A behavior check for an empty reply would need a probe turn, and no check should call `Stop` with context.

**A Cowork session in no room sends every prompt and response to the daemon.** Its hooks
are calls through the bridge (D-205, Cowork through a bridge on the user's machine), not
`plugin/hooks-handlers/run.sh`, so the check that ends the hooks of a Claude Code or Codex
session in no room never runs for it (read from `cmd/cogmer/cowork_setup.go`,
`coworkHookFile`, on 2026-11-16). The fix would be the same check in the bridge, before it
relays a call, which needs the bridge to read `bound-sessions/` under the state directory
of the daemon it relays to.

**A Cowork session that runs on the user's computer captures and injects through the plugin's hooks and cannot run any `/cogmer:` command without the bridge.** On 2026-11-18 on a Mac with Claude Desktop 5.11492.5, with "Only on this computer" on, the cogmer plugin's hooks ran on the Mac, a prompt hook's plain output reached the model, a session put in a room from a terminal had its prompts and replies captured, and the model quoted a colleague's turn word for word (`e7ddab7`). A command's `!` line printed "[shell command execution disabled by policy]", so `/cogmer:room-join` did not run, and the model's shell tool ran in a Linux aarch64 sandbox with no cogmer that could not reach the Mac's daemon (`e7ddab7`). Anthropic's skills page says a Cowork session on the desktop replaces every `!` command line with that placeholder, for every skill the user supplies, and names no setting that lets a user turn it off (https://code.claude.com/docs/en/skills.md, read on 2026-11-18). The Claude Code program decides this before it reads any setting. In Claude Code 2.1.303 and 2.1.327, read on the droplet on 2026-11-18, a session that Desktop marks with `CLAUDE_CODE_IS_COWORK` replaces every `!` line of a plugin's command whatever `disableSkillShellExecution` says, the setting can only turn the replacement on, and the code that loads a plugin's commands does not ask how the plugin was installed. The managed skills the page exempts are skill files in the managed settings folder, so an organization that deploys the plugin as managed does not exempt the plugin's commands. A Cowork user on their own computer is therefore reached for capture and injection with no bridge, no Desktop configuration and no second plugin, and cannot create, join or leave a room, name a colleague or check status from the session. This does not replace D-205 (Claude Cowork is a supported host through a bridge on the user's machine) for commands.

The options for commands, each with its cost:
- Keep the bridge for commands alone. A user then installs it as D-205 describes and only for the commands.
- Expose the commands as tools of an MCP server that the plugin bundles. Anthropic's page on plugin support says a local MCP server loads "when the Cowork session runs on your computer" (https://claude.com/docs/plugins/platform-support.md, read on 2026-11-16), and one started on the Mac on 2026-10-31 (`78477ee`), which is one run. It needs the server, a test of each tool, and a behavior check that the server still loads, and it might need no Desktop configuration.
- Offer creating, joining and leaving a room on the daemon's page. It needs no second plugin and no Desktop configuration, and the page cannot tell which session a user means to put in a room, so it would have to list the sessions it has heard from.

Local sessions stay supported although the maintainer's sessions ran in the cloud on 2026-11-19 with "Only on this computer" on (`263fd33`), since one organization's settings do not show what other users get. Where sessions run is told by the session-start hook's environment: a local session's Claude Code reported Darwin, the user's home and no `CLAUDE_CODE_REMOTE` (`e7ddab7`), and a cloud session's reported `host=vm`, Linux x86_64, `CLAUDE_CODE_REMOTE` true and `CLAUDE_CODE_ENTRYPOINT` `remote_cowork` (`9c88af4`). The local run's record does not say whether "Chat and Cowork unified" was on, and every cloud run had it on, so that setting may decide where a task runs; Anthropic's article says tasks in the unified experience run in the cloud (https://support.claude.com/en/articles/15811196, read on 2026-11-18). Setups not yet seen: an organization with unified off, Pro and Max, Enterprise, Windows, and a Mac with Intel.

Not known: whether the plugin starts a daemon in such a session, since the daemon was started first; which cogmer plugin ran, and whether a session with both the marketplace plugin and the bridge's plugin sends each turn to the daemon twice, once from each plugin's hooks; a session outside a project folder; whether a bundled server's tools run on the Mac with the same access as a hook. The steps are probe 1 in `docs/work/host-probes.md`, and a second run with both plugins installed would answer the first three.

**Only a daemon in the sandbox has been proven to capture a Cowork cloud session, and it reaches the relay only where an organization owner allows the relay's hosts.** The cloud-session decisions put a daemon in the sandbox: D-216 (a cloud session signs with a key of its own that the user's shared key vouches for), D-217 (a cloud session's daemon carries its relay connection as a WebSocket) and D-278 (the machines page offers the cloud value as a plugin for the user to upload once). In a Cowork cloud session that daemon reached the relay as a WebSocket (`7adc75c`). It reached the relay only after an organization owner allowed the relay's five hosts (`ce962eb`). A detached process that a hook started stopped about two and a half minutes after the last response (`c72ef75`), so a daemon there would have to be started again at each session and resume. The daemon's cloud key is readable by the model's shell, as D-216's Limits say. Since 2026-10-06 every new Cowork task on Pro, Max and Ultra runs in the cloud (https://support.claude.com/en/articles/15520349, read on 2026-11-18), and a Team organization moves to the unified experience by 2027-02-01 (https://support.claude.com/en/articles/18342806, read on 2026-11-18), in which tasks run in the cloud with no option to run them locally (https://support.claude.com/en/articles/15811196, read on 2026-11-18). We infer from these two changes, rather than from any count, that most Cowork users will be in cloud sessions. A Claude Code cloud session keeps the daemon in the sandbox either way, since its user sets the environment's network.

No hook in a Cowork cloud session reaches a server outside the sandbox that no owner allowed. On 2026-11-19, in cloud sessions of Claude Code 2.1.329-ccs on the Team plan, a plugin's `mcp_tool` hooks at `UserPromptSubmit`, `PostToolUse` and `Stop` reached neither a connector the user added in claude.ai nor a server on the user's computer through `remote-devices`, under the server names the session itself reported, while the model's own calls reached both and the same hooks reached the connector in Claude Code 2.1.328 on a droplet (`263fd33:docs/work/cowork-connector-probe-20261119T134000Z.txt`). Claude Code 2.1.328, the version found in the sandbox, runs a plugin's `mcp_tool` hooks in a cloud session only when they come from a repository's project settings, which a Cowork session has none of, and refuses a hook's call to `remote-devices` wherever `CLAUDE_CODE_REMOTE` is true, because a hook's call skips the permission check (read in the code on 2026-11-19, `263fd33:docs/work/cowork-connector-probe-20261119T134000Z.txt`). Neither an `http` hook nor a server a plugin declares by URL reached a host no owner had allowed, and the model was never offered the plugin's server. A `Stop` hook of type `agent` ran, and its model reported having Bash, Edit, NotebookEdit, Read, Write and the tool it answers with, and no MCP tool. Only the model's own tool calls leave the sandbox without an owner's allowance, so a route through a connector would have the session make a tool call that its user did not ask for at every turn.

What would serve more users, each with its cost:
- The daemon in the sandbox, which is built. It serves a Team or Enterprise organization whose owner allows the relay's five hosts, and nobody else until a Pro or Max session is shown to reach the relay.
- A request to Anthropic to run a plugin's `mcp_tool` hooks to a connector in a cloud session. A connector carries no secret into the sandbox and needs no owner, and its server would be one each user runs or one an operator runs for them (D-285, a user with no computer of their own has a machine hosted for them). It depends on Anthropic. The refusal of `remote-devices` is stated as a rule about permissions, so a request for the computer's servers is unlikely to succeed.

OpenAI's pages say plugin hooks do not run in cloud-orchestrated ChatGPT Work, which supports only MCP hooks that a workspace admin configures, and that personal accounts do not use them (https://learn.chatgpt.com/docs/hooks and https://learn.chatgpt.com/docs/plugins, read on 2026-11-18). An MCP route is therefore the only one into a ChatGPT Work cloud thread, and only for an organization.

Not known: the code of 2.1.329-ccs, which decides which hooks run and is not in the sandbox; whether an organization connector behaves as an individual one did; whether a Pro or Max cloud session can reach the relay; and which fields a ChatGPT admin's MCP hook passes. D-205's rejection of the daemon in the sandbox, that it "cannot reach a relay", is not true where an owner allows the relay (`7adc75c`).

**The first turn of a Cowork cloud session reaches UserPromptSubmit with an empty prompt.** On 2026-11-18 in the Team organization with "Chat and Cowork unified" on, a conversation stayed a chat, with no session and no hook, until a turn ran a plugin command or asked for a shell. Desktop then made a Cowork cloud session 1 to 3s after the send, and the session's hooks ran from that turn on: SessionStart had source `startup` when the command was the conversation's first message and `resume` when chat turns came before it (`ffe2f33`, `9c88af4:docs/work/host-probe-cloud-sessions-20261118T180600Z.txt`). On that first turn PreToolUse ran before UserPromptSubmit, and UserPromptSubmit carried a prompt of 0 characters, while in `ffe2f33` the first prompt the hooks saw was a 1198-character message that the model said continued the task. On 2026-11-19 the first turn's prompt was again 0 characters, every later prompt reached the hooks with its text, among them a plain chat message sent from the mobile app, and a session that started again in the same conversation after 12 idle minutes received its first prompt with 140 characters (`a6cffcc:docs/work/host-probe-phone-and-mid-conversation-20261119T051000Z.txt`).

The chat turns before a session are not a gap. Joining a room takes a command, which is itself the turn that makes the session, so those turns always come before the room, and §12a (room membership) begins a room's history at its creation and shares nothing a session did before. The session's transcript holds them all the same (`a6cffcc`), and capture leaves them there. The empty prompt loses nothing when the first turn is the room command. Not known: whether a session that starts again in a conversation stays in the room its earlier session joined, and what the 0-character prompt stands for.

**A user on a phone has no way to see the room's conversation.** A user on a computer reads the room on the daemon's page at 127.0.0.1. A session started on a phone, in Cowork or in Claude Code, has a daemon only in its sandbox (D-217, a cloud session's daemon carries its relay connection as a WebSocket), whose page no browser outside the sandbox reaches, so colleagues' turns reach the model as context and never the person. A user who works from a phone can therefore only ask the model what colleagues said, which paraphrases them and costs a turn. Anthropic's page on mods says what a mod draws appears in the terminal, the desktop app and VS Code, that over Remote Control from the mobile app it appears only in the terminal on the user's machine, and that a cloud session draws nothing (https://code.claude.com/docs/en/plugins/mods/overview.md, read on 2026-11-18). Claude Code 2.1.326's mod types name a `mobile` surface with a sheet of mods (read on 2026-11-18), and the page needs Claude Code 2.1.287 in a terminal and 2.1.286 in the desktop app, where the Cowork cloud sessions of `9c88af4:docs/work/host-probe-cloud-sessions-20261118T180600Z.txt` ran 2.1.42.

The options, each with its cost:
- A mod that draws the room from the sandbox daemon, once the mobile app draws mods for a cloud session. It needs that support, which the page does not announce, and a Cowork session on a version that has mods.
- The daemon's page served to the phone's browser through the relay. It needs an address on the internet, which exposes the room or puts a component somebody else operates between the user and the page.

A hook's `systemMessage` and a file in the session's outputs folder reach nobody. On 2026-11-19 a plugin's hook returned a `systemMessage` and added a line to a file in `/mnt/attach/outputs` at every prompt of a Cowork cloud conversation, and the transcript recorded each message, but neither Desktop nor the mobile app showed the message or the file, including at a prompt sent from the phone (`a6cffcc:docs/work/host-probe-phone-and-mid-conversation-20261119T051000Z.txt`).

**No reply from a Cowork cloud task is captured, because cogmer refuses the transcript path its Stop hook sends.** On 2026-11-24, in a Cowork cloud task on Claude Code 2.1.331 with the staged 0.11.0, the sandbox's `daemon.log` held "sent a transcript path cogmer will not open (transcript path refused: not a .jsonl name); the turn is not captured" after each of five turns, and the room held no reply from the task (`docs/work/staged-0.11.0-run-20261124T222000Z.txt`). `openTranscript` in `cmd/cogmer/transcript.go` checks the name's extension first, so an empty path or any path without `.jsonl` gives that line, and the line does not say which. The sandbox's `~/.claude/projects` held only the behavior check's probe transcript and nothing for the task's session, while a record from 2026-10-31 has a readable transcript there for the hooks of a Cowork cloud sandbox (`7adc75c`). What path the hook sends now is not known. D-262 (a transcript is opened only when it is a regular jsonl file) rejects a rule that the path sit under the host's transcript folder, because a cloud session and Cowork keep theirs where cogmer cannot name in advance, and a refused path fails open by design.

The options, each with its cost:
- Make the refusal line say whether the path was empty and what extension it had, and read the answer from another run. It costs a release and a run, and gives the form.
- Upload a probe plugin to the account that records `transcript_path` from the hooks of a task, as `docs/work/cowork-probe` did on 2026-10-31. It needs no release and costs an upload and a run.
- Change D-262's name test to a test of type and size once the form is known. It keeps the protection against a FIFO or an oversized file, and costs a change to a decision and to the tests.

**No run has shown that a Cowork cloud task's first prompt after a quiet spell reaches the room with the spool in place.** The sandbox's daemon is unreachable while the task idles, since the Mac's `daemon.log` recorded the cloud machine unreachable at 22:07:03 and reachable again at the next prompt, 4 minutes later (`docs/work/staged-0.11.0-run-20261124T222000Z.txt`). On 2026-11-24 a prompt sent at about 22:29:55 to the task while the daemon was asleep was never recorded, and one sent at 22:30:33 while it was awake was recorded at once, in a run driven through Claude in Chrome. The sequence numbers have no gap for the lost prompt, so it never reached the daemon, and the prompt hook makes one request with no retry, a second after the session-start hook starts a daemon that answers in about 0.85 seconds on a Mac (`cmd/cogmer/main.go`, read on 2026-11-24). D-294 (a prompt that finds no daemon listening just after one was started is held for it) has the hook write the prompt to a spool that the daemon records when it starts. Tests show the hook writing it, the daemon recording it once and dropping a stale one, and the start function marking a start. They do not show it in a task, where the daemon starts more slowly than on the Mac and the timing of the hooks was read from a log.

A run would stage a candidate with the change, let the task idle for four minutes with the daemon unreachable on the Mac, send one prompt, and look for it in the room, which is the test the browser route makes cheap. The window is the 15 seconds the marker lasts. If the sandbox's daemon takes longer than that to start, the prompt is dropped as before, and the limit needs lengthening.

**cogmer runs its behavior check in a Cowork cloud task, where it cannot work, and tells the user it failed.** In the task's `/cogmer:room-join` output and in the sandbox's `daemon.log`, "check: claude 2.1.331 could not run its probe: probe session failed: exit status 1" at 21:53:10, and the output said the probe session "hit an authentication error" (`docs/work/staged-0.11.0-run-20261124T222000Z.txt`). The check drives two Claude turns with `claude -p`, and the sandbox's Claude Code has no login of its own for that. It tries each time a version has not been checked, and collaboration goes on. A cloud session could skip it, as `COGMER_PREFLIGHT=off` does, since the check is of the Claude Code on the user's own computer; whether it is meant to skip there already was not read from the code.

**A Cowork cloud task's captured prompt begins with the host's own `<system-reminder>` text, and colleagues see it.** Both prompts the room recorded from the task began "<system-reminder>The user's timezone is Europe/Lisbon (UTC+00:00). Message sent at Tue 2026-11-24 21:55 local time.</system-reminder>" and then held what the user typed (`docs/work/staged-0.11.0-run-20261124T222000Z.txt`). That puts the user's timezone and the time of each message into the room, and into what a colleague's session is given. The hook's `prompt` field holds the reminder in this host. A leading block of that kind could be removed before a prompt is published, which costs a rule about text a host adds and an entry in the behavior registry that says the host adds it.

## ChatGPT Work as a host

A user of ChatGPT Work on their own computer shares a conversation with colleagues through the
Codex plugin cogmer already has.

**Whether cogmer's Codex plugin captures and injects in a ChatGPT Work thread is not known.**
OpenAI's page on hooks says that "Local-only Work and Codex threads continue to support their
existing hooks", that plain text a UserPromptSubmit hook writes "is added as extra developer
context", that Stop carries `last_assistant_message`, and that a plugin's hooks are reviewed and
trusted before they run. It also says that Work Cloud without local computer access does not
run them (https://learn.chatgpt.com/docs/hooks, read on 2026-11-16). Those are the events
cogmer's Codex hooks use (D-201, Claude Code and Codex are supported hosts).

A run would install the plugin, trust its hooks in a local Work thread, and check the room
commands, capture and injection with a colleague. It would also show whether a Work thread
carries `CODEX_THREAD_ID` (B28, `CODEX_THREAD_ID` in a shell tool equals the hook session
identifier) and whether a typed `$cogmer` command reaches the hook as Codex's does. If they
work, a ChatGPT Work user is reached with no new design, and the README and §29 (the
experience we want) would name ChatGPT Work as a way in. The steps are probe 3 in
`docs/work/host-probes.md`.

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

**A daemon switches its relay connection to a WebSocket only when `CLAUDE_CODE_REMOTE` is `true`, so no network evidence moves it.** The relay's own protocol starts as an HTTPS request that is switched into a stream, and a proxy that passes ordinary requests but not that switch stops it. `relayAsWebSocketInCloud` in `cmd/cogmer/tailcat.go` sets Tailscale's `TS_DEBUG_DERP_WS_CLIENT` before any relay connection in that case only (D-217, a cloud session's daemon carries its relay connection as a WebSocket), and a cloud session was observed to need it on 2026-10-18 (`e415f1d`). A host that renames the variable, or a Mac or Linux daemon behind a proxy of the same kind, would not switch and would not reach the relay. A Windows daemon cannot switch at all, since Tailscale compiles the WebSocket client only for Linux and macOS (`derp/derphttp/websocket.go` in the Tailscale module that `go.mod` names, read on 2026-10-28), and a Windows daemon reached the relay with the relay's own protocol on 2026-10-29 (`0e0c629`).

Tailscale built its WebSocket relay client for browsers, presumably because a browser cannot open the relay's switched connection: the commit "cmd/derper, derp/derphttp: add websocket support" of 2021-10-22 cites its issue "Run Tailscale in the browser", and the commit "derp/derphttp: don't link websockets other than on GOOS=js" of 2024-11-08 links the client only for browsers or the `ts_debug_websockets` tag, citing its issue about binary size (both in `tailscale/tailscale`, read on 2026-10-28). So the switch is an upstream debug setting made for another purpose. HTTP marks a protocol switch hop-by-hop, so an intermediary does not forward it unless built to (RFC 9110 section 7.8, read on 2026-10-28). Other networks that inspect TLS have been reported to break such connections: Tailscale's firewall page says FortiGate application control rules with certain configurations intercept a client's HTTPS connection to the control plane, so the client cannot connect and shows a certificate verification error, and a Fly.io user behind a TLS-inspecting network got "expected handshake response status code 101 but got 200" from a WebSocket dial on 2026-08-31, with the cause left open (read on 2026-10-28). Anthropic's cloud environments page says a session's outbound traffic passes through an HTTP/HTTPS proxy "for security and abuse prevention purposes", which blocks malicious requests and limits rates, and that an agent proxy adds a network secret to a request after it leaves the session's VM (read on 2026-10-28). That suggests the proxy reads each request, which would explain why it does not carry a protocol switch it has no rule for, and the page does not say so or mention switches.

Five ways a proxy can break the switch are plausible, from general knowledge and not checked: it recognizes only some upgrade protocols and rejects `DERP`; it parses every message as HTTP and closes the connection on bytes that are not; it strips the hop-by-hop `Connection` and `Upgrade` headers; it accepts HTTP/2 from the client and speaks HTTP/1.1 upstream, where the HTTP/1.1 switch has no counterpart; or it buffers, or closes a connection that stays open. The cloud session's proxy is a forward proxy at a local address (`e415f1d:docs/work/cloud-network-probe-20261018T163431Z.txt`) and passed a WebSocket. That fits the first way best, rules out stripping the headers from every request, and fits the second only if the proxy handles WebSocket on purpose. Whether the HTTP/2 way applies is not known. The last would stop a WebSocket as well, so a fallback would then need keepalives. A probe from a cloud session that sends the relay host `Upgrade: websocket`, `Upgrade: DERP` and a made-up protocol name, logs each status line or timeout, and then holds an idle WebSocket open for several minutes would say which, and would extend `docs/work/cloud-network-probe.sh`.

A fallback that is sticky and revisited would close the first two of those cases, and not the Windows one. After the relay's own protocol fails to connect within a timeout, the daemon sets the variable for the process and resets its cached relay clients (`tailcatDialer.Reset`), and it tries the relay's own protocol again after a while, so a switch made on false evidence ends. The variable stays the fast path for a cloud session. It costs one timeout in a restricted network, 10s being what the pings in `e415f1d` waited. It also costs a switch whenever a relay or a peer is down, which puts that daemon on a Tailscale setting with no stability promise until the retry, and a test needs a proxy that refuses the switch, which nobody has built. D-217 rejected a fallback that waited on every connection, and this one waits once. Nothing is planned until a network other than a cloud session is seen to break the switch, or the variable changes.

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

**`TestTheViewShowsAHeldImageAndSaysWhenOneIsComing` failed once in a full run of the suite and passes when run alone.** On 2026-11-03 `go test ./...` failed in it for turns 3 and 4, which showed only `<p>x</p>` where they should have said that an image could not be read and that it is not a kind cogmer sends, while a second run of the suite passed and the test alone passed 30 runs in a row (`b6466a3`). The cause is not known. One lead is in the code: `orderBy` in `cmd/cogmer/store.go` sorts events by their timestamp as text, and its comment says the format is fixed, while the timestamps are written with `time.RFC3339Nano`, which drops trailing zeros, so an event whose fraction ends in zeros can sort after a later event that shares its leading digits. Whether that explains this failure has not been tried, and the failure showed missing notes, not turns in another order.

**`TestADaemonBlockedByAnotherCogmerSaysSoAndNamesStop` failed once in a full run of the suite and passes when run alone.** On 2026-11-16 in a Claude Code cloud session, `go test ./...` on the tree committed as `584326b` failed in it after 0.02s, the next full run passed, and the test alone passed three runs in a row at about 7s each. The failure's message was not kept, since the run's output was filtered to the lines that name a test. One lead is in the code, and has not been tried: the test runs in parallel with others, and `freeAddr` in `cmd/cogmer/testhelpers_test.go` closes the listener it opened before returning the port, so another test can take that port before this test's daemon binds it. Running the suite several times with its full output kept would show where it fails, and a helper that hands a daemon a port it still holds, or a retry when the bind fails, would end the race if that is the cause.

**`TestARoomsPeersAreOnlyItsOwnGuests` failed once in a short run of the suite and passes when run alone.** On 2026-11-24 the short suite on the tree committed as `8233685` failed in it with `you already know a different key as "ready-albatross"`, the test alone passed 20 runs in a row, and the next run of the suite passed. The likely cause is in the test and has not been tried. It makes three keys at random and records each with `Allow` and no name, so each takes the name `PeerName` in `cmd/cogmer/peername.go` derives from it. `PeerNameSpace` gave 8280 names on 2026-11-25, so two of three random keys share a name in about one run in 2760, and the second is refused as a different key under a name already known (D-074, a name means one master key, and a collision is where a new master key surfaces). Giving each key a name of its own in the test would end it, and any other test that records several random keys under their derived names can fail the same way.

**`TestACloudSessionAllOfWhoseMachinesRefuseItSaysSoOncePerSession` failed once in a full run of the suite and passes when run alone, and the cause may also leave a real cloud session untold.** On 2026-11-25 the whole suite on the tree committed as `f821ec0` failed at `cloudrefusal_test.go:77`, "a new run of refusals did not tell the session again", with the box's load average at 5.27 a minute later. The test alone passed 20 runs in a row, and the next run of the suite passed. The run's log shows the machine refusing the second request's handshake ("may not connect: machine ... has been removed") and no line "every machine of yours refused" after it, which `cloudRound` in `cmd/cogmer/cloud.go` writes when a round finds every machine refusing, so the session's side did not count the second answer as a refusal. One lead has not been tried: `isRefusal` counts only a TLS alert the other end sent (`refusedError`), and in TLS 1.3 a client can finish its handshake before the server has checked the client's certificate, so depending on timing the client can meet the refusal as a closed or reset connection instead of the alert. If that is the cause, a cloud session whose environment was removed is sometimes not told that its turns are not shared (D-253, a cloud session says its turns are not being shared when every machine it asks refuses it). Running the test many times under load with the client's error logged would show which error the client met.

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
