# D-019 — No network provider is required

**Date:** 2026-09-16 · **Status:** active

**Decision.** No network provider is part of a room's identity, its membership or
replication, and none is a prerequisite: cogmer works with no virtual private network
when peers can reach each other. A connection tries the same network, then a private
network provider, then internet peer-to-peer, then a relay, and which one connected
appears in no room's identity and no event.

**Support.**
- Tailscale's free plan is for personal use, so a team evaluating cogmer could read it
  as requiring a paid service. `https://tailscale.com/pricing`.
- What synchronization means is independent of the transport that carries it. §4
  (networking).

**Rejected.**
- *Tailscale as the foundation.* It ties adoption to an account and a second daemon for
  the simplest case.

**Revisit when** a transport is added whose peers cannot be found as candidates.
