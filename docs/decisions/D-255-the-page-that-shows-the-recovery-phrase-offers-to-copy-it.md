# D-255 — The page that shows the recovery phrase offers to copy it and names a password manager as where to keep it

**Date:** 2026-10-26 · **Status:** active · **Areas:** identity, view

**Decision.** The page that shows the recovery phrase (D-230, a user identity is made in the daemon's page the first time the user needs one) has a button that copies the eighteen words, with no list numbers, and says to keep them in a password manager as a secure note. A password manager survives the loss of every machine, the case the phrase exists for, and no assistant reads it. The page tells the user that the words are on the computer's clipboard until the user copies something else.

**Rejected.**
- *A printable recovery sheet, and keeping the phrase on paper.* Paper is exposed to theft and loss, and a user who keeps the phrase in a password manager gets nothing from a sheet.
- *Hiding the button so that the phrase is never on the clipboard.* The user would select the words by hand, which copies the list numbers and leaves the same words on the clipboard.

**Limits.** Any program on the computer can read the clipboard while the words are on it, and the page cannot clear it. The phrase is as safe as the password manager the user keeps it in.

**Revisit when** a user is found to have lost the phrase from a password manager.
