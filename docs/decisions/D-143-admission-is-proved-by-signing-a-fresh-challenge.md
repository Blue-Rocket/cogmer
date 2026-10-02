# D-143 — Admission is proved by signing a fresh challenge

**Date:** 2026-09-16 · **Status:** active · **Areas:** admission, identity, trust

**Decision.** A peer that joins presents its identifier, the host finds it among the
room's guests and issues a fresh, unpredictable challenge, and the peer signs it with
the matching private key.

**Support.**
- Nothing secret passes in either direction, so the exchange needs integrity, which the
  signature supplies, and no confidentiality. §12 (forming a room).
- Proving possession of a key establishes who is asking, and never whether they may,
  which is the guest list's question. D-044 (sync requests are signed).

**Rejected.**
- *A proof that can be replayed.* An exchange that can be replayed is a bearer
  credential.

**Revisit when** admission has to work without the host able to answer a challenge.
