# D-097 — A pairing string carries a chosen name and never a guessed one

**Date:** 2026-09-20 · **Status:** active · **Areas:** pairing, identity

**Decision.** A pairing string may end in `#<name>`, as in `ed25519:KEY@ENDPOINT#Alice`,
carrying the name its sender chose. A guessed name is never carried. The receiver uses it as
the default label after what they typed and before asking, says so when it does, and reduces
it with `sanitizeName` before use.

**Support.**
- `#` matches the invitation format's use of the same character for a trailing identifier. A
  string without a name is ordinary, and the endpoint is untouched, `tc://` included, since
  the name is stripped before the address is split. `cmd/cogmer/main.go`, `parsePairing`, and
  `cmd/cogmer/sas_test.go`, `TestPairingStringsWithoutANameStillParse`.
- A username presented as a name is a claim about a person that nobody made, where the derived
  name is plain about being made by a machine, so `chosenName` returns nothing while
  `NameChosen` is false. D-095 (whether a name was chosen is recorded, and never inferred),
  `cmd/cogmer/sas_test.go`, `TestAPairingStringCarriesAChosenNameOnly`.
- The name is a claim by whoever sent the string, and a substituted string carries a
  substituted name, which is safe only because the label is written after the two words match,
  so that write must not move earlier. D-093 (the label is collected with the pairing string
  and written only when the words match).
- The name arrives from another machine and becomes a unique key and an argument to
  `resolvePeer`, so `sanitizeName` strips what would break the format, span lines or be
  invisible, collapses whitespace and caps the length, and whatever it does the endpoint
  survives. `cmd/cogmer/main.go`, `sanitizeName`, `cmd/cogmer/sas_test.go`,
  `TestANameFromAnotherMachineIsReduced`.
- Adopting somebody's description of themselves as the user's own label is a small act worth
  stating and not performing silently. `cmd/cogmer/main.go`, `runPair`.

**Revisit when** the string carries a third thing, since `#` would then be a separator with
two jobs and want a real encoding.
