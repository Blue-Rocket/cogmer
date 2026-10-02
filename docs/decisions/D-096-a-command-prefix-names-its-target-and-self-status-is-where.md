# D-096 — A command prefix names its target, and `/self-status` is where you are

**Date:** 2026-09-20 · **Status:** active · **Areas:** commands, identity

**Decision.** A command prefix names what the command is about: `peer-` for another user,
`room-` for a room and `self-` for the user, so `/self-name` sets the user's own name. The
user's own identity and pairing string are shown by `/self-status`, and no `peer-` command
prints them.

**Support.**
- The existing prefixes name what is acted on, as `peer-list`, `peer-forget`, `room-create`
  and `room-join` do, and a rule written to excuse an exception is one people stop trusting.
  `plugin/commands/`.
- "Status" means the current state of this thing in this set, and a user reads it for the same
  reason in `room-status` and `self-status`: what am I, where am I, can anybody reach me.
  `plugin/commands/self-status.md`.
- `/peer-pair` with nothing and `/peer-list` with no peers both explain and point at
  `/self-status`, and neither prints a string that is not about its target.
  `plugin/commands/peer-pair.md`.
- The terminal is not a surface to rely on, so a user's own identity needs a slash command.
  D-086 (the terminal is not a user experience, and the view is the surface).

**Rejected.**
- *A prefix that names the activity.* It excuses `/peer-pair` printing the user's own string,
  which is about the user and not about a peer, so the exception was the defect.
