# D-109 — Build for two people, and treat a design that cannot extend past two as a defect

**Date:** 2026-09-21 · **Status:** active · **Areas:** project

**Decision.** cogmer is built for two people in a room and spends nothing on a third. Any design
that cannot extend past two is a defect to be argued for explicitly, and not a saving to be taken
quietly.

**Support.**
- A cost differs from a ceiling. Synchronization is a pull against a per-peer watermark and guest
  lists belong to each peer, so both would do work that grows with the number of peers and neither
  breaks, and those are costs. A pairwise ceremony assumed to be the only shape of admission, or a
  room identity derived from two identifiers, would be ceilings, and so is describing the
  tool by a number of participants. D-159 (an invitation carries the room and the inviter, and
  joining admits the inviter).
- Transitive relay and immutable events with per-peer sequences pay for themselves only past two
  peers, and they were built for two because event identity cannot be retrofitted once events
  exist. §13 (transitive synchronization), §7 (event model).

**Rejected.**
- *Designing for any number of peers now.* Every hard problem so far has belonged to the pair in
  front of us, a generalization written before the second case is a guess, and it costs the
  evidence the prototype exists to gather, whether two people use it.
- *Optimizing for two and revisiting later.* It sounds identical and permits the ceiling,
  and once an event format or an identity scheme has one, immutability makes it permanent. D-058
  (an event verifies under the scheme it was signed with).

**Revisit when** evidence arrives that a third peer is wanted. That releases the first half of the
rule, to build only for two, and not the second, which has no expiry.
