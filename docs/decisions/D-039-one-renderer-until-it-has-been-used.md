# D-039 — One renderer until it has been used

**Date:** 2026-09-17 · **Status:** active · **Areas:** view

**Decision.** The browser view is the only renderer of a room. A terminal pane beside the
session, or a notification when a colleague's turn arrives, is built only once using the
view shows it is needed.

**Support.**
- The view was used on 2026-09-17 and judged the right avenue: a separate window is
  consulted rather than forgotten. `84a0751:docs/decisions.md`.
- The terminal is not a user experience. D-086 (the terminal is not a user experience;
  the view is the surface).

**Rejected.**
- *Building a terminal pane and a notification before the view had been used.* Each would
  encode a guess about how a room is watched, and have to be maintained whether or not
  the guess held.

**Limits.** It does not decide whether a colleague's arrival should be announced.

**Revisit when** use shows that the view is not consulted, or that a user misses a
colleague's turns for want of an announcement.
