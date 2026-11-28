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

**The suite fails in a pre-push hook run from a git worktree, because a test's git commands inherit the hook's `GIT_DIR`.** On 2026-11-24 on the droplet, `scripts/stage.sh` pushed from a worktree made with `git worktree add`, and `.githooks/pre-push` ran the suite and refused the push on `TestGitSkipsTheStateDirectoryInABulkAddAndRefusesAnExplicitOne` (`cmd/cogmer/stategitignore_test.go`, "a bulk add staged" the repository's own files). The test passes with `GIT_DIR` unset and fails with it set to the worktree's absolute git directory, which is what git gives a hook run in a worktree, in runs on the droplet (`docs/work/staged-0.11.0-run-20261124T222000Z.txt`). In a normal clone the hook's `GIT_DIR` is `.git`, relative, so the test's own `git init` in a temporary directory takes it for its own. Any test that runs `git` has the same exposure. A helper that runs git for tests could clear every `GIT_*` variable, which costs a change to the helpers and a test that sets `GIT_DIR` and expects the same result. `scripts/stage.sh` can be run from a normal clone meanwhile.

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

**The next release moves every installed machine to user identities, and nothing yet says in what order the three installs move.** 0.11.0 waits until a passkey is the user's master key, so that no user ever holds a recovery phrase (Draft 6 of `docs/work/cowork-first-run-design.md`). On 2026-11-20 `host-probes`, which holds `user-identity`, is 281 commits ahead of `origin/main` and merges into it with no conflict (`git merge-tree`). The wire version is 3 and the state directory carries data version 1, where 0.10.5 has neither. The first start of the new program copies the databases and then migrates them (D-256, the databases are copied before the state directory is migrated, and a program that cannot migrate it stops). One program serves every host, and it serves the plugin of each of the last three releases, so a host whose plugin has not updated keeps working against the migrated data (D-269, the program serves the plugin of each of the last three releases). A 0.10.5 machine answered by an upgraded one is told to upgrade, and turns either side sends meanwhile reach the other once both have upgraded and made their user identities, with nobody pairing again (`TestContainerUpgradeKeepsAPairedRoom`, which passed on this droplet on 2026-11-20 with Docker 29.1.3 and `main` at `357af6b`, the 0.10.5 code, along with the other six container tests). The installs are the maintainer's and the two beta users'. A release would:

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

On 2026-11-27 cogmer chose to run its own relay under `cogmer.ai` and serve its map, so that an owner allows two entries, `cogmer.ai` and `*.cogmer.ai`, which also cover the hosted machines, and to move the addresses users hold before users multiply. Which form an address takes is not decided, and the options, each with its cost:

- An address of keys only, on one relay region that cogmer runs at cogmer.ai. Nothing a user holds then names a server, so cogmer can move, add or replace relay servers without any user noticing; a region can be several meshed servers in several places. Every user depends on the one relay network cogmer runs unless the address can also name another relay. §4 (networking) says no network provider is required, so an address that names no relay would mean cogmer's region, and an optional relay in the address would serve a user who runs their own relay or would rather not use cogmer's. The address format changes, which D-091 (identity is advertised and location is discovered) and the description of an overlay address in §4 would follow.
- A region number and a map that cogmer serves, which tailcat supports through its `DERPMapURL` option. Host names leave the address and a relay can move by changing the map, but each address still pins a region number, so retiring a region still breaks the addresses that name it. This option is the least new code and a step towards the first option, and every daemon fetches the map at start, from a host an owner then has to allow.
- Keys only and a lookup of which relay a key is on now, from DNS records or a service at cogmer.ai. This option is the most flexible, and the service is a central component that learns who looks for whom, which D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person) asks to be justified to the user.

Any relay at cogmer.ai makes cogmer the operator that sees each machine's network address and key, which machines connect, when and how much passes, as D-192 says the present relay does. The user has to be told, and only a benefit to the user can justify it. The relayed traffic of every cloud session is cogmer's bandwidth to pay for, and the relay must stay up and limit abuse. It would cut what an owner allows to one or two cogmer names, and it would run beside the servers cogmer would run for D-285 (a user with no computer of their own has a machine hosted for them).

A relay cogmer runs could also give the page that reads a cloud session's room a far smaller client. The page loads tailcat's browser build, 27 MB, which carries Go's runtime, WireGuard and a TCP stack so that the page can dial port 4784 of the overlay as a machine would (D-301, a cloud session's daemon serves its room on a second overlay port), and compiling it took 16 of the 19.6 seconds to first data on an iPhone's first open (the item on the page that reads a cloud session's room). If cogmer's relay joined a browser's WebSocket to a connection the daemon keeps with it, the page and the daemon could run a Noise handshake end to end with the browser's own cryptography, checked against the daemon's key that the link carries, and exchange D-301's lines inside it, so the page would carry no WireGuard, TCP stack or Go runtime. It costs a second protocol beside the overlay, which the daemon and the relay both speak, and handshake code on both ends that needs tests and review, and it works only through cogmer's relay. Whether WebCrypto's X25519 is in the browsers users hold is to be checked. A spike on the droplet would answer whether it pays: a relay endpoint that joins a browser WebSocket to a daemon's connection, a JavaScript client that completes the handshake and reads one line of a room, and the first open on an iPhone timed against 19.6 seconds.

