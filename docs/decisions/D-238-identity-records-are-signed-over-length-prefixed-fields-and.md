# D-238 — Identity records are signed over length-prefixed fields and carried as JSON

**Date:** 2026-10-24 · **Status:** not built · **Areas:** identity, sync

**Decision.** A shared-key certificate, a machine certificate, a session certificate and a removal (D-219, a user is a master key that certifies machines through a shared key) are each signed with Ed25519 over a length-prefixed encoding of their fields behind a purpose tag of their own, as an event is (D-152, events are signed at origin over length-prefixed fields), and each carries the number of its signing scheme outside the signature and is verified under that scheme (D-058, an event is verified under the scheme it was signed with). On the wire and in storage a record is JSON, with keys and signatures in base64url. A record's hash, which names it in lists, fetches and acknowledgements, is the SHA-256 of its signing bytes, so a record has one hash whoever holds it.

**Rejected.**
- *A standard binary encoding such as CBOR.* It is more compact, and it is a second encoding beside the one every event is signed and carried in, with its own canonical form to get right before a signature over it means anything.
- *Hashing a record together with its signature.* Ed25519 signatures are deterministic, so the signing bytes name the record alone, and a hash that covers the signature would give two copies of one record two names if an encoding of the signature ever differed.

**Revisit when** identity records are found to make up a share of the traffic between machines that their size matters.
