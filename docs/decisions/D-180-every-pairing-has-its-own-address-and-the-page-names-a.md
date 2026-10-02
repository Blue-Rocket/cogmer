# D-180 — Every pairing has its own address, and the page names a pairing and never a peer

**Date:** 2026-09-20 · **Status:** active · **Areas:** pairing, view, trust

**Decision.** Each pairing page has its own address carrying a random 128-bit id that is the
capability to start that exchange. `/verify/start` and `/verify/confirm` take a pairing
identifier, `pairId`, that the daemon resolves, and they take no peer from the page.

**Support.**
- A pairing page is a live ceremony with a deadline and not a dashboard, so two must never
  share a page, and a page left open from an earlier attempt must never become a different
  one, since the whole security property is that the person knows which key they vouch for.
  `cmd/cogmer/pairview.go`, `cmd/cogmer/pairview_test.go`, `TestEachPairingGetsItsOwnURL`.
- A fixed address whose contents were rewritten would at best change a tab nobody looks at,
  and `open` makes a new tab every time even for an identical address.
  `f288913:docs/decisions/D-088-the-pairing-ceremony-lives-in-the-view-and-every-pairing.md`.
- A page that reached the endpoint could not begin an exchange for an arbitrary identifier,
  and an expired link fails as a link and starts nothing. `cmd/cogmer/pairview_test.go`,
  `TestPairingResolvesToItsPeer`, `TestExpiredPairingIsRefused` and
  `TestVerifyStartRefusesAnExpiredPairing`.

**Rejected.**
- *A fixed address that shows whichever pairing is current.* Its contents change under the
  tab.

**Revisit when** the view stops being loopback-only, since the pairing id would then be a
bearer token over a network and not a local one.
