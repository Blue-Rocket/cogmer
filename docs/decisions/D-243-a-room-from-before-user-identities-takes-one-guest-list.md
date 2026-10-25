# D-243 — A room from before user identities takes one guest list, founded by the users whose machines wrote its turns

**Date:** 2026-10-25 · **Status:** active · **Areas:** rooms, admission, sync

**Decision.** A room made before user identities, which has no founding invitation, is carried into them with one guest list computed as every other room's is (D-235, a room's one guest list, changed by any guest through events in the room, decides whose events count), whose founders are the users whose machines signed its turns under the scheme from before user identities, each such key standing for its user once it proves it belongs to one (D-242, a machine key from before the user identity carries its verification, label and turns). Every machine holding the room holds the same turns, so every machine computes the same founders. A machine that has the room, the first time it has a user identity, invites every user its own earlier list of the room admitted, so that a guest who joined and never spoke is on the list too. A founder lifts no revocation, as a founding self-invitation does not (D-240, a room's identifier is the hash of its creator and a random value), and a room with no turns has no founders and is not carried.

**Rejected.**
- *Creating every such room again.* The room's history and its place in every session that joined it are lost, and every user redoes the work, for a room the turns it holds can found.
- *Keeping each machine's own list for such a room, with an entry for a machine key admitting that key's user.* Two guests could hold different lists of the same room, and a revocation would reach only the machine that made it, for as long as the room is used.
- *Holding such a room suspended until every guest's machine has its user identity.* A machine that cannot read a turn leaves it on offer, so nothing is lost while some machines wait, and a suspension stops the sharing that works meanwhile.

**Revisit when** a room from before user identities is still in use after every user of cogmer has a user identity.
