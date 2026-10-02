# D-083 — The view is opened at a first pairing, not at room creation

**Date:** 2026-09-20 · **Status:** active (sequencing decision; not yet implemented)

**Context.** With auto-open shown to work (D-082), the remaining question was *when*
to spend it. The obvious answer was the first `create` or `join` — the moment a room
first exists and therefore first has something to show.

**That is the wrong moment, because it is not the first one.** Pairing precedes room
formation in the user's experience, and it precedes it by design: D-053 separates the
two scopes, pairing happens once with a colleague and outlasts every room, and D-054
makes verification a **gate** — an unverified peer is refused sync before any room
content moves at all. A person therefore meets this system at pairing, not at a room.
Opening the view at room creation would leave the very first unfamiliar step — a
two-word comparison read aloud on a call (D-055) — happening with nothing on screen,
and would then open a window for the second step, which is the more familiar one.

**So the first open belongs at the first pairing.** It is also the step that most
needs a surface: the two words have to be read by both people at once, and a terminal
is the wrong place to put something a non-person must find and compare under time
pressure.

**Not settled here:** whether pairing and verification are *driven* from the view or
merely displayed in it, and whether an open recurs on later pairings or happens only
once ever. Both were raised and neither is answered; do not build either on
speculation.

**Revisit when:** the pairing ceremony gets a surface, or D-055's ceremony changes.
