# D-090 — Attribution is structured; the derived name is a field, not part of a string

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** From `docs/residual-concerns.md`: what should happen when a peer's
self-identified name collides with one already in the list. Tracing it found three
different name spaces with three different answers, and one genuine hole.

**The derived name** (`quiet-otter`) can collide and that is accepted — D-050 says
names may collide and identities do not, nothing keys on a name (D-017), and it takes
roughly 107 known peers before a collision among 8,280 combinations is more likely
than not.

**The local label** you assign is settled by D-074: `known_peers.name` is unique and
`explainNameTaken` says that a changed key is indistinguishable from somebody else's
key sent in their name.

**The self-asserted display name has no constraint, correctly** — it is the peer's
name for themselves and two colleagues really may both be David. That already
happened: the first two-peer run had both daemons assert the same display name
because both ran under the same OS user.

**The hole was structural rather than a collision.** The injected block built one
string:

```go
speaker := fmt.Sprintf("%s (%s%s)", e.UserDisplayName, PeerName(e.PeerID), mark)
```

Peer-chosen free text immediately before a parenthetical the model is meant to read
as the derived identity. A display name of `Alice (quiet-otter)` produced
`Alice (quiet-otter) (prudent-wagtail, unverified)`. It escapes nothing, forges no
turn and leaves the block intact — so `TestACraftedDisplayNameCannotForgeASpeaker`
passed, because it tests escaping. Escaping stops a value leaving its field. It does
nothing about a value imitating the field beside it.

**So the derived name became a field**, as `verified` already was, and two other
concatenations went with it: the verified state (a boolean) and the `Claude-` prefix
that marked an assistant turn (what `kind` already says). Nothing of ours is glued to
their text any more, and a value cannot occupy another field's position.

This is the same lesson as D-040 (fence the block so content cannot close it) and
D-081 (encode turns as JSON so a value cannot escape its string), applied one level
further in: **do not concatenate our facts with their claims.**

**Still open, and not fixed here:** the view renders the two names in separate
elements already, so it does not have this bug — but it gives them similar visual
weight. Somebody skimming a room is a different reader from a model parsing JSON,
and nothing has been measured about what they actually notice.

**Revisit when:** another field is added that a peer can influence.
