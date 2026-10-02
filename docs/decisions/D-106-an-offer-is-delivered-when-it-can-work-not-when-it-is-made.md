# D-106 — An offer is delivered when it can work, not when it is made

**Date:** 2026-09-21 · **Status:** active · **Areas:** admission, pairing

**Decision.** Inviting an unverified peer records the admission at once and withholds the offer
and the fallback string. When the two complete verification the offers waiting for that peer are
delivered. The invite reads as admitted and queued and never as a refusal.

**Support.**
- Whom to admit is the host's judgment and is separate from whether the key is the person's, so
  the admission is recorded and only the notification waits, and it waits only as long as it
  would be useless. §25 (security), `cmd/cogmer/offer.go`.
- Wanting to start a room is what makes somebody willing to verify, since verification is a
  chore whose payoff is invisible until it is needed, so a queued invitation meets a user at the
  moment they are motivated, and the extra state is one pending offer with one flush point.
  `cmd/cogmer/membership.go`, `WithheldFor`.
- Without the wait, invite and accept would leave both people believing they are in a room that
  is permanently silent, and the join-time warning would be the only thing between them and
  that, carried by the weakest channel there is, a line a command printed and a model relayed.
  `cmd/cogmer/offer_test.go`.
- The fallback string is withheld on the same terms, because a guest who pasted it would join and
  get the same dead room. `cmd/cogmer/offer.go`.
- This peer is a guest of its own rooms and is never in its own peer list, so a check that a
  guest is verified excludes this peer and does not conclude that the creator is unverified.
  D-103 (an address belongs to a peer and is stored in one place).

**Rejected.**
- *Refusing to invite an unverified peer.* It is simpler, with one fewer state, and it blocks a
  user when they are motivated and sends them on an errand.
- *Delivering the offer marked unverified.* A notification that produces a dead room is worse
  than none, since it reads as progress and a sentence is all that corrects it.

**Revisit when** admission and verification stop being separable, since withholding then has
nothing left to sequence.
