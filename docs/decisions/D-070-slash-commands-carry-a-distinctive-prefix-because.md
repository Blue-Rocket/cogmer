# D-070 — Slash commands carry a distinctive prefix, because invocation is not namespaced

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Observed that command names need namespacing to avoid colliding with
other plugins. Checking the mechanism rather than assuming one: a subdirectory
changes how a command is **displayed** — `/build (project:ci)` — and not what a
person types. There is no `plugin:command` invocation form.

So two plugins defining the same name collide at the only thing a user touches. This
is not hypothetical: in the official marketplace today `hookify` and `ralph-loop`
both define `/help`.

**Decision.** `/team-*` becomes `/room-*`. `team` is about as collision-prone as a
prefix gets in a directory of developer tooling; `room` is the domain's own noun and
markedly less crowded.

**The prefix is tied to the protocol namespace, not the product name.** `room`
follows `protocolNamespace` (D-069), which is committed to never changing. Had it
followed the product name it would be hostage to a naming decision that has not been
made — and unlike the signing tags, nobody would notice until a collision.

**Unlike the couplings in D-069, this one is cheap to revisit.** Six markdown files
and no stored data depends on them, so if the settled name suggests a better prefix,
changing it costs a rename. It is being done now because the cost of *not* doing it
is a collision in somebody else's session, which is a bad way to find out.

**Revisit when** the name is settled, or if Claude Code gains a `plugin:command`
invocation form — at which point the prefix becomes belt and braces rather than the
only protection.
