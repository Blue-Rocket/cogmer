# D-226 — A cloud session takes the highest valid snapshot of its user's colleagues and rooms from any source

**Date:** 2026-10-19 · **Status:** not built · **Areas:** identity, sync, trust

**Decision.** A cloud session takes the highest valid snapshot of the user's verified colleagues and rooms from any source it can reach, which are a live machine of the user and a private repo added to the cloud environment, so that a session works with every machine off and learns of a removal when one is on. A snapshot is a complete set of the colleagues the user verified and the rooms with their guests, with a version number and a signed time, signed by a key that the user's master key has certified (D-219, a user is a master key that certifies machines through a shared key). The daemon refuses a snapshot whose signature or certificates do not check, refuses one whose version is lower than any it finds in the history of the repo that the session's clone holds, shows the snapshot's age in the view, and starts with no sharing when it finds no valid snapshot.

**Rejected.**
- *Only the repo.* A removal made after the last push reaches no session until the next push.
- *Only a live machine.* A session started while every machine is off would have no list.
- *Trusting the host key in an invitation string that the user pastes, with no snapshot.* Anyone who gets the user to paste a string could put text into the assistant, which is the case verification exists to block (D-054, verification gates synchronization and injection, not just a marker).

**Limits.** A snapshot lists colleagues' keys and addresses, and anything that runs in the session, including the model's shell, can read them. The history check sees only the commits a clone holds, so a rewritten history, or enough later commits, hides a newer version from it.

**Revisit when** a removal is found not to have reached a cloud session that used an older snapshot.
