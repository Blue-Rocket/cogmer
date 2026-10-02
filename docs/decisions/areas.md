# Areas

Each active decision names the areas it belongs to on its date line, and a decision may
belong to more than one. These are the areas a decision may name, and what each covers.

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
