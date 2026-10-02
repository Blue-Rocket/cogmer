# D-043 — A second host is prepared for by naming, never by machinery

**Date:** 2026-09-17 · **Status:** active · **Areas:** hosts, sync

**Decision.** Nothing in cogmer names the agent that produced a turn: the session a turn
came from is an origin session, opaque to the host. There is no field saying which host
produced an event, no architecture of adapters and no design for a second host until a
second host exists.

**Support.**
- Capture was solved in the first integration work, and every hard problem since has
  been specific to injecting into Claude Code: hooks display nothing, MCP never renders,
  a remote event must not start a turn, the injected block needs an unforgeable fence,
  and delivery is confirmed by evidence. D-033, D-036, D-035, D-040 and D-014.
- A second adapter would inherit the schema and none of that difficulty, so building one
  now would generalize from one host and no users. §3.8 (the host is launched and used
  unchanged).

**Rejected.**
- *A field naming the host, such as `source: claude-code | codex`, and an architecture
  of adapters.* It rests on the claim that collaboration across agents becomes nearly
  free once events are normalized, and what is hard does not transfer between hosts.

**Revisit when** a second host is chosen to be supported.
