# D-094 — The name you chose leads in the view, with the derived name beside it

**Date:** 2026-09-20 · **Status:** active · **Areas:** view, identity

**Decision.** The view leads with the label the user gave a colleague at pairing and shows the
derived name beside it. A label equal to the derived name counts as absent.

**Support.**
- The label is the one name that is both memorable and bound to a key, since the display name
  is the peer's own claim and the derived name is computed by nobody who chose it. D-074 (a
  name means one key, and a collision is where a key change surfaces).
- The derived name has momentary jobs, which are to disambiguate a collision and to anchor an
  alarm, and neither asks anybody to remember it, so a mnemonic nobody chose is not much of
  one. D-021 (peer names are word pairs derived from the identity, never chosen).
- It is shown beside the label because a key change surfaces on it and it separates two peers
  that claim one display name. `cmd/cogmer/ui.go`.
- A placeholder offered as a choice would be a lie, so a label equal to the derived name is
  treated as absent. `cmd/cogmer/ui.go`.

**Rejected.**
- *Leading with the peer's own display name.* It is a claim and not bound to a key.