On 2026-11-27 cogmer's relay was taken to replace tailcat's rather than run beside them, and a shape was proposed for it, not yet decided: one protocol through one relay for machines and pages alike, the relay joining two WebSockets as the spike's does and reading nothing, with addresses of keys only, and a machine reachable at a public name, which every hosted machine is, dialed there directly before the relay. A hosted machine has a name under `cogmer.ai` and a public address, so a computer, a cloud task and a page reach it with no relay, under the two entries an owner allows (the item on how hosted machines run), and a cloud user's traffic, which goes to their hosted machine for renewals and lists and to colleagues for their rooms, leaves the relay for the hosted half. Machines that are not reachable so, computers behind routers and cloud tasks, meet only through the relay, giving up the direct path the overlay finds between two computers when their networks allow one, which matters little for traffic of a few megabytes an hour (`docs/work/hosting-fitness-runs-2026-11-26-adaptive.md`). Peer connections keep TLS pinned to the colleague's key (D-101, peer connections are TLS pinned to a key this machine has recorded), carried through the relay or directly, and a page, which has no TLS client to run over a relayed connection, uses the spike's Noise handshake. A dialer learns that a machine has a public name only from the machine's record, signed under its user's shared key, which for a hosted machine certifies it as hosted (D-285, a user with no computer of their own has a machine hosted for them) and would carry its name, from which the waking name is derived. A cloud session has the records with its user's lists, a colleague's machine through pairing and sync, and a page from the passkey's user handle, which names a hosted machine. A dialer never probes for a direct path, and dials a machine whose record names none through the relay by its key. For a machine with a public name it dials the name and, if no connection is made within about a second, the relay as well, keeping whichever connects first, since an allowlist, a company's network or a sandbox's proxy can refuse the name. It remembers which worked for each machine, a cloud session's daemon in memory for the session, a page in its storage on each device and a machine with its kept connection (D-303, a machine keeps one connection to each address it polls), and one that fell back to the relay tries the name again after about ten minutes or a change of network. A machine with a public name also keeps its connection to the relay, for dialers whose direct path is refused. It reverses D-068 (tailcat is the cross-network transport, behind our own dialer) and rewrites D-104 (the overlay address holds no secret and is the same at every start), D-187 (the overlay's relay region is chosen once and stored with the node key), D-192 (the public relay is an accepted dependency), D-217 (a cloud session's daemon carries its relay connection as a WebSocket), D-301 (a cloud session's daemon serves its room on a second overlay port), D-288 (a Cowork cloud session that holds a cloud value starts the daemon), whose limits have the daemon reach a colleague only through the relay's hosts, and D-306 (a machine nudges the machines it shares a room with when it writes an event), whose limits have a cloud session nudge over its relay connection, and every address changes once, as moving to any relay of cogmer's already makes them. A second spike would show two machines exchanging a sync through the relay and a machine dialing a hosted one directly. Once the relay is built, its memory and CPU an hour while it holds 1, 100 and 1,000 idle page connections give the cost a hosted user takes for each hour a page is open, which the spike's relay, with no authentication, keepalive or limit on connections, would misstate. For a page that reaches a machine through it, the relay also holds the page's connection while the machine is stopped, and tells the page when the machine connects again, as the proxy in front of hosted machines does for a page that dials one directly (the item on how hosted machines run). The spike's relay closes the page's side when the machine's side closes.

A machine that starts advertising a new address moves the colleagues it reaches, since a stored address is replaced when a peer advertises another (`known_peers`, read on 2026-09-28), and an endpoint matters only until the first contact (D-018, identity and reachability are separate). Pairing strings and invitations not yet used when the old relay stops would fail.

A relay at cogmer.ai does not help where a sandbox's allowlist cannot be edited. Whether a Pro or Max user can add domains to the allowlist for a Cowork cloud session is not known, and running cogmer in a Cowork cloud session on a Pro account with the relay's hosts added would answer it. On a Team plan an entry `*.probe.cogmer.ai` reached `a.probe.cogmer.ai` and not `probe.cogmer.ai` (`docs/work/first-run-spike-C-20261126T1533Z.txt`), so the two entries are both needed.

**The stop hook takes longer as a session's transcript grows, because the daemon reads the whole transcript at every stop.** At each stop the daemon reads every line of the session's transcript to find the blocks the prompt hook injected, which is how it confirms delivery (§19, incremental context injection; `confirmDelivery` in `cmd/cogmer/daemon.go` and `InjectedBlocks` in `cmd/cogmer/transcript.go`, read on 2026-11-28 at `36162d5`). On 2026-11-28, with a turn a minute in each room, the stop hook's median grew from 0.067s at a quarter of a day to 0.247s at two days, when each session's transcript held 6.3MB (`docs/work/hosting-fitness-runs-2026-11-27.md`). A Claude Code transcript also holds every tool call and its result, so a working session's transcript grows faster than one of prompts and replies alone, and the session waits on its stop hook, which the values forbid slowing.

Reading only what follows the offset read at the session's last stop, and from the start again when the transcript is shorter than that offset, would keep each stop's work to the turn's own lines, at the cost of an offset kept for each session. Stopping once every pending injection is confirmed would also bound it, but a block missed at its own stop is confirmed at a later one, which a read that stops early must still find.

**A transcript that records a reply differently from what the session showed is captured wrong, with nothing to say so.** A turn's text is reassembled from the transcript and the Stop hook's payload (D-003, reassemble turns by unioning two incomplete sources), and nothing compares the two. Claude Code's hooks reference says the transcript "is written asynchronously and may lag" and that hooks needing a turn's final text "should use `last_assistant_message` on Stop … instead of reading the transcript" (https://code.claude.com/docs/en/hooks, read on 2026-11-28), and Codex's says its "transcript format isn't a stable interface for hooks and may change over time" (https://developers.openai.com/codex/hooks, read on 2026-11-28). Codex compressed, paginated and migrated its rollout files four times between 0.130 and 0.162, from 2026-05 to 2026-10. A simulation of 24 months of changes to both vendors' capabilities put a local Claude Code session's months of silent failure at 0.30 today and 0.24 with a check that publishes the payload's text and calls the transcript unreadable whenever its final text differs from `last_assistant_message`, and a Codex session's at 0.31 and 0.22 (`docs/work/capture-simulation/report.md`, with its assumptions).

The check costs about a week for each host and a behavior check of its own: the transcript's final text equals `last_assistant_message`. For Codex the simulation also counted delivery at Stop while the rollout file is unreadable, which D-202 (host-specific hooks use the shared peer protocol) rules out in its Limits, "Codex delivery has no trust fallback", so that half needs a decision.

**A behavior check runs when a room is formed under a Claude Code version the machine has not checked, not when a version is released, so a change waits for a user to meet it.** Claude Code published about 27 releases a month in 2026, as the npm registry's publish times for `@anthropic-ai/claude-code` show (counted in `docs/work/capture-simulation/report.md`). One change, hook output over 10,000 characters saved to a file from 2.1.89 on 2026-03-31, silently cut injection short for cogmer's users until they reported late turns on 2026-10-14 (`1df545d`). In the simulation, a month's delay before `cogmer doctor` meets a new version raised a local session's months of silent failure over 24 from 0.29 to 0.51 (`docs/work/capture-simulation/report.md`). A job that runs `cogmer behaviors` against each new Claude Code and Codex release, found from the npm registry and the GitHub releases API, would find a change before any user forms a room under it, at the cost of a scheduled job with Claude and Codex logins of its own.

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
different answer for close pairing than for long solo stretches. D-148 (arrival would
be announced by an operating-system notification) is the decision that would build one
on a computer, and a cloud session's daemon has no desktop to show it on.

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
the lifecycle needs, and nothing past that is built. D-183 (a receiver orders candidate
addresses by class and never by a configured sort) and D-091 (identity is advertised
and location is discovered) are the decisions that would build it.

**Whether to rebuild the post-quantum hedge.** D-104 removed the pre-shared key,
which was the only quantum-resistant element in the stack. It could be rebuilt at
our own layer from material the pairing exchange already produces — per peer, which
is better than one secret shared with everybody. Nothing depends on deciding this.

## First contact without a paste

A host can admit somebody who asks to join, without a pairing string being sent first.

**Whether a host approves an unsolicited join request.** A person who is not on a
room's guest list cannot ask to be let in; §12a (room membership) and D-051 (stranger
pairing is not a supported case) leave open whether they should be able to, and
nothing is built for it. D-144 (a refused peer's request goes to a host who is
present) is the decision that would build it.

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

**A Cowork-only user cannot start by installing the plugin: they also need a machine to hold their identity, the cloud value uploaded as a plugin, an allowlist their organization's owner edits, and each colleague verified with two words.** A user whose Cowork tasks run in the cloud and who has no computer running cogmer installs the plugin on their account (D-290, cogmer reaches a Cowork cloud session as the marketplace plugin installed on the user's Claude account), needs a hosted machine for their identity (D-285, a user with no computer of their own has a machine hosted for them), which is not built, uploads the cloud value as a plugin zip (D-278, the machines page offers the cloud value as a plugin for the user to upload once), on a Team plan needs an owner to allow the relay's hosts (`ce962eb`), and verifies each colleague with two words (D-054, verification gates synchronization and injection), while their first task says nothing (D-286, a Cowork cloud session with no cloud value fetches and starts nothing). Read on 2026-11-26.

