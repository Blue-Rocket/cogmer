# D-237 — The recovery phrase is eighteen words of the PGP word list

**Date:** 2026-10-24 · **Status:** active · **Areas:** identity

**Decision.** The recovery phrase is 18 words of the PGP word list (D-164, the two words come from the PGP biometric word list, alternating by position): sixteen carry 16 random bytes, 128 bits, and two carry the first two bytes of their SHA-256 as a check, with even and odd positions taken from the list's two halves. The two halves alternate, so a swapped or missing word makes a phrase that does not parse, and the check misses a wrong word that falls in the right half once in 65,536. The master key is derived from the 16 bytes and not from the words (D-219, a user is a master key that certifies machines through a shared key), so the list decides only how the phrase is written down and typed.

**Rejected.**
- *The BIP-39 English list, as 12 words.* A cryptocurrency wallet's recovery phrase has that form, and software that steals wallets searches clipboards, notes and screenshots for it, so a phrase a user saved in a notes file would be found and taken as a wallet, and its thief would be the user to every colleague.
- *The EFF long word list, as about 11 words.* It is shorter and resembles neither the pairing words nor a wallet phrase, but it is a second list to add, credit and test, for a phrase a user types only when creating the user identity and when recovering.

**Limits.** The two words of a pairing come from the same list, so a phrase word can look like a pairing word to somebody asked for "your words".

**Revisit when** users are found to mistake the phrase for the pairing words, or to give up typing 18 words.
