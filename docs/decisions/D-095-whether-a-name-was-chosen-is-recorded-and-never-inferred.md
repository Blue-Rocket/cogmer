# D-095 — Whether a name was chosen is recorded, and never inferred

**Date:** 2026-09-20 · **Status:** active · **Areas:** identity

**Decision.** `NameChosen` records whether the user's display name was chosen, and is
never inferred by comparing the name with the guess computed from the operating system
user. The user chooses the name in the daemon's page when the user identity is made
(D-230, a user identity is made in the daemon's page the first time the user needs one),
and any deliberate act of naming sets it, including naming it as the guess. A machine's
name is a different name: the one the user gives that machine, which its certificate
carries (D-219, a user is a master key that certifies machines through a shared key).

**Support.**
- The name has one function, which is to be seen by other people, and a user never sees
  themselves labeled, so there is no moment when its owner notices that it is wrong.
  `cmd/cogmer/identity.go`, `NameChosen`.
- Comparing against the guess would pester the person whose username is their name for ever,
  which is an always-on warning. D-089 (the reachability warning asks about the advertised
  address, and travels with the pairing string).
- Renaming changes nothing others rely on. Turns sent earlier keep the name they carried,
  because events are immutable, and the label a colleague gave is theirs, so a rename cannot
  change what anybody else calls the user. §7 (event model), D-094 (the name you chose leads
  in the view, with the derived name beside it).

**Rejected.**
- *Inferring that a name was chosen because it differs from the guess.* A user whose username
  is their name could never be recorded as having chosen it.

**Revisit when** something other than a person needs a display name, or the name travels
somewhere other than a pairing string and an event.
