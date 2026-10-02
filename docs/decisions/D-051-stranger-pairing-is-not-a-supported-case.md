# D-051 — Stranger pairing is not a supported case

**Date:** 2026-09-17 · **Status:** active

**Decision.** Pairing with someone unknown is not a design target. No affordance
presents it as intended, and no claim is made that verification protects it.

**Support.**
- Verification needs a channel on which the other party can be recognised, and people
  who have never met have none. §25 (security).
- A match between strangers shows that two parties hold the same key and says nothing
  about whose, so the ceremony would give the appearance of assurance, and people act on
  the appearance. §25.
- A room carries a working session, and admission sends a member's turns to another
  person's provider under that person's account. §28 (configuration).
- People who know one another and whose machines have not met are the ordinary first
  contact, since colleagues pair repeatedly, and they are not strangers in this sense.
  §12 (forming a room).

**Rejected.**
- *Designing the host-approval path as the way two strangers pair.* The ceremony takes
  nothing from verification when the two have never met, and the situations that want it,
  such as mentoring, an interview or a contractor's first day, almost always have a call
  available, so it buys convenience and not capability.

**Limits.** The mechanism does not forbid it. A host who approves a request from someone
unknown has paired with a stranger, and that is the host's judgement.

**Revisit when** there is a concrete use for pairing with someone unknown that a call
cannot serve, or a verification channel exists that does not depend on recognising the
other party.
