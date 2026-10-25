# D-250 — A removal of a cloud key names a cutoff for each of its session keys in each room

**Date:** 2026-10-25 · **Status:** not built · **Areas:** identity, sync, capture

**Decision.** A removal of a cloud key, under a removal scheme of its own beside the scheme for a machine (D-058, an event is verified under the scheme it was signed with), names, for each room and each session key certified under that cloud key, the last of that session key's turns that counts, so that retiring a cloud environment that was not stolen marks none of the turns its sessions wrote (D-220, a removal names the last event of a machine that counts, room by room). The removing machine proposes, for each session key, the last of its turns that machine has seen, and a session key the removal does not name counts none of its turns, as a machine's removal after a theft counts none. The page asks first whether the environment was lost or stolen, as it does for a machine, and proposes no turns when it was. A cutoff only moves earlier, so the lowest any removal with authority names for a session key in a room is the one that applies.

**Rejected.**
- *Every removal of a cloud key counting none of its sessions' turns.* Retiring an environment the user has stopped using would withdraw from every colleague's assistant everything its sessions wrote, which is the common case, while a theft is the rare one.
- *A cutoff given as a time, such as the session certificate's expiry.* The cloud key signs its sessions' certificates and their expiry, so a thief holding it could certify a session that appears to have ended before the removal.

**Limits.** A session key whose turns the removing machine never saw, in a room that machine is not in, is named by no cutoff, so its turns are marked.

**Revisit when** removing a cloud environment is found to mark turns the user expected to keep.
