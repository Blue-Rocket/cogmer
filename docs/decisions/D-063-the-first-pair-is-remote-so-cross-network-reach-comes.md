# D-063 — The first pair is remote, so cross-network reach comes before local discovery

**Date:** 2026-09-18 · **Status:** active (ordering)

**Context.** The phase order placed local discovery (Phase 12) before cross-network
reach (Phase 15), with 15's position marked conditional on "whether the first person
who is not the author sits on the same network". That fact is now known: the first
real peer is a colleague at the same company, both working from home. **They will
never share a network.**

**What that invalidates.** D-019 made local discovery the first transport to build,
reasoning that "two people on the same network is the simplest case and must be
the easiest". The claim is true and it is not about anybody who will use this. A
zero-configuration path that serves nobody is not a zero-configuration path; it is
an unused feature with a good argument behind it.

D-019's *requirement* is untouched and should not be read as weakened: no provider
belongs in room identity, membership or replication, and none may be a prerequisite.
The same-network case must still cost nothing when it arises. Only the build order
changes.

**Decision.** Phase 15 moves ahead of Phase 12 and becomes a prerequisite for Phase
13 rather than a conditional detour. Local discovery is not cancelled — it is the
right answer for a case that will arrive, and it is cheap once the transport seam
exists. It is simply no longer first.

**What this does to the tailcat evaluation (D-062).** It moves from "a good fit for
a case that may not arise" to the likely answer for the only case that matters, and
its costs move with it: 526 dependencies and a roughly doubled binary are now prices
being paid rather than risks being weighed. The offsetting fact is that every
alternative is worse here. A company VPN cannot be assumed. A public bind is not
available from a home connection. An SSH tunnel is what Phases 2 and 5 used and is a
person performing NAT traversal by hand, twice a day, forever.

**The DERP objection weakens for this specific pair**, which is worth recording
because it was the strongest one. Tailcat's relay dependency is for *rendezvous*,
and the relay can be self-hosted. A droplet already exists and has hosted both
cross-machine runs. Self-hosting DERP is a relay this pair controls, used to meet
and as fallback, with the working path still direct peer-to-peer — which is a
materially different thing from depending on a third party's infrastructure, and
from the central server §3.3 rejects.

**What has not changed.** A transport still decides nothing (D-019, D-062): a sync
request is signed, refused unless its peer is a guest, and refused unless a person
verified that peer. Reaching the door is not admission.

**Revisit when** a second pair appears who do share a network, or when the same-
network case becomes the common one. Phase 12 is then worth its cost, and the seam
this phase builds is what makes it small.
