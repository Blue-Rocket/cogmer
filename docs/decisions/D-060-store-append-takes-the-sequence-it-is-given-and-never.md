# D-060 — `Store.Append` takes the sequence it is given and never derives one

**Date:** 2026-09-18 · **Status:** active · **Areas:** sync, storage

**Decision.** `Store.Append` takes the sequence number as an argument and derives none.
The number comes from `Membership.ReserveSequence`, which records it in `membership.db`
before the event that uses it is published.

**Support.**
- A sequence derived from the highest one in the room's own database goes to zero when
  that database is lost, so the peer would reissue numbers that others hold as different
  events, which a receiver can only quarantine. D-029 (losing a room database does not
  end membership), D-027 (a sequence conflict is quarantined, not dropped).
- Reserving before publishing loses a number and never reissues one, and a lost number is
  harmless because a gap makes peers wait and not skip. `cmd/cogmer/offline_test.go`,
  `TestAReservationIsSpentEvenIfNothingIsWritten`.
- No call derives a sequence, so the dangerous path is absent as well as unused.
  `cmd/cogmer/store.go`, `Append`.

**Rejected.**
- *Deriving the next sequence from the room's own events.* The counter restarts at 1
  when the room is lost.

**Limits.** Reporting a lost room is D-169.

**Revisit when** a peer needs to reserve sequences for a room it has not joined, or
across two machines under one identity. The reservation would then have to be shared and
not local, which makes it a distributed counter.
