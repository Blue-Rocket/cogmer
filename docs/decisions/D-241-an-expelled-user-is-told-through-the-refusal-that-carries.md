# D-241 — An expelled user is told, through the refusal that carries the revocation

**Date:** 2026-10-24 · **Status:** not built · **Areas:** admission, sync, view, capture

**Decision.** A guest's machine that refuses a sync request because the requesting user has been revoked from the room sends, with the refusal, the revocation events that expelled them, and nothing else of the room, so that each of the expelled user's machines learns why the next time it asks, including one that was offline when the revocation was made (D-235, a room's one guest list, changed by any guest through events in the room, decides whose events count). The expelled user's machine checks the revocations' signatures, stores them, and says in the room's page, in the room status and at the next prompt of each of its sessions in the room that the user was removed from the room, by whom, and from which of their turns on, and the room up to the cutoff remains readable there (D-236, a revocation names the last turn of each of the revoked user's machines that counts). A machine that holds no revocation and whose requests a room's guests refuse says in the room status that the room's guests are refusing this machine, which usually means the user was removed, and guesses no further.

**Rejected.**
- *Telling the expelled user nothing.* Their room goes quiet and their colleagues look unreachable, which is the failure a quiet room is shown to avoid (D-194, a colleague this machine cannot reach is named, with how long, in the room status and the view).
- *Telling them without saying who.* The room's list shows every guest who revoked whom, so withholding it from the one user it concerns hides only what the others see.
- *The revoking machine pushing the revocation to each of the expelled user's machines until each acknowledges.* It needs a push path and resend state for machines the revoking machine may not know or reach, where each of them asks again soon and the refusal can answer.

**Revisit when** an expelled user's machine is found not to learn of its revocation for longer than its polling interval.
