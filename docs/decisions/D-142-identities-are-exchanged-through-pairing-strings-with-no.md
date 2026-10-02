# D-142 — Identities are exchanged through pairing strings, with no directory

**Date:** 2026-09-16 · **Status:** active

**Decision.** A peer learns another's identity from the pairing string that other peer
sends, and there is no directory of peers to consult.

**Support.**
- A pairing string carries an identifier, an address and a name, none of which is a
  secret. §12 (forming a room).
- Pairing happens once with each colleague and outlasts every room. §12.

**Rejected.**
- *A directory of peer identifiers.* Needing one to be set up before two people could
  work together would defeat an invitation, and a directory is a central component.

**Revisit when** users need to find a colleague's identity without the colleague sending
it.
