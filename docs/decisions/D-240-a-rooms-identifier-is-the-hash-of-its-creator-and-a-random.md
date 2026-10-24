# D-240 — A room's identifier is the hash of its creator and a random value, so only the founding self-invitation admits its creator

**Date:** 2026-10-24 · **Status:** not built · **Areas:** rooms, admission, identity

**Decision.** A room's identifier is the SHA-256 of its creator's identifier and a random value, and the room's first event is the creator's invitation of themselves, which carries that value, so the creator is on the room's guest list by the rule every guest is (D-235, a room's one guest list, changed by any guest through events in the room, decides whose events count). A user's invitation of themselves counts only when hashing that user's identifier with the value it carries gives the room's identifier, which only the creator can do, so nobody admits themselves to a room that exists. A user's invitation of themselves lifts no revocation, so a creator who is expelled is let back in only by another guest's invitation, and a self-invitation works once, when the room is founded. A room made before the user identity existed is not carried into it, and its users create it again.

**Rejected.**
- *A creator always on the list, as the signer of the room's first event.* The creator could never be expelled, which gives one guest a standing no other guest has, where no machine and no user is a room's authority (§3.3, no authoritative peer).
- *A random identifier, with a self-invitation counting when it is the room's first event.* Without a clock no machine can tell which event came first, so an expelled guest or anybody who learned the identifier could sign a self-invitation and claim it.
- *Carrying rooms made before the user identity into it.* Such a room has no founding invitation and so no guest list under the rule every other room follows, and the users who hold one upgrade together and can create it again.

**Revisit when** a room has to be founded by more than one user at once.
