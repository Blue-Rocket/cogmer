# D-073 — Forgetting a peer discards their admissions with them

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Tracing a guest's lifecycle through the code rather than the
specification. §12 promises:

> Forgetting and revoking are not the same act. Revoking withdraws admission to one
> room. Forgetting discards the identity itself, **so a later meeting is a first
> meeting again.**

`Forget` deleted the row in `known_peers` and left every row in `room_guests`.

**What that cost.** Two things, and the second is the serious one.

A forgotten peer still appeared in `guests` — a host reading the list would see
somebody admitted who is not known, which is a false affordance in the place a host
looks to decide who is in a room.

And **meeting them again readmitted them to everything**. `allow` reinstates the
identity; the old `room_guests` rows were still there; verification would follow;
and they would be a guest of every room they had ever been in, without the host
inviting them to any of it. The host is never asked, because from the code's point
of view nobody was ever un-invited. That is the opposite of a first meeting.

It was masked rather than harmless: a forgotten peer is unverified, so D-054's gate
refused their sync. The hole only opened at the moment they were verified again —
which is to say, at the moment the host believed they were starting over.

**Decision.** `Forget` deletes admissions and identity in one transaction. A later
meeting then really is a first meeting: known again, unverified again, admitted to
nothing until a host says so.

**Revoke is unchanged and remains the narrow act.** One room, admission only, the
identity untouched. The two were always meant to differ in scope rather than in
kind, and now they do: revoke removes one admission, forget removes the identity and
every admission it carried.

**Tests.** `TestForgettingDiscardsAdmissionToo` follows the whole arc — allow,
verify, invite to two rooms, forget, meet again — and requires that meeting again
grants nothing. `TestForgettingIsNarrow` requires that forgetting one peer disturbs
neither another peer nor the room's own creator, who is its first guest. The first
is confirmed to fail without the cascade.

**Revisit when** a peer needs to be forgotten on one machine while remaining a guest
elsewhere. Guest lists are per-peer (D-159, an invitation carries the room and the inviter), so two peers can already disagree about
who belongs; this changes nothing about that, and a host forgetting somebody does
not tell them so.