On 2026-11-26 the spikes of `docs/work/cowork-first-run-spikes.md` found, on the Team plan: no route but the user's own upload puts a per-user file on an account (`docs/work/first-run-spike-A-20261126T1558Z.txt`); a plugin's options are asked for nowhere and reach a cloud task unset (`docs/work/first-run-spike-B-20261126T1533Z.txt`); an allowlist entry reaches only the names it gives, a wildcard missing the name without a subdomain, TLS goes through unchanged to the hosts it lets through, and nothing persists from one new task to the next (`docs/work/first-run-spike-C-20261126T1533Z.txt`); a hook's messages reach the model and are never shown to the person (`docs/work/first-run-spike-D-20261126T1533Z.txt`); Safari on the user's iPhone returned a passkey's same 32-byte PRF secret at its creation and at two sign-ins (`docs/work/first-run-spike-E-20261126T1650Z.txt`); and a page at `127.0.0.1` had a passkey sign its challenge through a page under `cogmer.ai` by navigation alone, in Chrome and Safari on the Mac, with the passkey signing on other names under `cogmer.ai` and on the user's iPhone (`docs/work/first-run-spike-F-20261126T194604Z.txt`).

The route these leave is in `docs/work/cowork-first-run-design.md`: when the user creates or joins a room in a task, the command prints the cloud page's link; the first time, the page asks for a home for their identity, a hosted machine, which onboarding suggests, or a computer of their own; and the user answers a passkey prompt, which signs the task's session key for three days, after which the user's machines renew it, so no value is uploaded or stored, and only a task that joins a room is ever asked for a tap. Five drafts there would become decisions: the passkey certifying and the user's machines renewing, the passkey asked for only by a command that needs the user's identity, the page served under `cogmer.ai`, the page remembering the user's machines with a directory of their addresses that cogmer runs for what it does not remember, and an invitation link and its reply that verify two colleagues over a channel they trust with no call, each acted on by opening it and tapping the passkey, which reverses D-055 (a peer is verified in one way, the two-word comparison) and rewrites D-301 (a cloud session's daemon serves its room on a second overlay port) to let every machine whose user has a passkey take a link the passkey signed. A passkey is not counted as a cloud key under D-280 (a user has one cloud key at a time), so a user may register several and keep an uploaded value beside them. Whether Pro and Max plans have an allowlist is not known.

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

**A hosted machine (D-285, a user with no computer of their own has a machine hosted for them) has no operator certificate, and neither the passkey that is to be every user's master key nor the sign-in it gives is built.** The user chose on 2026-11-28 that a user's master key is a passkey for the site `cogmer.ai`, with no recovery phrase, which Draft 6 of `docs/work/cowork-first-run-design.md` sets out with its costs. The master key certifies a new shared key, which is how a user shuts out a machine that holds the old one (D-219, a user is a master key that certifies machines through a shared key; D-220, a removal names the last event of a machine that counts), so for a hosted user it is the only recourse against the operator, and a passkey's key never reaches a page the operator serves. The passkey signs a record's hash in place of a random challenge, as wallets that hold funds under a passkey do (https://docs.candide.dev/wallet/plugins/passkeys, read on 2026-11-18), and signs with P-256, which only a new version of the records can carry. A user signs in to the hosted machine's page with a passkey assertion over a challenge the machine issues, which the machine checks against the user's identifier. The hosting service also keeps an email address the user has proved, only to reach and bill the user, and never uses it as an identity or for recovery. The user chose both on 2026-11-28, and D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person) requires the list of hosted users' addresses to be disclosed.

A passkey belongs to one site for life, a site is a domain name and never an IP address, and a page may name any parent of its own name as the site (https://web.dev/articles/webauthn-rp-id, read on 2026-11-18). The daemon's page at `127.0.0.1` therefore uses the passkey through a page under `cogmer.ai`, which hands the signature back by navigation, with no hosts-file entry or local certificate (spike F in `docs/work/cowork-first-run-design.md`). The project registered `cogmer.ai` on 2026-11-19, and releases are still served from GitHub (`plugin/release-url.txt`). Losing the domain, such as by letting its registration lapse, would end every passkey made under it, so D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person) has to record the dependence.

A phone takes part only as the authenticator for a page on the computer, through the browser's offer to use a phone, which scans a code and checks that the two are near over Bluetooth, or not at all, when the passkey syncs to the computer through the user's Apple or Google account. A computer outside that account, such as a Linux laptop, uses the passkey only through the phone.

An operator is added with no release, as chosen provisionally on 2026-11-28: cogmer's root key, kept offline, signs each operator's certificate, a hosted machine carries its operator's certificate, and every daemon checks it against the root's public key, which the released program carries. A revocation is a list that the root signs, with a version number, which every daemon fetches from `cogmer.ai` and each release carries beside `checksums.txt`, so a revocation reaches a running daemon within minutes and an updated one with no fetch. An operator listed in the released program would need a release to be added, and a root key kept in cogmer's online service would let anyone who broke into the service make operators every daemon trusts. The fetch from `cogmer.ai` is a dependence every daemon has, which D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person) requires disclosed.

A hosted machine serves its user's rooms to the user's page on any device the user has signed in on, a phone among them, ahead of the user's own computer and of a cloud task's daemon (Draft 4 of `docs/work/cowork-first-run-design.md`).

**Nothing decides how hosted machines run or how each one's page gets its address.** A hosted machine (D-285, a user with no computer of their own has a machine hosted for them; D-287, a user with a computer of their own can also have a machine hosted) answers for its user while every other machine of theirs is off and keeps every turn of the user's rooms, by §6 (identity). The daemon serves one user, whose state is in one directory that `COGMER_HOME` names (`homeDir` in `cmd/cogmer/identity.go`, read on 2026-11-20). A hosted machine does not let a Cowork sandbox skip the allowlist. Today the sandbox's daemon reaches it through tailcat's relay, whose hosts an owner allows, and under the relay proposed in the item "A relay change would break addresses users already hold" it dials the hosted machine directly at its name under `cogmer.ai`, which an owner allows as well. The listener for other machines' browsers and the remote sign-in it needs would be off on a user's own computer, whose page answers only on that computer, so that a user who runs cogmer themselves configures nothing new. The choices, each with its cost:

- One daemon process for each user, with its own state directory, is chosen provisionally, on 2026-11-28, and the work on hosting makes that process cheaper rather than building toward one process for many users. It holds unless something shows it untenable, such as a cost for each user, after the work on memory, the scan at start and the cost of a poll in the items below, too high for hosting to be worth providing. One process for each user needs little new code, and a fault reaches one user's keys and turns, at the cost of each user's memory all the time. One process for many users is cheaper for each user, needs the daemon rewritten for many users, and lets one fault reach every user's keys and turns.
- What isolates one user's process from another's is chosen provisionally, on 2026-11-28: a container for each user under gVisor, a kernel in user space between the container and the host's kernel, on ordinary virtual machines, with packaging that depends on no provider, so that an operator could run a microVM for each user instead. A hosted machine holds its user's shared key and never the master key (Draft 6 of `docs/work/cowork-first-run-design.md`), so a process that breaks out of one container gets the other users' shared keys and turns, and speaks as each until that user's passkey signs a new shared key, but takes no identity. The container is the cheapest and densest choice and runs on any provider's virtual machines. It isolates less than a microVM, gVisor slows the system calls SQLite makes, and an escape reaches every user on the server. A microVM for each user, such as Firecracker on cogmer's own servers, gives each user a kernel of its own, and needs bare-metal servers for KVM and tooling of cogmer's own for volumes. A provider's microVMs, such as Fly Machines, add a provider every operator depends on, and whether Fly's proxy passes TLS through by name, as the waking name needs, is not known. Kubernetes is the most moving parts for a service this size. What gVisor costs the daemon's CPU, memory and hook times is not measured, and the choice holds only if that cost is small beside measurement 12's.
- Always running, or started when traffic arrives. A machine that is off cannot answer for its user or keep a turn, and tailcat's relays cannot wake one. Under the relay proposed in the item "A relay change would break addresses users already hold", a hosted machine is dialed directly at its public name, so the proxy in front of it starts it, and cogmer's relay starts it only for a connection that comes through the relay. A machine started on demand is woken by a colleague's nudge, since a nudge means a turn the machine has to keep, and by a request from its user's page, from a cloud session or from another of its user's machines. It is not woken by a colleague's poll. A stopped machine has written nothing for a poll to collect, and a colleague polls a quiet address every 20 to 60 seconds (D-305, an address is polled less often while its rooms are quiet), so a machine that polls woke would never stay stopped. Whatever stands in front of a stopped machine, the relay or the proxy for direct dials, answers a poll that the machine is asleep, and the machine polls its colleagues when it starts. Starting on demand belongs to hosting that cogmer or an operator runs, where the proxy in front is part of the hosting and the memory it saves is many machines' on one server; the program every machine runs does not stop itself when idle. A machine without a public name is reached only through the connection it holds to the relay, which ends when it stops, and nothing can start a process on someone else's computer, so it could stop only by leaving a smaller process running to hold that connection and start it. A machine with a public name could stop only behind a proxy or the system's socket activation on its waking name. A self-hosted machine is one machine, whose idle daemon held 11.66MB and took 15.7 CPU seconds an hour on 2026-11-27 (`docs/work/hosting-fitness-runs-2026-11-27.md`), so it always runs. Every connection between machines ends its TLS in the machine, pinned to the machine's key (D-101, peer connections are TLS pinned to a key this machine has recorded), and a machine sends its polls and nudges over one kept connection (D-303, a machine keeps one connection to each address it polls; D-306, a machine nudges the machines it shares a room with when it writes an event), so nothing in front of a machine can tell a nudge from a poll by reading it. Requests that wake a machine therefore go to a name of their own, such as `<id>-wake.cogmer.ai` beside `<id>.cogmer.ai`, both on port 443, since a sandbox's allowlist names hosts and its traffic leaves through a proxy that may pass only 443. The proxy reads the name a connection asks for in the clear, starts a stopped machine for a connection to the waking name and holds the connection until the machine answers, refuses a connection to the other name at once, which a colleague takes as a machine that does not answer, and passes both through to a running machine without reading them. The relay does the same, since a dial names the machine's key and the channel. It costs four things:
  - A sender keeps a connection for each channel to a hosted machine, which changes D-303 and D-306, whose nudges go over the connection a machine polls with.
  - Every kind of request is assigned a channel: nudges, list requests, renewals, the page, invitation confirmations, pairing and verification wake a machine, and polls do not. A request on the wrong channel keeps a machine running for nothing or fails while it is stopped.
  - Anyone can wake a machine, since the proxy cannot tell who connects without reading the TLS, although the machine still checks the pinned certificate once it answers. A machine that starts runs for a fixed interval before it may stop again, so frequent wakes at worst keep it running, at the cost of a machine that is always running, and never cost a start for each wake. The interval is set from what a start costs against an hour of running, which measurement 13 of `docs/work/hosting-fitness.md` gives.
  - A request on the waking channel waits for a start, so its timeout is longer than a start takes. A nudge is sent again until it is answered (`deliverNudge` in `cmd/cogmer/nudge.go`).
- Storage: each user's SQLite files in a directory of the user's own on the server's disk, with the encrypted copies that recovery restores in object storage. A shared database would need the store rewritten.
- Recovery when a server is lost is chosen provisionally, on 2026-11-28: the hosting service copies each user's state directory, the shared key with it, encrypted to object storage about every hour, and after a loss starts a container from the latest copy with a new machine key, which the copied shared key certifies, then removes the lost machine. A machine's count of the sequences it has issued in each room is kept in the membership index in its state directory, not in the room database (D-029, losing a room database does not end membership; the sequence lives with the machine; D-249, each key that signs in a room counts its own sequence from 1), so a machine started from an older copy under its own key reissues numbers that its colleagues already hold for other events. A new machine key counts from 1, so the restored machine reissues nothing, fetches what its colleagues received since the copy, and needs nothing of the user. It is an exception to D-229 (a machine is added from another of the user's machines with the two-word check), whose words guard a pairing string sent between two machines, which a restore from the service's own copy does not send, and the removal needs no new shared key, since the lost machine is destroyed (D-251, removing a machine that someone other than the user could still use always certifies a new shared key). It loses the turns only the lost machine held that were written since the last copy, and the copies hold the shared key, which the operator can read in the running machine anyway. The alternatives cost more: the state directory on a network volume or replicated storage keeps the same machine with no gap, but a volume for each user fits dense containers poorly and two copies running at once reissue sequences; a new shared key from the user's passkey for a fresh machine, with the rooms fetched back from colleagues, needs the user present and loses everything only that machine held; and the counts kept somewhere that survives the loss must be written before each number is used. The same reason forbids two copies of one machine running at once, so a hosted machine is made available by replacing it quickly, never by a standby that runs alongside it.
- An operator other than cogmer runs the same image on servers of its own with its own certificate, so the packaging cannot depend on one provider.
- Where the work is kept, chosen provisionally on 2026-11-28: the control plane, which is provisioning, accounts and sign-in to the service, billing, the proxy and its waking, the backup jobs, the deployment on a provider, the runbook for the root key, and monitoring, is built in a private repository with its own decisions and open items. This repository keeps everything a user or a daemon relies on: the program a hosted machine runs, the formats a daemon checks (the operator certificate, the root's public key, the revocation list, the passkey's records), the packaging an operator runs, and the account of what the hosting service holds and does not keep, which D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person) requires disclosed. A promise users cannot check in the code, such as encrypted backups or no request logs, is stated in that account. The provider and region, the root key's custody and ceremonies, provisioning, the names and certificates of hosted pages, where TLS ends, and charging for hosting are open there.

A hosted page has a name of its own under `cogmer.ai` that reveals nothing of its user's identifier, and no certificate is published for each user, so the Certificate Transparency logs do not list hosted users. Whether TLS ends at the proxy in front of hosted machines, which would then read a page's traffic, is chosen with the control plane, and the account of what the hosting service holds says which.

A hosted page is to be told of each change in its user's rooms with work that does not grow with the room's size. The loopback page's stream builds and sends the whole room on every change in any of the daemon's rooms (`handleStream` in `cmd/cogmer/ui.go`, read on 2026-11-27), and a room grows for as long as it is used, so that stream served from a server would cost more for each user every month. Sending the page only what changed does not do it alone. A turn's state changes with no new event when a session takes it, a colleague is renamed, a machine is removed or a picture arrives (D-308, the static page is sent a page of a room's turns and asks for the earlier ones), and finding those changes by comparing whole snapshots is the same work. The page has to be told of each such change by the code that makes it.

An open page does not keep a hosted machine running, since a user is unlikely ever to close the page, and a machine kept running by its page would run all day and night. The page's connection rests at whatever stands in front of the machine, which holds it while the machine is stopped: the proxy at the hosted machines' public names, through which a page reaches a hosted machine directly, or the relay, for a page that reaches the machine through it. While the machine is stopped nothing in its rooms changes except by a colleague's write, whose nudge wakes the machine, which then makes a new session with the page, since the old one ended when the machine stopped, and sends the page what changed. What the page shows that comes from no event, such as whether colleagues are online and the age of the lists, it shows as of when the machine stopped. A page that the browser reports hidden, or that nobody has touched for a while, asks for nothing, so that the machine can stop, and catches up when it is shown again. A machine's running hours then follow its rooms' activity, not whether its user's page is open, and a page opened on a stopped machine waits for the machine to start, which measurement 13 of `docs/work/hosting-fitness.md` times.

**A hosted machine's memory grows with its user's history, from about 13MB with no turns to 111MB idle after two days of turns, and only part of the growth is explained.** A hosted machine (D-285, a user with no computer of their own has a machine hosted for them) is a daemon for each user. On 2026-11-27, on a droplet made for the measurements and the build `36162d5` of `release-0.11.0`, which keeps a connection for each address (D-303, a machine keeps one connection to each address it polls), polls quiet rooms less often (D-305, an address is polled less often while its rooms are quiet) and nudges (D-306, a machine nudges the machines it shares a room with when it writes an event), twenty users with a turn a minute in each room cost the box 27.3 CPU seconds an hour each, held 13.35MB of PSS and sent 2.89MB an hour, with the overlay off, which fits about 525 users on a server of 4 vCPUs and 8GB (`docs/work/hosting-fitness-runs-2026-11-27.md`). On 2026-11-28 a group of four filled to two days of the same turns held 111.4MB of PSS idle and took 21.2 CPU seconds an hour idle, against 38.0MB and 13.0 at a quarter of a day, and a daemon started on those stores held 42MB to 85MB a minute later (the same document). Four daemons share the program's pages less than twenty do, and at 111MB a server holds about 70 users. A profile of one daemon on a two-day store on 2026-11-28 found about 4MB of live Go heap, as with an empty store, about 31MB of garbage from the scan at each start, which the Go runtime returned after about two minutes, and about 29MB of other anonymous memory, most likely SQLite's. On 2026-11-25, before connections were kept, a user cost the box 189 CPU seconds an hour (`f98fa3d:docs/work/hosting-fitness-runs-2026-11-25.md`). The relays cogmer uses are free and rate-limited (D-192, the public relay is an accepted dependency). The hook and page address must be loopback (`cmd/cogmer/main.go`, read on 2026-11-24), so each hosted page also needs a proxy or a change to that rule.

A profile of the memory on the stores measurement 10 left would say what the rest is. The options it would choose among: a smaller SQLite cache for each connection, or fewer connections; a memory limit for the Go runtime; and running the scan at start once, as the next item says. One connection that carries the polls of many daemons would cost a change to sync and a decision. Still to measure: a daemon under an OS user of its own and in a container under gVisor, which the choice of isolation rests on; the overlay with turns; bytes to a colleague on another machine; and an ARM server. The terms, the users each server holds, and the plan are in `docs/work/hosting-fitness.md`.

**Every start of a daemon reads and checks every stored event again, so a start costs CPU and memory in proportion to the store, which a hosted machine started on demand pays at every wake.** `scanWithheld` in `cmd/cogmer/propagation.go` runs once when the identity loop starts, lists every event of every room, and for each asks the membership index what the event's author's records are, which checks their signatures (`noteLacking`, read on 2026-11-28 at `36162d5`). It exists so that an event stored before the build that records what an event names, or a want lost with `membership.db`, is still fetched. On 2026-11-28 a daemon started on a store of two days of turns, 8,643 for each of four daemons, used about 7.6 CPU seconds in its first 8 seconds and left about 31MB of garbage, while it answered at `/healthz` within 0.1s, and measurement 13 found a daemon started on such stores held 42MB to 85MB a minute later (`docs/work/hosting-fitness-runs-2026-11-27.md`). Running the scan once for each build that changes what it looks for, recorded in the state directory, would spare every later start, at the cost of a marker that a restored copy of the directory has to keep.

**Each poll reads every stored sequence number of the room, so an idle daemon's CPU grows with its history.** `SyncState` in `cmd/cogmer/store.go` reads the signer and sequence of every event in the room, in order, to find the highest sequence held without a gap for each signer (`heldSequencesQuery`, read on 2026-11-28 at `36162d5`), and every poll calls it. On 2026-11-28 an idle daemon took 13.0 CPU seconds an hour at a quarter of a day of turns and 21.2 at two days (`docs/work/hosting-fitness-runs-2026-11-27.md`), and a profile of an idle daemon on a two-day store found `SyncState` the largest part of its own work. Keeping the highest sequence held without a gap for each signer in a table, updated as each event is stored, would make a poll's cost independent of the history, at the cost of a stored value that a migration fills for every store that exists.

**A hosted machine holds an overlay client for each colleague it dials, at about 42MB each, and nothing lets a colleague's machine carry the hosted machine's requests instead.** With the overlay on, a daemon in a group of four held 161MB, against 27MB with it off (`f98fa3d:docs/work/hosting-fitness-runs-2026-11-25.md`), because each colleague address gets a `tailcat.Client` of its own, a WireGuard engine and network stack with its own relay connection (`tailcatDialer.clientFor` in `cmd/cogmer/tailcat.go`, and `Client` in `github.com/tailscale/tailcat` v0.6.0, read on 2026-11-25). The library connects one client to one server and has no setting for memory, so one stack for many colleagues would need a fork or an engine of our own on the `tailscale.com` packages. A daemon dials another machine from eight places, read on 2026-11-25: room sync (`pullFrom`), image fetches (`requestBlob`), invitation offers (`deliverOfferTo`), identity records, removals and new shared keys (`postSigned`), the user's own lists (`exchangeWith`), the request to erase keys (`askToRetire`), verification (`RunVerification`) and a cloud session's lists (`fetchListsFrom`).

The options, each with its cost:
- The machine that cannot be dialed opens a TLS connection to the hosted machine and keeps it, and both ends multiplex it, so the hosted machine's dialer opens a stream on that connection instead of an overlay client. Every flow keeps its protocol, behind the `Dialer` interface (D-068, tailcat is the cross-network transport, behind our own dialer), and a laptop reaches the hosted machine's public address with plain TLS, as D-019 (no network provider is required) allows. It needs the connection kept for each colleague that D-303 (a machine keeps one connection to each address it polls) keeps; a multiplexing library; a rule for who dials, such as that a machine certified as hosted (D-285, a user with no computer of their own has a machine hosted for them) never dials; the kept connection closed when either machine is removed and kept open until an erase is acknowledged (D-252, a reachable machine is asked to erase its keys before it is removed); status that says when a colleague last connected rather than that it cannot be reached (D-061, a machine outage is reported when it starts; D-194, a colleague this machine cannot reach is named with how); D-116 (verification dials the peer it is verifying) worded to cover a stream on that peer's own connection; and overlay dialing kept for machines on older versions.
- One exchange carries both directions: the dialing machine's request also carries the events the answering machine lacks, and the answer says what it now holds. A pair of machines then makes one exchange a second instead of two. It meets D-031's objection to push, that a sender would have to know what the other holds, since every answer says it. It needs a new wire version, with the old exchange kept for older machines (D-065, the daemon reads a range of wire versions), and §16 (propagation) reworded.

A cloud sandbox reaches a hosted machine without the overlay only if its organization allows the hosting domain and its proxy passes the sandbox's TLS through. A Cowork sandbox reached only the hosts its organization allowed, through the session's proxy (`ce962eb`). Every connection between machines carries a client certificate and is pinned to a recorded key (D-101, peer connections are TLS pinned to a key this machine has recorded), which a proxy that decrypts and encrypts again cannot present. On 2026-11-26 the Cowork proxy passed TLS through unchanged to every host the organization allowed: a task saw the certificates the user's Mac saw, and a server of cogmer's saw each request come from an address in 160.79.106.0/24 with no forwarding header. It decrypted only `github.com` and `api.anthropic.com`, with certificate authorities of Anthropic's own (`docs/work/first-run-spike-C-20261126T1533Z.txt`). So a sandbox could hold a pinned connection with a client certificate to a hosted server on an allowed name. A sandbox keeps the overlay for colleagues on laptops either way, since D-248 (a cloud session asks its user's colleagues for a room) has it sync with each colleague directly. Syncing only through its user's hosted machine would remove the overlay from the sandbox, would reverse D-248, and would let its turns reach colleagues only while that machine is up.

An entry `*.probe.cogmer.ai` reached its subdomains and not `probe.cogmer.ai`, and an entry `probe.cogmer.ai` reached only that name, so a hosting domain takes two entries (`docs/work/first-run-spike-C-20261126T1533Z.txt`). Not known: whether a client certificate passes as the curl requests did, which only a run with one would show; whether Pro and Max users can allow a host; whether laptops reach a hosted server's port, which argues for 443; and how a kept connection fares when a laptop sleeps or changes network.

**A turn takes about 7.6KB in each daemon's store, of which 1.1KB is its signer's certificate chain repeated on both of its events, and a hosted machine keeps every turn.** On 2026-11-25 a room filled with 1441 turns of 4.5KB held 9.85MB in its events table and 1.10MB in the five indexes on it (`523cceb:docs/work/hosting-fitness-store-2026-11-25.md`). Each event carried 571 bytes of `chain`, which is the same for every event its signer signs, and 43 bytes of `room_id`, which is the same in every row of a room's database. `idx_events_signer` indexes the same columns as the index SQLite makes for the table's `UNIQUE(signer, peer_sequence)`. At the reference user's 4320 turns a day that is about 33MB a day and 12GB a year in each daemon, and a hosted machine keeps every turn of its user's rooms (D-285, a user with no computer of their own has a machine hosted for them). Each database's write-ahead log also stays at about 4MB once it has grown there, which does not grow with turns.

The options, each with its cost:
- Compress a turn's content. It is likely the largest saving, and is not measured, since the harness fills turns with random letters. It changes the stored row, which D-155 (the wire format is defined separately from the stored row) allows, and needs every store migrated and each row to say how it is stored, so that rows written before it still read.
- Store each certificate chain once for its signer, and name it from each event. It saves about 15% of a turn, and is a change to the stored row with the same migration.
- Drop `idx_events_signer`, which the unique constraint's index duplicates. It saves about 2% of a turn, and `migrate` and `rekeyEventsOnSigner` in `cmd/cogmer/store.go` both create it.
- Set a limit on the size of each database's log, so that it shrinks after a checkpoint. It saves about 4MB for each room, which matters for a user with many rooms and not for growth.
- Keep only a window of a room's history on a hosted machine, with older turns kept elsewhere or fetched from colleagues, which is a change to D-285.

Not known: how far a real turn's text compresses; whether `pending_injection` grows with a room where a host never confirms delivery, as it did in the harness, where each of its 20 rows for a session listed every turn offered and took 52KB.

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

**The page that reads a cloud session's room takes about 20 seconds to its first data the first time a phone opens it, almost all of it the browser compiling the module, and a blank page on an iPad is not explained.** On 2026-11-25 the page, published to `pages`, opened from the link in a cloud task's `/cogmer:room-join` on 0.11.7 and read the room through port 4784 of the task's daemon. It drew a prompt within a second, said the session was asleep after the daemon went unreachable, and recovered without a reload when a prompt woke the task (`docs/work/browser-spike-20261125.txt`). That run was in Chrome on a computer. On an iPad in Safari the page read the room of a session that was awake and drew a prompt within a second, and a copy of the page from before two fixes stayed on its asleep banner after the session was sent a message, which was not reproduced. After a long switch away the iPad showed a blank page that a reload did not clear, and it has not come back since. On 2026-11-26 on an iPhone over 5G, in what looks like Safari, a page opened while the task was asleep drew the room, including a picture attached to a prompt on 0.11.8, and showed a message that woke the task without a reload, and the rooms pane's Close button worked. On 2026-11-26 an iPhone on iOS 18.7, in a private tab on Wi-Fi, read "1.0s module fetched · 17.3s module ready · 17.3s dialing · 18.9s connected · 19.6s first data", so the 27 MB module took 16 seconds to compile and the connection 1.6 more. An earlier private tab on 5G fetched it in 0.3s, and this Mac's Chrome took 1.0s to fetch it and 6.9s to have it ready the first time and 0.2s afterwards. The same phone's second open read "0.0s module fetched · 0.1s module ready · 0.1s dialing · 1.0s connected · 1.6s first data", so the compiled module is kept and the cost is paid once. The banner says so on the first open.

A run would leave a page open through a long idle, since the page keeps one dial in flight and each failed dial lasts the module's full minute. On 2026-11-26 in Docker on this Mac, a cloud session's daemon with a 300-turn room spent 3.0% of a core on average with a client polling it as the page does for 50 minutes against 1.9% idle, with its two idle neighbours unchanged at 1.9%, and its memory rose from 49 MiB to 128 MiB in the first 40 minutes and held there. The 976 polls of each route took 36 ms on average, none failed, and the snapshots were 308 KB each, 353 MB for the 50 minutes. D-307 and D-308 cut that transfer. Nothing publishes the page as part of a release, so the protocol of a daemon's port and the page that reads it can fall out of step. The first publish took about 6.5 minutes on this Mac's connection and later ones 13 seconds, since GitHub holds the module.

**Cowork on a user's computer captures and injects through the bridge's `mcp_tool` hooks, though the plugin's own command hooks were seen to run there.** D-205 (Claude Cowork is a supported host through a bridge on the user's machine) chose the bridge before a run on 2026-11-18 saw a cogmer plugin's command hooks capture a session's prompts and replies on the Mac, and its prompt hook's plain output reach the model (`e7ddab7`). That run does not show which copy of the plugin ran, the uploaded one or the account's, and the plugin's commands were refused there ("shell command execution disabled by policy"), so joining a room needed a terminal. `mcp_tool` hooks changed or were fixed six times between Claude Code 2.1.281 and 2.1.328, from 2026-09-23 to 2026-11-18, and a reply the harness drops while the Stop call arrives is counted delivered and lost (the Limits of D-206, a Cowork turn counts as delivered when the stop call for its prompt arrives). The simulation put Cowork's capture working 86% of the time through the plugin's hooks against 79% through the bridge, and months of silent failure over 24 at 0.40 against 0.61 (`docs/work/capture-simulation/report.md`). Capturing and injecting through the plugin's hooks, with the bridge kept for commands, needs a decision beside D-205, and a run that shows the hooks firing, and which copy runs, across several Desktop versions.

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

