# D-105 — An invitation travels over the channel pairing already established

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** While tracing how a peer recovers from an address change, the
hand-carried invitation was defended on the grounds that it survives one-way
reachability: the string goes host to guest by human means, and the guest then dials
the host, which is the direction that works when the guest has moved. Answered that
the trade is the wrong way round — recovery from an address change is an outlier and
forming a room is an everyday act.

That is right, and the argument is stronger than it was put. The robustness being
defended protects a case D-104 largely removes: with a stable overlay address, a
host becomes unable to reach a guest only after a restart *and* an unreachable
relay. So the trade was an everyday inconvenience against an outlier of an outlier.

**§12 already describes the intended shape**, and the implementation is what
diverged. The specification has the guest typing a room name, with the host's daemon
recording the admission and the guest's daemon locating the room — and no account of
how it locates it. Lacking a mechanism, the implementation answered by printing a
full invitation string and requiring it. This supplies the missing mechanism rather
than changing the goal.

**An invitation becomes an offer delivered over the paired channel**, with the
string kept as a fallback when the host cannot reach the guest.

**Pushing an offer is not pushing an admission**, which is what makes it safe. The
guest-list row on the host's side is the admission, and it is the host's judgement
to make (D-024). Delivering an offer only tells the guest that row exists. Joining
remains the guest's own act, nothing is entered on their behalf, and reading
anything still requires verification (D-054). §3.7 is untouched: an offer arriving
is data being stored, not a turn being taken.

**The failure is better than the one it replaces.** A host who cannot reach the
guest is told at the moment of inviting and handed the string to send. Today the
string is the only path, and nobody learns whether it was needed.

**Rejected: the guest polling for offers.** It works in the direction that survives
a moved guest, and it matches the anti-entropy pull the rest of the system uses. It
was not taken because it requires every pair of paired peers to maintain contact
indefinitely, whether or not they share a room — standing traffic and standing
metadata, permanently, for something that happens rarely. Push has an immediate and
honest failure with a working fallback, and costs nothing between invitations.

**This mostly dissolves Phase 14.** "First contact without a paste" is the same
problem approached from the other end. Once an invitation reaches a paired guest
over the channel, the only remaining manual exchange is between people who have
never paired, which D-051 declines to support.

**Revisit when:** paired peers acquire a standing reason to poll each other, at
which point pull costs nothing extra and is the better shape.
