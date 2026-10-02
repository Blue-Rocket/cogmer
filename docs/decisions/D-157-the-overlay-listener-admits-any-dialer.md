# D-157 — The overlay listener admits any dialer

**Date:** 2026-10-01 · **Status:** active · **Areas:** transport, trust

**Decision.** The overlay listener accepts a tunnel from a dialer whatever key the
dialer presents. Whether the other side is a peer is decided above the tunnel, by the
TLS pin and the signed request, as on every other transport.

**Support.**
- A tailcat client given no key generates one at its first use, and cogmer's dialer
  gives it none, so the key a colleague's daemon dials with appears in nothing that
  colleague sends. https://pkg.go.dev/github.com/tailscale/tailcat@v0.6.0#Client
  (`Key`), and `tailcatDialer.clientFor` in `cmd/cogmer/tailcat.go`.
- A tailcat listener with a list of allowed clients ignores every other client and
  sends it no reply. https://pkg.go.dev/github.com/tailscale/tailcat@v0.6.0#Server
  (`AllowedClients`).
- During the TLS handshake, before any request is read, the peer listener refuses a
  client whose key this machine has not recorded. `serverConfig` and `pinnedVerifier`
  in `cmd/cogmer/peertls.go`, and D-101 (peer connections are TLS pinned).
- A stranger who completes a tunnel learns the identifier in this machine's
  certificate, and a pairing string carries that identifier beside the overlay
  address. §12, and `peerCert` in `cmd/cogmer/peertls.go`.
- The recorded peers are the one list that decides who is a peer, and a tunnel list
  keyed on a different key type would be a second that can disagree with it. D-062
  (tailcat's own client allowlist is not used).
- A listener admits a dialer it has never seen. `TestATunnelAdmitsADialerItHasNeverSeen`
  in `cmd/cogmer/tailcat_test.go`, which runs only with `COGMER_NETWORK_TESTS` set,
  because it needs the public relay.

**Rejected.**
- *Admitting only the listener keys of recorded peers, read from their overlay
  addresses.* A colleague's daemon dials with a key its address does not carry, so
  once one peer is recorded no colleague's dial is answered, and a verification waits
  out its timeout. https://pkg.go.dev/github.com/tailscale/tailcat@v0.6.0#Client
  (`Key`).
- *A persistent dialer key, carried in the pairing string beside the address.* The
  daemon opens one tunnel client per peer, so two colleagues whose addresses name the
  same relay region would put one key on two connections to that relay, and the
  tailcat documentation does not say how the relay treats that. `tailcatDialer` in
  `cmd/cogmer/tailcat.go`.
- *Dialling with the listener's own key.* The listener and every dialer would hold
  connections to the relay under one key, with the same undocumented result.
- *A dialer key for each colleague.* A pairing string is printed before its sender
  knows who will receive it, so it cannot carry a key made for them. `runWhoami` in
  `cmd/cogmer/main.go`, which prints the string and takes no recipient.

**Limits.** Anybody holding this machine's overlay address can open a tunnel and make
the listener perform a TLS handshake. An unrecorded key is refused there, and the
dialer receives a TLS error. This decides nothing about the post-quantum hedge the
pre-shared key provided (D-104, the overlay address is public and stable).

**Revisit when** tailcat documents how its relay treats one key on several
connections, or lets a listener admit a client by something a pairing string can
carry.
