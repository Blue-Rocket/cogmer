# D-109 — Two is the target and nothing rules out more

**Date:** 2026-09-21 · **Status:** active (standing constraint)

**Context.** D-032 (the order of work) records in an aside that "pairs remain the
target" and that Phase 6 waits for evidence a third peer is wanted. That is half a
principle, and the recorded half is the one that needs no enforcing. The half that
governs day-to-day work has been applied consistently and never written down: while
bearing down on two, **avoid anything that forecloses more**.

The two are easy to confuse and pull in opposite directions. "Two is the target"
argues for spending nothing on a third peer. "Nothing rules it out" argues for
noticing when a shortcut would make a third peer a rewrite rather than a feature.
Neither is the whole rule, and holding only the first is how a prototype acquires a
ceiling nobody chose.

**Decision.** Build for two people in a room. Spend no effort on a third. But treat
any design that *cannot* extend past two as a defect to be argued for explicitly,
not a saving to be taken quietly.

The distinction is between a **cost** and a **ceiling**. Sync is a pull against a
per-peer watermark and guest lists are per-peer (D-159, an invitation carries the
room and the inviter); both would be O(n) work with more peers and neither breaks — those are
costs, and they are fine. A pairwise ceremony assumed to be the only shape of
admission, or a room identity derived from exactly two identifiers, would be
ceilings. So would describing the tool by a count, which is why the word "several"
does not belong in its first line either.

This is also why several things already built are shaped the way they are.
Transitive relay (§13) and immutable events with per-peer sequences (§7) only pay
for themselves past two peers; both were built anyway, because retrofitting event
identity is not possible once events exist.

**Rejected.**

- *Design for N now.* Every hard problem so far has been specific to the pair in
  front of us, and a generalisation written before the second case is a guess. It
  also costs the thing the prototype is for: evidence about whether two people
  actually use it.
- *Optimise for exactly two and revisit later.* This is the position that sounds
  identical to the decision and is not. It permits the ceiling, and the moment an
  event format or an identity scheme has one, D-058 (signature schemes are added,
  never edited) and §7's immutability make it permanent.
- *Leave it in D-032's aside.* An aside inside a decision about phase ordering is
  not where somebody looks before choosing a data structure, which is the moment
  this rule applies.

**Revisit when** evidence arrives that a third peer is wanted — the trigger D-032
already names. Note that the trigger releases the first half of this rule and not
the second: the second half has no expiry.
