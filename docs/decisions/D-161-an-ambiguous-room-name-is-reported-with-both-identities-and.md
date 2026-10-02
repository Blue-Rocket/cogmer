# D-161 — An ambiguous room name is reported with both identities and never guessed

**Date:** 2026-09-17 · **Status:** active

**Decision.** Looking up a room by name has three outcomes: one room, no such room, and
more than one. For more than one, `FindRoom` names each room's identifier and asks which
was meant.

**Support.**
- Answering "no such room" for an ambiguous name would send a user looking for a room
  they are in, and returning whichever row came first would choose for them
  without saying so. `cmd/cogmer/membership.go`, `FindRoom`.
- The report is distinguishable from an unknown name and names both identifiers.
  `cmd/cogmer/membership_test.go`, `TestAnAmbiguousNameIsReportedNotGuessed`.
- Names collide by design. D-050 (room names may collide locally).

**Rejected.**
- *Answering that no such room exists.* It misdirects the user.
- *Taking the first match.* It chooses on the user's behalf without telling them.

**Revisit when** a user has to refer to a room somewhere that cannot ask which was meant.
