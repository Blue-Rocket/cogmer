# D-188 — A waiting offer is shown with the rooms and a withheld one beside the peer who can release it

**Date:** 2026-09-21 · **Status:** active · **Areas:** admission, pairing, commands

**Decision.** An offer waiting to be accepted is listed apart from the rooms that are joined. An
invitation withheld for want of a verification is shown beside the peer who can release it. It
names the colleague and what to type, `/peer-pair <name>`, and does not count rooms, and it stops
being said once pairing completes.

**Support.**
- Accepting an offer is the act that has not happened, so it belongs with rooms and apart from
  the joined ones, and a withheld invitation is invisible to the person it is for and wholly
  visible to the person who can release it. `cmd/cogmer/membership.go`, `Offers`.
- An ordinary room holds two people, so a count of withheld rooms is always one and carries
  nothing, where the colleague who is unfinished and the command to type are what a user can act
  on. `cmd/cogmer/main.go`, `cmd/cogmer/offer_test.go`.
- Pairing is the act a person recognizes and verifying is the word for a step inside it, so the
  prompt says there is work to do with the colleague and offers `/peer-pair`. D-107 (pairing again with a peer who is paired does nothing, loudly).

**Rejected.**
- *A tally of withheld rooms.* It is always one.
