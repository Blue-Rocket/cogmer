# D-110 — Host order: Claude Code, CoWork soon after, ChatGPT Desktop much later

**Date:** 2026-09-21 · **Status:** active (ordering; alters neither D-043 nor D-086)

**Context.** The order of target hosts was recorded nowhere. One sentence in D-086
(the terminal is not a user experience) carries it as context for a user-interface
argument — "Claude Code is the first host; Claude CoWork is wanted as a fast follow"
— and ChatGPT Desktop appears nowhere in the repository. An ordering that governs
scope should not be a supporting clause inside a decision about pairing surfaces.

**Decision.** Three hosts, in order: **Claude Code**, **Claude CoWork**, **ChatGPT
Desktop**. The interval from Claude Code to CoWork is short. The interval from
CoWork to ChatGPT Desktop is long, and long enough that the two are not planned
together.

**The intervals are the load-bearing part, not the order.** An order alone says only
that ChatGPT Desktop is third, which nothing acts on. The gap sizes are what decide,
for any given piece of host-independence, whether it is preparation or speculation.

**This licenses nothing.** D-043 (a second host is prepared for by naming, never by machinery) forbids
`source`/adapter machinery, and D-086 already considered this exact move and refused
it: "naming a second host does not change that." Naming a third does not either. The
reasoning is untouched — capture generalises, injection does not, and every hard
problem so far has been host-specific. This entry records an intention, not a
permission.

**What the short interval does support** is D-086's cheapest-preparation argument,
now with a nearer payoff. The daemon and its view are host-independent by
construction — a local service and a web page, not an extension of anything — so
every piece of experience living there is one a second host does not reimplement.
That was justified on today's host alone. A short gap to a same-vendor host makes it
a good bet as well as a justified one, which is a reason to prefer the view for new
surfaces, and not a reason to abstract anything.

**What the long interval settles** is that nothing is designed against ChatGPT
Desktop. Different vendor, unknown extension model, and the parts most likely to
differ are precisely the ones D-043 named as non-generalising: §3.5's thinking-block
exclusion is a fact about one transcript format, D-014 (delivery is confirmed by
observation) depends on evidence appearing in a specific file, and the whole
injection path assumes a hook that runs before a turn. A host with no hook
equivalent does not need a smaller adapter; it needs a different design, and that
design is not written before the host exists.

**A consequence for the specification, recorded and not resolved.** §3.8 — "Claude
Code is launched and used unchanged" — is the single test applied to every proposal,
and it is written as a statement about one named product. With a host order it must
either become host-general (*the host is launched and used unchanged; we install
only what it already loads*) or stand as a Claude Code rule at the head of a project
with three hosts. Deciding that is a separate pass; naming it here keeps it from
being decided by drift.

**Rejected.**

- *Leave it in D-086.* Where an ordering lives determines whether anybody finds it,
  and the person who needs this one is scoping a host, not choosing a pairing
  surface.
- *Put it in the phase order.* A phase needs a placement, and CoWork's placement
  depends on its extension model — which is D-086's own revisit trigger. An ordering
  of intent is not yet a phase.
- *Record the order and omit the intervals.* The order without them is inert: it
  cannot tell anybody whether host-independence work is early or premature, which is
  the only question the ordering is consulted for.

**Revisit when:** CoWork's extension model is known (D-086's trigger, unchanged); or
ChatGPT Desktop moves near enough that the long interval stops being the operative
fact; or a fourth host is wanted, at which point an ordering of intent has probably
become a roadmap and wants a different home.
