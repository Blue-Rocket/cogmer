# D-025 — A machine knows peers, and a room admits guests

**Date:** 2026-09-16 · **Status:** active · **Areas:** admission, pairing

**Decision.** There are two lists at two scopes. A machine's known peers are the
identifiers it has learned and the names it knows them by, durable and outlasting every
room. A room's guests are the known peers who may enter it, created with the room and
archived with it. Revoking withdraws a peer's admission to one room, and forgetting a
peer discards the identity and every admission it held.

**Support.**
- A user may know six colleagues and admit two of them to a room about customer data.
  §12 (forming a room).
- Both lists can be inspected and changed, with `/cogmer:peer-list`,
  `/cogmer:peer-forget`, `/cogmer:room-status`, `/cogmer:room-invite` and
  `/cogmer:room-revoke`. `plugin/commands/`.

**Rejected.**
- *One combined list.* It cannot express knowing someone without admitting them to
  every room.

**Revisit when** a user needs to admit a peer to a room without knowing it on this
machine.
