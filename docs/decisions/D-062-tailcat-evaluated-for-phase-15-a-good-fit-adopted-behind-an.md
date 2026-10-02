# D-062 — Tailcat evaluated for Phase 15: a good fit, adopted behind an interface if at all

**Date:** 2026-09-18 · **Status:** **adopted** — see D-068 for what was built and
what remains unproven. Was: investigated, not yet adopted, but the likely answer
rather than a contingency — D-063 established that the first pair are
remote, so this is the only case that matters. Its costs are prices to be paid, not
risks to be weighed.

**Context.** Phase 15 needs two peers behind different NATs to find a direct path.
Every cross-network run so far — Phase 2's and Phase 5's — used an SSH tunnel,
which is a person performing NAT traversal by hand and is not something a colleague
can be asked to do. Tailscale published **tailcat** in August 2026: Tailscale's data
plane (WireGuard, magicsock, DERP) with **no control plane, no account, no tailnet**.

**What it is.** A Go package and CLI. A server generates a keypair, picks a DERP
relay, and produces a `tc…` address carrying its public key, a path-discovery key
and DERP bootstrap info. A client dials that address through DERP; both sides then
attempt direct UDP, with DERP remaining as fallback. Userspace throughout — gVisor
netstack and wireguard-go — so no root, no TUN device, no routing-table changes.

**Why it fits this design unusually well.** The API is `net.Conn`-shaped:
`Server.OnTCP(port) func(net.Conn)` and `Client.DialTCPPort(ctx, port)`. Sync is
plain HTTP over TCP, so the integration is `http.Serve` over one and a
`DialContext` over the other. There is no protocol to redesign.

And it lands exactly where D-019 said a transport must: **under everything.** A
tailcat connection carries bytes and decides nothing. A sync request is still signed
(D-044), still refused unless its peer is a guest (D-045), and still refused unless
a person has verified that peer (D-054). D-019 named Tailscale explicitly as "one
provider among several, never a prerequisite", and this is the version of Tailscale
that can actually be one.

**Two distinctions to hold, because both look like contradictions and are not.**

A `tc…` address is something a person can hold, and holding it gets you a
connection. D-026 says nothing a person can hold admits them. Both are true: the
address reaches the **door**, and the guest list and the two-word comparison decide
whether anyone comes in. An intercepted address is worth what an intercepted IP
address is worth today.

Tailcat has its own WireGuard keypair. That is a **transport** identity and must
never be confused with a `peerId`, which is an Ed25519 key (D-042) and is what
signs events and is what a person verifies. Two keys, two jobs.

**Do not use `AllowedClients`.** Tailcat can restrict connections by peer public
key, which would be a third allowlist, keyed on a different key type, needing to
agree with `known_peers` and `room_guests`. Two lists that can disagree is precisely
the defect behind the asymmetric guest list (D-159, an invitation carries the room and
the inviter). One authority.

**Measured, not assumed.** Built against a trivial program, `CGO_ENABLED=0`, all
four targets:

| | tailcat hello-world | `cogmer` today |
|---|---|---|
| darwin/arm64 | 23.3 MB | 16.3 MB |
| linux/amd64 | 25.0 MB | |
| linux/arm64 | 23.3 MB | |
| windows/amd64 | 24.5 MB | |

Pure Go on every target, so cross-compilation survives. The cost is size and supply
chain: **526 dependencies against our current one**, and a binary likely to roughly
double. For something a colleague installs, both are real.

**Three risks to weigh at Phase 15, not now.**

- **No stability promise.** "The Go API, CLI flags and output, and wire format may
  all change." Load-bearing code behind an unstable API is why it should sit behind
  a narrow interface of our own — a Dial and a Listen — which costs almost nothing
  to write now and a great deal to retrofit.
- **DERP is a rendezvous dependency.** Direct paths are peer-to-peer once
  established, but two peers behind NAT cannot *meet* without reaching a relay. That
  is not a control plane and it is not local-first either. Self-hosting DERP is
  possible and is a server, which is the thing this project avoids. The honest
  framing: the cross-network case has always needed something in the middle, and
  DERP is a more honest version of the SSH tunnel we have been using.
- **Public relays are rate-limited with no SLA.** Room traffic is text and small, so
  throughput is unlikely to bind; availability might.

**What would make it the answer** was whether Phase 13's first outside user is
remote. **They are** (D-063): colleagues at one company, both working from home,
pairing daily. So Phase 15 is a prerequisite rather than a detour, and the
alternatives are all worse here — a company VPN cannot be assumed, a public bind is
not available from a home connection, and an SSH tunnel is a person performing NAT
traversal by hand twice a day.

The DERP objection also weakens for this pair specifically, which matters because it
was the strongest one. The relay is for rendezvous and can be self-hosted; a droplet
already exists and has carried both cross-machine runs. A relay this pair controls,
used to meet and as fallback with the working path still direct, is a different
thing from depending on a third party — and from the central server §3.3 rejects.

**Revisit when** Phase 15 begins, or if tailcat reaches a stable API. Re-measure the
binary then: a figure from a hello-world is a floor, not the number that matters.
