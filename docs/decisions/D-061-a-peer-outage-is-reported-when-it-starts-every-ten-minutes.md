# D-061 — A peer outage is reported when it starts, every ten minutes while it lasts, and when it ends

**Date:** 2026-09-18 · **Status:** active · **Areas:** transport, daemon

**Decision.** A peer that stops answering is logged once when it stops. While it
remains unreachable it is logged again every ten minutes, and when it answers again it is logged
once more with how long it was gone. A failed poll in between says nothing.

**Support.**
- Polling runs every second, so a line for each failed poll writes two lines a second
  that say the same thing and bury the transition, which is the line that matters.
  `cmd/cogmer/ui.go`, `markPeerUnreachable`.
- The ten-minute repeat lets somebody reading a log an hour later learn that the peer is
  still gone and not only that it once went. `cmd/cogmer/ui.go`, `downRepeat`.
- Coming back is reported only for a peer that had been reported missing.
  `cmd/cogmer/ui.go`, `markPeerSeen`.

**Rejected.**
- *A line for every failed poll.* A message repeated that often is not a signal.
