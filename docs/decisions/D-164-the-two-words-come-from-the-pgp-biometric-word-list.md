# D-164 — The two words come from the PGP biometric word list, alternating by position

**Date:** 2026-09-17 · **Status:** active

**Decision.** The first word comes from one list of 256 words and the second from
another, both from the PGP biometric word list, and neither list is the peer-name or the
room-name vocabulary.

**Support.**
- The list is phonetically distinct over a bad telephone line, which is what it exists
  for. `cmd/cogmer/sas.go`.
- Alternation makes a transposition detectable, since said in the wrong order the pair is
  not a valid rendering of anything. `cmd/cogmer/sas_test.go`,
  `TestTheTwoWordsComeFromDifferentLists`.
- Each word carries one byte, so each list must hold 256 distinct words, which the
  security argument assumes. `cmd/cogmer/sas_test.go`, `TestWordlistsAreWholeBytes`.
- A SAS sits on screen beside a derived peer name, and one mnemonic that could be
  mistaken for the other is how a person compares the wrong thing. `cmd/cogmer/sas.go`.

**Rejected.**
- *Words from the peer-name or room-name vocabularies.* A person could compare the name
  beside the words in place of the words.

**Limits.** Nothing checks that the two lists share no word with the peer-name or
room-name vocabularies.
