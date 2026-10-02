# D-056 — A session's room is fixed when its first hook fires, and never changes

**Date:** 2026-09-18 · **Status:** active · **Areas:** rooms

**Decision.** A session binds to a room at first sight, and the binding does not change. No exception
exists for a session that has received nothing, and nobody is asked at binding, since
nothing knows a session exists until its first hook fires.

**Support.**
- `RoomForSession` reports the bound room whatever the session has received, so no check
  on what reached a session stands between it and its room. `cmd/cogmer/membership.go`,
  `RoomForSession`, and `cmd/cogmer/membership_test.go`, `TestABoundSessionCannotBeMoved`.
- Moving a session that has received nothing needs a judgment made from outside about
  what a context window holds, which cannot be inspected, and one error is irreversible.
  D-016 (a session is in one room at most, and never joins a second).
- Correcting a room joined by mistake means starting a session, which costs less than
  that judgment. §12a (room membership).

**Rejected.**
- *Letting a session that has received no colleague's turns move to another room.* The
  rule has to be right about state nobody can inspect, and being wrong once cannot be
  undone.

**Revisit when** a case appears for moving a session that has received nothing, together
with a way to know that from outside the session that depends on no flag.
