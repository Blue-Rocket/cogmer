# D-044 — Sync requests are signed; authentication is not admission

**Date:** 2026-09-17 · **Status:** active · **Areas:** sync, identity, admission

**Decision.** Every sync request carries its own proof of who sent it, so that no exchange comes before it and no session is held between requests: it carries the caller's identifier, a timestamp and a nonce, signed with a purpose tag by the machine making it. The receiver checks the signature against that machine's key, whose certificates must end at the master key the identifier names (D-219, a user is a master key that certifies machines through a shared key), and refuses a timestamp more than two minutes off or a nonce it has seen, which together refuse a replayed request (`authTolerance` in `cmd/cogmer/auth.go`). Machines poll every second (§16, propagation), where a handshake to authenticate each poll would cost two round trips a second, and the receiver holds nothing between requests but the nonces of the last two minutes. When the connection carries a client certificate, the key that signed the request must also be the certificate's key (D-143, admission is proved by a signed request bound to the connection). Each request is checked in full whether its connection is new or kept from an earlier poll (D-303, a machine keeps one connection to each address it polls).

The window is two minutes because two machines synchronized by NTP were 408ms apart (`84a0751:docs/decisions.md`), so it allows clocks much further apart while keeping the set of seen nonces small. The list of what the caller holds is not signed, since a caller that altered it would gain nothing, as it may ask for everything (`cmd/cogmer/auth.go`).

**Rejected.**
- *Logging in once and presenting a token.* A machine would prove who it is once, by answering a challenge the receiver issues, and then send a token in each request that the receiver remembers as belonging to it. Both machines would then hold the session, and either one losing it, as a daemon does when it restarts, would leave them disagreeing until they log in again, and the challenge adds a round trip.

**Limits.** Signing establishes who is asking and never whether they may: a stranger who generated a key pair authenticated correctly and read a private room until admission existed. Admission is D-045 (a sync request is refused unless its peer is a guest).

**Revisit when** clocks between machines are found to differ by more than two minutes.
