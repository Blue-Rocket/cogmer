# D-065 — The daemon reads a range of wire versions, so upgrading is not a flag day

**Date:** 2026-09-18 · **Status:** active · **Areas:** sync

**Decision.** A build declares the newest wire version it speaks, `wireVersion`, and the
oldest it reads, `minWireVersion`, and accepts any version between. A peer outside the
range is reported with which side is older and what to do about it, and a peer that sends
no version is read as speaking version 1.

**Support.**
- A change that needs both users to upgrade at the same moment silences the room for
  whoever upgrades last, and the failure names a version and not what to do.
  `cmd/cogmer/protocol.go`, `minWireVersion`.
- Accepting events of an unknown shape silently is how a field comes to mean two things,
  so a peer outside the range is reported and its events are not read. `cmd/cogmer/sync.go`.
- A peer that predates the field spoke version 1. `cmd/cogmer/protocol.go`, `speaks`.
- A version on the wire is what lets the far end refuse a field it does not understand,
  and the wire format is separate from the stored row so that a column added for local
  bookkeeping cannot become protocol. D-155 (the wire format is defined separately from
  the stored row).

**Rejected.**
- *Requiring both sides to speak the same version.* Every protocol change becomes a flag
  day.
- *No version on the wire.* Nothing is then refused, and an event of an unknown shape is
  accepted as though understood.

**Limits.** The floor rises only when an older version cannot be understood, which is a
statement about the events on the wire and not about tidiness. Raising it to delete a
branch silences the room for whoever upgrades last.

**Revisit when** a change cannot be expressed so that a version 1 reader can skip it. The
floor rises then, the range narrows, and whatever announces a release has to say so.
