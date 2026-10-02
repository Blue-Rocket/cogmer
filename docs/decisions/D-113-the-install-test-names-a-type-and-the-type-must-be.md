# D-113 — The install test names a type, and the type must be documented

**Date:** 2026-09-21 · **Status:** active · **Areas:** hosts, behaviors

**Decision.** The project installs only artifacts of a type among the host's demonstrated and
documented extension mechanisms. In Claude Code those are hooks, skills, MCP servers and the plugin
that carries them.

**Support.**
- What an artifact does is unconstrained and what it is must be on the list, so a hook that
  replicates a conversation to a peer is a hook however little anybody anticipated one. §3.8 (the
  host is launched and used unchanged).
- Documented means the vendor says the mechanism exists on purpose and is meant to keep existing,
  and demonstrated means it works in the installed version, which `doctor` is the means of
  establishing, and a mechanism with one and not the other fails. `cmd/cogmer/doctor.go`.
- "Something the host would load anyway" is a claim about a host's behavior that nobody outside
  the vendor can establish, so against a host nobody here has examined it is unfalsifiable, where
  "is a require hook among the documented mechanisms" has a plain answer, which for
  `NODE_OPTIONS="--require shim.js"` is no. §3.8.
- The registry governs how a documented mechanism behaves, as with the tail of
  `Stop.last_assistant_message`, which is undocumented, and this rule governs what type of artifact
  is installed, so depending on undocumented behavior of a documented mechanism is the ordinary
  condition, and an undocumented artifact type has no check to make because nothing was promised.
  B04 (last_assistant_message contains only the turn's final text block).

**Rejected.**
- *"Something the host would load anyway".* It asks a reader to reason about a host's internals
  and not to look something up.
- *Demonstrated alone.* It admits `NODE_OPTIONS`, which demonstrably works, and shows that a
  mechanism exists today and never that anybody intends to keep it.
- *Documented alone.* It leaves the project installing against a page the shipped version does
  not honor, which is the failure `doctor` exists to catch.

**Revisit when** a host's documentation lags a mechanism this project needs and the mechanism is
plainly deliberate. The question is then whether "demonstrated and documented" should weaken to
"demonstrated and publicly intended", which is harder to apply and is the reason not to reach for it
first.
