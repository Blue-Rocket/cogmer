# D-045 — Rooms are records with a guest list; admission is enforced

**Date:** 2026-09-17 · **Status:** active · **Areas:** admission, rooms

**Decision.** A room is a record in `membership.db`, holding its identifier, its
generated name and its guest list, and a sync request is refused unless the peer that
signed it is a guest of the room.

**Support.**
- Authenticating a caller establishes who is asking, not whether they may. D-044 (sync
  requests are signed).
- A stranger who authenticated correctly reads nothing, and the host records that the
  peer is authenticated and not a guest. `cmd/cogmer/auth_test.go`.
- Membership lives beside the identity, outside every room's database. §22
  (persistence).

**Rejected.**
- *Admitting every authenticated caller.* It gives the system a name for its visitors
  and nothing more.

**Revisit when** a room needs to admit a peer by something other than a guest-list entry
or a host's approval.
