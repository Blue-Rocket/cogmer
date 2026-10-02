# D-181 — The ninety seconds of a pairing start when the user presses Start

**Date:** 2026-09-20 · **Status:** active · **Areas:** pairing, view

**Decision.** The page opens in a ready state with a Start button, and the 90 seconds of the
exchange start when the user presses it. A timeout returns the page to the ready state and
does not retry at once.

**Support.**
- Running the exchange when the page loads spends the deadline on however long two people
  take to get on a call, which is the coordination the deadline exists to bound.
  `cmd/cogmer/pairview.go`.
- The window is the daemon's verification timeout. `cmd/cogmer/verify.go`, `verifyTimeout`.

**Rejected.**
- *Starting the exchange when the tab loads.* The deadline is gone before the call begins.
