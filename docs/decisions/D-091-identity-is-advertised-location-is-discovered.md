# D-091 — Identity is advertised; location is discovered

**Date:** 2026-09-20 · **Status:** decided, not implemented · **Supersedes** part of D-089

**Context.** Two questions, a few hours apart. How can a peer's daemon know which
address suits the network topology between two particular machines? And does a
machine traveling between networks change the answer? It does, and it changes the
shape of the answer rather than a detail of it.

**The sender cannot know which address works, and should not try.** Topology is a
property of the *pair*. The sender knows its own interfaces and nothing about where
the receiver sits, so only the receiver can find out, by attempting. That is what
ICE and happy-eyeballs do, and what an overlay already does inside a single
endpoint: a relay first, direct once it can. Which is why the overlay path survives
NAT and the plain TCP path does not — the negotiation exists at one layer and is
absent at the other.

**Addresses are of two kinds, and only one of them can be advertised.**

| | names | survives a move | where it belongs |
|---|---|---|---|
| identity-shaped (`tc://…`) | a node, and where to find it | mostly — see below | the durable peer record |
| location-shaped (an IP and port) | a place | no | a candidate, with an expiry |

An advertisement outlives the fact it asserts. That is tolerable for something that
identifies a machine, which does not change when a laptop moves from a desk to a
hotel, and not tolerable for an address, which is true of one network position at
one moment.

**An overlay address is not purely an identity, and the difference matters.**
Decoding one gives a small CBOR map: three 32-byte public keys, and the node's home
relay — a hostname and its v4 and v6 addresses. The keys are the durable part. The
relay is not an identity but a **rendezvous**: where this node can currently be
found so that an introduction can happen, after which the path upgrades to direct
if the two ends can reach each other.

A rendezvous can go stale. The overlay picks the lowest-latency relay, and that
choice changes when a machine moves far enough — a different continent, sometimes a
different country. The keys in the advertised value stay correct and the relay named
beside them may no longer be the one that node is attached to.

In practice it usually still works, because the relays mesh and one that receives a
connection for a node attached elsewhere forwards it. That is a property of somebody
else's network rather than a guarantee this design holds, which is one of the
reasons the dependency on it is a precondition of Phase 13 rather than a settled
matter. So: treat the keys as durable and the rendezvous as best-effort, and do not
write anything that assumes an overlay address is timeless.

**A private address is not merely stale; it can be confidently wrong about a
different machine.** `192.168.1.42` at a coffee shop belongs to somebody else's
laptop, so attempting it does not fail — it succeeds against a stranger, and then
has to be unwound at a higher layer. Private ranges therefore never appear in a
pairing string, an invitation, or a durable record. They enter only as facts
discovered on the network we are on now (D-138), where discovery states where a
peer *is* rather than where a peer *was*.

This entry first argued that the cost was disclosure — that a signed sync request
(D-044) would deliver our identifier to whoever now holds the address, on every
poll. That was true when it was written and TLS removed it (D-101): the connection
is abandoned as soon as the far side presents a key this machine does not know,
which in TLS 1.3 is before this machine has sent a certificate of its own. Measured
rather than assumed. What a stranger at a reassigned address learns is that
something attempted a connection, and nothing about who. The argument from
correctness stands unchanged and is sufficient on its own.

**A remembered winner is a snapshot too.** Remembering which candidate worked is
right, and it must carry the time it worked, be tried first, and be discarded on
failure rather than kept. It is a hint, not a record.

**On ordering, asked directly: no, a daemon should not have a configurable sort.**

The receiver holds the information, so any order the sender expresses is a
preference rather than knowledge — and one the receiver would be honoring on the
word of the peer whose address it is.

Order by *class* instead, which is derivable rather than configured — same-host,
same-network, public, relayed — and try the first few concurrently, so a dead
candidate costs a round trip rather than a timeout. A learned winner then dominates
any static order after the first success, which makes a configured one dead weight
that is wrong in exactly the cases it was added for.

Every real requirement that arrives dressed as ordering is a **filter**: "never use
a relay" for a site that will not route through third-party infrastructure, or
"never expose a direct address". Those need *never*, and a preference order cannot
express never. Add filters if such a requirement appears; do not add a sort in
anticipation of one.

**D-089 rejected advertising this machine's LAN address, and was right.** Its stated
reason — that an address working for some colleagues and not others is worse than
one that visibly works for none — was the weaker argument, and an earlier form of
this entry overturned it on those grounds. The reason that holds is that a LAN
address is a claim about a network the machine may no longer be on, and the claim is
wrong about somebody else rather than merely wrong.

**None of this is implemented.** A peer is advertised at one address, that address
is recorded once, and nothing expires.

**Revisit when:** local discovery lands (D-138, local discovery), which is
what gives a location candidate a legitimate way in.
