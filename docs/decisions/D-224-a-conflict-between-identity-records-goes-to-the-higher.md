# D-224 — A conflict between identity records goes to the higher level, then to a removal, then to the lowest hash

**Date:** 2026-10-19 · **Status:** not built · **Areas:** identity, sync

**Decision.** When two signed identity records conflict, the one from the higher level wins, in the order master, shared key, machine (D-219, a user is a master key that certifies machines through a shared key). At one level a removal beats an addition, and then the lowest hash decides. Every receiver holding the same records therefore picks the same one, with no clock and no server.

**Rejected.**
- *A window of time in which a higher level can override a lower one.* It needs every receiver to agree when a record was made, and an author states the time of its own records.
- *The latest record wins.* It needs the same agreed time, and a thief could date a record later than the user's.

**Limits.** Two records signed by the master itself conflict at one level, and when a thief holds the phrase no rule favors the user.

**Revisit when** a conflict is found that the rule decides in a way users reject.
