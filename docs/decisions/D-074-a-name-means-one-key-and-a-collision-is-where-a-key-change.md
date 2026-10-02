# D-074 — A name means one key, and a collision is where a key change surfaces

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Tracing a peer identity's lifecycle. It is created once from
`identity.key`, never rotates, never expires, and the key is authoritative — if
`identity.json` disagrees it is corrected, because an identifier that is not this
key names an identity nothing can verify.

`pair` ends by telling a person:

> if this key changes, that is an alarm rather than a new first meeting.

**Nothing implemented that alarm**, and it is not obvious that anything could. A
peerId **is** a key (D-042), so "the same person with a new key" is not expressible:
to this machine that is a peer it has never seen, and it is refused as a stranger.
Correct, and it is not an alarm — it never mentions the person whose name is on the
old key.

**There is exactly one moment where the two can be connected**: when somebody
records the new key under a name they already use. That is also precisely what a
substitution looks like from the host's side — Mallory sends a pairing string in
Alice's name, and David types `pair <key> alice`.

**And it passed silently.** `known_peers.peer_id` is the primary key; `name` had no
constraint. A second row was created, `peers` listed two alices, and `resolvePeer`
returned whichever `KnownPeers()` yielded first — so `invite alice` could admit
either, with nothing said. That is the failure the entire verification apparatus
exists to prevent, reachable by typing a name twice.

**Decision.** A name may belong to one key. `Allow` refuses a name already held by a
different key and returns a typed `NameTakenError` carrying both, because the
explanation must show what changed.

The explanation says what the collision means rather than treating it as a naming
mistake: a changed key is indistinguishable from somebody else's key sent in their
name, so check on a call before recording it. And it gives the route — `forget` the
old one first, which since D-073 discards its admissions too, so the person is
invited again deliberately rather than inheriting rooms.

**`resolvePeer` refuses an ambiguous name rather than choosing.** Recording two keys
under one name is now prevented, but a database written before this could hold one,
and picking between them would admit a peer nobody named. Same shape as D-161 (an ambiguous room name is reported and never guessed): report
the ambiguity, name both, say what it probably means.

**What this does not do.** It does not detect a key change by itself — only a key
change that somebody tries to file under a familiar name. A peer that simply
presents a new key is still an unknown peer, refused without ceremony. That is the
honest limit of self-certifying identifiers, and it is why the alarm lives at the
moment of recording rather than at the moment of contact.

**Revisit when** identity rotation is wanted. There is none today: an identity is
created once and lives until `identity.key` is lost, at which point the peer is a
stranger to everyone and must pair again. Rotation would need a way for a new key to
be vouched for by the old one, which is a real design and not a small one.
