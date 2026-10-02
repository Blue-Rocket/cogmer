# D-131 — Delivery is trusted without evidence only once the evidence is known broken

**Date:** 2026-09-16 · **Status:** active · **Areas:** capture, behaviors

**Decision.** When a completed turn's transcript holds no injected block, the pending
turns remain pending and are offered again at the next prompt. The daemon counts them as
delivered without evidence only once `cogmer doctor` has recorded B20 as failing for
the installed Claude Code version.

**Support.**
- A transcript with no injected block looks the same whether the attachment format
  changed or the injection never reached Claude. B20.
- Counting a turn delivered on trust whenever no evidence is found loses context
  silently in the case D-014 (delivery from transcript evidence) exists to catch: the
  first implementation did that, and testing the failure path showed it.
  `84a0751:docs/decisions.md`.
- The daemon logs that it is offering turns again, and points at `cogmer doctor`.
  `cmd/cogmer/daemon.go`, `confirmDelivery`.

**Rejected.**
- *Counting a turn delivered whenever no attachment is found.* It is indistinguishable
  from the injection never arriving, so context would be lost silently.

**Limits.** Offering turns again can inject a colleague's turns more than once, which
is noisy rather than lossy.

**Revisit when** the check for B20 fails.
