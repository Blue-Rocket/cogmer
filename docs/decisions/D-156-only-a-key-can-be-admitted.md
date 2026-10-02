# D-156 — Only a key can be admitted

**Date:** 2026-09-17 · **Status:** active · **Areas:** admission, identity

**Decision.** Recording a known peer and admitting a guest both refuse an identifier that
names no key.

**Support.**
- `Allow` and `Invite` each refuse an identifier from which no public key can be read.
  `cmd/cogmer/membership.go`, `Allow` and `Invite`.
- Admission is proved by possession of the key the identifier names. D-143 (admission is
  proved by signing a fresh challenge).

**Rejected.**
- *Recording a name or an identifier from before keys, such as `alice` or `peer-8f3a…`.*
  Nothing could ever prove possession of it, so the entry could never do its job.

**Revisit when** a guest has to be recorded before its key is known.
