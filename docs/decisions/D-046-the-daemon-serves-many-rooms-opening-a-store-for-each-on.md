# D-046 — The daemon serves many rooms, opening a store for each on demand

**Date:** 2026-09-17 · **Status:** active · **Areas:** rooms, daemon

**Decision.** The daemon is the machine's local service and is not a room. It opens a
store for a room on demand, and starting the process creates no room.

**Support.**
- A room begins when somebody is invited into it, so a daemon that held a room from
  startup would hold one nobody created. D-022 (a room begins when someone is invited
  into it).
- A user can be in several rooms across different sessions, and a room outlives the
  session that created it. §12a (room membership).
- Which room a session is in is settled when its first hook fires, and a session in no
  room is an ordinary Claude Code session with nothing captured, injected or shared.
  D-056 (a session's room is fixed at first sight), §12a.

**Rejected.**
- *One room per daemon, chosen by an environment variable at startup.* The daemon
  would invent a room when pointed at one nobody had created, which D-022 forbids, and
  it could serve one room only.

**Revisit when** a user needs more than one daemon on a machine, such as one for each
account.
