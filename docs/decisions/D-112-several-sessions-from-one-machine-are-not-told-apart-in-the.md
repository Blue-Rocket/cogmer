# D-112 — Several sessions from one machine are not told apart in the block

**Date:** 2026-09-21 · **Status:** active (considered and not built)

**Context.** Nothing stops one machine having two live sessions in a room:
`session_rooms` is keyed on the session because sessions are members (§22), while
`room_guests` is keyed on `(room_id, peer_id)` because peers are admitted. They are
different questions and the schema answers them separately. `UndeliveredFor`
excludes `origin_session_id`, not the peer, so the two sessions correctly see each
other's turns.

But identity is per machine, so both carry the same `peerId`, the same derived
`peerName` and the same `label`. The injected block's JSON has `speaker`,
`peerName` and `label` and no session field, so a reader receives two threads
interleaved under one name. `originSessionId` is on the event and unsurfaced, so
the material for a discriminator is already there.

**Considered: a per-block discriminator derived from `originSessionId`**, stable
within a block and not the raw identifier, so threads separate without naming the
agent (D-043).

**Rejected, because it distinguishes without informing.** It reports process
topology, not work. It says turn 7 came from a different session than turn 6 and
nothing about whether that matters — two halves of one feature, or one continuing
after a crash, or two unrelated things all render the same. The reader cannot act
on it.

It is also worse than silence, because it invites an inference it cannot support:
two marked streams read as two topics, or as two people. That is the false
affordance this project already refuses in the view, appearing as a data field
rather than as a control.

**Consistent with how identity is displayed elsewhere.** The view leaves the derived
name off your own turns, where it identifies nothing you did not know, since it
exists to identify other peers (D-021, peer names are derived from the identity), and
D-099 prefers the label because a word pair means nothing to a person weeks later.
A session discriminator is the same kind of thing: an identifier the machine can
derive and the reader cannot use.

**How narrow the problem actually is.** A crash replacement is sequential — the
dead session emits nothing, so nothing interleaves. Unrelated work is
self-correcting, because a session in no room captures nothing and joining is
deliberate (D-064). That leaves deliberately split work, which is one collaboration
by construction, and where **the person knows why and the reader does not**. An
asymmetry of knowledge is not repaired by deriving a marker from a session
identifier.

**What would carry information is a purpose the person supplies** — "this is the
frontend half" — which is a feature with a real cost, and is not justified without
evidence that anybody runs split sessions.

**Revisit when** there is evidence that two live sessions from one machine is an
ordinary thing to do, rather than a thing the schema permits.
