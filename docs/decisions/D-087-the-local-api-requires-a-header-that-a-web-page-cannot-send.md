# D-087 — The local API requires a header that a web page cannot send

**Date:** 2026-09-20 · **Status:** active · **Areas:** view, trust

**Decision.** A route that changes state requires the method POST and the header
`X-Cogmer: 1`. A request whose `Origin` does not match is also refused, as a second layer.
Routes that only read are not guarded.

**Support.**
- Loopback keeps other machines out and does nothing about a page open in this machine's
  browser, which reaches 127.0.0.1 like any address, and a request with a `text/plain` body
  is sent without the browser asking permission, so one POST from a page on another port
  marked an unverified peer verified. `cmd/cogmer/localguard.go`, and
  `f288913:docs/decisions/D-087-the-local-api-requires-a-header-a-web-page-cannot-send.md`.
- A browser cannot send a custom header to another origin without a preflight, and the
  daemon answers a preflight with nothing, so the real request is never sent.
  `cmd/cogmer/localguard_test.go`, `TestLocalGuardDoesNotAnswerPreflight` and
  `TestLocalGuardRefusesAMissingHeader`.
- Requiring a header fails closed, since anything that cannot present it is refused whatever
  it is, where refusing a mismatched `Origin` fails open when the header is absent.
  `cmd/cogmer/localguard_test.go`, `TestLocalGuardRefusesAWebPage`.
- State-changing routes require POST, so no navigation or image source reaches a handler
  that reads a body. `cmd/cogmer/localguard_test.go`, `TestLocalGuardRefusesNonPost`.
- The reads `/healthz`, `/events`, `/stream` and the page are unguarded because the daemon
  sends no CORS headers, so a cross-origin page cannot read their responses, and guarding
  them would break the view. `cmd/cogmer/localguard.go`.

**Rejected.**
- *Requiring a `Referer`.* A page suppresses its own referrer with one `<meta>` tag, and an
  absent `Referer` has to be allowed because the CLI and the hooks send none, so the check
  passes when it should fail.
- *A redirect to control the referrer.* The referrer is the initiating document and survives
  a redirect. A client-side redirect would work, but `Referer` is a header that third parties
  may strip, such as the view's own `Referrer-Policy`, an extension or a proxy, and nothing
  strips a header it has never heard of.

**Limits.** A malicious program running as the user can set any header and could edit
`membership.db` directly, so the guard closes the threat of a web page and no other.

**Revisit when** the view stops being loopback-only. A secret in the address then takes the
guard's place and does not add to it, and D-077 (every room has its own address in the view)
holds the per-room addresses that carry it.
