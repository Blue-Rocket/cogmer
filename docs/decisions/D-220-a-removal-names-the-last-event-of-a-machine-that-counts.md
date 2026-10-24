# D-220 — A removal names the last event of a machine that counts, room by room

**Date:** 2026-10-18 · **Status:** active · **Areas:** identity, sync, capture

**Decision.** A removal of a machine names, for each room, the last event of that machine that counts, and the machine's later events are marked, shown in the room view and never given to an assistant. The cutoff is a sequence number for the room (D-029, the sequence lives with the machine), which the user's machine works out from the order it received the events. A machine can remove itself, and deletes its copy of the shared key when it does, so a retired machine certifies no other, a machine that holds the shared key of the current version can remove another, and the master can remove any machine and certify a new shared key in the same act (D-219, a user is a master key that certifies machines through a shared key). A user certifies a new shared key when a stolen machine held the old one, and a removal signed by an older shared key then counts for nothing, so the machine that certifies the new key shows the user every removal it holds, with the key that signed each, and signs again, under the new key, the removals the user confirms, so that a thief's removals made before the new key do not carry over. A cutoff only moves earlier, so the lowest one anyone with authority signed is the one that applies, and a removal after a theft marks all of the machine's events unless the user names a later point. The page that removes a machine asks first whether it was lost or stolen: if it was not, it proposes, in each room, the last of the machine's turns that the removing machine has seen, so none of the user's own turns are marked; if it was, it proposes that none count; and the user confirms the cutoffs room by room before the removal is signed.

**Rejected.**
- *Voiding everything a removed machine signed.* It discards the user's own events from before the theft, and a transcript that rewrites itself is not a record.
- *Refusing the machine only from the removal onward.* A thief's turns from before the removal would stand as the user's.
- *A cutoff given as a time.* An author states the time of its own events, so a thief can backdate them to fall before it.
- *Counting a removal signed by any shared key the master certified, of any version.* A thief whose stolen machine held the old shared key could sign a removal of every machine of the user with nothing counting, and since a cutoff never moves later, certifying a new shared key would not undo it.
- *A cutoff that can move later.* A thief who obtains the phrase could raise it to make forged events count, and a cutoff that moves both ways needs an order that the lowest-wins rule does without.
- *Asking the user for every room's cutoff with no proposal.* Few users can say which turn was a machine's last, and most removals are of a machine retired, not stolen.

**Limits.** An assistant that read a forged turn before the removal cannot unread it. A cutoff cannot be loosened, so a genuine turn marked by mistake is marked for good.

**Revisit when** marking everything proves too costly to users who lose a machine, or a cutoff has to move later.
