# D-016 — A session is in one room at most, and never joins a second

**Date:** 2026-09-16 · **Status:** active · **Areas:** rooms

**Decision.** A Claude Code session is a member of one room at most, and a session that
has been in one room never joins a second. It may leave its room and rejoin it.

**Support.**
- Every captured event belongs to one room, so a session in two would give no basis
  for choosing which, and a session receiving turns from two unrelated conversations
  could not keep them apart. §12a (room membership).
- Injected context cannot be withdrawn, so admitting a session to a second room would
  carry the first room's conversation into the second through the model's own output,
  without either room's members knowing. §12a.
- Delivery is recorded per event, so a session that rejoins receives what it missed and
  nothing else. D-014 (delivery from transcript evidence).

**Rejected.**
- *Letting a session into a second room with a warning.* The leak is silent, and falls
  on people who never saw the warning.
- *Letting a session that has received no colleague's turns move to another room.* The
  exception depends on knowing what a context window holds, and starting a new session
  corrects a room joined by mistake more cheaply.

**Limits.** A session's own prompts and responses restrict nothing, because that
content came from its own user.

**Revisit when** a way exists to scope or remove injected context within a live
session.
