# Open

Actions not yet taken and decisions not yet made. Nothing here is durable: every item
either becomes a decision, a change to the specification, or code — and is then
**deleted**, not marked done. The record of why lives in `decisions.md` and the
record of what the system is lives in the specification, so an item that has earned a
permanent home does not need one here too. Nothing here records status.

Items are grouped by milestone: a set of related functionality, named in the sentence
that opens its section. A milestone's section is deleted with its last item, and
nothing records it afterwards.

Being scratch is the point. Be untidy in it.

## Installing and the first commands

A new user installs the plugin, and the first commands they run answer instead of failing.

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
  no binary, as the first-use item under "Installing and the first commands" describes.
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
- Shorten the wait with a smaller download, as "Whether the download is slow enough to
  matter" describes.
- Ship the binary inside the plugin, ruled out there for what it adds to the
  history.

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

## Running the daemon

The daemon a machine runs is the right one, and says so when something is in its way.

**A port held by another cogmer daemon is reported as held by something else.**
When the peer-sync port is taken, `runDaemon` calls `reportDaemonBlocked` without
asking what holds it, and the message says "held by something that is not a cogmer
daemon" regardless. On 09-22 it was a cogmer daemon, one left running from a test
with a different `COGMER_HOME`, and the message sends the reader looking for a
different program. Only the hooks port is ever probed, through
`daemonAlreadyServing`, which reads `/healthz` over plain HTTP. The peer mux serves
`/healthz` too, but over TLS, and nothing asks it.

**A blocked address does not say that `stop` clears it.** When a daemon cannot bind,
`reportDaemonBlocked` prints an `lsof` line, where D-123 (finding a daemon by the
addresses it holds) has it name `stop` by its full path, from `invocation()`. Read
from the code on 10-01.

Three things about `stop` are undecided. The first is whether `stop` then starts this
installation's daemon. Clearing the way is almost always why somebody runs it, but a
person may also want it simply stopped. The second is whether it gets a slash
command. The person is in a session, not at a terminal, and the binary is not on
PATH, so from a terminal they have to type `~/.cogmer/bin/cogmer stop`. The third is
Windows, which has no `lsof`, so neither `stop` nor the replacement of a daemon of
another version (D-158) can find the process holding a port there. `netstat -ano`
gives the pid there, or `stop` can report the port and fall back to moving this
daemon aside.

## Reaching a peer on another network

Two peers on different networks, each behind its own router, reach each other with nothing configured.

**NAT to NAT, with a real colleague on a real home router.** The last unknown in the
transport, and not testable alone.

**Only one address is ever tried for a peer, and none is demoted, aged or
rehabilitated.** Section 4 of the specification, "The lifecycle of a peer address",
has an address used as one of several candidates tried together, demoted by
silence, invalidated by a wrong key, aged at a rate set by what it names, and
restored by evidence. `known_peers` holds one endpoint per peer with the time it was
recorded, replaced when the peer advertises another (read on 2026-09-28). D-103 (an
address belongs to a peer, and is stored in one place) gave each address the owner
the lifecycle needs, and nothing past that is built.

**Re-pick the overlay relay when it cannot be reached.** D-104 pins it so the
address is stable. Nothing re-picks, so a machine that relocates past its pinned
relay is unreachable and nothing says so. Change driven by failure, never by
preference.

**Say when your own address changes.** Every pairing string and invitation already
handed out is then stale, and only the daemon can know.

**Whether the public relay is an acceptable dependency.** Reaching a peer across NAT
works through a public relay operated by a third party, used with no account and no
configuration of ours. It has to be settled before somebody else's conversation
crosses it. It has three parts, with different answers: whether depending on a relay
nobody here operates fits §4's rule that no transport is a prerequisite; what the
relay observes, since it cannot read what it carries but sees which nodes talk, when,
and how much; and whether unconfigured use of somebody's free infrastructure is
something to build a product on.

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

**Nothing lets a room show its conversation without injecting it.** Section 28 of the
specification has a setting per room that keeps a room visible to its members while
injecting none of it into their sessions, for a room whose conversation should not
reach another user's model provider. No such setting exists in the code (read on
2026-09-29), and no decision records it.

