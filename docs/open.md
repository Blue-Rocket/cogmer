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

**One peer that is offline delays sync with every other, and rooms are asked of peers who
are not in them.** `RunSync` in `cmd/cogmer/sync.go` asks each address one after another,
and `syncTargets` returns every address this machine has recorded, so a peer paired once
and never finished costs a dial that times out in about ten seconds on every round. In
`~/.cogmer/daemon.log` on 2026-10-12 the first failure for such a peer and the first for
the colleague a room was shared with are ten seconds apart. Each request to a peer also
names a room it is not a guest of, in a signed request, so that peer learns the room's
identifier. With one colleague the cost is about ten seconds in the view, and it grows
with the number of peers ever paired who are offline: six pairings with four offline
would lag the view by about forty seconds a round and could leave a colleague's last turn
out of what a prompt injects.

The fix has two parts. For each room, ask only that room's guests, which `RoomPeers`
already maps to addresses, and ask the addresses in parallel so an unreachable one waits
alone. Before changing which peers are asked, read whether relaying a third member's
events (§13) relies on asking a peer that is not a guest of the room, since nothing has
run that path.

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

An interactive Claude Code session on 2026-10-14 showed "Stop hook error: MCP server
'remote-devices' not connected" for a Stop hook that names a server that is absent. B31 (a hook of type
mcp_tool that names a server that is not there shows nothing) says nothing shows, and its check runs
with `claude -p`. Find out whether an interactive session differs, and if it does, whether the Cowork
plugin's hooks show the error when the bridge has not started. A user whose bridge is down would then
see an error on every prompt.

When a Cowork session has been seen to work with them, D-205 and D-207 become active. The probe left an
entry named `cogmer-host-probe` in the user's Claude Desktop configuration, the probe plugin installed in
Cowork, and the files `~/.cogmer-host-probe.*`, to be removed by the user.

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

## The room view

A reader of the shared transcript sees a turn as its author meant it to look.

**Render Markdown in the shared transcript.** The page's script, read in `cmd/cogmer/ui.html` on
2026-10-13 and not run, turns only these into markup: fenced code blocks, inline code, `**bold**`,
and a list of lines that each begin with a hyphen or an asterisk. A heading, a numbered list,
emphasis with underscores, a quotation, a table and a link all show as the characters typed, and
an assistant's turn is often written in them. The page escapes a turn before it applies any markup,
because a colleague's turn is untrusted text, and it makes no request outside the machine, so a
renderer has to keep both. Three questions follow from that. Whether to extend the hand-written
renderer or to carry a Markdown library inside the binary, which would have to be read and kept
current for the whole of what it accepts. How a link is shown: as text with its address, or as a
link the reader can follow, limited to `http` and `https`, opened with no referrer, since a
`javascript:` address or a link that carries the reader's page with it is how a colleague's text
would act in the reader's browser. And an image, which would load from an address a colleague
chose and so tell them when the reader looked, and is shown as its address and never fetched. No
test runs the page's script, so a renderer needs one that does, against turns that hold markup
meant to break out of the page. The Markdown export keeps the text as written.

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
