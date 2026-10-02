# D-018 — Identity and reachability are separate

**Date:** 2026-09-16 · **Status:** active · **Areas:** transport, identity

**Decision.** A machine's label is for attribution and display, and nothing routes by
it. An endpoint is where a peer can be reached. It is opaque to the protocol, and the
daemon discovers it from the transport in use rather than deriving it from the
hostname. An endpoint in an invitation is a hint needed only until the joining peer
learns the room's members, and an invitation may carry several.

**Support.**
- A machine answers to several names, and the one that resolves may be an address no
  remote peer can reach: on one machine `os.Hostname()` returned `macbookpro.lan`,
  `scutil --get LocalHostName` returned `pushover`, and the name that resolved was a
  LAN address. `84a0751:docs/decisions.md`.
- An endpoint need be correct only once, because a peer that has joined learns how to
  reach the other members. §12 (forming a room).

**Rejected.**
- *Resolving the machine's label as a hostname.* The first machine tried disproved it.

**Revisit when** a transport is added whose peers have no addressable endpoint.
