# D-073 — Forgetting a peer discards their admissions with them

**Date:** 2026-09-19 · **Status:** active · **Areas:** admission, identity

**Decision.** `Forget` deletes a peer's identity and every admission it carries in one
transaction, so a later meeting is a first meeting: known again, unverified again, and
admitted to nothing until a host says so. Forgetting acts on the colleague as a user: it
covers every machine of theirs, and it applies on every machine of the user who forgets,
since the lists belong to that user (D-228, a user's colleagues, verifications, labels and
rooms belong to the user and are copied between the user's machines). It never stands for
removing one machine, which only the user whose machine it is can do (D-220, a removal
names the last event of a machine that counts).

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

**Revisit when** a colleague needs to be forgotten by one user while remaining a guest in
another user's room. Guest lists belong to each user, so two users can disagree about who
belongs (D-159, an invitation carries the room and the inviter, and joining admits the
inviter).
