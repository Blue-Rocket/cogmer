# D-251 — Removing a machine that someone other than the user could still use always certifies a new shared key

**Date:** 2026-10-26 · **Status:** not built · **Areas:** identity, view

**Decision.** The page that removes a machine asks whether someone other than the user could still use it to act as the user. Someone other than the user could still use a machine when it works or could be made to work, has not been wiped, and is not in the user's hands. An answer of yes, or that the user does not know, removes the machine only together with a new shared key, which the recovery phrase certifies (D-220, a removal names the last event of a machine that counts, room by room). The page offers no plain removal of such a machine, so a user without the phrase leaves the machine as it is. The rule does not apply when the user says the machine was wiped or destroyed, or when the machine acknowledged erasing its keys (D-252, a reachable machine is asked to erase its keys before it is removed), and that removal needs no phrase. It does not apply to a cloud environment either, since a cloud key never holds the shared key (D-219, a user is a master key that certifies machines through a shared key), so removing the cloud key with a cutoff for each session key is the whole remedy (D-250, a removal of a cloud key names a cutoff for each of its session keys in each room).

**Rejected.**
- *Asking whether the machine was lost or stolen.* A lost machine that works and has not been wiped is as dangerous as a stolen one, and a user can wave it through as lost where nobody will use it.
- *A plain removal that leaves the old shared key with the machine.* Whoever holds the machine can use that key to add machines of their own, so the removal would look like protection and give none.
- *A new key that the page only suggests.* A user who is unsure would skip it, and a wrong answer costs more than a new key does.

**Limits.** The page takes the user's word that a machine was wiped, and a machine that acknowledges erasing its keys is trusted only as far as the program running on it.

**Revisit when** users are found unable to remove a machine for want of the phrase.
