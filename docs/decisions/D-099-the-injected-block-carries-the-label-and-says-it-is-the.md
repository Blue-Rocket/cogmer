# D-099 — The injected block carries the label, and says it is the name to use

**Date:** 2026-09-20 · **Status:** active (implemented) · **Completes** D-094

**Context.** D-094 put the label in the view and deferred the same field in the
injected block, because `FormatTeamContext` took `verified func(string) bool` across
twenty-two call sites and adding a second function was a signature decision rather
than a field addition.

**The gap was an inconsistency, not an omission.** The view said "alice" and the
model said `Ec2-user` or a word pair, in the same conversation, about the same
person.

**`PeerFacts` groups the two lookups** — `IsVerified` and `Label` — because they are
one idea: what the receiving side knows, as against what the peer asserts. That is
the distinction the block is built on (D-090), and naming it means a third such fact
costs an implementation and nothing at any call site. `Membership` satisfies it
directly. Six callers passed a function and now pass the membership or a `fixedFacts`
stating the same thing about every peer; the rest passed nil and were untouched.

**The framing says what the field is for.** A field the model is given and told
nothing about is a puzzle, so the block states that the label is the name its own
user calls that person by, and that it is preferred over both the self-asserted
`speaker` and the derived `peerName`.

**All three travel.** The label is preferred, not substituted: `speaker` is what the
peer claims and `peerName` is what can be checked, and dropping either to save a
field would remove the thing the preference is safe because of.

**An absent label is an absent field**, not an empty one — `omitempty` — because a
blank value invites a model to wonder what it means.

A test asserts the label reaches the block, that the other two names survive
alongside it, that the framing explains it, and that nothing is emitted when there
is no label. The last check initially scanned the whole block and matched the
framing's own use of the word, which is a reminder that a test over rendered text
should read the payload.

**Revisit when:** a fourth peer fact appears, which should now be an implementation
change only.
