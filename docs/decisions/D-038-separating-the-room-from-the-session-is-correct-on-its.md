# D-038 — Separating the room from the session is correct on its merits

**Date:** 2026-09-17 · **Status:** active · **Areas:** rooms, view

**Decision.** The room is shown outside the session, in a view of its own, and would be
even if Claude Code could show it inside one.

**Support.**
- A session is read closely and a room is glanced at, so interleaving them would bury the
  room inside the session and interrupt the session with arrivals not addressed to it.
  §17 (shared conversation UI).
- With three colleagues, a session carrying their turns would become unreadable, and the
  cost would fall on the user's own working view. §17.
- The model wants a colleague's turns in its context at a turn boundary, and the user
  wants them available to glance at, so injection serves the model and a view serves the
  user. §17.

**Rejected.**
- *Showing colleagues' turns inside the session, if Claude Code allowed it.* It is the
  interleaving above.

**Revisit when** users say they need colleagues' turns inside their own session.
