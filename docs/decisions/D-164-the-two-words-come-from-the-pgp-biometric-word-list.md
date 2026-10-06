# D-164 — The two words come from the PGP biometric word list, alternating by position

**Date:** 2026-09-17 · **Status:** active · **Areas:** pairing

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
- Both lists equal a copy of the published list, which matches Decred's copy word for word, so a word typed by
  hand or swapped for another fails a test. `cmd/cogmer/testdata/pgp-word-list.tsv`, `cmd/cogmer/sas_test.go`,
  `TestTheWordListsAreThePGPList`.
- No word of either list is a word of a name vocabulary. Each vocabulary can be edited without knowing the
  others, so one test reads all four. `cmd/cogmer/sas_test.go`, `TestNoPairingWordIsAlsoAWordOfANameVocabulary`.

**Rejected.**
- *Words from the peer-name or room-name vocabularies.* A person could compare the name
  beside the words in place of the words.

**Limits.** The copy is one published list, and no standard says which copy is canonical: another copy found
online differs from the other three in many places. Builds up to 0.7.4 carry four even words that are not the
list's, at bytes AD, AE, D0 and D1, so about 1.6% of pairings between one of those and a later build show
different first words, which looks like an interception. They also carry three peer-name words and four
room-name words that overlap the lists, changed in place so that no vocabulary changes length. A peer whose
derived name falls on one of the three peer-name entries, about 3% of peers, is named differently by those
builds. Room names are random and are not affected.
