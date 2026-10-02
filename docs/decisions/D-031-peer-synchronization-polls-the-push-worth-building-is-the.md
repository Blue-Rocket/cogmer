# D-031 — Peer synchronization polls; the push worth building is the local UI's

**Date:** 2026-09-16 · **Status:** active

**Decision.** Peers synchronize by polling, and push between peers is not built without a
reason beyond latency. The push that is built is the daemon's to the local view, by
server-sent events over loopback.

**Support.**
- An event reaches the other peers in about half a second, while it reaches a
  colleague's Claude only at their next prompt, minutes later during a long turn, so
  push would speed up the half that nobody is waiting on. §16 (propagation).
- A peer that was absent recovers by asking, so reconnecting needs no queue of retries
  and no tracking of what each peer was owed. §16.
- The local view updates live, and a user watching it notices a delay a machine does
  not. §16, and §17 (shared conversation UI).

**Rejected.**
- *Push between peers, for faster propagation.* Polling every second meets the
  propagation target, and pushing would need each sender to know who is connected and
  what each holds, state that can be wrong.

**Revisit when** something needs propagation between peers faster than a second for a
reason other than conversation, such as showing presence, or when rooms are large enough
that constant polling is wasteful.
