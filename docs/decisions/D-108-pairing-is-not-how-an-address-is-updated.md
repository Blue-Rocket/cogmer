# D-108 — Pairing is not how an address is updated

**Date:** 2026-09-21 · **Status:** active (implemented) · **Supersedes** part of D-107

**Context.** D-107 made a pairing string offered for an already-paired peer update
the stored address while skipping the ceremony, on the grounds that a colleague who
moved has no other way to tell you. Rejected on sight, and both halves of the
reasoning turn out to be wrong.

**It is the wrong shape.** Pairing is a security act, and a security act that
silently writes network state is a quiet mutation of the kind this design keeps
removing. It also teaches the wrong reflex: re-pasting somebody's string becomes a
maintenance chore, when handing over a pairing string should be rare, deliberate,
and attached to a ceremony. A string that gets pasted routinely stops being
treated as significant.

**And the gap it filled does not exist.** The case was two peers paired with no
room between them, one of whom moved, so nothing polls and nothing heals. But
nobody needs the address in that state. The next thing that happens is an
invitation, and a host who cannot reach a guest is told so and given a line to send
by hand (D-105). The guest pastes it, joins, and their first synchronization
carries their current address back — repairing the record as a side effect of the
thing they were trying to do anyway.

**An address matters only when it is used, and every use repairs itself.** That is
the general form, and it is why no command for setting one is needed either. A
dedicated address-update command would be a mechanism for a problem that resolves
itself, and one more thing to explain.

**What remains true from D-107:** already paired says so and stops, and `--again`
exists for a comparison somebody has reason to repeat. Only the address write is
withdrawn.

**Revisit when:** a peer can move while sharing no room with anybody and still need
to be reached — which would mean something other than an invitation had come to
depend on a stored address.
