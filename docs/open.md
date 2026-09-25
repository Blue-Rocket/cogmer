# Open

Actions not yet taken and decisions not yet made. Nothing here is durable: every item
either becomes a decision, a change to the specification, or code — and is then
**deleted**, not marked done. The record of why lives in `decisions.md` and the
record of what the system is lives in the specification, so an item that has earned a
permanent home does not need one here too. Nothing here records status.

Being scratch is the point. Be untidy in it.

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

**Most decision entries hold more than one decision, or history, and none is split
yet.** W-40 in `docs/writing.md` says an entry records one decision. Reading all 126
entries in full, as they stood at commit `f11e871`, gave this, with the number of
decisions in parentheses. `docs/work/decision-split.md` holds the observations for
each entry: where each decision sits, the alternative each extra one has, the parts
that are not decisions, and what each citation means.

- split (59): D-002 (2), D-008 (2), D-010 (2), D-014 (2), D-015 (2), D-016 (2),
  D-017 (4), D-018 (2), D-019 (3), D-020 (2), D-021 (4), D-023 (3), D-024 (2),
  D-025 (3), D-030 (2), D-032 (2), D-033 (2), D-035 (2), D-036 (2), D-037 (2),
  D-040 (2), D-041 (3), D-042 (4), D-043 (4), D-045 (2), D-046 (4), D-050 (2),
  D-052 (4), D-053 (3), D-056 (2), D-057 (3), D-059 (2), D-060 (2), D-061 (4),
  D-062 (2), D-066 (2), D-067 (3), D-069 (2), D-074 (2), D-075 (2), D-077 (2),
  D-080 (3), D-081 (2), D-082 (2), D-084 (2), D-086 (2), D-088 (3), D-090 (2),
  D-091 (2), D-093 (2), D-094 (2), D-095 (3), D-096 (2), D-098 (2), D-100 (2),
  D-104 (3), D-106 (3), D-117 (3), D-121 (2);
- become tombstones under W-37 (a reversed decision becomes a tombstone): D-028,
  which D-029 reversed, and D-047, which D-055 reversed; D-076 is one already;
- one decision, with history or findings to move out (57): every entry not listed
  here;
- one decision and nothing to move (9): D-005, D-007, D-009, D-012, D-013, D-112,
  D-124, D-125, D-126.

Splitting all 59 adds 86 entries, and 146 citations of them mean something other
than the first decision, so each needs checking against its few words. Four
questions come before any split:

- whether an extra decision that repeats an existing entry folds into it instead of
  taking a number: D-080's prefix rule into D-096, D-020's guest-list preference
  into D-024, D-056's rule into D-016;
- whether D-002, D-062, D-070 and D-078 become tombstones or are rewritten to what
  still holds (W-37, W-38), since later entries hold most of what they decided;
- which decision keeps the number where most citations mean one the title does not
  name, as in D-043, whose citations mostly mean "no adapter machinery for a second
  host", and D-080, split 8 to 7;
- whether D-018, D-024 and D-025 are rewritten for what D-026 (no join token)
  reversed before they are split.

Five entries lack a **Decision.** field: D-068, D-083, D-086, D-088 and D-089.

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

**Whether decisions move to one file per record before the log is split.** Every
citation of a decision is prose: a number and a few words typed at each use. Checking
them showed four costs, recorded in `docs/work/decision-split.md`. A citation says
nothing about how it relates to what it cites, so what each of 146 citations means
needs a reader. Its few words are retyped each time and drift from the title. Finding
every citation of an entry takes a grep that also hits test fixtures, code comments
and quotations. A line-number citation moves with every edit above it.

The options, each with its cost:

- keep one file, and write citations as links to stable anchors: cheap, and GitHub
  resolves them, but the relation between records stays untyped;
- one file per decision, such as `docs/decisions/D-054.md`, with a front-matter
  header for the ID, title, status, date, the entry that replaced it, and typed links
  (supports, restates, reverses), and prose below it: the integrity checks become
  exact, a split becomes a new file and edited links, and each record gets its own
  history, at the cost of moving 126 entries and changing every tool that reads the
  log;
