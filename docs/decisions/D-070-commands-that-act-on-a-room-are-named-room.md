# D-070 — Commands that act on a room are named room-*

**Date:** 2026-09-19 · **Status:** active · **Areas:** commands, rooms

**Decision.** A command that acts on one room carries the prefix `room-`, as in
`/cogmer:room-create`.

**Support.**
- `room` is the domain's own noun for what the command acts on, and a prefix names the
  target of its command. D-096 (a command prefix names its target).
- Every room command is a file named for it. `plugin/commands/room-create.md`.

**Rejected.**
- *`team-`.* The commands act on a room and not on a team, and a prefix that names
  something else misleads.

**Limits.** Claude Code adds the manifest's name in front of every command, which is
D-118 (the plugin manifest name is the command namespace, and commands keep their prefixes under it).
