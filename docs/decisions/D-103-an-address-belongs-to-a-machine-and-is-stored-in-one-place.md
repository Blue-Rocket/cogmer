# D-103 — An address belongs to a machine and is stored in one place

**Date:** 2026-09-21 · **Status:** active · **Areas:** transport, admission

**Decision.** A machine's address is stored once, in `known_peers`. A room's sync targets are
the machines from which a session has joined the room, this user's own machines included and
only this machine excluded, each joined to the address recorded for it, with the exclusion of
this machine written explicitly. A colleague's machine learns the address of a machine newly
joined to the room from the signed address in that machine's sync requests (D-160, the address
in a sync request is signed).

**Support.**
- An address that arrives with an identity was stored in a table with no column for its owner,
  so verification dialed every address because it could not ask which was whose, an unexpected
  key could not be an alarm, room addresses accumulated, and invalidation had nowhere to live.
  `cmd/cogmer/membership_test.go`, `TestAnAddressBelongsToAPeer`.
- The join yields identity with every address, so a failure can be attributed, a machine that
  moves overwrites one row, and expiry has something to attach to. `cmd/cogmer/sync.go`,
  `syncTargets`, `cmd/cogmer/membership_test.go`, `TestRoomPeersAreItsGuests`.
- No room holds an address for a non-guest, since `verifyRequest` refuses a request from anybody
  who is not a guest before an address is recorded, and joining receives the room's guest list
  from the inviting machine as it records that machine's address (D-159, an invitation carries
  the room and the inviter, and joining receives the room's guest list from the inviting
  machine). `cmd/cogmer/auth.go`, `verifyRequest`.
- A machine that is known and is in no room is not asked, and a guest's machine is asked only
  about the rooms its user is a guest of, so a stranger is not sent a room's identifier. `cmd/cogmer/sync.go`,
  `syncTargets`, `roomsToAsk`, `cmd/cogmer/syncscope_test.go`,
  `TestAPeerInNoRoomIsNotAskedAndAGuestIsAskedOnlyAboutItsOwnRooms`.
- A user is a guest of their own rooms, and this machine is never in its own known-peers list,
  so the join excludes this machine by saying so and does not rely on a missing row that
  somebody will later add for another reason. `cmd/cogmer/membership.go`, `RoomPeers`.
- With one store no reconciliation between two is needed, and no argument remains that one
  might hold a fresher address than the other. `cmd/cogmer/membership.go`, `RoomPeers`.

**Rejected.**
- *Leaving the user's own machines out of the sync targets.* A user's sessions in one room
  share context with one another as they do with a colleague's, so a turn written on one of the
  user's machines has to reach the others that joined the room.
- *Every machine of a guest as a sync target.* A machine from which no session has joined a
  room holds nothing of it, and asking it would send the room's identifier where nothing reads
  it.

**Revisit when** an address is legitimately held for something that is not a machine.
