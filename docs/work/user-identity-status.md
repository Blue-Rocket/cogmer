# Status: building the user identity

This is working material for the open item "A user's machines are separate identities to
colleagues, and a cloud session has no identity of its own to sign with" in `docs/open.md`,
and is deleted with it. It records where the build on the `user-identity` branch stands, so
that an interrupted session can resume from the pushed branch. The design is
`docs/work/user-identity-design.md`, and the brief is `docs/work/user-identity-brief.md`.

Merged: the design document.

In progress: the first run's foundation. One agent builds the phrase and the master key,
the records and the chain, and another builds the precedence rule and the cutoff.

Next: the second run, which starts only when the person who started the session says so.

The questions recorded are in `docs/open.md`, in the item that opens "Building the user
identity needs answers that no decision gives", and the reasoning for each choice is in
section 7 of the design.

Tests: `go test ./...` has not yet run on this branch. The writing and citation checks
pass.
