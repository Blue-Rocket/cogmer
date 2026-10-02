# D-030 — Two listeners: hooks on loopback, peer sync separately

**Date:** 2026-09-16 · **Status:** active · **Areas:** daemon, transport, trust

**Decision.** Hooks and the local view are served on one listener, `COGMER_ADDR`, which
refuses to bind anything but loopback. Synchronization with peers is served on a separate
listener, `COGMER_PEER_ADDR`.

**Support.**
- Reaching the hook interface is equivalent to being the local user: it publishes into
  the room and returns the room's conversation. `cmd/cogmer/main.go`, `runDaemon`.
- Tests check that neither listener serves the other's routes.
  `cmd/cogmer/listener_test.go`.

**Rejected.**
- *One listener, with the routes filtered by path.* The boundary would be a line of code
  someone can edit without seeing what it guarded, rather than a network interface.

**Revisit when** hooks need to be reached from another machine.