**A room never closes, so nothing is archived.** D-132 (a closed room's log is archived)
has a closed room kept as an archive that can be read but never rejoined, and the
specification freezes a room's sequences at that point. Membership ends per person
through `leave` and `revoke`; the room itself has no closed state.

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


## Compaction

A colleague's turns survive a compaction, and a room shows when one happened.

**B19 checks a colleague's turn that was the conversation's subject, never one that was
incidental.** The probe in `cmd/cogmer/probe.go` injects "The codeword for this check is
…" and asks for the codeword after compacting. Test B in
`84a0751:docs/phase0a-findings.md` showed that a turn the conversation never discussed
survived too, which is the case a real room produces, and only a person re-running it by
hand checks it again. A `--deep` probe that runs unrelated turns between injecting and
compacting would check it, at the cost of more turns in a check that takes about 40s.

**Whether automatic compaction can start during a turn is unknown.** Automatic
compaction never fired in the compaction probe of 2026-09-16, even at 378,916 tokens
against a 100k threshold, so `84a0751:docs/phase0a-findings.md` shows only that none
started during that turn. Nobody has checked what turn reassembly does with a compaction
marker and summary record in the middle of the turn it reassembles.

**D-007 (record `COMPACTION` events as observability) is not built.** No code records
a compaction, so if the summarizer stops keeping a colleague's turns, a room's
history shows nothing at the moment it happened. §7 lists the event type among those
to design for later. Either the event is built, or D-007 is withdrawn.

## What reaches the machine, and what leaves it

Nothing outside a room can read it, alter a colleague's words, or make a record a colleague holds unreadable, and what leaves the machine is known.

**A web page can read the local API, and only the `Origin` check keeps it from
writing, once its site's name points at 127.0.0.1.** A site can load its page from
its own address on port 4782, then point its name at 127.0.0.1 (DNS rebinding). The
browser then treats the page's requests to the daemon as same-origin, so it sends
`X-Cogmer` without asking and lets the page read the replies. `/`, `/events` and
`/stream` have no check, so the page can read a room once it knows the room's
address, which was not checked. The state-changing routes are refused only because
`guardLocal` in `cmd/cogmer/localguard.go` rejects the page's `Origin`, a check that
passes a request with no `Origin` at all. The daemon answers a request whose `Host`
names another site: on 2026-09-24, `curl -H 'Host: evil.example:4782'` got 200 from
`/` and `/healthz`. None of this was reproduced in a browser.
`docs/explanations/dns-rebinding.md` walks through each scenario.

The fix is to refuse, on every local route, any request whose `Host` is not
`127.0.0.1`, `localhost` or `[::1]` with the daemon's port. A page can reach the
daemon under its own name, or name the daemon in `Host` and be treated as another
origin, but not both. Demonstrate it before and after, as D-087 (the local API
requires a header a web page cannot send) was demonstrated: a hosts-file entry
pointing a made-up name at 127.0.0.1 reproduces the end state of rebinding. D-087
calls the header check the one the defence rests on and the `Origin` check a second
layer, so the change rewrites that part of it under W-38 (a partial change rewrites
the earlier entry).

**The boundary around injected turns is removed from their text, when it could be
chosen to be absent from it.** `FormatTeamContext` in `cmd/cogmer/daemon.go` generates
a random value for the boundary, then deletes any copy of that value from each
colleague's turn, so a turn that contains it arrives altered. A random 20-character
value almost never occurs in a turn, so the effect is rare, but the text is changed
when it does. Choosing a value that occurs in none of the turns in the block, and
generating another in the rare case that one does, would leave every turn as written
and need no removal. `TestTeammateContentCannotEscapeTheBlock` in
`cmd/cogmer/transcript_test.go` checks the boundary and should pass unchanged. D-040
(the injected block is fenced with an unforgeable value) records the removal, so the
change rewrites that part of it under W-38 (a partial change rewrites the earlier
entry).

**No test fails when a change to the signing bytes or the wire format stops an
existing record from being read.** Every signing test in `cmd/cogmer/keys_test.go`
signs a fresh event and then verifies it, so an edit to `signingBytesV3` or
`eventBytes` changes the signer and the verifier together and every test still
passes (read from the code on 2026-09-24). An event signed under scheme v3 before the
edit, held on a colleague's machine, would then fail to verify, with an error that
reads as a forgery. D-058 (signature schemes are kept, never replaced) depends on
that edit never happening, and nothing detects it. The wire format has the same gap:
nothing decodes a sync message written at `minWireVersion` with the current code.

The fix is a v3 event signed once and checked in, as a constant or a file under
`cmd/cogmer/testdata/`, with a test that `Verify` accepts it, and the same for a sync
message at each version `speaks` accepts. Each new scheme or wire version adds its
own frozen record when it ships.

**What leaves the machine is recorded only for 0.6.0.**
`84a0751:docs/what-leaves-findings.md` read every outbound path on 2026-09-21, at 0.6.0.
Two have changed since: releases come from GitHub (`plugin/release-url.txt`), and peers
reach each other through Tailscale's DERP relays, choosing a region in
`loadTailcatRegion` in `cmd/cogmer/tailcat.go`, which may fetch a relay map from a
server nobody has named (read on 2026-09-25, not traced further). D-115 (a centralized
component must trace to a disclosed tradeoff) rests on that record. Reading every
outbound path again at the current version, including where `loadTailcatRegion` fetches
from, would say what leaves now. Three of the properties that do not leave are kept only
by the absence of code, and each could be a test instead, such as one that the embedded
view holds no absolute URL.

**Whether to rebuild the post-quantum hedge.** D-104 removed the pre-shared key,
which was the only quantum-resistant element in the stack. It could be rebuilt at
our own layer from material the pairing exchange already produces — per peer, which
is better than one secret shared with everybody. Nothing depends on deciding this.

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
only process is `CLAUDE.md`'s "Read `docs/decisions.md` before proposing a change", a
log of about 6,800 lines, and it names none of the other documents. On 2026-09-25 and
2026-09-28 a session proposed moving implementation detail out of the specification,
restating decisions in it, and keeping the two-peer result because a decision said so,
and each was caught by the maintainer rather than by a process. A skill run before a
design is proposed, like `writing-review`, would take the areas the proposal touches
from `CLAUDE.md`'s "Where to read before changing something" and a search of the
documents, and report what the proposal must satisfy and what it conflicts with in the
values, the specification, the decisions and their **Rejected.** fields, the behaviour
registry, the patterns and `open.md`, and what records it would need. Each point would
quote its source, and a test would drop a quote that is not in the file. It would run on
any proposal that needs a decision entry or a change to the specification. It is only
as good as the decision log, so it waits until the log is rewritten, and it checks
consistency, not judgement: whether to proceed stays the maintainer's call.

**Nothing records which decisions and rules the maintainer ratified.** A Claude session
wrote most of `docs/decisions.md` and `docs/writing.md`, and an entry the maintainer
approved reads the same as one a session made and logged by itself. Later sessions then
defend earlier sessions' choices as settled: on 2026-09-25 D-122 (the two READMEs) put
the two-peer result in `README.md`, W-75 carried the requirement over, and it was
defended twice before the maintainer removed it. A field on each decision and each rule
saying whether the maintainer ratified it would show which choices rest on a person's
judgement. What it costs is a pass over the existing entries to set the field, which
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
does that cogmer relies on is in the behaviour registry, `cmd/cogmer/behaviors.go`, and
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
`CLAUDE.md`'s "Where to read before changing something" points at entries by area, so a
fact reaches a session when the work touches what it constrains rather than on every
turn (W-26).

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

The answers settled on 2026-09-29 change those counts, and they are rules in
`docs/writing.md` now: the log stays one file (W-28), a repeated decision folds into the
entry that holds it, an entry is rewritten to what still holds before it is split (W-40),
and a plan becomes a "removed" tombstone (W-37). Applied to the notes, 52 entries remain
to split, adding 78 entries rather than 86:

- no longer split: D-098 and D-100, moved to `docs/writing.md`; D-002 and D-032,
  removed as plans; D-020 and D-056, whose second decisions fold into D-024 and D-016;
  and D-062, whose first decision D-068 holds, so it is rewritten to its second;
- still split with one decision fewer: D-080, whose prefix rule folds into D-096;
- D-061 splits into its four decisions about the system, and its title about Phase 7
  goes.

The 146 citations that mean something other than an entry's first decision have not
been recounted.

Five entries lack a **Decision.** field: D-068, D-083, D-086, D-088 and D-089.

**Six citations in `docs/decisions.md` credit an entry with something it does not
hold.** Each resolves, so the citation test passes, but the source says something
else. "D-016 fixes that at the first prompt" describes what D-056 (a session's room
is fixed at first sight) decided. D-017's Context says "D-015 gave rooms a generated
id plus a human-chosen label", though D-015 holds no label. "The invitation format
from D-017" names a format D-017 does not hold. "D-099 prefers
the label because a word pair means nothing to a person weeks later" gives a reason
D-094 holds. "`common.sh` and D-107 both attribute to §3.1" is untrue of D-107,
which cites no § section. D-110 calls the thinking-block exclusion "§3.5's", but
§15 (capturing Claude responses) holds it. These are fixed when the entries that hold
them are rewritten.

**Three decisions' Status lines report what is no longer so.** D-041 (ship as a Claude
Code plugin) says "specified; not implemented", D-057 (a slash command is a thin
wrapper over the CLI) says "commands not yet built", and D-083 says "not yet
implemented", yet the plugin, its commands and the first opening of the view at
pairing all exist.

