# D-086 — The terminal is not a user experience, and the view is the surface

**Date:** 2026-09-20 · **Status:** active · **Areas:** view, commands, hosts

**Decision.** What a user does belongs in the view, the browser page the daemon serves, and
the terminal is an operator surface for diagnostics, the daemon's lifecycle, a machine's
identity and testing.

**Support.**
- Every room-scoped and peer-scoped act has a slash command, and the pairing ceremony is
  daemon-side and spoken over HTTP, with `/verify/start` returning the words and
  `/verify/confirm` recording the answer, so moving it off the terminal is a change of
  interface and not of protocol. `cmd/cogmer/verify.go`, `handleVerifyStart` and
  `handleVerifyConfirm`.
- Not passing through the model and being a terminal are different properties, and the view
  has the first without the second. D-175 (a terminal command exists for diagnostics, the
  daemon's lifecycle, a machine's identity and testing).
- The ceremony is unchanged by moving its display: two words from a live exchange, compared
  aloud on a call by both people at once, with no way to skip the call. D-055 (a peer is
  verified in one way, the two-word comparison).
- The daemon and its view are a local service and a web page, so every piece of experience
  placed there is a piece a second host does not have to reimplement, and it gets there
  without an adapter or a speculative abstraction. D-043 (a second host is prepared for by
  naming, never by machinery).

**Limits.** The pairing ceremony itself lives in the view as D-088 (the pairing ceremony
lives in the view, and every pairing gets its own URL) states.

**Revisit when** the extension model of a second host is known.
