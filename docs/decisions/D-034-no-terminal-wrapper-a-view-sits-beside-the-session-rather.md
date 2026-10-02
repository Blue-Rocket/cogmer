# D-034 — No terminal wrapper; a view sits beside the session rather than around it

**Date:** 2026-09-17 · **Status:** active

**Decision.** No pseudo-terminal wrapper launches Claude Code. A view of the room is a
separate program beside the session, never around it.

**Support.**
- A user starts the host the same way they would without cogmer. §3.8 (the host is
  launched and used unchanged).
- Claude Code runs as a desktop application, a web application and an editor extension
  as well as in a terminal, and a pseudo-terminal reaches only the terminal. §3.8.
- A wrapper on Windows would need ConPTY, a different interface from the Unix
  pseudo-terminals, so the display path would diverge by platform. D-001 (Go, not
  TypeScript/Node or Python).
- A passthrough prototype under a real 24×80 pseudo-terminal was byte-for-byte identical
  to running `claude` directly, so the obstacle is not feasibility.
  `84a0751:docs/decisions.md`.

**Rejected.**
- *Wrapping Claude Code in a pseudo-terminal, with colleagues' turns drawn in a reserved
  band.* It changes how a user starts Claude Code, reaches only the terminal, and needs
  a second implementation on Windows.
- *A wrapper for terminal users and something else for everyone else.* Two display
  paths, the harder one reaching fewer people.

**Revisit when** starting Claude Code through a wrapper stops being a change to how a
user starts it.
