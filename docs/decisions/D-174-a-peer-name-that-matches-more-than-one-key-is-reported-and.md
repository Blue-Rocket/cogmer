# D-174 — A peer name that matches more than one key is reported and never chosen

**Date:** 2026-09-19 · **Status:** active · **Areas:** identity

**Decision.** `resolvePeer` refuses a name that matches more than one key, lists each key,
and says what that probably means, and it never picks one. When no key it matches carries
that name as a label the user gave, every match is a name derived from its key, which
nobody chose, so the message says the keys share it by chance and that giving one a name
tells them apart, and it says nothing of a substituted key (D-074, a name means one master
key, and a collision is where a new master key surfaces).

**Support.**
- A store written before names were limited to one key may hold two under one name, and
  choosing between them would admit a peer nobody named. `cmd/cogmer/main.go`, `resolvePeer`.
- Two keys under one name is what a substituted key looks like once recorded, so the
  message tells the user to ask the person which key is theirs over a channel they can
  recognize them on. `cmd/cogmer/main.go`, `resolvePeer`.

**Rejected.**
- *Taking the first match.* It chooses on the user's behalf without saying so, as D-161
  (an ambiguous room name is reported with both identities and never guessed) refuses to
  for rooms.
