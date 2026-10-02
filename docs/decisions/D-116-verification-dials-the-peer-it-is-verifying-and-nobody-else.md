# D-116 — Verification dials the peer it is verifying, and nobody else

**Date:** 2026-09-21 · **Status:** active · **Areas:** pairing, transport

**Decision.** `verifyTargets(peerID)` returns the address recorded for that peer, plus the
addresses in `COGMER_PEERS`, and nothing else.

**Support.**
- The dial is pinned to the key being verified, and every other recorded address belongs to a
  different key and is refused at the handshake, so a sweep spends a dial and a timeout per peer
  on candidates excluded by construction. D-103 (an address belongs to a peer and is stored in one
  place), `cmd/cogmer/verify.go`, `verifyTargets`.
- Pairing is two people on a call comparing two words, and a sweep made it visible to everybody
  else on the list every time, as a connection from a recognizable address at a recognizable
  moment. `cmd/cogmer/verify_test.go`, `TestVerifyDialsOnlyTheNamedPeer`.
- An address that answers with an unexpected key while verifying a named peer is what substitution
  looks like, and the user is told and the attempt is not retried, which cannot be said while a
  wrong key is the ordinary outcome of most dials. §4 (networking), `cmd/cogmer/peertls.go`,
  `pinnedVerifier`.
- `COGMER_PEERS` addresses name no peer, since they are configured for the machine, so one of them
  may be the peer wanted and only dialing can tell, where every other source attributes an address
  to somebody. `cmd/cogmer/verify.go`, `verifyTargets`.
- A fallback sweep cannot succeed. Only one side needs a usable address, so a peer that moved
  reaches this one by dialing inward, and where both sides are stale the addresses a sweep would
  try belong to other peers and are refused. D-092 (pairing proceeds when the string carries no
  address, because only one side needs to dial).

**Rejected.**
- *A fallback sweep across every known address.* It leaks the pairing and cannot succeed.

**Revisit when** an address source appears that attributes an address to nobody, as `COGMER_PEERS`
does, or a peer's recorded address can be refreshed by something other than pairing.