**Whether a daemon is a cloud session's is asked in ten files in several forms that disagree.** Read from the code on 2026-11-26: `inCloudContainer()` is true when `CLAUDE_CODE_REMOTE` is `true` or the cloud key's variable is set, `d.cloud != nil` is true once the daemon holds a cloud session, `isCloudSession()` is true when it holds one or saw the variable, and `selfcommands.go`, `tailcat.go` and `main.go` read the environment themselves. The daemon drops the key from its own environment once it has read it, so `inCloudContainer()` was false in a daemon whose host sets only the key, and that daemon served no page until `1c42e2e`. The hosts are `claudeHost`, `codexHost` and `coworkHost`, and `coworkHost` is Cowork on the user's computer through the bridge. A Cowork cloud task runs `claudeHost` in the sandbox, and its entry point was `remote_cowork` on 2026-11-26, which no code reads, so a Cowork cloud task and a Claude Code cloud session are not told apart. What a Claude Code cloud session on the web looks like in its sandbox is not known.

What differs about a cloud session falls in two groups. The daemon's placement decides whether it sleeps and spools prompts (D-294, a prompt that finds no daemon listening just after one was started is held for it), carries its relay connection as a WebSocket (D-217, a cloud session's daemon carries its relay connection as a WebSocket), runs no behavior check (D-297, the behavior checks are not run in a cloud container), and speaks with a cloud key and serves the page (D-216, a cloud session signs with a key of its own that the user's shared key vouches for; D-301, a cloud session's daemon serves its room on a second overlay port). An application in a placement decides where its attachments land (D-304, a cloud session's attached images are the fresh files in its uploads folder that it has not sent) and that its hooks send no transcript path (D-295, a turn with no transcript path is recorded from the Stop hook's last message). D-203 (a host is declared by its entry point and resolved per request, and the daemon holds none) puts the host with the request, and a placement belongs to the daemon, not the request.

