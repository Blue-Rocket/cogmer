# D-077 — Every room has its own URL, and no ambient value picks one

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Two objections, and the second landed on something introduced an hour
earlier: per-room URLs are mandatory, the machine-level pointer still has no
evident use, and **an environment variable is inadequate on a machine running
several sessions**.

The third is the serious one. `COGMER_ROOM` is **ambient** — exported once in
a profile, or inherited by every session a machine starts — and D-076 had just
placed it *above* the session's own room. So a single export would have answered
for every session regardless of which room it was in: exactly the failure D-064
removed, reinstated at higher precedence, in the same afternoon it was fixed
elsewhere. It is gone, along with the dead `LoadConfig` that also read it, and a
test now fails if anything reads it again.

**A room is named per invocation, or it is the session's.** Nothing ambient decides
who is admitted to what.

**Per-room URLs.** `/room/<name>` serves that room; `/` lists rooms and chooses
none, redirecting only when there is exactly one — a list of one is a question
nobody needs asked. The events endpoint and the event stream take the room as a
parameter, so two tabs stream two rooms and neither changes identity when somebody
creates a third. `watchLine` hands over the room's own address, which makes the line
printed at `create` a stable link rather than a window whose contents can move.

**What that settles about the pointer.** The view was its last consumer that could
not be given a session, and per-room URLs remove the need: the address carries the
room. What remains is a fallback for a terminal, where there genuinely is no session
to ask — and it is now reached only there, after the session has been asked and
found absent rather than merely unhelpful.

**The pattern worth naming, because it recurred three times in a day.** Ambient
state that answers "which room" is wrong wherever a session could have been asked:
once in the view, once in `where`, once in `currentRoom`. Each time it looked like a
convenience and behaved like a silent wrong answer. A session is the unit of
membership (§22), so anything that can ask a session must.

**Rejected.**
- *Ranking a room named in an environment variable above the session's own room.* An
  exported variable has the same value in every session a machine starts, so it
  would choose one room for all of them, and a command run in one room would act on
  another without an error. `84a0751:docs/room-choice-findings.md`, "An ambient
  variable ranked above the session answers for every session".

**Revisit when** a terminal command needs a room and there are several. Today the
fallback answers; the honest alternative is to require naming it, and the friction
only appears once rooms accumulate — which nothing yet archives.
