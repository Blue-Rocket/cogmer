# D-192 — The public relay is an accepted dependency, and `plugin/README.md` discloses it

**Date:** 2026-10-02 · **Status:** active · **Areas:** transport, trust

**Decision.** The overlay is on by default and introduces two peers through the free relays that
tailcat's default relay map lists. `plugin/README.md` tells a user what the relay sees and that
`COGMER_TAILCAT=off` turns the overlay off.

**Support.**
- D-115 (a centralized component must trace to a disclosed tradeoff that benefits the person)
  asks three questions. The relay traces to D-068 (tailcat is the cross-network transport). It
  benefits the user, since two people behind separate routers cannot otherwise connect, and the
  first pair is remote. It is disclosed in `plugin/README.md`.
  `f288913:docs/decisions/D-063-the-first-pair-is-remote-so-cross-network-reach-comes.md`.
- The library's documentation offers free, rate-limited relays to anyone, and names
  `https://tailcat.dev/derpmap.json` as the default map, so using them with no account and no
  configuration is the use the library describes. tailcat v0.6.0, `README.md`.
- The relay carries WireGuard packets between the two machines and cannot read them. It sees
  both machines' keys and network addresses, when they connect and how much passes. With the
  pre-shared key disabled (D-104), an operator who sees both keys can open a tunnel to the
  daemon, and what stops it there is the pinned TLS key, the signed request, the guest list and
  verification, which a tunnel connection meets as a TCP one does. D-157 (the overlay
  listener admits any dialer), D-044 (sync requests are signed), D-045 (rooms are records with a
  guest list), D-054 (verification gates synchronization and injection).
- §4 (networking) makes no provider a prerequisite. With `COGMER_TAILCAT=off` the daemon
  advertises its TCP address, or `COGMER_PEER_ENDPOINT`, and contacts no relay.
  `cmd/cogmer/tailcat.go`, `tailcatEnabled`, and `cmd/cogmer/main.go`, the daemon's start.
- A relay outage ends collaboration across networks and affects nothing else in a session, which
  is the whole of the loss §3.1 (first, do no harm) allows.

**Rejected.**
- *The overlay off until the user asks for it.* The first pair is remote, so the default product
  would fail at its first step.

**Limits.** The documentation promises no availability and says the relays are rate-limited. Two
peers on one network also meet through the relay, because a pairing string carries one endpoint.
D-104 gives up the only quantum-resistant element, so a recording of relayed traffic is not
protected against a later quantum computer.

**Revisit when** the relay list moves from `tailcat.dev`, the free service ends or adds terms, or
the pairing string carries a direct endpoint beside the overlay one.
