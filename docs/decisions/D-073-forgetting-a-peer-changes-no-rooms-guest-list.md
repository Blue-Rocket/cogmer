# D-073 — Forgetting a peer changes no room's guest list

**Date:** 2026-09-19 · **Status:** active · **Areas:** admission, identity

**Decision.** Forgetting a colleague is the forgetting user's act alone, and leaves every
room's guest list as it is, since a room's list belongs to the room (D-235, a room's one
guest list decides whose events count). `Forget` deletes the colleague's identity on every
machine of the user who forgets (D-228, a user's colleagues, verifications, labels and rooms
belong to the user and are copied between the user's machines). The user's machines then
stop recognizing the colleague, and the colleague's turns stop reaching that user, since a
colleague the user does not know is unverified to them (D-054, verification gates
synchronization and injection, not just a marker), while the other guests keep working with
the colleague. Taking somebody out of a room is a revocation. A colleague the user pairs
with again is known again and unverified again, as at a first meeting, and is a guest of
every room they were never revoked from. Forgetting acts on the colleague as a user and
covers every machine of theirs, and it never stands for removing one machine, which only
the user whose machine it is can do (D-220, a removal names the last event of a machine
that counts).

**Support.**
- Forgetting discards the identity itself, so a later meeting is a first meeting again,
  where revoking withdraws admission to one room. §12 (forming a room).
- Forgetting one peer disturbs neither another peer nor the room's own creator, who is its
  first guest. `cmd/cogmer/membership_test.go`, `TestForgettingIsNarrow`.

**Rejected.**
- *Forgetting discarding the colleague's admissions.* A colleague paired again is in the
  rooms they never left because a guest invited them and nobody revoked them, which the
  room's list records, and with one list per room, discarding the admissions would take the
  colleague away from every other guest.
- *Forgetting revoking the colleague from every room the two share.* It removes the
  colleague for every other guest, possibly from rooms the user has half forgotten.
- *Forgetting asking, room by room, whether to revoke the colleague.* Taking somebody out
  of a room is a revocation, which the user makes in that room.

**Limits.** Revoking removes one admission to one room, for every guest, and leaves the
identity. A user forgetting somebody does not tell them so. A room's list shows a forgotten
colleague as a guest though the user's machines do not recognize them, and this entry does
not decide how the list labels such a guest.

**Revisit when** users who forget a colleague are found to expect the colleague gone from
the rooms they share.