**The view leaves the derived name off a user's own turns, and no decision records
why.** Commit `33ce996` made the choice. D-021 (peer names are derived from the
identity) is cited for it, but D-021 says only what the name is for; it says nothing
about a user's own turns.

**`cogmer`'s usage text describes a current room, which D-080 removed.** `usage` in
`cmd/cogmer/main.go` says `join` makes "a room current, so new sessions join it", and
`leave` leaves "the current room". A room is the session's own (D-064), so a user
reading the help is told how something works that does not exist (read on
2026-09-25).

**Two comments describe the current-room pointer D-080 removed.** `main.go:878` and
`membership.go:686-688` still explain a machine-wide current room. D-080 (a terminal
command exists to be tested or to work when the plugin cannot) deleted
`SetCurrentRoom` and `CurrentRoom`, so a maintainer reading either comment is told
about a mechanism that does not exist. Found on 09-23, reading every decision for
the split list.

**`TestAnAddressBelongsToAPeer` fails when two random keys derive the same name.** It
admits two fresh identities under `PeerName` of each, and when the two word pairs
collide, `Allow` refuses the second: "you already know a different key as
\"glad-bobcat\"". It failed once and passed six times in a row on 2026-09-25. Giving
the two peers fixed, distinct names would remove the chance.

**Five behaviour checks have no test that makes them fail.** D-130 (every behaviour
check has a test that makes it fail) requires one for each, and no test names B03,
B05, B07, B11 or B20 (read on 2026-09-29). Nothing checks the requirement, so a check
for any of the five could pass whatever Claude Code did. A test for each, and a test
that every entry in the registry is named by one, would close it.

