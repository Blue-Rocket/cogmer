# D-151 — The daemon outlives its session, and can be found and stopped

**Date:** 2026-09-17 · **Status:** active · **Areas:** daemon

**Decision.** The daemon outlives the session that started it, and the user whose machine
it runs on can find it and stop it.

**Support.**
- A room may have members in several sessions on one machine. §29 (the experience we
  want).
- `stop` finds a daemon by the addresses it holds. D-123 (`stop` finds a daemon by the
  addresses it holds).

**Rejected.**
- *Starting a daemon with each session and stopping it at the end.* Starting it again and
  again is worse than leaving it running, and a room's other sessions would lose it.

**Revisit when** the daemon has to stop when its last session ends.
