# D-254 — A recovery phrase is read with the list numbers before its words ignored

**Date:** 2026-10-26 · **Status:** active · **Areas:** identity

**Decision.** A number before a word, such as "7." or "7)", is ignored when a recovery phrase is read (D-237, the recovery phrase is eighteen words of the PGP word list). The page shows the words as a numbered list, so a phrase copied from the page carries the numbers. No word of the list is made of digits, so ignoring a number drops no word.

**Rejected.**
- *Showing the numbers as text that a selection leaves out.* It fixes only the page that shows the phrase, and a phrase the user copied into a note, numbered, would still be refused.

**Revisit when** a number that was ignored is found to have hidden a word the user left out.