**Two tests in `ui_test.go` assume an order the store does not promise.**
`TestTheUnverifiedMarkerInTheViewIsAFact` takes the last event of a snapshot to be the
user's own turn, and `TestTheViewCarriesTheNameYouChose` takes the first to be the named
peer's. Events are ordered by timestamp and then by peer, and each test appends its
events within one second under random peer identifiers, so the order varies from run
to run. Each failed once in six or fewer runs on 2026-09-28 and 2026-09-29. Finding
each event by its content or its peer, or giving the events distinct timestamps, would
remove the chance.

**`TestTheThreeEndingsOfAPairing` fails about one run in four.** Its subtest
"matched writes the name and the verification" fails at cleanup with "TempDir
RemoveAll cleanup: unlinkat …/.cogmer: directory not empty", so something is still
writing into the test's state directory after the subtest returns, most likely a
goroutine the pairing starts. It failed 2 of 8 runs on 09-23 on committed code, so
it is not caused by a recent change. A flaky test trains a maintainer to rerun
failures instead of reading them. The fix is to make the test wait for whatever
writes, or to stop that writer before the subtest returns.

**Move `plugin/commands/*.md` to the `skills/<name>/SKILL.md` layout.** The
documentation calls `commands/` legacy and says the two are loaded identically. D-119
deliberately did not do it in the same pass, because a directory restructure is a bad
thing to bury a §3.1 fix inside. `TestNoCommandIsModelInvocable` reads the old path
and would need to follow.

**Whether `verify` still earns its place.** `/cogmer:peer-pair <name> --again` now
does the same job, and `verify` has no slash command, so it may be a subcommand
nobody has a route to.
