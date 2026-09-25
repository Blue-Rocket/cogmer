# How a web page attacks a local service, and what stops it

Written 2026-09-24 against `cmd/cogmer/localguard.go` at commit 5e35257. The
scenarios follow from the code and from how browsers are specified to behave; none
was reproduced in a browser. `curl -H 'Host: evil.example:4782'` against the
installed 0.7.1 daemon got 200 from both `/` and `/healthz`, so the daemon answers a
request whose `Host` names another site.

## The cast

The same cast appears in every scenario.

- **The user** runs cogmer, and has a browser open on the same machine.
- **The daemon** is cogmer's local server, listening at `127.0.0.1:4782`.
- **The attacker** controls a website at the name `evil.example`, and controls the
  DNS server that answers lookups for `evil.example`.
- **The attacker's page** is a web page the attacker wrote. It contains a script,
  which the browser runs as soon as the page loads.

**One browser rule matters throughout.** The browser labels every page with an
*origin*, made of three parts: the scheme, the name and the port, such as
`http://evil.example:4782`. The IP address is not part of the origin. When a page's
script sends a request to a URL with a different origin, the browser applies
restrictions. When the URL has the same origin as the page, the browser applies
none.

---

## Scenario 1: a cross-site POST, against a daemon that checks nothing

1. The user visits `https://evil.example`.
2. The attacker's page runs a script that sends
   `POST http://127.0.0.1:4782/verify/confirm` with a `text/plain` body naming a
   peer.
3. The browser compares origins. The page is `https://evil.example` and the URL is
   `http://127.0.0.1:4782`, so the two differ.
4. The browser has one exception for different origins: a request shaped like an
   ordinary form submission is sent without asking. A POST with a `text/plain` body
   and no unusual headers fits that shape, so the browser sends the request.
5. The daemon accepts the connection, because the connection comes from this
   machine.
6. The daemon marks the peer verified.
7. The browser refuses to show the daemon's reply to the script. That refusal
   doesn't matter, because the peer is already verified.

**Result:** the attack works. It was demonstrated against the daemon before the
custom header check existed.

## Scenario 2: the same attack against the daemon with the custom header check

The daemon now refuses any state-changing request that lacks the header
`X-Cogmer: 1`.

1. The user visits `https://evil.example`.
2. **Without the header:** the attacker's page sends the same POST as in Scenario 1.
   The browser sends the request. The daemon finds no `X-Cogmer` header and refuses
   the request.
3. **With the header:** the attacker's page sends the same POST with `X-Cogmer: 1`
   added.
   - The origins still differ, and a request with an unknown header doesn't fit the
     form-submission shape. So before sending the POST, the browser asks
     permission. It sends `OPTIONS http://127.0.0.1:4782/verify/confirm` with
     `Access-Control-Request-Headers: x-cogmer` and `Origin: https://evil.example`.
   - The daemon's reply grants no permission.
   - The browser never sends the POST.
4. The cogmer CLI and the hooks aren't browsers. They send `X-Cogmer: 1` directly,
   nothing asks permission first, and the daemon accepts their requests.

**Result:** the attack fails. The defence depends on step 3, where the browser asks
permission because the origins differ.

## Scenario 3: DNS rebinding against the daemon as it is today

1. The attacker's DNS server is set to answer lookups for `evil.example` with the
   attacker's own IP address, say `203.0.113.5`. Each answer carries a very short
   lifetime, such as a few seconds, so the browser soon looks the name up again.
2. The attacker runs a web server at `203.0.113.5`, port **4782**. The port matters:
   the origin includes the port, and the port must match the daemon's.
3. The user visits `http://evil.example:4782`. The browser looks up `evil.example`,
   gets `203.0.113.5`, and loads the attacker's page from the attacker's server. The
   page's origin is `http://evil.example:4782`.
4. The attacker changes the DNS server's answer for `evil.example` to `127.0.0.1`.
5. The attacker's page waits until the earlier answer expires, then sends
   `POST http://evil.example:4782/verify/confirm` with `X-Cogmer: 1`.
6. The browser compares origins. The page is `http://evil.example:4782` and the URL
   is `http://evil.example:4782`. **The two are identical**, so the browser asks no
   permission and sends the request exactly as written, custom header included.
7. The browser looks up `evil.example` again, now gets `127.0.0.1`, and connects to
   `127.0.0.1:4782`. That address is the daemon.
8. The daemon accepts the connection, because the connection comes from this
   machine. **Binding to loopback doesn't help here:** the browser is on this
   machine.
9. The request arrives with `Host: evil.example:4782`,
   `Origin: http://evil.example:4782` and `X-Cogmer: 1`.
10. The custom header check passes.
11. The `Origin` check refuses the request, because `http://evil.example:4782` isn't
    one of the daemon's own origins. Browsers attach `Origin` to every POST,
    including a POST to the page's own origin.
12. **But reads have no check.** The attacker's page sends
    `GET http://evil.example:4782/events?room=…`. Again the origins are identical,
    so the browser sends the request and **lets the script read the reply**. The
    script can then send the conversation to the attacker.

**Result:** the custom header does nothing, because the browser never asked
permission. The `Origin` check, which the design treats as a minor second layer, is
the only thing stopping writes. Nothing stops reads, as long as the attacker's page
can learn a room's address. That part hasn't been confirmed.

## Scenario 4: DNS rebinding against a daemon that checks `Host`

The daemon now refuses any request whose `Host` header isn't `127.0.0.1:4782`,
`localhost:4782` or `[::1]:4782`.

- Steps 1 to 8 happen exactly as in Scenario 3.
- 9. The request arrives with `Host: evil.example:4782`. The browser fills in `Host`
  from the name in the URL, and browsers forbid a page's script from changing
  `Host`.
- 10. The daemon refuses the request. It refuses reads the same way.

**Result:** the attack fails.

## Scenario 5: the attacker tries to pass the `Host` check

1. To produce `Host: 127.0.0.1:4782`, the attacker's page has to send its request to
   the URL `http://127.0.0.1:4782/…`.
2. The page's origin is `http://evil.example:4782` and the URL's origin is
   `http://127.0.0.1:4782`. The names differ, so the origins differ.
3. The origins differing puts the attacker back in Scenario 2. The browser asks
   permission before sending `X-Cogmer`, and the daemon grants none. For a read, the
   browser sends the request but withholds the reply from the script.

**Result:** the attack fails.

---

## Why binding to loopback and the `Host` check do different jobs

Binding decides which connections the daemon accepts: only connections from this
machine. The browser is on this machine, so in every scenario the browser's
connection is accepted. Binding never sees the name the browser used. `Host` is the
only part of the request that carries that name.

**Summary:** the custom header works only when the browser sees two different
origins. DNS rebinding makes the browser see one origin, while the request actually
reaches the daemon. The `Host` check catches exactly that case, because the
attacker's name travels in `Host` and a page's script can't change it.
