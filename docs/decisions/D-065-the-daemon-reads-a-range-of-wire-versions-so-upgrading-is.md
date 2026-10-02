# D-065 — The daemon reads a range of wire versions, so upgrading is not a flag day

**Date:** 2026-09-18 · **Status:** active (implemented) · **reconstructed 2026-09-21**

**This entry was never written**, for the same reason and in the same commit as
D-064. Reconstructed from `protocol.go:19-27` and `sync.go:250-256`, which carry
the reasoning nearly in full.

**Context.** `wireVersion` names the protocol a build speaks. Comparing it for
equality is the obvious implementation and makes every protocol change a flag day:
both people must upgrade at the same moment or the room goes silent, and the
failure names a version rather than saying what to do.

**Decision.** A build declares the newest version it speaks (`wireVersion`) and the
oldest it can still read (`minWireVersion`); `speaks` accepts anything between.
Today that is 2 and 1. A peer outside the range is reported rather than guessed at,
because silently accepting unknown-shaped events is how a field comes to mean two
things — and the error says which side is older and what to do about it.

**Zero means one.** A peer predating the field spoke v1, so an absent version reads
as 1 rather than as unknown.

**Why this stopped being optional at Phase 11.** A flag day is tolerable while one
person builds both sides, because both sides upgrade when he says so. Phase 11
shipped the plugin, which is the point at which the two sides belong to two people
upgrading on their own schedules. The constraint did not change; the number of
people did.

**Why the floor moves rarely.** Raise `minWireVersion` only when an older version
genuinely cannot be understood — a statement about the events on the wire, not
about tidiness. Raising it to delete a branch is how the room goes silent for
whoever upgrades last.

**Rejected** — what the code rules out, not what was considered:

- *Hard equality on the version.* The flag day above.
- *No version on the wire at all.* Then there is nothing to refuse, and an
  unknown-shaped event is accepted as though understood. D-155 (the wire format is
  defined separately from the stored row) keeps the two structs separate precisely so a column added
  for local bookkeeping cannot become protocol by accident; a version is what makes
  that refusable at the far end.

**Revisit when** a change cannot be expressed so that a v1 reader can skip it. The
floor rises then, the range narrows, and whatever announces a release has to say
so.
