# D-103 — An address belongs to a peer and is stored in one place

**Date:** 2026-09-21 · **Status:** active · **Areas:** transport, admission

**Decision.** A peer's address is stored once, in `known_peers`. A room's sync targets are its
guests other than this peer, each joined to the address recorded for them, with the exclusion
of this peer written explicitly.

**Support.**
- An address that arrives with an identity was stored in a table with no peer column, so
  verification dialed every address because it could not ask which was a peer's, an unexpected
  key could not be an alarm, room addresses accumulated, and invalidation had nowhere to live.
  `cmd/cogmer/membership_test.go`, `TestAnAddressBelongsToAPeer`.
- The join yields identity with every address, so a failure can be attributed, a peer that moves
  overwrites one row, and expiry has something to attach to. `cmd/cogmer/sync.go`, `syncTargets`,
  `cmd/cogmer/membership_test.go`, `TestRoomPeersAreItsGuests`.
- No room holds an address for a non-guest, since `verifyRequest` refuses a request from anybody
  who is not a guest before an address is recorded, and joining admits the host as it records
  them. `cmd/cogmer/auth.go`, `verifyRequest`.
- A user is a guest of their own rooms and is never in their own known-peers list, so the join
  excludes this peer by saying so and does not rely on a missing row that somebody will later
  add for another reason. `cmd/cogmer/membership.go`, `RoomPeers`.
- With one store no reconciliation between two is needed, and no argument remains that one
  might hold a fresher address than the other. `cmd/cogmer/membership.go`, `RoomPeers`.

**Revisit when** an address is legitimately held for something that is not a peer.
