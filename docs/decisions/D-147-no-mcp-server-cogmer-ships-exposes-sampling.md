# D-147 — No MCP server cogmer ships exposes sampling

**Date:** 2026-09-17 · **Status:** active

**Decision.** Any MCP server cogmer ships exposes no sampling, the MCP mechanism by which a
server asks the client to run inference.

**Support.**
- Through sampling, a peer's daemon could cause inference in an interactive session by way
  of a server, which §3.7 (a remote event never drives an interactive session) forbids.
- Whether Claude Code implements sampling has not been tested.
  `84a0751:docs/decisions.md`.

**Rejected.**
- *An MCP server that exposes sampling.* It would be a route by which a peer drives a
  session.

**Revisit when** cogmer ships an MCP server.
