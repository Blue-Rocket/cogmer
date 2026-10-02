# D-108 — Pairing is not how an address is updated

**Date:** 2026-09-21 · **Status:** active · **Areas:** pairing, transport

**Decision.** Neither pairing nor any dedicated command updates a peer's stored address. The
peer's next synchronization repairs it.

**Support.**
- Pairing is a security act, and one that silently writes network state is a quiet mutation, and
  it teaches that re-pasting a string is a maintenance chore, where handing over a pairing string
  should be rare, deliberate and attached to a ceremony. `cmd/cogmer/main.go`, `runPair`.
- An address matters only when it is used, and every use repairs itself. A host who cannot reach a
  guest is told so and given a line to send by hand, the guest pastes it and joins, and the
  guest's first synchronization carries their current address back. D-105 (an invitation is
  delivered to a paired guest as an offer over the paired channel).

**Rejected.**
- *Writing the address from a string pasted for a paired peer.* It is a pairing that
  skips its ceremony and writes state.
- *A command that sets an address.* It is a mechanism for a problem that resolves itself, and one
  more thing to explain.

**Revisit when** a peer can move while sharing no room with anybody and still need to be reached,
which would mean something other than an invitation had come to depend on a stored address.
