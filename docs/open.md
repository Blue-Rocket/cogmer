# Open

Work and unanswered questions. Nothing here is durable: every item either becomes a
decision, a change to the specification, or code — and is then **deleted**, not
marked done. The record of why lives in `decisions.md` and the record of what the
system is lives in the specification, so an item that has earned a permanent home
does not need one here too.

Being scratch is the point. Be untidy in it.

**Where things stand** is the exception: it is edited rather than emptied, because
there is always a current phase. It lives here so that nothing which goes stale with
every week of work sits in a file loaded into every session.

## Where things stand

Every numbered phase in §31 is complete or deliberately dissolved. Phase 5 closed
review finding B2 (injection order) by verification, which one live peer could never
have exercised: the injected block then holds only that peer's turns, already in
sequence.

**Phase 13 — somebody else uses it — is the live one, and it is no longer blocked on
anything here.** The repository exists, is public, carries the marketplace manifest
beside the plugin, and serves releases the installer verifies against its pins
(D-121). What it needs is a person who did not write this: installing, pairing,
joining, working, and saying what they hit in the order they hit it. It is the only
phase that can fail in a way none of the others detect, and the failure looks like
somebody quietly not using it again.

Outstanding beneath that: local network discovery, and a host approving an
unsolicited join request — the second undecided rather than pending (§12a, D-051).

The `cogmer` binary is **not** tracked — `bin/` is ignored, because it is 17MB
per commit and `go build` reproduces it.

## Decided and not built

**Re-pick the overlay relay when it cannot be reached.** D-104 pins it so the
address is stable. Nothing re-picks, so a machine that relocates past its pinned
relay is unreachable and nothing says so. Change driven by failure, never by
preference.

**Say when your own address changes.** Every pairing string and invitation already
handed out is then stale, and only the daemon can know.

**A room never closes, so nothing is archived.** D-015 (rooms are session-scoped)
has a closed room kept as an archive that can be read but never rejoined, and the
specification freezes a room's sequences at that point. Membership ends per person
through `leave` and `revoke`; the room itself has no closed state.

