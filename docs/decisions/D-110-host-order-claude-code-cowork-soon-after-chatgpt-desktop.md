# D-110 — Host order: Claude Code, CoWork soon after, ChatGPT Desktop much later

**Date:** 2026-09-21 · **Status:** active · **Areas:** hosts, project

**Decision.** The hosts are Claude Code, then Claude CoWork after a short interval, then ChatGPT
Desktop after a long one, long enough that the last is not planned with the others. The order
licenses no adapter machinery.

**Support.**
- The intervals decide whether a piece of host independence is preparation or speculation, and an order alone says only
  that one host is third, which nothing acts on. D-043 (a second host is prepared for by naming,
  never by machinery).
- A second host is prepared for by naming and never by machinery, and naming a third changes
  nothing about that, since capture generalizes, injection does not, and every hard problem so far
  has been specific to one host. D-043 (a second host is prepared for by naming, never by
  machinery).
- The daemon and its view are a local service and a web page, so every piece of experience that
  lives there is one a second host does not reimplement, which is a good bet with a short gap to a
  host of the same vendor, and a reason to prefer the view for new surfaces and none to abstract
  anything. D-086 (the terminal is not a user experience, and the view is the surface).
- Nothing is designed against ChatGPT Desktop. The parts most likely to differ are specific to one
  host: that thinking blocks are excluded from capture is a fact about one transcript format, that
  delivery is confirmed by evidence in a specific file, and that injection assumes a hook that runs
  before a turn, so a host with no hook equivalent needs a different design and not a smaller
  adapter. `cmd/cogmer/transcript.go`, D-014 (derive delivery state from transcript evidence, not
  from recorded intent), D-043 (a second host is prepared for by naming, never by machinery).

**Rejected.**
- *Leaving the order inside D-086.* Where an ordering lives decides whether anybody finds it, and
  the person who needs it is scoping a host and not choosing a pairing surface.
- *Putting the order in a plan of work.* A phase needs a placement, and CoWork's depends on its
  extension model. An ordering of intent is not yet a phase.
- *Recording the order and leaving out the intervals.* It cannot say whether host-independence work
  is early or premature, which is the only question the ordering is consulted for.

**Revisit when** CoWork's extension model is known, or ChatGPT Desktop moves near enough that the
long interval stops being the operative fact, or a fourth host is wanted, when an ordering of
intent has probably become a roadmap and needs a different home.
