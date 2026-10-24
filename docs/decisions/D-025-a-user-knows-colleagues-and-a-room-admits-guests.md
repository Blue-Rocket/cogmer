# D-025 — A user knows colleagues, and a room admits guests

**Date:** 2026-09-16 · **Status:** active · **Areas:** admission, pairing

**Decision.** There are two lists at two scopes, and both belong to the user (D-228, a
user's colleagues, verifications, labels and rooms belong to the user and are copied between
the user's machines). Each of the user's machines holds the known peers, and a room's guests
are held by the user's machines that have joined the room. A user's known peers are
the identifiers the user has learned and the names the user knows them by, durable and
outlasting every room. A room's guests are the known peers who may enter it, created with
the room and archived with it. Revoking withdraws a peer's admission to one room, and
forgetting a peer discards the identity and every admission it held, for every machine of
that colleague and on every machine of the user (D-073, forgetting a peer discards their
admissions with them). Forgetting is never how one of a colleague's machines stops
counting: only that colleague removes it (D-220, a removal names the last event of a
machine that counts).

**Support.**
- A user may know six colleagues and admit two of them to a room about customer data.
  §12 (forming a room).
- Both lists can be inspected and changed, with `/cogmer:peer-list`,
  `/cogmer:peer-forget`, `/cogmer:room-status`, `/cogmer:room-invite` and
  `/cogmer:room-revoke`. `plugin/commands/`.

**Rejected.**
- *One combined list.* It cannot express knowing someone without admitting them to
  every room.

**Revisit when** a user needs to admit a peer to a room without first knowing them.