**Local network discovery** (D-019's zero-configuration path) is not built.

**`cogmer stop` (D-123, finding a daemon by the addresses it holds).** Specified and
not built. It needs `runStop` and a SIGTERM handler in `main.go`, since §3.7 keeps
`os/exec` and `syscall` out of the daemon's files, plus the blocked-address message
naming `stop` by its full path, and a test that it declines a process not named
`cogmer`. A test harness that listens on a port under another name is enough for
that.

Three things about it are undecided. The first is whether `stop` then starts this
installation's daemon. Clearing the way is almost always why somebody runs it, but a
person may also want it simply stopped. The second is whether it gets a slash
command. The person is in a session, not at a terminal, and the binary is not on
PATH, so from a terminal they have to type `~/.cogmer/bin/cogmer stop`. The third is
Windows, which has no `lsof`. `netstat -ano` gives the pid there, or `stop` can
report the port and fall back to moving this daemon aside.

## Commands that mislead

**`/cogmer:self-status` on first use fails instead of answering.** It is the first
command a new person runs, because the pairing string is what they came for, and it
runs before there is a binary. There are two ways that happens. The download takes
time: 29MB, measured at 16s on 09-22. And a plugin installed inside a running
session fetches nothing at all, because the fetch is started by the SessionStart
hook and that session started before the plugin existed. Installing and then
immediately trying the command, in the same session, is probably the commonest
first use, and on 09-22 it was David's.

The cause is in how Claude Code handles a command's `!` lines. A `!` line that exits
non-zero abandons the command, and no model turn runs. In an interactive session the
person sees the line's output under "Shell command failed for pattern …", labelled
`[stderr]`; with `claude -p` the result is the empty string. Both were observed on
09-22. `cli.sh` exits 1 from every branch where there is no binary, so the
installing, failed and stalled explanations D-075 (say which of three reasons it
is) wrote do reach the person, but framed as a crash, and the model never sees them,
so it can neither explain them nor act on them.

Five changes would make first use work. `cli.sh` should always exit 0 and send the
binary's stderr to stdout, so that every explanation reaches the model. Nothing else
reads its exit code, since the hooks go through `run.sh`.

When nothing has been fetched, `cli.sh` should start the fetch itself instead of
telling the person to start a new session. `install.sh` already takes a lock and
decides for itself whether there is anything to do, so a command can run it exactly
as the hook does. A command somebody typed is the right place to spend a download,
which the session-start hook has to do detached so that it slows nothing.

While an install is in progress, the command should wait for it within a bound and
then answer with the pairing string, rather than saying to try again in a moment.
The person asked for that string, and would otherwise type the command a second
time to get it. Waiting inside a command somebody typed is arguably not what §3.1
(first, do no harm) means by slowing a session, but that argument has to be settled
first. A bound of around 30s covers the 16s measured.

When the install failed or stalled, the command should say what to do as well as
what happened. The failed branch already names `~/.cogmer/install-state` and the
one-hour cooldown. The stalled branch says to start a new session, which cannot
help if the crash left `.install.lock` behind, because every later install finds
the lock and leaves.

The behaviour belongs in `behaviors.go` with a negative test, a command whose `!`
line exits 1 producing no turn, because it fails silently and the registry exists
for exactly that.

The same cause hits other commands. `/cogmer:room-status` outside a room fails
the same way, because its `guests` line exits 1, and that is the state everybody is in
just after installing. Any `log.Fatal` a command reaches does the same. The `cli.sh`
change covers every one of them, and each command still needs checking, because
some exit non-zero on purpose. A fix reaches nobody until the version moves (D-120,
the manifest version pins an installed plugin).

**A port held by another cogmer daemon is reported as held by something else.**
When the peer-sync port is taken, `runDaemon` calls `reportDaemonBlocked` without
asking what holds it, and the message says "held by something that is not a cogmer
daemon" regardless. On 09-22 it was a cogmer daemon, one left running from a test
with a different `COGMER_HOME`, and the message sends the reader looking for a
different program. Only the hooks port is ever probed, through
`daemonAlreadyServing`, which reads `/healthz` over plain HTTP. The peer mux serves
`/healthz` too, but over TLS, and nothing asks it.

**An old daemon keeps serving after an update, and nothing shows it.** `/healthz`
reports `ok`, `peerId` and `rooms` but no version, and `daemonAlreadyServing`
accepts any daemon that answers. So after an update installs a new binary,
`start_daemon_if_needed` finds the old daemon answering and leaves it serving, and
the new binary does not run until that process dies of something else. Read from
the code on 09-22, not yet observed. A daemon that reports its version would let
the hook replace one older than the binary it just installed, using `stop` (D-123).

**`whoami` prints a pairing string it knows nobody can use.** `printPairingInvitation`
prints the string first and then, when `pairingReachable` fails, a `NOTE:` after it,
so a loopback address goes out looking sendable. On 09-22 `/cogmer:self-status` did
exactly that: it advised holding off, then gave the string as "provisional" and
"safe to send". When no address is reachable there should be no string to copy. The
note for an unpublished address also says to start a session, which misleads when a
session has started and the daemon is blocked: `pairingReachable` does not read
`daemon-state`, which already has the real reason.

**The username notice prints twice in `whoami`.** `runWhoami` prints `nameLine`, and
then `printPairingInvitation` prints it again while the name is not chosen.

**`leave`, run twice, says "this session is not in a room."** True, and unhelpful
to somebody who left it a moment ago: it reads as a failure and sends them looking
for a problem. It should say it has already left. `LeaveSession` returns that error
from `RoomForSession` finding nothing, so the distinction to draw is between never
having been in one and having left it — the row is still there with `left_at` set,
so both are answerable.

**`create`, run twice, silently makes a second room.** Inherent — a room is a new
thing each time — but it is the command where a mistaken repeat costs most, because
the second room is indistinguishable from the first to everybody except its
creator, who now has two and is talking in one of them.

Nothing currently helps. The session invoking it is already bound to a room when
this happens, and binding is what `BindSession` already knows about, so a second
`create` from a session already in a room could say which room that is and ask.
That is the one place a non-idempotent command could cheaply refuse a mistake it
now makes in silence. The command's documentation warns against retrying, which is
the weakest possible form of the check.

**Move `plugin/commands/*.md` to the `skills/<name>/SKILL.md` layout.** The
documentation calls `commands/` legacy and says the two are loaded identically. D-119
deliberately did not do it in the same pass, because a directory restructure is a bad
thing to bury a §3.1 fix inside. `TestNoCommandIsModelInvocable` reads the old path
and would need to follow.

## Undecided

**Asking for a new session to finish an install is too much.** After `/plugin
install` the person believes it is installed, and Claude Code agrees: since
v2.1.221 a plugin installed mid-session is live in that session, with its commands
and its per-prompt hooks. Only SessionStart has not run, and nothing re-runs it,
since Claude Code runs no plugin code at install and reloading plugins does not
fire it. So the binary is never fetched, the daemon never starts, and the standing
policy for room content (D-081) is never given. Telling somebody to start again to
finish something they think has finished costs them the session they were working
in, and the README currently does exactly that.

The possible mitigations:

- Start the fetch from the per-prompt hook when there is no binary, detached, so the
  first thing typed after installing begins the download without waiting on it.
  Unchecked: whether a slash command fires that hook at all.
- Have `cli.sh` start the fetch and wait for it within a bound when a command finds
  no binary, as the first-use item under "Commands that mislead" describes.
  `install.sh` already starts the daemon when the download lands, so starting the
  daemon needs nothing further.
- Let the session you installed from enter rooms without the session-start policy.
  D-081 could not show that policy helping: in-block framing alone produced the
  same refusal of a hostile turn. Rerunning that hostile-turn test in a session
  that installed the plugin mid-session would say whether this is safe.
- Otherwise, have `join` and `create` refuse in a session that never got the policy,
  and offer `/clear`, which fires SessionStart but discards the conversation so far.
  Everything before entering a room, such as pairing, `self-status` and `self-name`,
  reads no room content and needs no policy, so the cost falls only on entering a
  room. It needs a marker the SessionStart hook writes per session.
- Shorten the wait with a smaller download, parked under "Needs somebody else".
- Ship the binary inside the plugin, ruled out there for what it adds to the
  history.

**Whether to rebuild the post-quantum hedge.** D-104 removed the pre-shared key,
which was the only quantum-resistant element in the stack. It could be rebuilt at
our own layer from material the pairing exchange already produces — per peer, which
is better than one secret shared with everybody. Nothing depends on deciding this.

**Whether `verify` still earns its place.** `/cogmer:peer-pair <name> --again` now
does the same job, and `verify` has no slash command, so it may be a subcommand
nobody has a route to.

**What a person actually notices in the view.** D-090 left this open: the display
name and the derived name are separate elements but carry similar weight, and
nothing has been measured about whether a reader distinguishes them.

**Whether arrival wants announcing.** D-039 (the browser view is the settled
avenue) left this open. Do not build a notification on speculation — it has a
different answer for close pairing than for long solo stretches.

**Whether a peer event may cause a *separate* run.** §3.7 forbids one in an
interactive session and leaves this open. Addressing another person's Claude
would need it, and it raises its own questions — whose subscription, what tool
access, what was agreed to — none of them answered.

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

## Cleanup

`WithheldFor` is used only by its own test — a leftover from the count that was
dropped in favour of naming the person.

`/cogmer:room-list` and `/cogmer:self-status` were both named without confirmation.

## Needs somebody else

**NAT to NAT, with a real colleague on a real home router.** The last unknown in the
transport, and not testable alone.

**Whether the public relay is an acceptable dependency.** Parked deliberately as a
precondition of Phase 13, where it stops being theoretical.

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

## Lost, and needs recovering from David

An item on **idempotency** was in `residual-concerns.md` and is gone: the file was
deleted on the strength of a reading from earlier in the session, and being
untracked there is no copy. `/cogmer:peer-pair` repeated on a completed pairing was
worked in D-107 and D-108, which may or may not be what it said.

## Older, from the specification review

**A1** — §19 still does not say when delivery advances. Reopened because the
implementation settled it and the specification never caught up.
