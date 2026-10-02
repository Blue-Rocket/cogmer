# D-179 — An operating-system notification announces and never carries what must be known to have landed

**Date:** 2026-09-20 · **Status:** active · **Areas:** view, capture

**Decision.** A notification from the daemon announces something, and nothing that must be known to
have arrived is left to one.

**Support.**
- The daemon cannot learn whether a notification was shown, suppressed or dismissed unseen,
  since the notification database is unreadable and the focus state is protected, so its
  delivery cannot be observed. `f288913:docs/decisions/D-082-the-daemon-can-reach-a-person-directly-there-is-no-setsid.md`.
- Delivery that matters is derived from evidence and not from the sending. D-014 (derive
  delivery state from transcript evidence, not from recorded intent).

**Rejected.**
- *Relying on a notification as delivered.* Nothing could show whether it was.

**Limits.** A notification posts under the identity of the tool that raised it, and clicking
it does nothing.
