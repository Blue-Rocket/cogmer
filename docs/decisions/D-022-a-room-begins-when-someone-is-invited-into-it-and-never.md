# D-022 — A room begins when someone is invited into it, and never earlier

**Date:** 2026-09-16 · **Status:** active

**Decision.** A room's identifier and name are generated when a user creates the room,
and creating a room is the act of inviting someone into it. The creating session
becomes the first member, and the room's history begins there. Before that, nothing is
captured, injected or shared.

**Support.**
- A room that began when a session started would hand the first colleague invited
  everything the user had done alone, by an act that looks like saying hello, and the
  disclosure could not be withdrawn. §12a (room membership).
- Throughout the specification, a room's beginning is the room's, never a member
  session's. §12a.

**Rejected.**
- *Creating a room when a session starts.* It is the disclosure above.
- *Creating a room lazily, at the first captured event.* The same outcome, reached less
  visibly.
- *Offering to include earlier turns when inviting.* It asks a user to decide what to
  share under time pressure, which is a poor moment to ask.

**Limits.** A new room does not hold the conversation that made a user want to invite
someone, and earlier turns are never contributed by default.

**Revisit when** contributing chosen earlier turns to a room is designed.
