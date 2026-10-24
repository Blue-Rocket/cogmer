# D-220 — A removal names the last event of a machine that counts, room by room

**Date:** 2026-10-18 · **Status:** not built · **Areas:** identity, sync, capture

**Decision.** A removal of a machine names, for each room, the last event of that machine that counts, and the machine's later events are marked, shown in the room view and never given to an assistant. The cutoff is a sequence number for the room (D-029, the sequence lives with the machine), which the user's machine works out from the order it received the events. A machine can remove itself, a machine that holds the shared key can remove another, and the master can remove any machine and certify a new shared key in the same act (D-219, a user is a master key that certifies machines through a shared key). A user certifies a new shared key when a stolen machine held the old one. A cutoff only moves earlier, so the lowest one anyone with authority signed is the one that applies, and a removal after a theft marks all of the machine's events unless the user names a later point.

**Rejected.**
- *Voiding everything a removed machine signed.* It discards the user's own events from before the theft, and a transcript that rewrites itself is not a record.
- *Refusing the machine only from the removal onward.* A thief's turns from before the removal would stand as the user's.
- *A cutoff given as a time.* An author states the time of its own events, so a thief can backdate them to fall before it.
- *A cutoff that can move later.* A thief who obtains the phrase could raise it to make forged events count, and a cutoff that moves both ways needs an order that the lowest-wins rule does without.

**Limits.** An assistant that read a forged turn before the removal cannot unread it. A cutoff cannot be loosened, so a genuine turn marked by mistake is marked for good.

**Revisit when** marking everything proves too costly to users who lose a machine, or a cutoff has to move later.
