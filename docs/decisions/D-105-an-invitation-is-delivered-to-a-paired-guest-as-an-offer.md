# D-105 — An invitation is delivered to a paired guest as an offer over the paired channel

**Date:** 2026-09-21 · **Status:** active · **Areas:** admission, pairing, transport

**Decision.** The host's daemon delivers an invitation to a paired guest as an offer, a message
telling the guest that they have been admitted to a room, over the connection pairing set up.
The host is given a string to send by hand only when the daemon cannot reach the guest.

**Support.**
- Delivering an offer is not pushing an admission. The row on the host's guest list is the
  admission, and whom to admit is the host's judgment, so an offer tells the guest that the row
  exists, joining remains the guest's own act and reading anything still requires
  verification. D-024 (admission is a guest list), D-054 (verification gates synchronization
  and injection).
- An offer that arrives is data being stored and not a turn being taken. §3.7 (a remote event
  never drives an interactive session), `cmd/cogmer/offer.go`, `handleOffer`.
- A host who cannot reach the guest is told at the moment of inviting and handed the string to
  send, where a string as the only path means nobody learns whether it was needed.
  `cmd/cogmer/offer.go`.
- Recovery from an address change is an outlier and forming a room is an everyday act, and with
  a stable overlay address a host fails to reach a guest only after a restart and an
  unreachable relay. D-104 (the overlay address holds no secret and is the same at every
  start).

**Rejected.**
- *The guest polling for offers.* It requires every pair of paired peers to keep contact
  indefinitely, whether or not they share a room, which is standing traffic and standing
  metadata for something that happens rarely, where a push fails at once and visibly and costs
  nothing between invitations.

**Limits.** The only remaining manual exchange is between people who have never paired. D-051
(stranger pairing is not a supported case).

**Revisit when** paired peers acquire a standing reason to poll each other, since pull then
costs nothing extra and is the better shape.
