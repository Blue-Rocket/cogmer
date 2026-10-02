# D-159 — An invitation carries the room and the inviter, and joining admits the inviter

**Date:** 2026-09-17 · **Status:** active · **Areas:** admission, rooms

**Decision.** An invitation carries the room's name, the room's identifier, an address
where the inviting peer can be reached, and the inviting peer's identifier, all of them
public. Joining records the room, admits the inviter to its guest list, and stores the
address against them.

**Support.**
- A guest did not create the room and cannot invent its identity, so the identity comes
  from the invitation. `cmd/cogmer/main.go`, `runJoin`.
- Synchronization is a pull in both directions, and a guest list is held by each peer,
  so a guest that recorded the room without its host could read the host and never be
  read. `cmd/cogmer/main.go`, `runJoin`.
- Storing an address updates a row that must exist, so joining admits the
  inviter before it stores the address. D-103 (an address belongs to a peer, and is
  stored in one place).
- Nothing in an invitation admits anybody, because there is no token to hold. D-026
  (there is no join token at all).

**Rejected.**
- *An invitation that leaves out the inviter's identifier.* The guest's guest list would
  refuse the host's requests, and collaboration would run one way.

**Limits.** How an invitation reaches the guest is D-105 (an invitation travels over the
channel that pairing established).

**Revisit when** a guest has to join a room with no inviter named.
