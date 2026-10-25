# D-242 — A machine key from before the user identity carries its verification, label and turns to the first user it proves it belongs to

**Date:** 2026-10-25 · **Status:** not built · **Areas:** identity, pairing, admission

**Decision.** A machine that used cogmer before its user had an identity keeps its key and is certified under the user with that same key, so when it presents its chain in a request it signs with that key (D-044, sync requests are signed), a colleague's machine that held the key as a colleague's identifier treats the key as that user's: the verification, the label, every guest-list entry naming the key and every turn the key signed pass to the user. The key itself vouches for the user, which nobody without the key can do, and the key is what the colleague verified, so comparing words again would confirm nothing new and upgrading costs no pairing. The first user a key proves it belongs to is its user, and a second user it proves it belongs to passes nothing over and is reported as a conflict that asks the colleagues to pair (D-052, verifying a peer is its own act run by both users at once), since only a copied or stolen key can be certified under two users.

**Rejected.**
- *Pairing every colleague again after the upgrade.* The words would confirm a key the colleague has verified, and a label held for that key would sound the alarm for a key substituted in transit (D-074, a name means one master key).
- *The latest user a key proves it belongs to.* A thief holding a copy of the key could take over the owner's verification by creating a user of their own after the owner did.

**Revisit when** a user's machine has to change its key while keeping its colleagues' verification.
