# D-053 — `pair` is the act with a colleague, and `invite` is the room-scope act

**Date:** 2026-09-18 · **Status:** active · **Areas:** pairing, admission, commands

**Decision.** `cogmer pair <identifier>[@address] [name]` records the colleague, records a
bootstrap address for one of the colleague's machines and runs the two-word comparison, in
one command that both users run at once on a call. A user pairs with each colleague once,
and a machine either of them adds later speaks under that pairing (D-219, a user is a
master key that certifies machines through a shared key). `invite` is scoped to a room, so
a pairing is made before any room exists.

**Support.**
- A user's known peers and a room's guests are two lists at two scopes. D-025 (a user
  knows colleagues, and a room admits guests).
- An address recorded at pairing is recorded before any room exists, belongs to one of the
  known colleague's machines, and is among the synchronization targets, so a verification
  can precede the room it protects. `cmd/cogmer/sync.go`, `syncTargets`.
- An address in a pairing need be correct only once. D-018 (identity and reachability are
  separate).

**Rejected.**
- *One flat vocabulary in which recording a peer sits beside inviting to a room.* The two
  acts differ in scope, and a command that does not say which sends a user to the wrong
  one.

**Limits.** The surface that shows the words is D-088 (the pairing ceremony lives in the
view). Recording a peer without verifying it is D-165.

**Revisit when** a colleague's machine can be found without a typed address, since a
pairing string might then reduce to the identifier alone.
