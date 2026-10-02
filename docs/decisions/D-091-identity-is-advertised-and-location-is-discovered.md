# D-091 — Identity is advertised and location is discovered

**Date:** 2026-09-20 · **Status:** not built · **Areas:** transport, identity

**Decision.** An address that names a node, such as an overlay address, is advertised and
stored durably. A location, an IP address and port, enters only as a candidate found on the
network this machine is on now, with an expiry. A private range never appears in a pairing
string, an invitation or a durable record.

**Support.**
- Which address works is a property of the pair, the sender knows its own interfaces and
  nothing about where the receiver sits, and only the receiver can find out by attempting.
  The overlay does this inside one endpoint, a relay first and direct once it can. RFC 8445
  (ICE), https://www.rfc-editor.org/rfc/rfc8445.
- An advertisement outlives the fact it asserts. That is tolerable for what identifies a
  machine, which does not change when a laptop changes network, and not for an address,
  which is true of one position at one moment. D-103 (an address belongs to a peer, and is
  stored in one place).
- An overlay address holds three public keys and the node's home relay, a hostname with its
  addresses. The keys are durable, and the relay is a rendezvous that can go stale when the
  machine moves far enough, so the keys are durable and the relay is best effort.
  `f288913:docs/decisions/D-091-identity-is-advertised-location-is-discovered.md`.
- A private address is confidently wrong about a different machine, since `192.168.1.42` at a
  coffee shop belongs to somebody else's laptop and an attempt succeeds against a stranger.
  D-138 (local discovery locates a room, and never admits anyone).

**Rejected.**
- *Advertising this machine's LAN address.* It is a claim about a network the machine may have
  left, and on another network the same address is a different machine, so the claim
  is wrong about somebody else.

**Limits.** A remembered address that worked is a snapshot too, and is tried first and
discarded on failure. D-183 (a receiver orders candidate addresses by class and never by a
configured sort).

**Revisit when** local discovery is built, which gives a location candidate a legitimate way
in. D-138 (local discovery locates a room, and never admits anyone).
