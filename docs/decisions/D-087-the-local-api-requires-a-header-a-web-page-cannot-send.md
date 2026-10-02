# D-087 — The local API requires a header a web page cannot send

**Date:** 2026-09-20 · **Status:** active (implemented and demonstrated)

**Context.** D-086 makes the browser view the place people *do* things rather than
only read them. Checking what that would expose found that no handler on the local
HTTP server checked `r.Method` or `Origin` — not one.

**It was exploitable, and was demonstrated rather than argued.** A page served from
another port marked an unverified peer **verified** with a single POST: no ceremony,
no two words, no call. Loopback binding keeps other machines out and does nothing
about a page in this machine's browser, which reaches 127.0.0.1 like any address.
`json.Decode` ignores `Content-Type`, so `text/plain` makes it a "simple request"
that is sent without the browser asking permission first. The response is unreadable
cross-origin — and irrelevant, because the side effect has already landed.

That bypasses D-054, the gate every other guarantee depends on. `/hook/prompt` and
`/hook/stop` are the same class and arguably worse: what is published there reaches
teammates' context windows.

**The rule is REQUIRE, not refuse.** A request must positively present
`X-Cogmer: 1`. A browser cannot send a custom header to another origin without
a preflight, and we answer preflights with nothing, so the real request is never
sent. Measured: the `OPTIONS` arrived carrying
`Access-Control-Request-Headers: x-cogmer`, and **no POST followed**.

Requiring presence is what makes it fail **closed**. The alternative first
considered — refuse a mismatched `Origin` — is fail-open on absent, and would have
needed a standing argument that no browser can ever produce a header-less
cross-origin POST. Requiring a header needs no such argument: anything that cannot
present it is refused, whatever it is. `Origin` is kept as a second layer because it
costs three lines and fails the request earlier.

**Rejected: `Referer`, and rejected on measurement.** Proposed as a way to require
presence. A page suppresses its own referrer with one `<meta name="referrer">` tag —
measured: `Referer` absent, `Origin` still present — and an absent `Referer` must be
treated as allowed, because the CLI and hooks send none. The check would pass exactly
when it needed to fail. In the realistic case it is worse: a hostile page is on
`https://` and we are on `http://`, and the default `strict-origin-when-cross-origin`
policy drops `Referer` on that downgrade without the attacker trying.

**Rejected: a redirect to control the referrer.** Also measured. A 307 through our
own origin left `Referer` as the attacker's page, because the referrer is the
*initiating document* and survives the redirect. A **client-side** redirect would
have worked — a document we serve that navigates onward does become the referrer —
and that is sound. It was still not taken: `Referer` is a header third parties are
entitled to strip. Our own view's `Referrer-Policy`, a privacy extension, or a
corporate proxy would remove it from our **own** requests and break the view, with a
failure that reads as a bug rather than a policy. Nothing strips a header it has
never heard of.

**Also fixed:** state-changing routes require POST, so no navigation or `<img src>`
variant reaches a handler that reads a body.

**Not guarded, deliberately:** `/healthz`, `/events`, `/stream`, and the page itself.
These are reads, and a cross-origin page cannot read their responses — we send no
CORS headers, so the browser withholds the response from the script. Guarding them
would break the view, which fetches them as ordinary GETs.

**What this does not defend against:** a malicious program running as this user. It
can set any header, and could edit `membership.db` directly in any case. The threat
closed here is a web page, which is the one the browser enforces a boundary for.

**Revisit when:** the view stops being loopback-only, at which point a secret in the
URL (D-077's per-room addresses are the natural carrier) replaces this rather than
supplementing it.
