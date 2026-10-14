# D-210 — The daemon says where a connection to a peer fails, and discards a connection that is stuck

**Date:** 2026-10-14 · **Status:** active · **Areas:** transport, daemon

**Decision.** When a peer reached over the overlay has not answered, the daemon logs at the start of the outage and at each repeat where the connection fails: the relay the peer's address pins, a relay ping on the connection the daemon holds, a relay ping on a new connection, and a path-discovery ping, with a reading of the three. After the peer has failed for three minutes, and at most every five, the connection held to it is discarded and built again at the next poll. `cogmer diagnose-endpoint` runs the same diagnosis for one endpoint.

**Support.**
- A failed poll logs its error and nothing about where the connection stalls, and the error is the request's timeout, which is the same whether the relay does not know the peer, the tunnel does not complete or the peer's server does not answer. `cmd/cogmer/ui.go`, `markPeerUnreachable`.
- The overlay library offers a ping through the relay that waits for the peer's answer, a path-discovery ping that says whether traffic goes direct or by relay, and a reader for the relay an address names. `cmd/cogmer/tailcat.go`, `Diagnose`.
- A connection to a peer is built once and kept, since building one takes seconds, so one that stuck would fail for as long as the daemon ran. `cmd/cogmer/tailcat.go`, `tailcatDialer`, `Reset`.
- The diagnosis runs in the background and for one address at a time, since it waits on the network, and a held connection that is discarded is built again at the next poll. `cmd/cogmer/peerdiagnosis_test.go`, `TestAPeerThatStopsAnsweringIsDiagnosedAtTheStartAndAtEachRepeat`, `TestAConnectionHeldToAPeerThatKeepsFailingIsDiscarded`.

**Rejected.**
- *Discarding the connection at every failed poll.* A first connection costs seconds, so the poll would never finish paying it. `cmd/cogmer/tailcat.go`, `tailcatDialer`.
- *Diagnosing inside the poll.* It waits up to fifteen seconds on the network, which would hold every poll behind it.

**Limits.** Only the overlay has a diagnosis, and a plain TCP address has neither. The reading is an inference from three pings, and it says nothing of the peer's own dial to this machine, which that peer's daemon diagnoses. Whether discarding a held connection recovers a peer that has stopped answering has not been observed.

**Revisit when** a log shows a discarded connection recovering a peer, or failing to, or the diagnosis proves too slow or too noisy.
