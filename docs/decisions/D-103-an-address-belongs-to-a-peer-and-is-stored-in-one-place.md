# D-103 — An address belongs to a peer, and is stored in one place

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** Asked why a newer address would not update both stores, and then more
pointedly why an address is stored in more than one place at all. Following that
found a modelling error underneath several problems that had been treated as
separate.

**Addresses live in two tables.** `known_peers.endpoint` holds one per peer, written
at pairing. `room_peers(room_id, endpoint)` holds addresses per room, with **no peer
column**, written when a member synchronizes and when an invitation is accepted.

**Both writers hold the identity and discard it.** The synchronization handler calls
`AddRoomPeer(room.RoomID, req.Endpoint)` one line after `verifyRequest` has
cryptographically established `req.PeerID`. The join path calls it two lines from
`Allow(host, "")`, where the host's identifier came out of the invitation. There is
no path by which an address arrives without an identity attached; the identity is
thrown away at the moment of writing.

**Four problems, one cause.** Each of these had been treated as its own defect:

- verification dials every address this machine knows, because it cannot ask which
  one is a particular peer's;
- an unexpected key therefore cannot be an alarm, because dialling the wrong peer is
  the expected case rather than a surprise;
- room addresses accumulate for ever, because `INSERT OR IGNORE` cannot overwrite
  per peer when there is no peer;
- invalidation has nowhere to live, since expiring "that peer's stale address"
  requires knowing whose it is.

**The replacement is a join that already has both halves.** `Guests(roomID)` returns
peer identifiers, and `known_peers.endpoint` holds an address per peer. So *where do
I poll for this room* becomes *the guests of this room, excluding me, and the address
recorded for each* — which yields identity along with every address, so a failure can
be attributed, a peer that moves overwrites one row rather than adding another, and
expiry has something to attach to.

**Checked: no room holds an address for a non-guest.** `verifyRequest` refuses a
request from anybody who is not a guest of that room before `AddRoomPeer` is reached,
and the join path admits the host in the same breath as recording them. The join is
therefore exact rather than approximate.

**One wrinkle, deliberately made explicit.** `Invite` requires only that an
identifier name a key, and `CreateRoom` invites the creator, so **you are a guest of
your own rooms and are never in your own known-peers list**. The join drops you
because no address row exists for you, which is the right outcome reached by
accident. Exclude self by saying so, rather than relying on a missing row that
somebody will later add for an unrelated reason.

**This removes work rather than adding it.** The pruning machinery for accumulated
room addresses becomes unnecessary — one row per peer, replaced when they move, gone
when they are forgotten. So does any reconciliation between the two stores, and so
does the argument for verification's fallback sweep: that argument was that
`room_peers` might hold a fresher address than `known_peers`, and with one store that
divergence cannot occur.

**What the §4 lifecycle assumed.** It was written describing an address that enters
from several sources, ages, and is discarded — all of which is right, and all of
which the schema cannot express while an address has no owner. The prose described a
model the tables could not hold.

**Revisit when:** an address is legitimately held for something that is not a peer.
