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

## Safe for somebody else's conversation

Nothing outside a room can read it, alter a colleague's words, or make a record a colleague holds unreadable, and what leaves the machine is known.

**What leaves the machine is recorded only for 0.6.0.**
`84a0751:docs/what-leaves-findings.md` read every outbound path on 2026-09-21, at 0.6.0.
Two have changed since: releases come from GitHub (`plugin/release-url.txt`), and peers
reach each other through Tailscale's DERP relays, choosing a region in
`loadTailcatRegion` in `cmd/cogmer/tailcat.go`, which fetches the relay map from
`tailcat.dev` (read on 2026-10-02). D-115 (a centralized
component must trace to a disclosed tradeoff) rests on that record. Reading every
outbound path again at the current version would say what leaves now. Three of the properties that do not leave are kept only
by the absence of code, and each could be a test instead, such as one that the embedded
view holds no absolute URL.

**The specification does not say that Claude's thinking blocks are never published.**
`cmd/cogmer/transcript.go` drops them and `cmd/cogmer/transcript_test.go` requires that, and
the specification does not mention them (read on 2026-10-01). Section 15 (capturing Claude
responses) says a response is published whole and never summarized, which reads as including
everything Claude produced. A reader of the specification could conclude the opposite of what
the code does, about content a user would not expect to leave their machine.

The fix is a sentence in section 15 stating that a response never includes its thinking blocks,
and a reason for it, which the code comment gives only as "deliberately never published".

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

## Checks and comments that say what is true

A decision, a comment or a test says what the code does, and a change that breaks it is caught.

**Five behavior checks have no test that makes them fail.** D-130 (every behavior
check has a test that makes it fail) requires one for each, and no test names B03,
B05, B07, B11 or B20 (read on 2026-09-29). Nothing checks the requirement, so a check
for any of the five could pass whatever Claude Code did. A test for each, and a test
that every entry in the registry is named by one, would close it.

**Nothing tests that a lost room is reported when it is opened.** D-169 (a lost room
state is reported when the room is opened) has `reportLostState` in
`cmd/cogmer/daemon.go` compare the highest sequence the store holds for this peer with
the highest recorded as issued, and no test in `cmd/cogmer` calls it or looks for its
"ROOM STATE LOST" line (read from the code on 2026-10-01). A change that stopped the
comparison, or that skipped the call on open, would leave a user with a quiet room and
no notice, and every test would pass.

The fix is a test that issues events in a room, deletes the room's database, opens the
room again, and requires the report, together with a test that a room holding what was
issued reports nothing.

**The tests for D-164 (the two words come from the PGP biometric word list, alternating
by position) check less than the decision states.** `TestWordlistsAreWholeBytes` and
`TestTheTwoWordsComeFromDifferentLists` in `cmd/cogmer/sas_test.go` require that each
list holds 256 distinct words and that the two lists share none (read from the code on
2026-10-01). Nothing requires that `sasEven` and `sasOdd` are the PGP biometric word
list, so an edit that replaced words with others, or one that merged duplicates away
and refilled the list, passes every test. Nothing requires that no word is also in
`peerAdjectives`, `peerAnimals`, `roomSky` or `roomLand`, which the decision names and
its **Limits.** records as unchecked. A user comparing two words beside a derived peer
name could then compare the wrong one.

The fix is a test that the two lists equal a copy of the PGP list kept under
`cmd/cogmer/testdata/`, and a test that no word of either list appears in the four
name vocabularies. D-164's **Limits.** is deleted when the second test exists.

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

**`cogmer`'s usage text describes a current room, which D-080 removed.** `usage` in
`cmd/cogmer/main.go` says `join` makes "a room current, so new sessions join it", and
`leave` leaves "the current room". A room is the session's own (D-064), so a user
reading the help is told how something works that does not exist (read on
2026-09-25).

**Two comments describe the current-room pointer D-080 removed.** `main.go:878` and
`membership.go:686-688` still explain a machine-wide current room. D-080 (there is no
current room, and a room-scoped command gets its room from its session) deleted
`SetCurrentRoom` and `CurrentRoom`, so a maintainer reading either comment is told
about a mechanism that does not exist. Found on 09-23, reading every decision for
the split list.

**The view leaves the derived name off a user's own turns, and no decision records
why.** Commit `33ce996` made the choice. D-021 (peer names are derived from the
identity) is cited for it, but D-021 says only what the name is for; it says nothing
about a user's own turns.

**D-007 (record `COMPACTION` events as observability) is not built.** No code records
a compaction, so if the summarizer stops keeping a colleague's turns, a room's
history shows nothing at the moment it happened. §7 lists the event type among those
to design for later. Either the event is built, or D-007 is withdrawn.

**Nothing lets a room show its conversation without injecting it.** Section 28 of the
specification has a setting per room that keeps a room visible to its members while
injecting none of it into their sessions, for a room whose conversation should not
reach another user's model provider. No such setting exists in the code (read on
2026-09-29), and no decision records it.

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

**NAT to NAT, with a real colleague on a real home router.** The last unknown in the
transport, and not testable alone.

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
address belongs to a peer, and is stored in one place) gave each address the owner
the lifecycle needs, and nothing past that is built.

**Re-pick the overlay relay when it cannot be reached.** D-187 pins it so the
address is stable. Nothing re-picks, so a machine that relocates past its pinned
relay is unreachable and nothing says so. Change driven by failure, never by
preference.

**Say when your own address changes.** Every pairing string and invitation already
handed out is then stale, and only the daemon can know.

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
