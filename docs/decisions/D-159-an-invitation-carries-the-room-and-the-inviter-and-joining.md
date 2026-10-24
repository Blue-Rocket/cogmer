# D-159 — An invitation carries the room and the inviter, and joining receives the room's guest list from the inviting machine

**Date:** 2026-09-17 · **Status:** active · **Areas:** admission, rooms

**Decision.** An invitation carries the room's name, the room's identifier, an address
where the inviting machine can be reached, and the inviting user's identifier, all of them
public. Joining records the room, receives the room's guest list from the inviting
machine, and stores the address against the inviting machine, and the joining user's
admission is the inviter's invitation on that list (D-235, a room's one guest list decides
whose events count). A session on another of the guest's machines
joins the room by name, from the list of rooms the user is a guest of that the user's
machines share (D-228, a user's colleagues, verifications, labels and rooms belong to the user and are copied between the user's machines), and needs no invitation.

**Support.**
- A guest did not create the room and cannot invent its identity, so the identity comes
  from the invitation. `cmd/cogmer/main.go`, `runJoin`.
- Synchronization is a pull in both directions, and every machine in the room refuses a
  request from a user who is not on the room's guest list, so a guest that recorded the room
  without a list holding its host could read the host and never be read.
  `cmd/cogmer/main.go`, `runJoin`.
- Storing an address updates a row that must exist, so joining records the
  inviting user before it stores the address. D-103 (an address belongs to a machine and is
  stored in one place).
- Nothing in an invitation admits anybody, because there is no token to hold. D-026
  (there is no join token at all).

**Rejected.**
- *An invitation that leaves out the inviting user's identifier.* The joining machine could
  not tell whether the machine that answers at the address is the inviter's, or whether the
  invitation on the list it receives was made by the colleague who invited its user.

**Limits.** How an invitation reaches the guest is D-105 (an invitation is delivered to a
paired guest as an offer over the paired channel).

**Revisit when** a guest has to join a room with no inviter named.
