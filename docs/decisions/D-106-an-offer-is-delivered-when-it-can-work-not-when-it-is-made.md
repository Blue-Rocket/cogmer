# D-106 — An offer is delivered when it can work, not when it is made

**Date:** 2026-09-21 · **Status:** active (implemented) · **Refines** D-105

**Context.** D-105 makes an invitation an offer pushed over the paired channel.
Inviting an unverified peer is permitted and inert (§25), so the two together would
make a room that does nothing easier to create than one that works.

**What today's friction was hiding.** A host reads the warning, copies a string,
sends it to the guest. Even a host who ignores the warning passes through a human
moment where noticing is possible. Push removes that: invite, accept, and both
people believe they are in a room which is permanently silent. The join-time warning
becomes the only thing between them and that, and it travels by the weakest channel
there is — a line a command printed, relayed by a model.

**So the admission is recorded and the offer is withheld.** Inviting an unverified
peer still writes the guest-list row, immediately. Whom to admit remains the host's
judgement and §25's separation of the two questions is untouched. What waits is the
*delivery of a notification*, and it waits only as long as it would be useless.

**Why the gate stays open at all, which is the load-bearing half.** Refusing to
invite an unverified peer would be simpler — one fewer state for a pair to be in,
and no queue to flush. It is declined because **wanting to start a room is what
makes somebody willing to verify**. Verification is a chore whose payoff is
invisible until it is needed, and attempting a room is the moment it acquires one.
A refusal blocks a person exactly when they are motivated and sends them off to do
an errand; a queued invitation meets them there. Expect the simplification to be
proposed on the grounds of fewer combinations, and that is the answer: the extra
state is one pending offer with one flush point, bought with the only moment this
system gets somebody's attention for free.

**Which decides the wording, not only the behaviour.** The difference between a
block and a path is whether the invitation succeeded and is waiting. It must read as
admitted, queued, and one step from done — with that step offered where it is
stated, since the two-word check opens in a browser from the same place (D-088). An
invite that reports a refusal throws away the advantage this entry exists to keep.

**Verification gains an effect: it flushes what was waiting.** When two peers
complete the two-word comparison, offers already recorded for that peer are
delivered. So the sequence a host expects — invite, then verify — produces the room
at the end of it, rather than producing a silent one in the middle and repairing it
later.

**The fallback string is withheld on the same terms.** It would otherwise be a way
to reach the outcome the withholding exists to prevent: a guest who pastes a string
joins and gets the same dead room. Inviting an unverified peer therefore produces
neither a delivery nor a string, and says what to do instead.

**Rejected: deliver it, marked unverified.** The guest's daemon could receive the
offer and say that nothing will sync until both verify. It was not taken because a
notification that produces a dead room is worse than no notification — it reads as
progress, and the only thing correcting it is a sentence somebody may not read. An
offer that arrives when it works needs no warning at all.

**Self is a guest of its own rooms and is never in its own peer list** (D-103), so
any check of "is this guest verified" must exclude self rather than conclude that
the creator is unverified and withhold from them.

**Where the two waiting things surface, since they point in opposite directions.**
An offer waiting to be accepted is a room, and belongs with rooms — it is listed
apart from the rooms already joined, because accepting is the act that has not
happened. An invitation withheld for want of a verification is not visible to the
person it is for and is entirely visible to the person who can release it, so it
belongs beside that peer.

**Name the person, do not count the rooms.** A tally was tried first and is close
to useless here: the ordinary room holds two people, so the number is always one
and carries nothing. What somebody can act on is which colleague is unfinished and
what to type. It must also stop being said once the pairing completes, rather than
persisting as a record of work already done.

**The appeal is to finish pairing, not to verify.** Pairing is the act a person
recognises; verifying is our word for a step inside it, and nobody thinks "I must
verify Alice". So the prompt says there is work to do with Alice and offers
`/peer-pair alice`.

That required the command to accept a name. It previously took only a pairing
string, so `/peer-pair alice` hashed the literal text into an identifier and
invented a peer nobody had met — worse than refusing, because it produced a
plausible stranger. It now resolves a recorded peer and runs the ceremony, which is
the second half of an act rather than a new one, so it asks for no label: they were
named when they were recorded.

**Revisit when:** admission and verification stop being separable — if a future
change makes one imply the other, withholding has nothing left to sequence.
