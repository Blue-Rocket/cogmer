# D-071 — Leaving is a pause, and the row that records it is a tombstone

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Tracing the room lifecycle against the code found that `leave` did not
leave. It cleared the machine-level current-room pointer and nothing else: sessions
already bound stayed bound and kept publishing, the sync loop kept polling, and the
room's `state` column — written once as `joined` and never read — stayed as it was.
Its message was accurate about *new* sessions and easy to misread as having stopped
participating.

Three requirements were then stated: leaving must not foreclose rejoining the same
room, must not obscure the room's history, and must not affect the visibility of the
transcript as it had accumulated.

**The third one caught a live defect.** The browser view follows the current-room
pointer, and clearing that pointer was the whole of leaving — so leaving **blanked
the view**. The accumulated transcript disappeared at the moment a person stopped
adding to it, which is the opposite of what leaving should mean. Leaving no longer
touches the pointer at all; that pointer says what commands and the view are
looking at, which is a different question from whether a session is participating.

**The first one sharpened the invariant.** §12a forbids a session **moving to
another room**, because injected context cannot be withdrawn. Returning to the room
it was already in is not a move — the context it holds came from that room. So
rejoining is permitted, and only a different room is refused.

**The trap, which required the tombstone.** `BindSession` refuses a second room by
checking whether a `session_rooms` row exists. Had leaving *deleted* the row, then
leave-then-join would be exactly the move §12a prevents, with two extra keystrokes
— the session still holding the first room's turns and now publishing into a second
by way of the model's own output. The row is therefore kept and marked `left_at`,
so that two questions stop sharing one answer: **is this session in a room**, and
**which room has it ever been in**. A test asserts that a plain delete reopens the
hole.

**Leaving is per-session**, because §22 puts membership there: "Membership is held
by a Claude Code session, identified by its session ID." A peer with three sessions
in a room leaves three times, and that follows from where membership lives rather
than being a quirk. `/room-leave` runs inside the session it removes; at a terminal
there is nothing to leave, and it says so rather than reaching into sessions nobody
is looking at.

**What leaving deliberately does not touch.** The room's events. The guest list —
admission is the host's to withdraw with `revoke`, and leaving is your own act.
Synchronization, which continues: whether a peer should stop serving a room it has
left is a separate question, and answering it carelessly would silently degrade
somebody else's room.

**Still not implemented, and now the only part of §22's lifecycle that is not.** A
room never closes. Nothing archives, nothing goes dormant, and `state` remains
inert. That matters once rooms accumulate, and nobody has enough for it to bite.

**Revisit when** Phase 13 says whether rooms accumulate the way §22 assumed. If a
pair creates one room a week and never closes any, dormancy is the next piece; if
they reuse one room for months, it is not.
