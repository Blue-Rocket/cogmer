# D-239 — An event a machine signs for its user names the user, the signing key, the shared-key version and the removals seen, all as signed fields

**Date:** 2026-10-24 · **Status:** not built · **Areas:** identity, sync

**Decision.** An event under signing scheme v4 names the user as its peer, and carries as fields of their own, each covered by the signature, the key of the machine or session that signed it, the version of the user's shared key that machine holds and the hashes of the removals it has seen (D-221, an event whose identity version the receiver lacks is withheld from the assistant until fetched). It carries the certificates that tie the signing key to the user beside the signed fields, as D-219 (a user is a master key that certifies machines through a shared key) has every event do, so that an event can be checked wherever it arrives, relayed or fetched again after a lost database (D-029, losing a room database does not end membership). The event's sequence is the signing machine's own in the room, so an event is identified by its signing key, room and sequence (D-027, a sequence conflict is quarantined, not dropped). An event signed under v3 verifies as it does now (D-058, an event is verified under the scheme it was signed with).

**Rejected.**
- *Carrying the shared-key version and the removals seen in the event's metadata.* The metadata is signed, but a check that decides whether an event reaches an assistant would then rest on parsing free-form JSON, where a signed field of its own is read the same way by every receiver.
- *Sending only the signing key and the version with each event, and fetching the certificates as identity records.* Events would be smaller, but an event could not be checked until its certificates arrived, and a store could keep each certificate once whichever way they travel.

**Revisit when** the certificates carried with events are found to be a large share of a room's traffic.
