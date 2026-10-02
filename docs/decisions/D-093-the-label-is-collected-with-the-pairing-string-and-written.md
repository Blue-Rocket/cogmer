# D-093 — The label is collected with the pairing string and written only when the words match

**Date:** 2026-09-20 · **Status:** active · **Areas:** pairing, identity

**Decision.** The name a user gives a colleague is taken with the pairing string and held in
the pending pairing, and it is written only when the two words match. `NameFree` checks for a
clash before the ceremony, and both the page and the terminal run through one pairing record.

**Support.**
- Asking is not asserting. The label claims that this key is Alice, which is unjustified
  until the comparison succeeds, and that is true of writing it and not of asking for it.
  `cmd/cogmer/verify.go`.
- A second completion point forces a bad choice between defaulting the label to the derived
  name, which is the forgettable thing being escaped, and withholding the verification until
  the label exists, which holds the security-relevant act hostage to a convenience field.
  `cmd/cogmer/verify.go`.
- A clash discovered after the words matched would leave a verified peer and an unusable
  name, so the check comes first. `cmd/cogmer/membership.go`, `NameFree`, and
  `cmd/cogmer/pairview_test.go`, `TestATakenNameIsRefusedBeforeTheCeremony`.
- One pairing record serves both surfaces, so what a match, a mismatch and an abandonment
  mean is decided once. `cmd/cogmer/pairview.go`.
- The endpoint is recorded with the row, since `SetPeerEndpoint` updates a row that has to
  exist and returns nil when it does not, which would lose the only address anybody had.
  `cmd/cogmer/membership.go`, `SetPeerEndpoint`.
- A user learns that a name is required from the command, which names both arguments, from
  `/peer-pair` with nothing, which explains both halves of pairing, and from a string given
  without a name, which asks for one. `plugin/commands/peer-pair.md`.

**Rejected.**
- *An optional label that defaults to the derived name.* A user who never thought about a name
  got the label that stops meaning anything, so the default was the failure.
- *Asking for the name after the words matched.* It adds a second completion point.

**Limits.** What each ending of a pairing leaves behind is D-184 (a pairing's three endings
leave different state).

**Revisit when** another flow gains a second interaction. Its endings should be enumerated and
named for it, and not left to whatever the code does.
