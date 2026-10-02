# D-029 — Losing a room database does not end membership; the sequence lives with the identity

**Date:** 2026-09-16 · **Status:** active

**Decision.** A peer that loses a room's database keeps its membership. Its highest issued
sequence for each room is stored outside the room's database, in the membership index,
with the identity, and each sequence number is reserved there before the event that uses
it is published.

**Support.**
- A user's own turns and the colleagues' turns injected into their session are in that
  session's context, under `~/.claude/projects/`, and survive the loss, so the room's
  database holds the record, which other members can supply again. §8 (event identity
  and ordering).
- A session never joins a second room, so a peer whose membership ended could not
  collaborate again from that session at all. D-016 (a session never joins a second
  room).
- Losing identity is the safe failure, since the peer becomes a new peer with a new
  sequence space, and losing a room while keeping the identity is the dangerous one,
  since that peer could reissue numbers others hold. §8.
- Reserving before publishing means a failure between the two leaves a number unused,
  which is harmless. `cmd/cogmer/daemon.go`, `appendLocal`, which calls
  `ReserveSequence` before `Append`.

**Rejected.**
- *Ending the peer's membership in the room.* It would cost the user the session and its
  working context, in exchange for a record other members hold.
- *A sequence epoch, an incarnation number raised on recovery.* It needs a field on every
  event and a synchronization exchange keyed by peer and epoch, while keeping the counter
  avoids the restart and changes no event or protocol.
- *Keeping the counter in the room's database with a backup elsewhere.* Two copies can
  disagree, and the disagreement is the failure.

**Limits.** Events return only from peers that still hold them, so a member recovering
alone has a correct sequence and an empty history until others reconnect. Delivery state
is lost with the database, so some colleagues' turns are injected twice.

**Revisit when** a peer is found to have reissued a sequence number after recovering.
