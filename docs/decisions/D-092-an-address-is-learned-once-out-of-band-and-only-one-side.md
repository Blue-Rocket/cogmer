# D-092 — An address is learned once, out of band, and only one side needs one

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asked when and how an address is learned for pairing. Tracing it found
the answer is narrower than expected, and that one consequence of it was being
actively refused.

**When: once, at `pair` time. How: carried by a person.** The address is the `@…`
tail of the pairing string a colleague sends by whatever channel they like.
`parsePairing` splits on the last `@`, `SetPeerEndpoint` stores it, and
`syncTargets` offers it to `RunVerification` later. There is no other path —
`RemoteAddr` is never read anywhere in the codebase, so nothing is learned from a
connection, and there is no discovery to learn it from (D-138's local discovery
path is not built).

**Only one side needs a usable address.** `RunVerification` checks for an inbound
exchange before it dials: if the other side ran the whole exchange against this
session, the revealed nonce is already here and the same two words come out. So an
asymmetric arrangement works, and it is the one a colleague behind a NAT that cannot
be traversed actually needs — they can be the side that publishes nothing.

**And `pair` refused to allow it.** A pairing string with no address printed "there
is nowhere to reach them yet" and returned before starting a session, so the inbound
exchange had nothing to arrive at. A working arrangement was made to look broken, and
the person was sent to fetch an address they did not need. It now says which of them
has to dial and starts anyway.

Asserted by `TestOnlyOneSideNeedsAnAddress`, which gives one daemon no addresses at
all and requires both to reach the same words.

**Seven more instruction sites were still naming a bare `cogmer`** (D-085),
including the one in this path. Fixed, and the ones about pairing now point at
`/peer-pair` rather than at `verify`, since the ceremony is in the view (D-088).
Operator surfaces — the daemon log, `doctor`'s stderr, the generated behaviours
header — keep the short form, because you are already at a prompt when you read them.

**Revisit when:** D-091 lands, at which point a pairing string carries a set and
"no address" becomes "no address that worked", which is a different message.
