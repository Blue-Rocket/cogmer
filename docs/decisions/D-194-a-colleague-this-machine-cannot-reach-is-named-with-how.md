# D-194 — A colleague this machine cannot reach is named, with how long, in the room status and the view

**Date:** 2026-10-12 · **Status:** active · **Areas:** sync, commands, view

**Decision.** The daemon names each peer it tries by the person, a label this user gave or else the
derived name, and records how long it has failed to reach them and the last reason. `/cogmer:room-status`
says, for each colleague, whether this machine can reach them, and for one it cannot, for how long, and
what that means for the room. The view says "unreachable for" and the time.

**Support.**
- A colleague this machine cannot reach looks the same as a quiet room, and the only trace was a line in
  `daemon.log`. A guest saw only their own turns for over ten minutes, every pull timing out, with nothing
  said anywhere a person looks. §3.1 (first, do no harm), `cmd/cogmer/ui.go`, `peerStatus`.
- The daemon keeps its state per address, and a person is not an address, so the name comes from the peer
  an address was recorded for. The view named a peer by 189 characters of address. D-103 (an address
  belongs to a peer, and is stored in one place), `cmd/cogmer/membership.go`, `EndpointOwners`.
- A peer that answered in the last 15 seconds is online, so one failed poll does not read as an outage.
  `cmd/cogmer/reach_test.go`, `TestAColleagueIsNamedAndTimedWhileUnreachable`.
- A command is a separate process and the daemon is the one that tries, so the command asks the daemon,
  over a read route that needs no header, as the view's reads do. D-087 (the local API requires a header
  that a web page cannot send), `cmd/cogmer/main.go`, `fetchPeers`.
- The words a person is told, and that a reachable colleague is never called unreachable, are held by
  tests. `cmd/cogmer/reach_test.go`, `TestReachLineSaysWhetherAColleagueCanBeReached` and
  `TestTheGuestListSaysWhoCannotBeReachedAndWhatThatMeans`.

**Rejected.**
- *Leaving it in the log.* That is what hid it.

**Limits.** The state is this machine's view and lives in memory, so a restarted daemon begins with no
history. A colleague whose daemon can reach this machine and not the other way is reported unreachable
while their turns still arrive here, which is true of this machine's attempts and may read oddly.
