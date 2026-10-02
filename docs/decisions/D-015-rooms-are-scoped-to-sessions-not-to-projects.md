# D-015 — Rooms are scoped to sessions, not to projects

**Date:** 2026-09-16 · **Status:** active · **Areas:** rooms

**Decision.** A room is a set of linked Claude Code sessions, identified by a generated
identifier and entered by invitation. It closes when every member has explicitly left,
or when it has been dormant long enough that resuming it is not plausible.
Nothing about a room is derived from a directory, a repository or a project.

**Support.**
- A room that outlasted its sessions would face a new session with weeks of unseen
  events, and injection would have to truncate them. §21 (context window management).
- A Claude Code session's identifier survives resumption and compaction, so membership
  held by a session is stable across a laptop sleeping and a session being resumed.
  B14.
- A session whose process exits is absent, not gone. §12a (room membership).

**Rejected.**
- *Rooms scoped to projects, resolved from the working directory.* It needs a
  configuration file, a rule for walking up directories, a guard that is off by
  default, and validation of room names against path traversal, all to infer what an
  invitation states outright.
- *Standing team rooms.* They produce the catch-up problem, and widen whose model
  provider receives a user's conversation through injection, for a collaboration that
  needs only a live pairing.

**Limits.** Anti-entropy reconciles only interruptions inside a live pairing, not
hours or days apart, so a durable team memory would have to be built over archived
rooms rather than live ones.

**Revisit when** catching up asynchronously on weeks of a team's conversation proves
more valuable than bounded context.
