# D-177 — A busy daemon port is told apart: one of ours exits quietly, anything else is reported

**Date:** 2026-09-20 · **Status:** active · **Areas:** daemon

**Decision.** When the daemon cannot bind its port it asks `/healthz` who holds it. If the
holder is a cogmer daemon it exits quietly. Otherwise it writes `daemon-state`, which the
session-start hook relays to the model.

**Support.**
- Several sessions start at once and race to start the daemon, and one wins, so
  losing the race is the ordinary outcome. §29 (starting the daemon).
- Anything else holding the port is a real fault, and the only route from a detached daemon
  to a person is by way of what the model says. D-033 (the room cannot be displayed inside
  Claude Code), D-036 (MCP logging notifications are not a display channel), D-075 (the
  install has a state, and absence is not an inference).
- `daemonAlreadyServing` reads `/healthz`, which names the peer the daemon belongs to.
  `cmd/cogmer/main.go`, `daemonAlreadyServing` and `daemonStateFile`.

**Rejected.**
- *Treating every bind failure as fatal.* The ordinary race would then read as a fault.
