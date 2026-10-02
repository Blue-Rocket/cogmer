# D-021 — Peer names are word pairs derived from the identity, never chosen

**Date:** 2026-09-16 · **Status:** active · **Areas:** identity

**Decision.** A peer's name, an adjective and an animal such as quiet-otter, is derived
by `PeerName()` from a hash of the peer's identifier when the identity is loaded. It is
never chosen and never stored, and attribution rests on it rather than on the display
name a peer asserts.

**Support.**
- The injected block tells a colleague's Claude who said each turn, so a name a peer
  could choose would put words in a colleague's mouth in a form the model reads as the
  colleague saying them. §20 (attribution in injected context).
- A hash covers the whole identifier, so a difference anywhere in it changes the name.
  `cmd/cogmer/peername.go`, `PeerName`.
- The word lists name people, so they hold nothing unkind applied to a person, and a
  test checks them. §6 (naming people), and `cmd/cogmer/peername_test.go`.

**Rejected.**
- *Display names a peer chooses.* They are the impersonation above.
- *Storing the name with the identity.* A stored name can drift from the identity it
  represents.
- *Indexing the word lists by the first and last bytes of the identifier, so that a
  person could check the name against the bytes.* Two identifiers differing only in a
  middle byte would derive the same name, because slicing sees sixteen bits and nothing
  between the ends.

**Limits.** A derived name can be ground: an identity can be generated again and again
until its name matches a target, so a name is a mnemonic for a verified identity,
never an introduction to a stranger (D-139).

**Revisit when** peer names are needed at a scale where collisions become common, since
the lists hold 8,280 combinations (D-140).
