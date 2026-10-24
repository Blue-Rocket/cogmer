# D-233 — Every turn carries the name of the machine it came from, as a field the user asserts

**Date:** 2026-10-24 · **Status:** active · **Areas:** capture, view, identity, trust

**Decision.** Every turn carries the name of the machine it came from, the name the user gave the machine when its key was certified (D-219, a user is a master key that certifies machines through a shared key), so that a reader can tell which of a user's machines a turn came from. The name is a field of its own, in the view and in the injected block, and is marked there as asserted by the turn's user. It is never joined to the user's name, the label or the derived name. The user chooses the name, so the field is another field a peer can influence, which is the condition on which D-090 (attribution in the injected block is structured, and nothing of ours is joined to a peer's text) is revisited, and the field meets it in the way D-090 requires: a separate field, with nothing cogmer states concatenated to it.

**Rejected.**
- *Showing a machine's name only on turns from a cloud key.* Turns from two of a user's ordinary machines in one room would look alike, and a reader could not tell which session wrote which.
- *Never showing a machine's name.* A colleague could not tell which of the user's sessions a turn came from, and the user could not tell their own turns from different machines apart.
- *Joining the machine's name to the user's name, as in "Alice (laptop)".* The user chooses the machine's name, so it could imitate the derived name or the verified state beside it.

**Revisit when** a machine's name is found to mislead a reader about whose turn it is.
