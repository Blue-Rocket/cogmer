# D-140 — Peer names come from 8,280 combinations

**Date:** 2026-09-16 · **Status:** active · **Areas:** identity

**Decision.** Peer names are drawn from 92 adjectives and 90 animals, which give 8,280
combinations, rather than from two lists of 256 words each.

**Support.**
- With 8,280 names, the chance that two of 10 peers share one is about 0.5%, of 25 about
  3.6%, and of 100 about 45%, from the sizes of the lists in `cmd/cogmer/peername.go`.
- Pairs are the case that exists. D-109 (two is the target, and nothing rules out more).
- The lists name people, so each word is curated against being unkind to a person.
  §6 (naming people).

**Rejected.**
- *Two lists of 256 words each.* They would give 65,536 combinations and push collisions
  out by an order of magnitude, at the cost of curating 512 words against that
  constraint.

**Revisit when** rooms regularly hold enough peers that names collide.
