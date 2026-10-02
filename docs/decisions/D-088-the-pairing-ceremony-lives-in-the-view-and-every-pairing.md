# D-088 — The pairing ceremony lives in the view, and every pairing gets its own URL

**Date:** 2026-09-20 · **Status:** active (implemented and verified end to end)

**Context.** D-086 established that the terminal is not a user experience and that
the only thing still requiring one was a single `fmt.Scanln` — the y/N answer in
`verifyWith`. D-087 gave the view an origin boundary, which was the prerequisite for
letting it do anything rather than only show things.

**D-055 is unchanged, and that is the point.** Two words, from a live commit/reveal
exchange, compared aloud on a call, by both people at once, no fallback. What moved
is the surface. A terminal was never the ceremony — it was the only thing available
that was not the model (D-080, as
`f288913:docs/decisions/D-080-there-is-no-current-room-a-terminal-command-exists-to-be.md` held it), and the view is the other one.

**Every pairing gets its own URL**, and this is load-bearing rather than tidy:

- A pairing page is a live ceremony with a deadline, not a dashboard. Two must never
  share a page, and a page left open from an earlier attempt must never quietly
  become a different one — the entire security property is that the person knows
  *which key* they are vouching for.
- It is what makes a second pairing visible. A fixed address whose contents we
  rewrote would, at best, change a tab nobody is looking at.

The id is 128 random bits, because it is the capability: holding it is what lets a
page start that exchange.

**Measured, and it refined the premise.** `open` creates a **new tab every time**,
even for an identical URL — tab count climbed 2→3→4→5 across repeated opens — and
focus was taken in all six trials, same-URL and unique-URL alike. So in Chrome the
silent-background-rewrite case does not arise from `open` itself. Unique URLs are
still right: they do not depend on that behavior, and the correctness argument above
stands on its own.

**The page names a pairing, never a peer.** `/verify/start` and `/verify/confirm`
grew an optional `pairId`, and the daemon resolves it. So even a page that reached
the endpoint could not begin an exchange for an arbitrary identifier, and an expired
link fails as a link rather than silently starting something new.

**The 90-second window starts when the person is ready, not when the tab loads.**
The first build ran the exchange on load, which spends the deadline on however long
it takes two people to get on a call — exactly the coordination the deadline exists
to bound. The page now opens in a *ready* state with a Start button, and a timeout
returns there rather than to an immediate retry.

**The terminal path is kept, and is not legacy.** `--terminal` forces it, and it is
also the automatic fallback when no browser can be opened — a machine reached over
SSH, a container, a server. `openInBrowser` returns an error rather than failing
quietly, precisely so the caller can choose that branch.

**Verified end to end**, not only by unit test: two real daemons on one machine, a
genuine commit/reveal between them, matching words rendered in both browser pages,
and both sides recording `verified` only after a human clicked. Nothing was recorded
before the click.

**Also settled from the list D-083, as `f288913:docs/decisions/D-083-the-view-is-opened-at-a-first-pairing-not-at-room-creation.md` held it, left open:** later pairings do re-open the view, because
each one is a new URL and therefore a new tab. Whether the view *drives* or merely
*displays* is now answered — it drives, since the confirmation is the one bit that
was keeping a terminal in the path.

**Revisit when:** the ceremony gains a step that a page cannot host, or the view
stops being loopback-only (which would make the pairing id a bearer token over a
network rather than a local one).
