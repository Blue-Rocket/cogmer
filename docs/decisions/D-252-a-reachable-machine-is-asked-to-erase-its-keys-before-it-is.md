# D-252 — A reachable machine is asked to erase its keys before it is removed

**Date:** 2026-10-26 · **Status:** not built · **Areas:** identity, sync

**Decision.** When the user says a machine is in their hands, the removing machine first asks the machine being removed, over a connection that its certificate still allows, to erase its private keys, including its copy of the shared key, and to acknowledge, and signs the removal only after the acknowledgement. The acknowledgement is what lets a machine count as wiped without the user's word (D-251, removing a machine that someone other than the user could still use always certifies a new shared key), and a machine that does not answer is treated as one that someone other than the user could still use. No connection from a removed machine is ever accepted for this, since a connection from a removed machine is refused (D-101, peer connections are TLS pinned to a key this machine has recorded).

**Rejected.**
- *Telling a machine after its removal.* A removed key is refused at the connection, and a route that it could still reach would be open to a thief.
- *Erasing without an acknowledgement.* The removing machine could not tell an erased machine from one that never received the request.

**Limits.** An acknowledgement comes from the program on the machine, so a machine in hostile hands can give it falsely. The rule is for a machine the user holds.

**Revisit when** a machine is found to keep a key after it acknowledged erasing it.
