# D-144 — A refused peer's request goes to a host who is present

**Date:** 2026-09-16 · **Status:** not built · **Areas:** admission, pairing

**Decision.** A peer that is not a guest is refused, and the host is told that it asked,
with the identifier and derived name it presented. The host may admit it, by its
identifier, which makes it known and a guest in one act. The refused peer is told it was
refused and shown its own identifier. A request is never queued: if the host is absent,
it fails.

**Support.**
- It serves two colleagues whose machines have not met, which is an absence of a recorded
  key, not of trust. §12 (forming a room).
- The name beside a request is a claim made by whoever sent it, and the identifier is the
  fact. §6 (identity).

**Rejected.**
- *Refusing in silence.* Neither side could proceed.
- *Queuing requests for an absent host.* An admission completed without the host's
  attention is a token in another form.
- *Approving by the name shown.* The name is the requester's claim.

**Limits.** It does not decide whether a request may arrive when the host is not
expecting one.

**Revisit when** colleagues are found to need first contact without a pairing string.
