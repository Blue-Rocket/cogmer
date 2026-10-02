# D-088 — The pairing ceremony lives in the view

**Date:** 2026-09-20 · **Status:** active · **Areas:** pairing, view

**Decision.** The two-word comparison is shown and confirmed on a page the daemon serves in
the browser. The terminal path remains for `--terminal` and for a machine that cannot open
a browser, and it is the automatic fallback there.

**Support.**
- Nothing in the ceremony needed a terminal beyond the answer to whether the other said the
  same two words, and the view has an origin boundary, which letting it do anything and not
  only show things required. D-087 (the local API requires a header that a web page cannot
  send).
- The ceremony is unchanged by moving its display: two words from a live exchange, compared
  aloud by both people at once. A terminal was never the ceremony. D-055 (a peer is verified
  in one way, the two-word comparison), D-086 (the terminal is not a user experience, and the
  view is the surface).
- `openInBrowser` returns an error and does not fail quietly, so the caller can choose the
  terminal branch. `cmd/cogmer/browser.go`, `openInBrowser`.
- Both paths need the daemon, so `beginCeremony` checks that one is serving before it begins,
  and says the one true thing when it is not. `cmd/cogmer/main.go`, `beginCeremony`.
- Each pairing opens a new tab, since each is a new address, so a later pairing re-opens the
  view, and the view drives the ceremony because the confirmation was the one thing that kept
  a terminal in the path. D-180 (every pairing has its own address), `cmd/cogmer/main.go`,
  `beginCeremony`.
- Two real daemons on one machine ran a genuine exchange, matching words appeared in both
  pages, and both sides recorded the peer as verified only after a person clicked, with
  nothing recorded before.
  `f288913:docs/decisions/D-088-the-pairing-ceremony-lives-in-the-view-and-every-pairing.md`.

**Rejected.**
- *The terminal as the only place for the ceremony.* A user may have never opened one, and the
  words must reach a person's eyes without passing through the model.

**Revisit when** the ceremony gains a step that a page cannot host.
