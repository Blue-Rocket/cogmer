# D-068 — Tailcat is the cross-network transport, behind our own dialer

**Date:** 2026-09-18 · **Status:** active (implemented); **NAT-to-NAT still unproven**

**Context.** Phase 15. Two people working from home are behind two routers and
neither can bind an address the other can reach (D-063, as
`f288913:docs/decisions/D-063-the-first-pair-is-remote-so-cross-network-reach-comes.md`
held it). Phases 2 and 5 substituted
an SSH tunnel, which is a person performing NAT traversal by hand.

**What was verified, stated precisely because the first version of this was
overstated.** A spike connected a home Mac to the droplet in 291 ms with nothing
configured but one address — and **the droplet has a public IP, so that was
NAT-to-public, which is an ordinary outbound connection.** It proved tailcat builds,
runs and handshakes; it proved nothing about NAT traversal.

The full integration was then run between the two machines with **no tunnel**:
pairing (the two-word exchange over the tunnel), room creation, invitation, joining,
capture, and convergence on both sides. DERP carried the introduction and the path
upgraded to direct UDP — `now using <droplet>:34746`, the address redacted: it is a
personally-operated machine and this repository is public. Zero unreachable errors.

**Still not tested: neither side able to accept inbound.** That is the case that
exists, and a stateful firewall on a public IP is a poor imitation of a consumer
router — no symmetric mapping, no CGNAT, no ALGs. It will be tested with the real
peer, on real routers, and until then the claim here is that our integration works
between two machines, not that NAT traversal does.

**What is nearly certain regardless.** Any router permits outbound connections,
which is all a relay needs, so the floor is a working path with relay latency.
The uncertainty is how often the upgrade to direct succeeds, which is a question
about latency rather than correctness, and for turns of about a kilobyte is unlikely
to matter.

**Two defects found by running it, neither visible from reading it.**

**A client is not a connection.** `clientFor` built a fresh `tailcat.Client` per
dial, and constructing one starts a userspace WireGuard stack, probes the network,
chooses a DERP region and handshakes — seconds of work. At a one-second poll every
round paid first-contact cost and none ever finished: the first attempt was still
handshaking when the next began, and every sync timed out against a peer that was
plainly reachable. Clients are now cached per peer for the life of the daemon, and
the sync timeout is 30s to bound first contact rather than the ordinary round.

**A timeout is the signature of the packet filter.** Connections reached netstack
and `OnTCP` was never called. The answer was in tailcat's source: *"Unlike OnTCP's
nil-handler response, packets dropped by the filter get no RST; a client dialing a
filtered port times out."* `ServedTCPPorts` is now set explicitly. Two gates that
fail differently — the filter drops silently, `OnTCP` returning nil resets promptly
— and knowing which one is refusing is the difference between a diagnosis and a
week.

**A third defect, found and unrelated to tailcat.** The hook payload field is
`session_id`, and a hand-written test harness had been sending `sessionId`. Under
the old binding rule an empty session id was silently bound to the current room, so
it appeared to work — **including in the Phase 5 two-machine run, whose events were
all captured under an empty origin session.** That run's convergence finding stands,
because convergence is what it measured; its session attribution was degenerate and
injection would not have worked. D-064's explicit binding is what surfaced it.

**Costs, measured.** 566 dependencies against 40 before; 21 MB stripped against
11 MB. Both are prices D-063, as
`f288913:docs/decisions/D-063-the-first-pair-is-remote-so-cross-network-reach-comes.md` held it, already accepted, and both are worth restating at the
point they were actually paid.

**Confined to one file.** Tailcat promises no API stability, so `tailcat.go` holds
all of it behind the `Dialer` interface and a `net.Listener`. One set of routes
serves both transports, which is what makes "a transport decides nothing" true
rather than merely intended: a tailcat request passes the same signature check, the
same guest list and the same verification as a TCP one, because it arrives at the
same handler.

**Revisit when** the real peer test runs. What to watch: whether the path goes
direct or stays on the relay, and what a relayed round trip costs a turn.
