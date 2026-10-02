# D-139 — A name is shown as itself only for a verified peer

**Date:** 2026-09-16 · **Status:** active

**Decision.** A peer's name is shown as itself only when its identity has been verified.
Any other peer is shown as unverified, with its identifier, and an unverified speaker is
marked inside the injected text, not only in an interface.

**Support.**
- A derived name can be ground, so it is a mnemonic for a verified identity, never an
  introduction. D-021 (peer names are derived from the identity).
- The model reasons about who said a thing, so a mark only in the view would protect the
  wrong reader. §20 (attribution in injected context).
- An unverified peer's turns are not exchanged at all, so the mark is a backstop. D-054
  (verification gates synchronization and injection).

**Rejected.**
- *Marking an unverified speaker only in the view.* The model, which reasons about the
  speaker, would never see it.

**Revisit when** a turn from an unverified peer is found in injected context without its
mark.
