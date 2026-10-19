# D-219 — A user is a master key that certifies machines through a shared key

**Date:** 2026-10-18 · **Status:** not built · **Areas:** identity, pairing, admission

**Decision.** A user is a master key derived from one recovery phrase that no machine stores, so that a colleague verifies the user once and any machine the user adds later speaks for the user. The master key certifies a shared machine-signing key, which carries a version number, and the shared key certifies each machine's own key, with the name the user gave the machine and a serial number. A machine signs its events with its own key and its own sequence, and sends the two certificates with them, so a receiver checks three links back to the master key it recorded.

**Rejected.**
- *An identity that belongs to one machine, so that a user with two machines is two peers.* A colleague verifies each machine separately, and a user who has lost every machine cannot be recognized again.
- *Receivers that accept only a machine's own key.* Each machine a user adds would need its own two-word comparison with every colleague, and a connection, an admission proof, a sync request or an event from the new machine would be refused until it had one.
- *Copying one machine's private key to every machine.* Two writers under one key issue the same sequence numbers, which a receiver sets aside as conflicts (D-027, a sequence conflict is quarantined), and nothing can remove one machine.
- *A login from a provider such as Google.* The user becomes a name the provider controls and can take away, and the provider takes part in every machine the user adds.
- *Machines that certify one another, with no master key.* A new machine needs an existing machine to approve it, so a user who has lost every machine cannot add one.
- *Machines that vote on removals, each signing with its own key.* A user with two machines cannot outvote a thief, and a machine cannot be added with none.

**Limits.** Whoever holds the phrase is the user. Theft of the phrase ends the identity, and colleagues verify a new one. A machine the phrase is typed into can copy it. A machine that holds the shared key can add machines until the master certifies a new shared key, and a colleague who has not heard of the new key accepts the old one until they do.

**Revisit when** a phrase is stolen, or users hold enough machines for a vote among them to be workable.
