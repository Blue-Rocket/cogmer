# D-166 — A column that encodes a rule the system does not hold is dropped by migration

**Date:** 2026-09-18 · **Status:** active · **Areas:** storage, rooms

**Decision.** A column that encodes a rule the system does not hold is dropped, not left
in place. `migrateMembership` drops `session_rooms.injected` from every store that has
it.

**Support.**
- The migration drops the column and keeps the binding it carried. `cmd/cogmer/membership.go`,
  `migrateMembership`, and `cmd/cogmer/membership_test.go`,
  `TestMigrationDropsTheInjectedColumn`.

**Rejected.**
- *Leaving the column.* It is harmless, since it has a default and nothing writes it,
  but a column encoding a removed rule is a rule somebody will later find and reinstate,
  and a second mechanism for one rule is how one of them rots.
