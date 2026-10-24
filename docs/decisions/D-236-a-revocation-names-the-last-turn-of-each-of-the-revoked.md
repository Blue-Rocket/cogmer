# D-236 — A revocation names the last turn of each of the revoked user's machines that counts

**Date:** 2026-10-24 · **Status:** active · **Areas:** admission, sync, capture

**Decision.** A revocation names, for each machine of the revoked user, the last of that machine's turns that counts, by the machine's sequence in the room (D-029, the sequence lives with the machine), as the revoking machine saw it, so that every guest holds the same history of the room. The revoked user's later turns are treated as a non-guest's (D-235, a room's one guest list decides whose events count): not stored, not shown and never given to an assistant. When two revocations of one guest name different turns for a machine, the lower one applies, and a later invitation that lifts the revocation lets the user's turns after the cutoff count again, and ordinary synchronization fetches them from the machines that hold them, since no receiver stored them (D-054, verification gates synchronization and injection, not just a marker). Everything the revoked guest wrote up to the cutoff remains in the room. A removal of a machine sets its cutoff the same way (D-220, a removal names the last event of a machine that counts, room by room).

**Rejected.**
- *No cutoff.* Each guest would refuse the revoked guest's turns from whenever the revocation reached it, so guests would hold different histories.
- *Hiding everything the revoked guest ever wrote.* It rewrites the room and withdraws what the other guests have read, and an assistant cannot unread a turn (§25, security).
- *A cutoff given as a time.* An author states the time of its own events, so a revoked guest could backdate later turns to fall before it.

**Limits.** An assistant that was given a turn after the cutoff, before the revocation reached its machine, cannot unread it.

**Revisit when** guests find that a cutoff taken as the revoking machine saw it leaves out turns the other guests had read.
