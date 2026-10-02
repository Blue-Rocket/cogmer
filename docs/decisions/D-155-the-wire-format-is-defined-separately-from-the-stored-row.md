# D-155 — The wire format is defined separately from the stored row

**Date:** 2026-09-17 · **Status:** active

**Decision.** The wire format is its own type, in `cmd/cogmer/protocol.go`, with explicit
conversion to and from the stored row.

**Support.**
- A column can be added to the store for local bookkeeping without telling any peer, and
  a wire field cannot change without every peer agreeing, so one struct for both would
  let a convenient column become protocol without anyone deciding it. `cmd/cogmer/protocol.go`.
- A test checks that converting an event to the wire format and back drops nothing.
  `cmd/cogmer/protocol_test.go`, `TestWireRoundTripPreservesEverything`.

**Rejected.**
- *One struct for the stored row and the wire.* The next column added for local
  bookkeeping would become protocol silently.

**Revisit when** the stored row and the wire format need to carry the same thing.
