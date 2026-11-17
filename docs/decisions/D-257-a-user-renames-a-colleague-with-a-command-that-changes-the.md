# D-257 — A user renames a colleague with a command that changes the label and nothing else

**Date:** 2026-10-27 · **Status:** active · **Areas:** commands, identity

**Decision.** `/cogmer:peer-rename <peer> <new name>` changes the name the user gives a colleague, found by the colleague's current name, derived name or key, and changes nothing else: whether the colleague is verified and every room they are in are as they were, since a label names a key and grants nothing (D-093, the label is collected with the pairing string and written only when the words match). The new name goes to the user's other machines as any label does (D-228, a user's colleagues, verifications, labels and rooms belong to the user and are copied between the user's machines). A name that another colleague has is refused, naming that colleague, because two keys under one name is how a substituted key surfaces (D-074, a name means one master key, and a collision is where a new master key surfaces).

**Rejected.**
- *Changing the name by pairing again.* It repeats a ceremony that needs the colleague on a call, to change a word that only the user sees.
- *Pointing the user at `cogmer allow` with the colleague's full key.* It overwrites a known label, but it records a key without verifying it and tells a verified colleague to finish verifying.

**Revisit when** a user is found to need a colleague's name cleared back to the derived name.
