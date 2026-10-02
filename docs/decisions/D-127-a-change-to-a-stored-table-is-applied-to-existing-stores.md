# D-127 — A change to a stored table is applied to existing stores when they open

**Date:** 2026-09-24 · **Status:** active

**Decision.** A column or constraint added to a stored table is also applied by the
migration its store runs when it opens: `migrate()` for a room store and
`migrateMembership()` for the membership store. A schema change is complete only when
that migration carries it.

**Support.**
- `CREATE TABLE IF NOT EXISTS` leaves an existing table unchanged, so a column added to
  that statement is missing from every store created before it, and the failure
  appears at the first query that names the column, not when the store opens.
  `84a0751:docs/decisions.md`, D-043's paragraph "A bug this surfaced".
- `migrate()` runs each time a room store opens, and reads the table's columns to add
  the ones it lacks. `cmd/cogmer/store.go`.
- SQLite has no way to drop a constraint, so relaxing one rebuilds the table, which
  `migrateMembership()` does. D-050 (room names may collide locally), and
  `cmd/cogmer/membership.go`.

**Rejected.**
- *Leaving an older store in the shape it was created in.* A user who has been working
  in a room would lose it to a change that mattered only to the maintainers. The
  comment above `migrate()` in `cmd/cogmer/store.go`.

**Limits.** Nothing checks that a schema change reaches the migration. A test that
opens only freshly created stores cannot see a column missing from an older one.

**Revisit when** a change to a stored format needs more than adding columns or
rebuilding a table, such as a change to the data a store holds.
