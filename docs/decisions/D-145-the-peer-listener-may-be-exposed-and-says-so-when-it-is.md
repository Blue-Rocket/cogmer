# D-145 — The peer listener may be exposed, and says so when it is

**Date:** 2026-09-16 · **Status:** active · **Areas:** transport, trust

**Decision.** The peer listener defaults to loopback. It may be bound where other machines
can reach it, and the daemon then notes at start that it is reachable.

**Support.**
- Exposing the peer listener is how a second machine reaches this one. §4 (networking).
- Every peer connection is TLS pinned to a key this machine knows, so a stranger
  completes no handshake. D-101 (peer connections are TLS pinned to a verified key).

**Rejected.**
- *Exposing the peer listener by default.* Nothing is reachable until someone decides it
  should be.
- *Refusing to expose it until peers were authenticated.* It would have meant building
  identity before ever running across a network, which is what identity is for.

**Revisit when** exposing the peer listener is found to reveal something a pinned
handshake does not protect.
