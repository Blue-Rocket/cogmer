# D-073 — Forgetting a peer discards their admissions with them

**Date:** 2026-09-19 · **Status:** active · **Areas:** admission, identity

**Decision.** `Forget` deletes a peer's identity and every admission it carries in one
transaction, so a later meeting is a first meeting: known again, unverified again, and
admitted to nothing until a host says so.

**Support.**
- Forgetting discards the identity itself, so a later meeting is a first meeting again,
  where revoking withdraws admission to one room. §12 (forming a room).
- If the admissions stayed, meeting the peer again would readmit them to every room they
  had been in without the host inviting them to any, and the host is never asked, since
  nobody was ever un-invited. `cmd/cogmer/membership_test.go`, `TestForgettingDiscardsAdmissionToo`.
- A peer forgotten and left in the guest list would appear admitted though unknown, a
  false affordance where a host looks to decide who is in a room. `cmd/cogmer/membership.go`,
  `Forget`.
- Forgetting one peer disturbs neither another peer nor the room's own creator, who is its
  first guest. `cmd/cogmer/membership_test.go`, `TestForgettingIsNarrow`.

**Limits.** `revoke` removes one admission to one room and leaves the identity. A host
forgetting somebody does not tell them so.

**Revisit when** a peer needs to be forgotten on one machine while remaining a guest
elsewhere. Guest lists belong to each peer, so two peers can disagree about who belongs
(D-159, an invitation carries the room and the inviter, and joining admits the inviter).
