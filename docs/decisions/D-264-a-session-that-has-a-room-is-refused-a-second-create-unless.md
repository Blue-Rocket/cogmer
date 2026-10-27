# D-264 — A session that has a room is refused a second create unless the command carries --another

**Date:** 2026-10-27 · **Status:** active · **Areas:** commands, rooms

**Decision.** `create` run in a session that has been put in a room, whether it is still in it or has left it, makes no room and says which room the session has. `create --another` makes one anyway. The flag is the user's to give: the command files tell the model to ask the user and to add the flag only after the user says so, as `peer-pair` asks for a name. The flag travels as an argument, so it needs nothing of its own in the Cowork route (D-207, a Cowork command is run by the daemon from the text of the prompt) or in the skill table. A room made with `--another` cannot take this session, since a session that has left its room is refused any other, and a session is never moved to a second room (D-072, a refused join is explained and not only refused), so the user enters it from another session. A second room is a new room each time, and its creator then holds two and talks in one, which looks the same to everyone else as the first.

**Rejected.**
- *Making the command idempotent by returning the room the session is in.* A user who wants a second room has no way to get it from the command.
- *Asking for confirmation during the command.* A command run by a host has no way to ask, and the model would answer for the user.
- *A longer warning in the command file.* The model reads it after the second room exists.

**Revisit when** a user is found to make a room from a session that cannot enter it, and not to want the refusal.
