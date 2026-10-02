# D-154 — An event whose signature fails is refused, not quarantined

**Date:** 2026-09-17 · **Status:** active · **Areas:** sync, identity

**Decision.** An event whose signature does not verify is refused and logged, and never
kept aside.

**Support.**
- A failed signature has no harmless reading, unlike a sequence conflict, which may be a
  peer that lost its state. D-027 (a sequence conflict is quarantined, not dropped).
- A peer that impersonated another and offered an event signed by nobody was refused, and
  nothing reached the room. `84a0751:docs/decisions.md`.

**Rejected.**
- *Quarantining it as a conflict is quarantined.* There is nothing to tell apart later.

**Revisit when** a failed signature is found to have a harmless cause.