One placement for the daemon, worked out once at start from the environment and the key and asked as questions (whether a behavior check may run, whether prompts are spooled, whether the page is served, where attachments land), would leave the three forms no way to drift apart. What an application does in a placement stays a method of its host that takes the placement. A host type for each pair, such as a Cowork cloud host beside a Cowork host, would multiply the types and mix the two groups. The hosts' names stay as they are, since stored records and session identifiers carry them, such as the `cowork:` prefix. The change touches the ten files and is covered by the container and cloud tests, and it changes nothing a user sees.

**A cloud session's turn records only its final text, not the text written before each tool call, since the session has no transcript to read.** A turn with no transcript path is recorded from the Stop hook's last message (D-295, a turn with no transcript path is recorded from the Stop hook's last message). Claude Code's MessageDisplay hook, from 2.1.152 on 2026-05-26, runs "for every assistant message that streams text", once for each batch of completed lines in an interactive session and once for each whole message under `-p` (https://code.claude.com/docs/en/hooks, read on 2026-11-28), so it could supply that text. Its input carries a `turn_id` but no `prompt_id` to pair with the prompt and Stop hooks, its `message_id` "can't be correlated with transcript message ids", whether it runs for subagents is not documented, and "Claude Code holds each batch until your hook returns", so a hook that is slow at any moment delays what the person reads, which the values forbid. The simulation put full capture in a Cowork cloud session at about 60% of the time with it, against none today (`docs/work/capture-simulation/report.md`). A probe of its timing and of what it fires for, in an interactive session, the desktop app, a cloud session and Cowork, would show whether it can be used, and a behavior check with a negative test would have to hold it.

**A silent failure of capture or injection in a cloud session is found only when a user notices, since no behavior check runs there.** No behavior check runs in a cloud container (D-297, the behavior checks are not run in a cloud container). The simulation put a cloud session's months of silent failure over 24 at about 1.1, against 0.2 to 0.3 on a user's computer, depending most on how long users take to notice, which it assumed to be 1.5 months (`docs/work/capture-simulation/report.md`). A daemon in a cloud session could check its own hooks' invariants on every turn: Stop arrives with a `prompt_id` of an open prompt (B29), the transcript path is empty or present as the surface makes it, the prompt is not empty after the first turn, and the last message is not empty. It could tell the user of a violation once, through a hook's `systemMessage`, which the person sees and the model does not (`11b1294`).

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
Cowork reached a colleague with its text and no picture. On 2026-11-26 a picture pasted into a Cowork cloud task, whose room held a colleague's machine, was saved in the task's sandbox at `/mnt/user-data/uploads/<8 hex digits>-image.png`, a different folder from the one of 2026-10-15. The model described the picture, and the room recorded the turn's text with no `[Image #N]` placeholder and no image, so the page drew none. The file kept the time it was saved through a sleep and a wake, and the task's entry point was `remote_cowork`. D-304 (a cloud session's attached images are the fresh files in its uploads folder that it has not sent) is built for it and tested against a folder laid out as that task's was. On 0.11.8, staged to `Blue-Rocket/cogmer-staging`, a real task sent a picture attached to a prompt and the page of the same account drew it; a colleague on another machine has not seen one. In containers, a cloud session's daemon whose hooks send no transcript path sent a picture saved where Claude Code saves one to a colleague, and took one from the colleague, and the receiver's view named it by its hash and served its bytes (`b5c417a`).
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
following it. The page has to keep the unread count and the jump button right for the turns it hides. On the static page the filter sees only the turns the page holds, since D-308 (the static page is sent a page of a room's turns and asks for the earlier ones) sends the newest page and the rest on request, so a person's turns hidden there could leave a page of nothing and a button asking for more. A
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
