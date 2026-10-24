# D-194 — A colleague this machine cannot reach is named, with how long, in the room status and the view

**Date:** 2026-10-12 · **Status:** active · **Areas:** sync, commands, view

**Decision.** The daemon names each colleague it tries by the person, a label this user gave or
else the derived name, and records, for each of the colleague's machines, how long it has failed
to reach that machine and the last reason. `/cogmer:room-status` and the view show each colleague
as one line, reachable when any of their machines that joined the room can be reached from this
machine. Only when none can does the line say that the colleague is unreachable, for how long,
what that means for the room, and when they last made a request of this machine, and the view says
"unreachable for" and the time. The user's own machines are listed one by one, by name, with
whether this machine can reach each: those that joined the room in the room status, and all of
them in `/cogmer:self-status`.

**Support.**
- A colleague this machine cannot reach looks the same as a quiet room, and the only trace was a line in
  `daemon.log`. A guest saw only their own turns for over ten minutes, every pull timing out, with nothing
  said anywhere a person looks. §3.1 (first, do no harm), `cmd/cogmer/ui.go`, `peerStatus`.
- The daemon keeps its state per address, and a person is not an address, so the name comes from the user
  an address was recorded for. The view named a peer by 189 characters of address. D-103 (an address
  belongs to a machine and is stored in one place), `cmd/cogmer/membership.go`, `EndpointOwners`.
- A machine that answered in the last 15 seconds is online, so one failed poll does not read as an outage.
  `cmd/cogmer/reach_test.go`, `TestAColleagueIsNamedAndTimedWhileUnreachable`.
- A command is a separate process and the daemon is the one that tries, so the command asks the daemon,
  over a read route that needs no header, as the view's reads do. D-087 (the local API requires a header
  that a web page cannot send), `cmd/cogmer/main.go`, `fetchPeers`.
- A colleague who reached this machine a minute ago has a daemon that is running, so the fault is in
  reaching it, which is a different search from a daemon that was never started. The daemon records each
  authenticated request by the machine that made it, and an unsigned one that names a machine counts for
  nothing. `cmd/cogmer/sync.go`, `cmd/cogmer/selfcheck_test.go`,
  `TestOnlyAnAuthenticatedRequestCountsAsHearingFromAColleague`.
- The words a person is told, and that a reachable colleague is never called unreachable, are held by
  tests. `cmd/cogmer/reach_test.go`, `TestReachLineSaysWhetherAColleagueCanBeReached` and
  `TestTheGuestListSaysWhoCannotBeReachedAndWhatThatMeans`.

**Rejected.**
- *Leaving it in the log.* That is what hid it.
- *One line for each of a colleague's machines.* A colleague's turns reach this machine through any
  of their machines in the room, so a line for an unreachable laptop beside a reachable desktop
  reports a fault the user has no reason to act on.
- *The user's own machines as one line, as a colleague's are.* The user acts on their own machines,
  and needs to know which one to start or reconnect.

**Limits.** The state is this machine's view and lives in memory, so a restarted daemon begins with no
history. A colleague whose daemon can reach this machine and not the other way is reported unreachable
while their turns still arrive here, which is true of this machine's attempts and may read oddly.
