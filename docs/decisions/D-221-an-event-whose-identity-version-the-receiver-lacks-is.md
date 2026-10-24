# D-221 — An event whose identity version the receiver lacks is withheld from the assistant until fetched

**Date:** 2026-10-18 · **Status:** not built · **Areas:** identity, sync, capture

**Decision.** Each event carries, in its signed metadata, the version number of the user's shared machine-signing key that its sender holds (D-219, a user is a master key that certifies machines through a shared key) and the removals the sender has seen. A receiver that lacks that version or those removals fetches the version and the removals from any machine, because the master key or the shared key signed them and the receiver can check the signatures. Until the receiver has them, it shows the event in the room view and withholds it from the assistant, under the tag and notice of a turn not yet given to the assistant. An assistant cannot unread a forged turn, so the hold is on delivery to the assistant.

**Rejected.**
- *Delivering the event with a marker that the sender's identity is unconfirmed.* A marker is not a control, and the turn is in the assistant's context by then.
- *Holding for a limited time and then delivering.* The hold applies when the records cannot be fetched, which is when delivery is least safe.
- *Asking only the sender for the records.* A sender that is a thief withholds them, and a sender that is offline cannot supply them.

**Limits.** A matching version shows only what the sender states. A thief using a stolen machine states the old version, and a colleague who has not heard of the new one accepts it, so the view never presents a match as current. A held event has no time limit.

**Revisit when** events are found held for longer than the users involved can tolerate.
