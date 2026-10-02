# D-069 — Every signing tag uses one namespace that never changes

**Date:** 2026-09-19 · **Status:** active · **Areas:** identity, trust

**Decision.** Every signature covers a domain-separation tag that begins with
`protocolNamespace`, which is `peer-room`, arbitrary on purpose, and never changes. The
product name appears in no signed bytes.

**Support.**
- Domain separation needs stability and uniqueness and does not need meaning. A tag that
  carried the product name would have to be carried forever once real events existed,
  because an event is immutable and cannot be re-signed. D-058 (an event verifies
  under the scheme it was signed with), D-029 (losing a room database does not end membership).
- The event tag, the sync-request tag, the offer tag and the verification tags all begin
  with it. `cmd/cogmer/keys.go`, `cmd/cogmer/auth.go`, `cmd/cogmer/offer.go`.
- A tag that protects a live exchange, such as a sync request or a verification, needs no
  new version when it changes, since nothing signed under it outlives the exchange.
  `cmd/cogmer/auth.go`.

**Rejected.**
- *A namespace that names the product.* A rename would then force a new signing scheme
  that every peer must carry for good.
