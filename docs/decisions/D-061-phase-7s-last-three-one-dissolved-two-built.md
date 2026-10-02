# D-061 — Phase 7's last three: one dissolved, two built

**Date:** 2026-09-18 · **Status:** active (implemented)

**Context.** Phase 7 listed a local outbound queue, peer health, and context-size
controls. Taking them in turn produced one deletion and two findings.

### The outbound queue is dissolved, not deferred

It presupposed push: something must hold what a peer has not yet been told.
Synchronization is a pull (§10, D-031), so an event is durable the moment it is
committed locally and "what did you miss" is a question asked rather than state
anyone keeps. A queue would be a second record of what the event store already
holds, and the two would drift — the first time a peer was dropped from one and not
the other, silently.

Recorded in §7 as struck through with the reason, rather than removed. An item that
vanishes reads as forgotten.

### Peer health: report transitions, not polls

The sync loop logged `unreachable` once per address per poll. At a one-second
interval that is two lines a second saying the same thing, which was observed
filling a log through the Phase 5 partition. **A message repeated that often is not
a signal — it buries the one line that matters, which is the transition.**

Outages are now reported when a peer stops answering, again every ten minutes while
it continues, and once when it returns with how long it was gone. The repeat exists
because somebody reading a log an hour later should learn the peer is *still* gone
rather than only that it once went.

Also fixed: `peerStatus` enumerated only `COGMER_PEERS`, so every peer learned
by pairing or by joining a room was **invisible in the browser view** — which since
D-053 is most of them. It now reports every address the daemon would try.

### Context size: the limit that was missing bounded nothing

`maxInjectedEvents` capped turns and `maxInjectedChars` capped each turn, and
neither bounds the block. Forty turns of eleven thousand characters pass both.
Measured: **404,635 characters** would have been injected into a context window.

A whole-block budget now applies after the event cap, measured on the rendered
turns rather than estimated from their inputs, dropping from the front so the newest
conversation survives — a referent is most likely to point at recent turns. All
three limits are configurable, as §21 asked and as none of them were, and a limit
that parses as zero or as nonsense is ignored rather than obeyed: injecting nothing
looks exactly like a room where nobody is talking.

§21 is amended to say what each limit is *for*, since the one that was missing is
the one the list made sound redundant.

**On estimated token count**, which §21 also asked for: derived from characters at
four to one rather than measured. A tokenizer would have to track a model this
system does not choose, and the figure is wanted for judgement rather than
arithmetic.

**Revisit when** a room routinely exceeds the block budget. §21 treats these as a
safety valve, and under session-scoped rooms hitting one means an unusually long
pairing — so hitting one *often* means the rooms are not as session-scoped in
practice as D-015 assumes, which is worth knowing.
