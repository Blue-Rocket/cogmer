# D-229 — A machine is added from another of the user's machines, with the two-word check

**Date:** 2026-10-24 · **Status:** not built · **Areas:** identity, pairing, view

**Decision.** A user adds a machine from one of their machines that has the user identity: the new machine's page shows its pairing string, the user enters it on the existing machine, both pages show two words for the user to compare, and when the user confirms, the existing machine certifies the new one (D-219, a user is a master key that certifies machines through a shared key) and sends it the shared key, the user's machine list with addresses and the user's lists (D-228, a user's colleagues, verifications, labels and rooms belong to the user and are copied between the user's machines). The words come from the two machine keys under a purpose tag of their own, so that neither this exchange nor a pairing between colleagues (D-048, verification is a live commit-then-reveal exchange that derives two words) can pass for the other. Typing the recovery phrase into a new machine's page is for recovery only (D-223, the recovery phrase is typed only into the daemon's browser page). Each exchange of machine lists also repairs the addresses it carries, since a user's machines may share no room through which to learn one another's addresses.

**Rejected.**
- *Typing the recovery phrase into each machine the user adds.* The phrase would be typed more often and into more machines, and any machine it is typed into can copy it.
- *Certifying the machine whose string was entered, with no comparison.* A string substituted on its way between the two machines would have a stranger's machine certified as the user's.
- *Deriving the words as a pairing between colleagues derives them.* One exchange could then be presented as the other, so a user who believes they are pairing with a colleague could certify a stranger's machine.

**Revisit when** users are found to recover with the phrase on a new machine while one of their machines could have added it.
