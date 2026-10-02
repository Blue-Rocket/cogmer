# D-189 — Nothing is carried forward from a former name

**Date:** 2026-09-22 · **Status:** active · **Areas:** identity, storage, release

**Decision.** The superseded event signing scheme is deleted, and `signingBytes` knows one scheme and
refuses every other version, including the zero value. No state directory under a former name is
looked for, moved or mentioned.

**Support.**
- The only installation that ever existed was on the author's machine and was deleted and not
  migrated, so there is nothing to be compatible with. `cmd/cogmer/keys.go`, `signingBytes`.
- Silently guessing a scheme would verify bytes whose provenance nobody checked, so a missing or
  unknown version is refused. `cmd/cogmer/keys_test.go`, `TestAnEventWithNoRecordedSchemeIsRefused`.
- Schemes are still added and never edited, and the switch that makes that possible remains and is
  tested, so what changed is that the set of schemes worth keeping is empty. D-058 (an event verifies
  under the scheme it was signed with).

**Rejected.**
- *Keeping the old scheme and migrating the state directory.* Both would be work for a population of
  one machine.

**Revisit when** somebody other than the author holds a room. The latitude used here, deleting a
scheme and deleting state, is then gone, and D-058's rule applies with nothing to soften it.
