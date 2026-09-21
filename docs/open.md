# Open

Work and unanswered questions. Nothing here is durable: every item either becomes a
decision, a change to the specification, or code — and is then **deleted**, not
marked done. The record of why lives in `decisions.md` and the record of what the
system is lives in the specification, so an item that has earned a permanent home
does not need one here too.

Being scratch is the point. Be untidy in it.

## The specification currently states something false

**Verification dials every address this machine knows.** `RunVerification` is handed
`syncTargets()`, so verifying one colleague dials every other. `PeerEndpoint` exists
now (D-103) and nothing uses it for this. Try the peer's own address first and keep
the sweep as a fallback.

**And so §4 promises an alarm that cannot be delivered.** It says an unexpected key
while verifying a named peer is worth telling the person about. It is not told, and
cannot be while wrong keys are the expected outcome of most dials. Fixing the sweep
is what makes the sentence true; nothing else needs writing.

## Decided and not built

**Re-pick the overlay relay when it cannot be reached.** D-104 pins it so the
address is stable. Nothing re-picks, so a machine that relocates past its pinned
relay is unreachable and nothing says so. Change driven by failure, never by
preference.

**Say when your own address changes.** Every pairing string and invitation already
handed out is then stale, and only the daemon can know.

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
