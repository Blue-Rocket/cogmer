# D-009 — A failed behavior check never blocks the room

**Date:** 2026-09-16 · **Status:** active

**Decision.** A failed behaviour check reports which behaviour changed and what that
breaks, and the room forms anyway.

**Support.**
- A session is never worse for having cogmer installed, and collaboration that is
  unavailable leaves Claude Code working normally. §3.1 (first, do no harm).

**Rejected.**
- *Refusing to form the room.* A degraded feature would become a broken session, which
  §3.1 forbids.

**Revisit when** a check fails whose failure would make collaboration harm the session
rather than degrade it.
