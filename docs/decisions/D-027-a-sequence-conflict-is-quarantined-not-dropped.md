# D-027 — A sequence conflict is quarantined, not dropped

**Date:** 2026-09-16 · **Status:** active

**Decision.** Each event received is classified as stored, duplicate or conflict, where a
conflict is the same peer and sequence number arriving with a different event
identifier. A conflicting event is set aside with both identifiers, and `cogmer
conflicts` shows it.

**Support.**
- A conflict means a peer lost its state or an event was forged, and the kept event is
  the only evidence that tells the two apart. §8 (event identity and ordering).
- A peer and its sequence number are an event's identity, so a receiver that changed
  them would disagree with every other peer's copy. §13 (transitive synchronization).
- A kept conflict nobody can see would be as silent as a dropped one. `cmd/cogmer/main.go`,
  the `conflicts` command.

**Rejected.**
- *Ignoring the conflict as a duplicate.* It is unrecoverable: the sender believes it
  shared the event, the receiver never sees it, and anti-entropy cannot fill a gap where
  the sender's highest sequence is below what the receiver reports holding.
- *Overwriting the event held.* The incoming event has no better claim, and events are
  immutable.
- *Giving the incoming event a free sequence number.* It would preserve the event and
  change its identity, which makes the receiver's copy disagree with every other peer's.

**Limits.** A local event takes its sequence from the membership index (D-029), so a
conflict on a local event means the local store is inconsistent rather than that a peer
misbehaved, and it is reported that way.

**Revisit when** conflicts are found to have a cause other than lost state or forgery.
