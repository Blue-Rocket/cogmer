# D-174 — A peer name that matches more than one key is reported and never chosen

**Date:** 2026-09-19 · **Status:** active · **Areas:** identity

**Decision.** `resolvePeer` refuses a name that matches more than one key, lists each key,
and says what that probably means, and it never picks one. When no key it matches carries
that name as a label the user gave, every match is a name derived from its key, which
nobody chose, so the message says which keys are verified, that two keys share a derived
name by chance about once in 8,280 pairs and that a key can also be made until it does
(D-021, peer names are word pairs derived from the identity, never chosen), and that
giving one a name tells them apart. A derived name reaches a colleague with a label only
when no colleague without one is shown by it, so after one of the keys is named the
derived name reaches the other.

**Support.**
- A store written before names were limited to one key may hold two under one name, and
  choosing between them would admit a peer nobody named. `cmd/cogmer/main.go`, `resolvePeer`.
- Two keys under one name the user chose is what a substituted key looks like once recorded, so the
  message tells the user to ask the person which key is theirs over a channel they can
  recognize them on. `cmd/cogmer/main.go`, `resolvePeer`.

**Rejected.**
- *Taking the first match.* It chooses on the user's behalf without saying so, as D-161
  (an ambiguous room name is reported with both identities and never guessed) refuses to
  for rooms.
