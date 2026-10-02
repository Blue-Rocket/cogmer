# D-072 — A refused join is explained and not only refused

**Date:** 2026-09-19 · **Status:** active · **Areas:** rooms, commands

**Decision.** When a session that has been in one room is refused another, the refusal
explains the mechanism and gives the way out. It says that the first room's conversation
would reach the second through what the session says next, which cannot be taken back, and
that a second Claude Code session costs nothing and this one may still rejoin its room.

**Support.**
- The user has done nothing wrong, the refusal comes from an invariant they have not read,
  and nothing they can do will make it work, so a rule without a route leaves them stuck
  in the one situation where being stuck is permanent. `cmd/cogmer/membership.go`,
  `BoundElsewhereError`.
- The error carries both rooms, because a caller that cannot name the room a person was
  trying to reach cannot say what to do instead. `cmd/cogmer/membership.go`,
  `BoundElsewhereError`, and `cmd/cogmer/membership_test.go`, `TestARefusedJoinCanBeExplained`.
- `/room-join` relays the whole explanation, since a model that condenses it would reduce it to the one-line refusal. `plugin/commands/room-join.md`.

**Rejected.**
- *A one-line refusal that states the rule.* It leaves the user with no route.
