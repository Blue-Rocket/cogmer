# D-187 — The overlay's relay region is chosen once and stored with the node key

**Date:** 2026-09-21 · **Status:** active · **Areas:** transport

**Decision.** The relay region the overlay chooses is stored beside the node key and reused at
every start.

**Support.**
- The published address names a rendezvous, where the node can be found so that an introduction
  can happen. Left unpinned the library re-chooses by latency at every start, with a random
  fallback when the probe fails, so a start on another network could publish a different
  address and strand everybody holding the old one. `cmd/cogmer/tailcat.go`,
  `tailcatRegionFile`.
- A machine that travels keeps an address its colleagues can use, at the cost of a relay that
  may not be the nearest, and the relay carries an introduction after which the path upgrades
  to direct, so the penalty is a slower handshake and not a slower session.
  `cmd/cogmer/tailcat.go`.
- Disabling the pre-shared key made two starts identical on one network, and it does not
  follow that a start on another network picks the same relay. D-104 (the overlay address
  holds no secret and is the same at every start).

**Rejected.**
- *Choosing a region by latency at every start.* It is the library's default, and the address
  then belongs to the last startup and not to the identity.
