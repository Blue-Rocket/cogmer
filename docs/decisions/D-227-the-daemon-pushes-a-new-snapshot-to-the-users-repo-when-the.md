# D-227 — The daemon pushes a new snapshot to the user's repo when the user's list changes, once the user has opted in

**Date:** 2026-10-19 · **Status:** not built · **Areas:** sync, daemon

**Decision.** Once the user has opted in, the daemon makes a new snapshot whenever the user's verified colleagues or rooms change, commits it to the user's snapshot repo and pushes it with the user's own git setup, so that a cloud session started with every machine off has a current list (D-226, a cloud session takes the highest valid snapshot of its user's colleagues and rooms from any source). Until the user opts in, the daemon runs no git command.

**Rejected.**
- *The user pushing each snapshot by hand.* The list changes while the user is busy with something else, and a snapshot one verification behind leaves a colleague untrusted in the next cloud session.
- *A credential of cogmer's own for the repo, such as a deploy key.* It is a second secret to protect, and the user's own git setup is enough.

**Limits.** The push runs unattended with the user's credentials. A push that fails leaves the repo holding the last good snapshot, and the view says the snapshot is older than the list.

**Revisit when** pushes are found to fail often enough that users stop trusting the snapshot's age.
