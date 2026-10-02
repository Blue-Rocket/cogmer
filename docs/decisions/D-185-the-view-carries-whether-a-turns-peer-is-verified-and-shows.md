# D-185 — The view carries whether a turn's peer is verified, and shows the marker only when it is not

**Date:** 2026-09-20 · **Status:** active · **Areas:** view, trust

**Decision.** Each event the view receives carries whether its peer is verified, and the
unverified marker appears only when that field says so.

**Support.**
- An unverified peer's events are refused before they are stored, so everything displayed is
  verified, and a marker on every remote turn would sit permanently on. D-054 (verification
  gates synchronization and injection, not just a marker).
- A warning that is always on trains a reader to ignore the one that matters, so the marker is
  a fact and seeing it means a filter failed. D-089 (the reachability warning asks about the
  advertised address, and travels with the pairing string), `cmd/cogmer/ui.go`, `uiEvent`.

**Rejected.**
- *A marker on every remote turn.* It is the always-on warning.
