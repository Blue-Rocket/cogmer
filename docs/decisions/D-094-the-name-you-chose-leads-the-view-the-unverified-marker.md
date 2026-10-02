# D-094 — The name you chose leads the view; the unverified marker becomes a fact

**Date:** 2026-09-20 · **Status:** active (implemented, with one half deliberately deferred)

**Context.** Raised that a derived name like `quiet-otter`, however charming, has no
durable connection to a person across weeks of inactivity. Checking where each name
actually goes found that the one name which is both memorable and bound to a key
reached nobody.

| name | chosen by | bound to one key | reached the view |
|---|---|---|---|
| display name | the peer | no, it is a claim | yes, leading |
| derived name | nobody, computed from the key | yes | yes, secondary |
| the label you chose | you, at pairing | yes, D-074 makes it unique | **no** |

`known_peers.name` was read only by the CLI `peers` listing and `resolvePeer`. The
view led with the peer's own claim and anchored on a word pair nobody picked.

**The label now leads**, with the derived name kept beside it: it is still what a key
change surfaces on (D-074), and it still separates two peers claiming one display
name. A label equal to the derived name is treated as absent, because a placeholder
offered as a choice would be a lie.

**The derived name is doing a job it was never given.** Its actual jobs are
momentary — disambiguate a collision, anchor an alarm. Neither asks anybody to
remember it. D-021 (peer names are derived from the identity) calls it "a mnemonic for an identity already verified",
and a mnemonic nobody chose, for a key they will never look at, is not much of one.

**And the unverified marker was on every remote turn.** `uiEvent` had no verified
field, so the view could not ask; a comment written before D-054 explained that every
remote peer was unverified. D-054 now refuses an unverified peer's events outright,
so everything displayed is verified by construction and the marker was permanently
on. Same failure as D-089: a warning that is always on trains somebody to ignore the
one that matters. It is a fact now, and seeing it means a filter failed.

**Deferred: carrying the label into the injected block.** The model should say
"Alice" where the person says "Alice", and the field belongs there for the same
reason it belongs in the view. `FormatTeamContext` takes `verified func(string) bool`
and has twenty-two call sites, so adding a lookup is a signature decision rather than
a field addition, and it deserves its own entry rather than being smuggled in here.

**Not done, and blocked: defaulting the label to the name the peer chose.** Proposed
that the pairing string carry it, so `/peer-pair <string>` could offer a sensible
default. **A peer does not choose their name.** `UserDisplayName` is `$USER` with the
first letter capitalised, computed once at identity creation, with no command, flag
or environment override to change it — which is why two daemons asserted the same one
during the first two-peer run. Shipping that in a pairing string would be worse than
the derived name: `quiet-otter` is honest about being machine-made, while `Ec2-user`
or `Person` looks like a claim about a person. Choosing a name has to exist before
a pairing string can carry one, and the moment for it is the half of `/peer-pair`
that already explains what to send.

**Revisit when:** a peer can choose their own name, at which point the pairing string
should carry it and the label should default to it.
