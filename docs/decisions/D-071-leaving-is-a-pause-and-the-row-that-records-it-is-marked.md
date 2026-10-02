# D-071 — Leaving is a pause, and the row that records it is marked and not deleted

**Date:** 2026-09-19 · **Status:** active · **Areas:** rooms

**Decision.** Leaving is per session and is a pause. The session stops participating and
may rejoin the room it was in and no other, so its `session_rooms` row is marked
`left_at` and not deleted.

**Support.**
- A session may not move to another room, because injected context cannot be withdrawn,
  and returning to the room it was in is not a move, since the context it holds came from
  that room. §12a (room membership).
- Binding refuses a second room by finding a row, so deleting the row on leave would make
  leave then join the move the rule forbids, with two extra commands. `cmd/cogmer/membership_test.go`,
  `TestLeavingIsNotALaunderedMove` and `TestLeavingDoesNotForecloseReturn`.
- Membership is held by a session, so a peer with three sessions in a room leaves three
  times, and `/room-leave` runs inside the session it removes. At a terminal there is
  nothing to leave, and the command says so. §3.6 (session-scoped rooms),
  `cmd/cogmer/main.go`, `runLeave`.
- Leaving does not touch the room's events, so the transcript is as readable as it was a
  moment earlier. `cmd/cogmer/membership_test.go`, `TestLeavingLeavesTheRoomVisible`.

**Rejected.**
- *Deleting the row when a session leaves.* It reopens the move §12a forbids.

**Limits.** Leaving does not touch the guest list, since withdrawing admission is the
host's act with `revoke`, and it does not stop synchronization. Whether a peer should stop
serving a room it has left is not decided here.
