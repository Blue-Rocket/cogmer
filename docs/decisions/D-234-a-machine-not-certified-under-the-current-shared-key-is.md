# D-234 — A machine not certified under the current shared key is suspended until it is certified again

**Date:** 2026-10-24 · **Status:** not built · **Areas:** identity, sync, capture, view

**Decision.** When a user certifies a new shared key, a machine that is not certified under it is suspended: its turns appear nowhere, not in the room view, not in an export and never in injected context, so that a machine the user does not recognize, such as one a thief added with the old key, cannot pollute the conversation (D-219, a user is a master key that certifies machines through a shared key). A receiver does not store its new events, hides the ones it stored before the new key reached it, serves none of them to other machines, and refuses its connections, sync requests and admission proofs, as it does an unverified author's (D-054, verification gates synchronization and injection, not just a marker). A suspended machine that was the user's is added again from one of the user's current machines (D-229, a machine is added from another of the user's machines, with the two-word check), and its certificate under the current key restores its turns, so a suspension is lifted where a removal is final (D-220, a removal names the last event of a machine that counts, room by room). Every receiver holding the same certificates computes the same suspension, with no clock.

**Rejected.**
- *Keeping a suspended machine's stored events and refusing only its new ones.* Without a clock a receiver cannot tell an old event from a new one, so receivers would disagree, and a thief's events stored before the new key arrived would keep counting.
- *Treating a suspended machine as removed with a cutoff of 0.* A cutoff never moves later, so the user's own turns from a machine of theirs that the certifying machine had not heard of would be marked for good.
- *Showing a suspended machine's turns marked, as a removed machine's later turns are.* A machine the user never recognized has no turns of the user's to show, and its marked turns would still fill the room view and the export.
- *Letting a machine under the old key work until something removes it.* A machine a thief added would speak as the user after the user certified a new key, which is what certifying one is for.

**Limits.** An assistant that read a suspended machine's turn before the new key reached its receiver cannot unread it.

**Revisit when** a user's own machine is found suspended often enough that adding it again is a burden.
