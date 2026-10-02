# D-053 — `pair` is the machine-scope act, and `invite` is the room-scope act

**Date:** 2026-09-18 · **Status:** active · **Areas:** pairing, admission, commands

**Decision.** `cogmer pair <identifier>[@address] [name]` records the peer, records a
bootstrap address and runs the two-word comparison, in one command that both users run
at once on a call. `invite` is scoped to a room, so a pairing is made before any room
exists.

**Support.**
- A machine's known peers and a room's guests are two lists at two scopes. D-025 (a
  machine knows peers, and a room admits guests).
- An address recorded at pairing belongs to the known peer, which exists before any
  room, and synchronization targets include it, so a verification can precede the room it
  protects. `cmd/cogmer/sync.go`, `syncTargets`.
- An address in a pairing need be correct only once. D-018 (identity and reachability are
  separate).

**Rejected.**
- *One flat vocabulary in which recording a peer sits beside inviting to a room.* The two
  acts differ in scope, and a command that does not say which sends a user to the wrong
  one.

**Limits.** The surface that shows the words is D-088 (the pairing ceremony lives in the
view). Recording a peer without verifying it is D-165.

**Revisit when** a peer can be found without a typed address, since a pairing string
might then reduce to the identifier alone.
