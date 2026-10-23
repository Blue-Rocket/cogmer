# Status: building the user identity

This is working material for the open item "A user's machines are separate identities to
colleagues, and a cloud session has no identity of its own to sign with" in `docs/open.md`,
and is deleted with it. It records where the build on the `user-identity` branch stands, so
that an interrupted session can resume from the pushed branch. The design is
`docs/work/user-identity-design.md`, and the brief is `docs/work/user-identity-brief.md`.

Merged: the design document, and the recovery phrase and master key in `userphrase.go`.
A phrase prints and marshals as a redacted text, and no parse error quotes a word. The
records and the chain are in `idrecords.go`, with frozen vectors in
`testdata/identity_vectors.json`. A record under a scheme this build does not know hashes to
zero, so `Resolve` uses a hash only after the record verifies.

In progress: the precedence rule and the cutoff in `idstate.go`, in one agent.

Next: the second run, which starts only when the person who started the session says so.

The questions recorded are in `docs/open.md`, in the item that opens "Building the user
identity needs answers that no decision gives", and the reasoning for each choice is in
section 7 of the design.

Tests: `go test ./...` passes on `beabb2a` except
`TestTheRealOverlayCheckPassesForALiveServerAndFailsForADeadOne`, which reaches the public
relay and timed out after 10s in this cloud container on 2026-10-23. That commit changes
only documents, so the test fails the same way on `main`. Its guard dials `tailcat.dev:443`
over TCP, which the container's proxy answers, while the relay's own protocol gets no answer
here. The build runs the suite with that one test skipped.
