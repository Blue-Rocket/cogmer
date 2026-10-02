# D-093 — A two-interaction flow must choose what an unfinished second one means

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asked whether the local label should be a required argument to pairing,
after observing that a derived name like `quiet-otter` has no durable connection to a
person across weeks of inactivity. The answer turned on a general principle stated
during the discussion: anything that happens in two user interactions has to choose
what happens if the second is not completed.

**Optional did not mean unlabelled.** `Allow` filled an empty name with the derived
one, so somebody who never thought about a name got exactly the label that stops
meaning anything. The default was the failure mode.

**Asking after the words matched was worse, and was the first plan.** It creates a
second completion point, and both answers to an unfinished one are bad: default the
label to the derived name, which is the thing being escaped, or withhold the
verification until named, which holds the security-meaningful act hostage to a
convenience field. The flow should not have had two completion points.

**Collecting is not asserting.** The objection to taking the name up front was that
the label claims "this key is Alice", which is unjustified until the comparison
succeeds. That is true of *writing* it, not of *asking* for it. The name is now taken
with the pairing string, held in the pending pairing, and written only on a match.

**The three endings are now distinct, and were not before:**

| second interaction | what remains |
|---|---|
| never happened | the key, unverified, labelled with the derived placeholder — resumable |
| words matched | the key, verified, labelled with the chosen name |
| words differed | nothing, if this pairing created the row |

The last was the substantive bug. Abandoning and mismatching left identical state —
a recorded unverified peer wearing a colleague's name — so a **detected interception**
left the attacker's key on the list under the name meant for the real person. Not a
breach, since D-054 refuses everything from an unverified peer, but D-074's
uniqueness then blocked pairing with the real Alice until somebody worked out they
had to `forget` first. The attack you correctly detected became a second,
unrelated-looking problem at the worst moment.

**A mismatch removes only what the pairing created.** Re-verifying a colleague of two
years and seeing different words is an alarm about an existing relationship, not a
reason to discard it and every admission it holds (D-073).

**The name clash is checked before the ceremony, not after.** `NameFree` is split out
of `Allow` for this. Discovering the clash once the words had matched would leave a
verified peer and an unusable name — one more unfinished thing, which is the shape
this entry exists to avoid.

**Both surfaces now run through one pairing record.** The terminal path mints a
pairing and confirms against its id exactly as the page does, so what a match, a
mismatch and an abandonment mean is decided once rather than twice.

**A hazard found while moving the write:** `SetPeerEndpoint` is an `UPDATE … WHERE
peer_id = ?`, which affects nothing and returns nil when the row does not exist yet.
Recording the peer daemon-side moved the row's creation after the CLI's call to it,
which would have silently lost the only address anybody had. The endpoint is now
recorded with the row.

**How somebody learns the requirement**, since a required argument nobody is told
about is just a failure: the slash command's `argument-hint` names both, running
`/peer-pair` with nothing explains both halves of pairing, and giving a string
without a name produces the command echoed back with the string already in it and
one word missing. The command doc tells the model to ask rather than to pick one.

**Revisit when:** another flow gains a second interaction — the three endings should
be enumerated for it too, and named, rather than left to whatever the code does.
