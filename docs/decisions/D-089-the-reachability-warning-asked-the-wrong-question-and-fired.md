# D-089 — The reachability warning asked the wrong question, and fired always

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asked whether a loopback address in a pairing string was simply "the
daemon wasn't running", and whether enough was done to avoid it. Reading the code to
answer found three defects, one of which made the existing warning worthless.

**The warning tested the bind address, not the advertised one.**

```go
if isLoopback(peerAddr()) {          // the address this daemon BINDS
```

`peerAddr()` is `127.0.0.1:4783` by default and stays loopback even when tailcat has
negotiated a perfectly routable `tc://` endpoint — which is the **ordinary** case,
since tailcat is on unless switched off. So the warning fired on every pairing string
ever printed, including all the working ones. Observed directly: `whoami` printing a
`tc://` string with "nobody else can reach it" underneath it. A warning that is
always on is not a warning; it trains somebody to ignore the real case.

It now asks about `AdvertisedEndpoint()` and treats a negotiated tailcat endpoint as
routable by construction.

**It was attached to a command rather than to the string.** Three commands print a
pairing string — `whoami`, `pair` with no arguments, and `peers` when empty — and
only `whoami` warned. The check moved into `printPairingInvitation`, so it travels
with the thing it is about.

**The two causes need different answers, and got one.** "No daemon has ever run, so
this address is a guess" is fixed by starting a session. "A daemon ran and negotiated
no route out" is fixed by setting `COGMER_PEER_ADDR`. Sending somebody to the
second when the first is true wastes their time on a setting that is not the problem.
`endpointRecorded()` distinguishes them.

**A fallback that needed the thing that had just failed.** `beginCeremony` fell back
to the terminal when `/pair/new` failed — but the terminal ceremony posts to
`/verify/start`, so with the daemon down it failed too, one message later. Two
failures reading as two problems. It now checks `daemonAlreadyServing` up front and
says the one true thing. The terminal fallback remains for what it is actually for:
a machine that cannot open a browser.

**Found by a test, immediately:** `ParseEndpoint` accepts anything without a scheme
as a TCP endpoint, and `isLoopback` answers `false` for what it cannot parse — so a
malformed address read as "not loopback" and therefore as fine. A bare TCP endpoint
must now parse as host and port.

**Have we done everything to avoid a loopback endpoint?** Nearly. Tailcat is on by
default and supplies a routable endpoint without configuration, which is why the
false warning mattered so much — it was obscuring that the ordinary path works. What
remains is the case where tailcat cannot negotiate: the string is then genuinely
unusable and we now say so precisely.

**Not taken: advertising the machine's LAN address as a fallback.** It is easy to
detect and it is right only for peers on the same network — which D-138 names as the
zero-configuration path and which is not built. Advertising an address that works for
some colleagues and silently not others is worse than an address that visibly works
for none. This belongs with local discovery, not ahead of it.

> **The conclusion holds; the reason is restated in D-091.** A LAN address is not
> rejected because it works for some colleagues and not others, but because it is a
> claim about a network this machine may no longer be on — and on a different
> network the same address is a different machine. A location is discovered, never
> advertised.

**Revisit when:** local network discovery lands, or tailcat stops being on by
default.
