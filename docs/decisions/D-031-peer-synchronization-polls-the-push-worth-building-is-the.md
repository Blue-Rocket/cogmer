# D-031 — Peer synchronization polls; the push worth building is the local UI's

**Date:** 2026-09-16 · **Status:** active · **Areas:** sync, view

**Decision.** Machines synchronize by polling, and push between machines is not built without a
reason beyond latency. Between machines, the one thing pushed is a nudge, which carries no
events and tells a machine to poll the sender now, so that quiet rooms can be polled slowly
for their cost without slowing delivery (D-305, a machine nudges the machines it shares a room
with when it writes an event). The other push that is built is the daemon's to the local view,
by server-sent events over loopback.

**Support.**
- While a room is active, an event reaches the other machines in under a second, while it
  reaches a colleague's Claude only at their next prompt, minutes later during a long turn, so
  push would speed up the half that nobody is waiting on. §16 (propagation).
- Each address is polled in a loop of its own, so a machine that does not answer delays no other, and it is polled less often, up to every fifteen seconds, or every sixty once its rooms are dormant (D-304, an address is polled less often while its rooms are quiet), until it answers. `cmd/cogmer/sync.go`, `RunSync`, `cmd/cogmer/syncworkers_test.go`, `TestAPeerThatDoesNotAnswerDoesNotDelayOneThatDoes`.
- A machine that was absent recovers by asking, so reconnecting needs no queue of events
  and no tracking of what each machine was owed. §16.
- The local view updates live, and a user watching it notices a delay a machine does
  not. §16, and §17 (shared conversation UI).

**Rejected.**
- *Pushing events between machines, for faster propagation.* Pushing events would need
  each sender to know who is connected and what each holds, state that can be wrong, and
  a nudge followed by the receiver's poll meets the propagation target without it.

**Revisit when** something needs propagation between machines faster than a second for a
reason other than conversation, such as showing presence, or when polls that return
nothing are measured to be most of what a machine spends, with quiet rooms polled less
often (D-304, an address is polled less often while its rooms are quiet).
