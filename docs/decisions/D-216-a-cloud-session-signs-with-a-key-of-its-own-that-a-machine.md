# D-216 — A cloud session signs with a key of its own that a machine identity vouches for

**Date:** 2026-10-18 · **Status:** not built · **Areas:** identity, sync, admission

**Decision.** A cloud session signs its events, its admission proofs and its connections
with a key it generates when it starts, and that key carries a chain of two certificates
to a machine identity (§6): the machine identity signs a cloud key with no expiry, the
cloud key is the one secret stored in the cloud environment, and the cloud key signs each
session's key with an expiry. A receiver accepts a key whose chain ends at a machine
identity it has verified, and refuses one whose chain does not, so that a colleague
verifies the machine once (§25), never each session, and sees a cloud session's turns
under that machine's label and derived name. Each session's key numbers its own events,
so cloud sessions that run at the same time never claim the same sequence number. The
machine identity or the cloud key itself revokes the cloud key with a signed event
published in every room the machine is in, and a receiver that holds the revocation
refuses every event and connection under that cloud key that reaches it afterwards.

**Rejected.**
- *Copying the machine's private key into the cloud environment.* Two writers under one
  identity issue the same sequence numbers, which a receiver sets aside as conflicts
  (D-027, a sequence conflict is quarantined), and a copied key never expires, so whoever
  reads it holds the identity until every colleague forgets the peer.
- *A second permanent identity for cloud sessions.* Colleagues pair with it as a separate
  peer, two cloud sessions running at once collide on its sequence numbers, and a stolen
  key holds until every colleague forgets the peer.
- *The cloud key signing events itself.* Concurrent cloud sessions would share its
  sequence numbers.
- *A cloud key that expires.* Renewing it needs the machine that issued it, which a user
  away from that machine cannot reach.
- *Pairing each cloud session.* The comparison needs both people on a call (D-052,
  verifying a peer is its own act, run by both users at once), once for every session.
- *A cloud session that relays every exchange to the machine's daemon.* It works only
  while that machine is awake and reachable, which is the dependence this decision
  removes.

**Limits.** The cloud key is readable by anything that runs in the session, including the
model's shell, so a model steered by room content can copy it, and whoever holds it can
act as that machine in every room the machine is admitted to until each colleague's
daemon holds a revocation. A colleague's daemon learns of a revocation only by syncing a
room with a peer that holds it, and events accepted before then remain, since events are
immutable. A colleague whose cogmer predates the scheme refuses a delegated event as a
version problem (D-058, an event is verified under the scheme it was signed with). This
entry does not decide how a cloud session learns which colleagues its machine verified
and which rooms it is in, how it reaches peers, or how its hooks load.

**Revisit when** a cloud session can sign with a secret it cannot read, or a cloud key is
used by someone other than the user who issued it.
