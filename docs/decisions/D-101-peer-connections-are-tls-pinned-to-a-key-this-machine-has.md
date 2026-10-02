# D-101 — Peer connections are TLS pinned to a key this machine has recorded

**Date:** 2026-09-20 · **Status:** active · **Areas:** trust, transport, identity

**Decision.** Every peer connection is TLS with a required client certificate, over every
transport including the overlay. Each side accepts only a key this machine has recorded: the
named peer's key when verifying, and any recorded peer's key when syncing. No setting turns
the encryption off.

**Support.**
- Room content is a person's prompts and whatever their Claude said back, so confidentiality
  is a requirement, and a signature proves authorship and does nothing to stop a third party
  reading. §25 (security).
- A peer's identifier is an Ed25519 key confirmed by the two-word comparison, so there
  is no stranger for an authority to vouch for. The check disables the authority machinery and
  runs one of its own, whether the key in the certificate is the key known, which is
  stricter than the machinery it leaves out. `cmd/cogmer/peertls.go`, `pinnedVerifier`.
- The certificate is a container that the format requires, generated self-signed at startup
  from the identity key, and the pin is on the key inside it, so regenerating it costs nothing
  and its expiry is meaningless. `cmd/cogmer/peertls.go`.
- A client certificate is required, since a peer with nothing to pin has no business
  completing a handshake and refusing there is earlier and clearer than refusing after a body
  is read. `cmd/cogmer/peertls.go`.
- The test is that the key is recorded and not that it is verified. Verification happens over
  this connection, so requiring it to connect would make the only route to verification
  unreachable. D-054 (verification gates synchronization and injection) stops an unverified
  peer's events a layer up. `cmd/cogmer/peertls_test.go`, `TestARecordedButUnverifiedPeerCanConnect`.
- When verifying, the named peer's key is demanded, since an address can change hands and
  without the demand a ceremony could begin with a stranger. When syncing, a room stores
  addresses and not a map from member to address, so any recorded peer is accepted, and the
  signature inside the request says which one. D-044 (sync requests are signed),
  `cmd/cogmer/peertls_test.go`, `TestAnAddressAnsweringForAnotherKeyIsRefused`.
- Go decides whether to encrypt by the URL scheme and not by the TLS config, so peer URLs are
  `https://peer/...`, with a placeholder host that is never resolved, since a change to
  `http://` would send every connection in plaintext and break nothing. `cmd/cogmer/transport.go`,
  `peerURL`, `cmd/cogmer/peertls_test.go`, `TestPeerURLsAreHTTPS`.
- Over the overlay, which is encrypted itself, TLS runs as well, so nobody has to reason about
  which dial took the safe path. `cmd/cogmer/transport.go`.
- A stranger completes no handshake. `cmd/cogmer/peertls_test.go`, `TestAStrangerCompletesNoHandshake`.

**Rejected.**
- *A key exchange built directly on the identity keys.* It avoids a reviewed implementation and
  not certificates, and means writing a handshake, nonce discipline, a replay window and
  rekeying, where the primitives are sound and the composition fails silently.
- *Pinning to verified keys.* It deadlocks, since nobody could ever be verified.
- *A guard that refuses bare-TCP connections to other machines, with a setting to turn it off.*
  A bare TCP connection carries a TLS session, so the guard would refuse the connections TLS made
  safe, and a security property with an off switch is one somebody eventually switches off.

**Limits.** The encryption protects content in transit and nothing at rest, since events are
stored unencrypted in each machine's room database, and it does not protect a room from its own
members. §25 (security).
