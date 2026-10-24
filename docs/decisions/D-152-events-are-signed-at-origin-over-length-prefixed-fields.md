# D-152 — Events are signed at origin over length-prefixed fields

**Date:** 2026-09-17 · **Status:** active · **Areas:** identity, sync

**Decision.** Every event is signed by the machine that created it, over a length-prefixed
encoding of its own fields with a leading purpose tag, and every receiver checks it
against that machine's key, whose certificates must end at the master key the
originating peer's identifier names (D-219, a user is a master key that certifies
machines through a shared key).

**Support.**
- A relaying machine can then carry an event and cannot author one. §13 (transitive
  synchronization).
- The purpose tag binds a signature to this purpose and version, so a signature made here
  cannot be replayed as one made over something else. `cmd/cogmer/keys.go`,
  `protocolNamespace`.

**Rejected.**
- *Concatenating the fields directly.* A boundary could move: content ending in one value
  and a session beginning with another could swap without the signature changing.

**Revisit when** an event has to carry a field the signature does not cover.
