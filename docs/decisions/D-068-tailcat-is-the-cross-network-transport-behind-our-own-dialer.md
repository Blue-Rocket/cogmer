# D-068 — Tailcat is the cross-network transport, behind our own dialer

**Date:** 2026-09-18 · **Status:** active · **Areas:** transport

**Decision.** Tailcat carries synchronization between peers on different networks, and
all of it is confined to `cmd/cogmer/tailcat.go` behind the `Dialer` interface and a
`net.Listener`. One set of routes serves both transports.

**Support.**
- Two users working from home are each behind a router, and neither can bind an address
  the other can reach, so a connection between them needs traversal that a user would
  otherwise do by hand with a tunnel. `f288913:docs/decisions/D-063-the-first-pair-is-remote-so-cross-network-reach-comes.md`.
- Tailcat promises no API stability, so a breaking change has one place to land.
  `f288913:docs/decisions/D-062-tailcat-evaluated-for-phase-15-a-good-fit-adopted-behind-an.md`.
- A tailcat request passes the same signature check, guest list and verification as one
  over TCP, because it arrives at the same handler, which is what makes it true that a
  transport decides nothing. `cmd/cogmer/tailcat.go`, D-044 (sync requests are signed),
  D-045 (rooms are records with a guest list), D-054 (verification gates synchronization
  and injection).
- Constructing a tailcat client starts a userspace WireGuard stack and takes seconds, so a
  client is cached for each peer and the first contact has a timeout of its own.
  `cmd/cogmer/sync.go`, `cmd/cogmer/transport.go`.
- A dial to a port the packet filter does not serve times out with no reset, so the served
  port is set explicitly. `cmd/cogmer/tailcat.go`, `ServedTCPPorts`.

**Rejected.**
- *An SSH tunnel.* A person performs traversal by hand, which for a pair who pair daily is
  the product failing at its first step.

**Limits.** Any router permits outbound connections, which is all a relay needs, so a
working path with relay latency is the floor, and how often the path becomes direct is a
question about latency and not correctness. The measurements, the dependency count and the
defects found while running it are in
`f288913:docs/decisions/D-068-tailcat-is-the-cross-network-transport-behind-our-own-dialer.md`.
