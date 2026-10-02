# D-101 — Peer connections are TLS pinned to the key already verified

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asking what a `tc://` address contains led to noticing that `peerURL`
was `http://` and that no `https` appeared anywhere in the codebase.
Confidentiality of room content had never been stated as a requirement, and the
specification discussed it only for pairing strings — where the correct answer is
that none is needed, because the payload is public keys. Room content is a
person's prompts and whatever their Claude said back. Confidentiality is a
requirement.

### What was wrong

When two daemons talked, the request went out as ordinary HTTP over a TCP socket. A
`POST /sync` carrying a JSON body travelled as readable text, and anyone positioned
to carry those bytes — a router, an access point, anyone on the same segment — could
read the conversation.

Signing did not help, and it is worth being exact about why. A signature is a proof
of authorship: it lets a receiver confirm David wrote this event and that nobody
altered it on the way. It does nothing to stop a third party reading it. A signed
postcard is still a postcard.

### What TLS changes

Almost nothing about our HTTP. The routes are the same, the bodies are the same, the
status codes are the same. What changes is that before any of it, the two sides run
a handshake, and afterwards every byte travels inside an encrypted channel.

A dial now goes: open a TCP connection or a tailcat tunnel, which are the same from
here; handshake, in which each side sends a certificate, checks the other's, proves
it holds the matching private key, and agrees an encryption key for this connection
only; then `POST /sync` down that channel; then the response back the same way.
Nothing above the socket changed — `handleSync` still receives an ordinary
`http.Request`.

This happens over **every** transport, including the overlay, which is already
encrypted. The duplication is cheap and it buys the property that matters: nobody
has to reason about whether a particular dial took the safe path.

### What "pinned" means, and why there is a certificate at all

A browser connecting to a bank asks whether the certificate was signed by an
authority it trusts and whether the name matches the site it asked for. That
machinery — authorities, trust stores, expiry, revocation — exists because the
browser has never met that bank and needs a third party to vouch for it.

This is the opposite situation. We already know exactly which key the other machine
holds: a `peerId` **is** an Ed25519 public key (D-042), already confirmed to be that
person's by the two-word comparison (D-055). There is no stranger to vouch for.

So the authority machinery is skipped and one question is asked instead: is the key
inside this certificate the key we already know? That is what `InsecureSkipVerify`
together with a custom `VerifyPeerCertificate` does. The name misleads — it disables
the checks that assume an authority, and our own check runs in their place. It is
stricter than what it replaces, not weaker: an authority will vouch for millions of
keys, and this machine accepts about five.

There is a certificate at all only because the TLS wire format requires one. It is a
container. Each daemon generates a self-signed one at startup from its identity key,
and nobody pins the certificate — they pin the key inside it. That is why
regenerating it on every restart costs nothing, and why its expiry date is
meaningless here.

A client certificate is **required** rather than requested. A peer with nothing to
pin has no business completing a handshake, and refusing there is earlier and
clearer than refusing after a body has been read.

**The test is that the key is recorded, not that it is verified**, and that is not a
weakening. Verification happens *over* this connection: it is how a recorded peer
becomes a verified one. Requiring verification to connect would make the only route
to verification unreachable, so nobody could ever be verified and pairing would
never complete. Tightening this check would read as an obvious hardening and is a
deadlock.

An unverified peer connecting is harmless because the gate that matters is a layer
up: D-054 refuses its events in both directions, so it may complete a handshake and
a ceremony and still move no transcript until a person has compared two words.

Note that being recorded is per machine. Running `/peer-pair` records the colleague
on *this* machine and puts nothing on theirs, which is why the person who types
first cannot connect until the other types too — not because the pin is strict, but
because the far side has no record of them yet. The attempt retries, so typing a
few seconds early is ordinary rather than an error.

### The two situations a dial can be in

**Verifying a specific person.** We know exactly who we are calling: the ceremony is
with David and we hold David's key, so we demand that key. A machine at that address
presenting anything else is refused. This matters because an address can quietly
change hands — a home address gets reassigned, somebody else's machine answers — and
without the demand we would begin a ceremony with a stranger.

**Syncing a room.** We do not know who will answer. A room stores the addresses
where its members listen, not a mapping from member to address, so there is no
particular key to demand. Any key belonging to a peer this machine knows is
accepted, and everybody else refused. Identity is still pinned down a moment later,
because the request inside the connection is signed and the signature says exactly
who sent it (D-044). The handshake narrows it to "a known peer"; the signature makes
it "this peer".

### Why the URL scheme is load-bearing

Go's HTTP client decides whether to encrypt by looking at the URL scheme, not by
whether a TLS config was supplied. An `http://` URL opens the connection and starts
speaking HTTP immediately, ignoring `TLSClientConfig` entirely. An `https://` URL
performs the handshake with that config first and speaks HTTP inside the result.

Peer URLs are `https://peer/sync`. The hostname is a placeholder that is never
resolved, because `clientFor` installs a `DialContext` that ignores the address in
the URL and calls our own dialler. The scheme does one job: it tells Go to run the
handshake.

Changing it back to `http://` would send every connection in plaintext, leave the
certificate config silently unused, and **break nothing** — sync would work
perfectly and unencrypted. That is the worst failure shape available, so a test
asserts the scheme on its own.

### Why this superseded the interim guard

Before TLS existed, a rule was added refusing any bare-TCP connection to an address
not on this machine, on the reasoning that bare TCP carried cleartext and loopback
was the only safe case — an SSH tunnel makes both ends look like loopback and
encrypts between them.

That rule was about a property of bare TCP: that it is unencrypted. The property no
longer holds, because a bare TCP connection now carries a TLS session.

Keeping it would have been actively wrong rather than merely redundant. It would
have gone on refusing exactly the connections TLS had made safe, so building TLS
would have bought nothing — peers on different networks would still be blocked, and
the only way through would have been the escape hatch, which turns the check off for
everybody. The guard went, and the environment variable with it. There is now no
setting that disables encryption, which is the point: a security property with an
off switch is one somebody eventually switches off.

### Considered and rejected: deriving a key exchange from the same keys

The keys are already there, so an exchange could be built directly on them and
certificates avoided. Rejected, because what that avoids is not certificates but a
reviewed implementation: it means writing a handshake, nonce discipline, a replay
window and rekeying, where the primitives are sound and the composition is what
fails silently. Pinning already achieves the property the idea was reaching for —
that the transport's identity and the event's identity are the same key, confirmed
by the same comparison.

### What this does not do

It protects content **in transit only**. Events are stored unencrypted in each
machine's room database, so anyone who can read the disk can read the rooms.

It does nothing to protect a room from its own members. A peer admitted to a room
may read it; that is what admission means. This stops outsiders carrying the bytes,
not insiders holding them.

### A note for whenever a second person runs this

The handshake is a hard version boundary. A peer built before this cannot connect at
all, rather than degrading to something that works less well. Nobody else has run
it, so the cost is zero now and would not be later.

**Revisit when:** content needs protecting from a room's own members, or at rest.
