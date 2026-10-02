# D-133 — Presence is not membership

**Date:** 2026-09-16 · **Status:** active

**Decision.** Membership is held by a session's ID, and ends only when the session
explicitly leaves or the room closes. Presence says whether a member can be reached now,
and lapses whenever a process exits, a machine sleeps or a network drops. A room closes
when every member has left, or after a dormancy long enough that resuming it is not
plausible, and the threshold is generous.

**Support.**
- A Claude Code session's ID survives its process exiting and being resumed, and
  survives compaction. `84a0751:docs/phase0a-findings.md`, and B14.
- A killed process, or a machine that loses power, sends no signal that its session
  ended. §12a (room membership).
- A closed room can never be rejoined, and its sessions can join no other room. D-016 (a
  session never joins a second room).

**Rejected.**
- *Treating a process exiting as leaving.* Two people closing their terminals for lunch
  would close the room, and neither could rejoin it or join another.

**Revisit when** Claude Code sessions cannot be resumed.
