# D-113 — The install test names a type, and the type must be documented

**Date:** 2026-09-21 · **Status:** active (amends §3.8; tightens D-111)

**Context.** D-111 stated the install clause as "something the host would load
anyway". That says what the test is asking and not how to check it: *would load
anyway* is a claim about a host's behaviour, and behaviour is the one thing nobody
outside the vendor can establish. Against a host nobody here has examined — D-110
names two — it is unfalsifiable, which is fatal for a test whose whole merit is
being applicable without argument.

**Decision.** This project installs only **artifacts of a type included in the
host's demonstrated, documented extension mechanisms**. In Claude Code: hooks,
skills, MCP servers, and the plugin that carries them.

Two halves, both load-bearing.

- **Type, not instance.** What an artifact *does* is unconstrained; what it *is*
  must be on the list. A hook that replicates a conversation to a peer is a hook,
  however little anybody anticipated one. This is not a rule to do only what the
  host imagined.
- **Documented, and demonstrated.** Documented is the vendor saying the mechanism
  exists on purpose and is meant to keep existing. Demonstrated is that it works in
  the version installed, which `doctor` is already the means of establishing. A
  mechanism with one and not the other fails.

**What this catches that the old phrasing did not.** `NODE_OPTIONS="--require
shim.js"` against an npm-distributed host: the runtime loads arbitrary code into the
process, no launch command changes because the variable can be exported from a
profile, and rendering inside the session — the thing D-036 and D-038 keep running
into — becomes possible. Asked as "would the host load it anyway", it invites an
argument about what counts as the host, since Node genuinely does load it. Asked as
"is a require hook among Claude Code's documented extension mechanisms", it is
simply no.

**This does not conflict with the behaviour registry**, which is the first
objection a reader will raise, because `behaviors.go` records undocumented
behaviours this project depends on and there are many. The axes are different. The
registry governs how a *documented mechanism behaves* — hooks are documented,
`Stop.last_assistant_message` carrying only the tail (B04) is not. This rule governs
what *type of artifact* is installed. Depending on undocumented behaviour of a
documented mechanism is the ordinary condition here, and is exactly why the registry
has checks with negative tests. An undocumented artifact type has no such recourse:
there is nothing to check against, because nothing was promised.

**Rejected.**

- *"Something the host would load anyway"* (D-111's phrasing). Right in intent, and
  it asks a reader to reason about a host's internals instead of looking something
  up.
- *Demonstrated alone.* Admits `NODE_OPTIONS`, which demonstrably works.
  Demonstration establishes that a mechanism exists today and never that anybody
  intends to keep it.
- *Documented alone.* Costs nothing to drop the other word and leaves us installing
  against a documentation page the shipped version does not honour — which is the
  failure `doctor` exists to catch, so the second half is already paid for.

**Revisit when** a host's documentation lags a mechanism this project needs and the
mechanism is plainly deliberate. The question then is whether "demonstrated,
documented" should weaken to "demonstrated and publicly intended", which is harder
to apply and is the reason not to reach for it first.
