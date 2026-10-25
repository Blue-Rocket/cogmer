# D-143 — Admission is proved by a signed request bound to the connection

**Date:** 2026-09-16 · **Status:** active · **Areas:** admission, identity, trust

**Decision.** A machine proves who is asking to enter a room with each request it makes,
signed by its own key over its user's identifier, a timestamp and a nonce, and carrying
the certificates that tie the key to the master key the identifier names (D-219, a user
is a master key that certifies machines through a shared key; D-044, sync requests are
signed). When the connection carries a client certificate, the key that signed the
request must be the certificate's key, so that a request captured on its way to one
machine cannot be presented to another within the timestamp's tolerance. Nothing secret
passes in either direction, so the proof needs integrity, which the signature supplies,
and no confidentiality (§12, forming a room).

**Rejected.**
- *A fresh challenge the host issues and the joining machine signs.* Machines poll, so a
  challenge adds a round trip to every request, and the timestamp, the nonce and the
  binding to the connection refuse a replayed request without one.
- *A proof that can be replayed.* An exchange that can be replayed is a bearer
  credential.

**Limits.** The proof says who is asking and never whether they may enter, which is the
guest list's question (D-045, rooms are records with a guest list).

**Revisit when** a request has to be accepted over a connection that carries no client
certificate.
