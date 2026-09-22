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

Phases 8 (local UI), 9 (identity) and 10 (pairing and room identity) are done, in
§31's stated order rather than numeric order:

```
Phase 8 local UI  →  Phase 9 identity  →  Phase 10 pairing + room identity
     →  Phase 5 offline  →  Phase 7 hardening + recovery from local loss
```

Phase 10 is done **except** a host approving an unsolicited join request, and
whether that should exist is undecided rather than pending (§12a, D-051).

**Phase 5, offline and reconnection, is next.** It is also the only thing that can
close review finding B2 (injection order), which cannot be exercised with one live
peer: the block then contains only that peer's turns, already in sequence. It needs
a peer reconnecting with a backlog alongside a live one.

Phase 0 (integration spike) and Phase 0a (compaction probe) are complete —
`docs/phase0-findings.md` and `docs/phase0a-findings.md`. Injected context survives
compaction, so the watermark is correct as written. Peer networking works: two peers
synchronized over the open internet in 0.44 s with identical event ordering, in
`docs/phase2-experiment.md`, which is also where §30's question is answered
affirmatively.

The `claude-team` binary is **not** tracked — `bin/` is ignored, because it is 17MB
per commit and `go build` reproduces it.

## The specification currently states something false

**Verification dials every address this machine knows.** `RunVerification` is handed
`syncTargets()`, so verifying one colleague dials every other. `PeerEndpoint` exists
now (D-103) and nothing uses it for this. Try the peer's own address first and keep
the sweep as a fallback.

**And so §4 promises an alarm that cannot be delivered.** It says an unexpected key
while verifying a named peer is worth telling the person about. It is not told, and
cannot be while wrong keys are the expected outcome of most dials. Fixing the sweep
is what makes the sentence true; nothing else needs writing.

**"Local-first" claims more than we deliver, wherever it appears without §3's
definition.** §3 defines it narrowly — a session survives every peer disappearing —
and under that definition every use inside the spec is sound. Three uses reach
people who will never see the definition and read as the ordinary, much stronger
claim: the spec's own first line, `README.md:3`, and `plugin/README.md:4`. Against
it: two peers behind NAT cannot meet without a relay operated by somebody else
(recorded in D-105, the overlay transport); D-029 (losing a room database) makes
recovery a refetch from peers, so the durable copy is distributed rather than local;
nothing before the first invitation is captured at all (D-022); and offline is Phase
5, unbuilt. Say peer-to-peer and serverless, which are true without qualification,
and keep "local-first" where §3's meaning is on the page.

**`plugin/README.md` still opens with "local-first".** `README.md` and the spec's
first line are the other two unqualified uses; §3 defines the term narrowly and
every use inside the spec is sound.

**D-076 is cited three times and does not exist, and should not be written.** No
commit in all 142 ever contained the heading; `323f46d` (2026-09-20) added the
citations in code and did not touch `decisions.md`. It held that a room-scoped
command inside a session acts on that session's room first — but D-080 records that
it also put `CLAUDE_TEAM_ROOM` above the session's own room, "exactly the failure
D-064 removed, reinstated at higher precedence, in the same afternoon it was fixed
elsewhere." Wrong within hours, and its surviving rule is stated better by D-077
(every room has its own URL).

The two mentions in `decisions.md:3788,3790` are correct as they stand: they narrate
what D-076 did, and rewriting history to point elsewhere would be worse than a
dangling number. Only `membership_test.go:697` is a true dangling reference — a
reader is sent to D-076 for reasoning that was never written. Point it at D-077 and
say the rule outlived the entry.

## Decided and not built

**Re-pick the overlay relay when it cannot be reached.** D-104 pins it so the
address is stable. Nothing re-picks, so a machine that relocates past its pinned
relay is unreachable and nothing says so. Change driven by failure, never by
preference.

**Say when your own address changes.** Every pairing string and invitation already
handed out is then stale, and only the daemon can know.

**Rooms are not yet what D-015 (rooms are session-scoped) describes.**
`CLAUDE_TEAM_ROOM` selects a single room per daemon process; there is no invite, no
membership tracking, and no archive. That is Phase 1 work — do not treat the current
shape as the design.

**Plugin packaging (D-041, installation is one line)** is not built, nor is local
network discovery (D-019's zero-configuration path), nor detection of a lost room
database (review C-1, Phase 7).

## Commands that mislead

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

## Undecided

**Whether to rebuild the post-quantum hedge.** D-104 removed the pre-shared key,
which was the only quantum-resistant element in the stack. It could be rebuilt at
our own layer from material the pairing exchange already produces — per peer, which
is better than one secret shared with everybody. Nothing depends on deciding this.

**Whether `verify` still earns its place.** `/peer-pair <name> --again` now does the
same job, and `verify` has no slash command, so it may be a subcommand nobody has a
route to.

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

`/room-list` and `/self-status` were both named without confirmation.

`VERSION` is 0.6.0 and has not moved across a large amount of work.

## Needs somebody else

**NAT to NAT, with a real colleague on a real home router.** The last unknown in the
transport, and not testable alone.

**Whether the public relay is an acceptable dependency.** Parked deliberately as a
precondition of Phase 13, where it stops being theoretical.

## Lost, and needs recovering from David

An item on **idempotency** was in `residual-concerns.md` and is gone: the file was
deleted on the strength of a reading from earlier in the session, and being
untracked there is no copy. `/peer-pair` repeated on a completed pairing was worked
in D-107 and D-108, which may or may not be what it said.

## Older, from the specification review

**A1** — §19 still does not say when delivery advances. Reopened because the
implementation settled it and the specification never caught up.
