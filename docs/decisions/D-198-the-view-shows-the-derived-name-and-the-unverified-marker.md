# D-198 — The view shows the derived name and the unverified marker only on other people's turns

**Date:** 2026-09-17 · **Status:** active · **Areas:** view, identity, trust

**Decision.** The room view shows a turn's derived name and its unverified marker only when the turn
was written by somebody other than this user's own peer. A turn this user wrote shows a name and a
time.

**Support.**
- On a turn the reader wrote, the derived name and the marker identify nobody the reader does not
  know and attest to nothing. The first screenshot of real use showed three identifiers for one
  entity on one line, a name with "Claude" before it, a derived name and a badge, for a turn that was the
  reader's. `33ce996`, `cmd/cogmer/ui.html`.
- The derived name matters on other peers, whose identities nothing about the turn itself verifies,
  since attribution rests on it and a changed key is noticed against it. D-021 (peer names are word
  pairs derived from the identity, never chosen), §6 (naming people).
- Whose turn it is comes from the peer identifier the daemon sends as `selfPeerId`, and never from the
  display name, which two peers can assert alike. `cmd/cogmer/ui.go`, `uiState`,
  `cmd/cogmer/ui.html`.

**Rejected.**
- *Showing both on every turn.* On the reader's own turns they add identifiers and no information.

**Limits.** No test runs the view's script, so the rule is held by `cmd/cogmer/ui.html` alone.