- decisions as data, with the Markdown generated for reading, as `behaviors.go` and
  `docs/relied-on-behaviors.md` already work: the most checkable, and the least
  pleasant to write in.

Whichever is chosen comes before any split. Splitting 59 entries and repointing their
citations in the present format would be redone in the new one. The choice also
decides what the decision log's table of contents is: a generated block in one file,
or a generated index file for a directory.

**Whether a host approves an unsolicited join request.** A person who is not on a
room's guest list cannot ask to be let in; §12a (room membership) and D-051 (stranger
pairing is not a supported case) leave open whether they should be able to, and
nothing is built for it.

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

**Citations credit an entry with something it does not hold.** Each resolves, so the
citation test passes, but the source says something else:

- `CLAUDE.md`, "No adapter machinery for a second host (D-043, D-113)": D-113 does not
  state it; D-110 (host order) does.
- `CLAUDE.md`, "is fixed at its first prompt and never changes (D-016)", and
  `docs/decisions.md`, "D-016 fixes that at the first prompt": D-056 (a session's
  room is fixed at first sight) decided that.
- `CLAUDE.md`, "No join tokens, and no join-by-name on a trusted network (D-024,
  D-026)", and `docs/spec-review.md`, "The bearer token was never necessary (D-024,
  D-025)": D-024 keeps a join code, and D-026 (no join token) removed it.
- `cmd/cogmer/tailcat.go`, "a peerId is an Ed25519 key … (D-020)": D-020 never names
  Ed25519; D-042 (peer identity is an Ed25519 key pair) decides it.
- In `docs/decisions.md`: D-017's Context, "D-015 gave rooms a generated id plus a
  human-chosen label", though D-015 holds no label; "The invitation format from D-017",
  though D-017 holds no invitation format; "Already decided (D-052, §29): interactive,
  blocking on another person…", words D-053 holds; "D-099 prefers the label because a
  word pair means nothing to a person weeks later", a reason D-094 gives; and
  "`common.sh` and D-107 both attribute to §3.1", though D-107 cites no § section.

**Three decisions' Status lines report what is no longer so.** D-041 (ship as a Claude
Code plugin) says "specified; not implemented", D-057 (a slash command is a thin
wrapper over the CLI) says "commands not yet built", and D-083 says "not yet
implemented", yet the plugin, its commands and the first opening of the view at
pairing all exist.

**The view leaves the derived name off a user's own turns, and no decision records
why.** Commit `33ce996` made the choice. D-021 (peer names are derived from the
identity) is cited for it, but D-021 says only what the name is for; it says nothing
about a user's own turns.

**Two comments describe the current-room pointer D-080 removed.** `main.go:878` and
`membership.go:686-688` still explain a machine-wide current room. D-080 (a terminal
command exists to be tested or to work when the plugin cannot) deleted
`SetCurrentRoom` and `CurrentRoom`, so a maintainer reading either comment is told
about a mechanism that does not exist. Found on 09-23, reading every decision for
the split list.

**`TestTheThreeEndingsOfAPairing` fails about one run in four.** Its subtest
"matched writes the name and the verification" fails at cleanup with "TempDir
RemoveAll cleanup: unlinkat …/.cogmer: directory not empty", so something is still
writing into the test's state directory after the subtest returns, most likely a
goroutine the pairing starts. It failed 2 of 8 runs on 09-23 on committed code, so
it is not caused by a recent change. A flaky test trains a maintainer to rerun
failures instead of reading them. The fix is to make the test wait for whatever
writes, or to stop that writer before the subtest returns.

## Needs somebody else

**Somebody who did not write cogmer uses it.** Phase 13 in `docs/phases.md`. The
repository is public, carries the marketplace manifest beside the plugin, and serves
releases the installer verifies against its pins (D-121, the repository is both the
release host and the marketplace). What it needs is a colleague installing, pairing,
joining, working, and saying what they hit in the order they hit it. It is the only
remaining work that can fail in a way nothing else detects, and the failure looks like
somebody quietly not using it again.

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
