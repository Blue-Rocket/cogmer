# D-116 — Verification dials the peer it is verifying, and nobody else

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** `handleVerifyStart` passed `d.syncTargets()` to `RunVerification`, so
verifying one colleague opened a connection to every address this machine knew.

**Most of those addresses could not answer.** `clientConfig(peerID)` pins the dial
to the key being verified, and since D-103 (one address store) put every address in
`known_peers`, each other entry belongs to a different key and is refused at the
handshake. Room peers are a join against the same table, so they add nothing. The
sweep spent a dial and a timeout per peer on candidates excluded by construction.

**The leak is what makes it worth a decision.** Pairing is two people on a call
comparing two words, and this made it visible to everybody else on the list, every
time, as a connection from a recognizable address at a recognizable moment. §4
already ensures no identity goes with it — the connection is abandoned before this
machine presents anything of its own — but timing and an address are still more
than Bob needs to know about Alice.

**It also cost §4 its alarm.** §4 holds that an address answering with an
unexpected key during synchronization is routine, while the same event while
verifying a named peer "is what substitution looks like from the inside, and it is
worth saying so to the person rather than retrying quietly." That could not be said
while a wrong key was the *ordinary* outcome of most dials in a verification.
Against one address recorded for one peer it is the exception §4 describes, and the
machinery already exists: `pinnedVerifier` reports "expected X and reached Y: the
address answers for a different key", `RunVerification` returns it, and
`handleVerifyStart` puts it in front of the person. Nothing needed writing — the
sweep was what made the sentence false.

**Decision.** `verifyTargets(peerID)` returns the address recorded for that peer,
plus `COGMER_PEERS`, and nothing else.

`COGMER_PEERS` stays because those addresses **name no peer**: they are
configured for this machine, so one of them may be the peer wanted and only dialling
can tell. Every other source attributes an address to somebody, and somebody else's
address is never a place to look for this one.

**Why there is no fallback sweep.** A stale address was the reason to keep one, and
it is not a reason. Only one side needs a usable address, so a peer that moved
reaches this one by dialling inward while this side waits on the inbound check.
Where both sides are stale a sweep does not help either, because the addresses it
would try belong to other peers and are refused. A fallback that cannot succeed is
a fallback in name.

**Test.** `TestVerifyDialsOnlyTheNamedPeer` asserts the **absence** of the other
peer's address rather than the presence of the right one — a test checking only for
the right address would pass unchanged if the sweep returned. Confirmed to fail
when the sweep is restored.

**Revisit when** an address source appears that attributes an address to nobody, as
`COGMER_PEERS` does, or when a peer's recorded address can be refreshed by
something other than pairing — which would make the recorded address likelier still
and the case for anything else weaker.
