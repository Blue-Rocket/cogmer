# Decisions

Each file here records one decision as it stands now: what we do, the facts that
support it with their sources, the alternatives rejected and why, and the condition that
would reopen it. The rejected alternatives stop a later reader from proposing something
already ruled out. `docs/writing.md` gives the template.

A file is named for the decision's number and a slug of its title, such as
`D-048-verification-is-a-live-commit-then-reveal-exchange.md`. A new decision takes the
next number, and a number is never reused.

An entry's revisit condition names the automated check that would show it wrong, where
one exists: `cogmer doctor`, or an entry in the behaviour registry in
`cmd/cogmer/behaviors.go`. When that condition occurs, the decision is due for review.
It is not automatically wrong.

Each active decision names the areas it belongs to on its date line, and a decision may
belong to more than one. The areas are these.

| Area | What it covers |
|---|---|
| rooms | what a room is, its identifier and name, its life, and which session is in it |
| admission | who may enter a room: guest lists, invitations, joining and forgetting a peer |
| identity | a peer's key, its name, and what its signatures cover |
| pairing | pairing two machines and verifying a peer with the two words |
| sync | replicating events between peers: sequences, the wire format and recovery from loss |
| transport | addresses, reachability, the overlay and the connections between peers |
| capture | reading a session's transcript, delivery state, and injecting colleagues' turns |
| daemon | the local daemon and its hooks: starting, finding, stopping and failing open |
| view | the local HTTP API and the browser view |
| commands | slash commands, the plugin manifest and the product name |
| release | installing, versions and releasing |
| storage | the stored tables, their migration and their archive |
| behaviors | the behaviour registry and the checks that watch Claude Code |
| hosts | hosts other than Claude Code |
| trust | what untrusted input and a local web page may do |
| project | what cogmer is for, and the choices that shape all of it |

To list the active decisions in an area, search the date lines, such as
`rg -l 'Status:\*\* (active|not built).*Areas:.*\bpairing\b' docs/decisions`.
