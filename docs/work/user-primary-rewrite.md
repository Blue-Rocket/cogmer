# Plan: rewriting the decisions and the specification with the user as the primary entity

This is working material for the open item "A user's machines are separate identities to
colleagues, and a cloud session has no identity of its own to sign with" in `docs/open.md`,
and is deleted with it. The answers it carries out are in section 2 of
`docs/work/user-primary-inventory.md`, and the decisions each change touches are in section
1 of that file.

## How each changed decision is written

- A decision that changes in part is rewritten to state only what holds (W-38), keeps every
  fact, reference and number (W-39), and changes its title when the title states something
  that no longer holds, renaming its file to match. Every citation of a renamed decision is
  checked against its few words (W-11).
- A decision that states the design keeps its status. Where it then differs from the code,
  the open item in `docs/open.md` lists it, as it lists D-044, D-101, D-141, D-143 and D-152,
  since a defect in the present code is written only there (W-27).
- A peer is a user as cogmer knows them, and a machine is one of a peer's devices (W-01).

## The new decisions

Each takes the next free number when it is written, N1 first.

| name | title |
|---|---|
| N1 | A user's colleagues, verifications, labels and rooms belong to the user and are copied between the user's machines |
| N2 | A machine is added from a machine the user already has, with the two-word check |
| N3 | A user identity is made in the daemon's page the first time the user needs one |
| N4 | A user's machines are managed from one command that opens the view |
| N5 | A user's own turn from another session is injected as a report marked as the user's |
| N6 | Every turn carries the name of the machine it came from, as a field the user asserts |

N1 holds R2 and the shared room list from R6. N2 holds the part of R6 on adding a
machine and the separate purpose tag from R8. N3 holds the rest of R6. N4 holds R9.
N5 holds R4. N6 holds R5.

## Who rewrites what

- Identity and pairing: D-025, D-042, D-048, D-052, D-053, D-073, D-074, D-095, D-097,
  D-141, D-186, D-222, and the new N1 to N4.
- Rooms, sync and the view: D-024, D-027, D-029, D-060, D-086, D-090, D-099, D-103, D-105,
  D-112, D-135, D-149, D-159, D-194, D-198, D-200, D-204, D-214, D-225, and the new N5 and
  N6.
- The specification: sections 6, 12, 12a, 13, 17, 20, 25 and 29, and any other passage that
  makes a machine the participant.
- Wording only, after the two decision groups merge: the decisions listed as wording only in
  section 1 of the inventory.
