# D-135 — A room's name is unique only among one user's rooms

**Date:** 2026-09-16 · **Status:** active · **Areas:** rooms

**Decision.** A user's live rooms have distinct names, and no name is unique beyond one
user. A machine minting a name avoids every name in the list of rooms the user is a guest
of, which the user's machines share (D-228, a user's colleagues, verifications, labels and rooms belong to the user and are copied between the user's machines), and generates another on a collision. Two of the
user's machines that mint the same name while out of contact leave a name that is
reported as ambiguous, with both identifiers (D-161, an ambiguous room name is reported
with both identities and never guessed).

**Support.**
- Rooms are created on machines that do not coordinate, so names cannot be globally
  unique. §12 (forming a room).
- A name is only ever resolved against one user. §12.
- The two lists hold 88 and 87 words, which give 7,656 combinations, and a user is a
  guest of few live rooms at once. `cmd/cogmer/roomname.go`.

**Rejected.**
- *Globally unique names.* They cannot be had without coordination, and nothing needs
  them while a name is resolved against one user.
- *Names unique only among one machine's rooms.* A session on another of the user's
  machines joins a room by name from the user's list, so two rooms with one name on
  different machines would make that name ambiguous wherever the list reaches.

**Revisit when** a name has to be resolved beyond one user.
