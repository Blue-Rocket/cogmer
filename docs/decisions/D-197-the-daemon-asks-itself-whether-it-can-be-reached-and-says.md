# D-197 — The daemon asks itself whether it can be reached, and says when that changes

**Date:** 2026-10-12 · **Status:** active · **Areas:** transport, daemon, commands

**Decision.** While it runs, a daemon with an overlay reads the relay it is attached to every minute and
makes a connection to its own address through the relay every twelve minutes, with a random offset. A failed
connection is tried again 30 seconds later and counts only if it fails again. The daemon logs when it stops
being reachable, at a slow repeat while that continues, and when it is reachable again, and
`/cogmer:room-status` and `/cogmer:self-status` say so and for how long. The daemon restarts nothing.

**Support.**
- A colleague's daemon, three weeks old, could not be reached through the relay for sixteen minutes while it
  could reach others, and nothing on either machine said so until a person went looking. It was reachable
  again two seconds after a session started a new one. D-196 (an overlay address is reachable when a connection from
  outside gets through it), `cmd/cogmer/ui.go`, `RunSelfWatch`.
- Reading the attached relay opens no connection, so it can be asked often. A connection through the relay is
  one more from this machine to a free, rate-limited service, so it is asked rarely. D-192 (the public relay
  is an accepted dependency, and `plugin/README.md` discloses it), `cmd/cogmer/tailcat.go`, `newRelayProbe`.
- A laptop that has just woken fails a check once and is fine, so one failure is asked again before it counts.
  A relay missing for two reads running is the same, so it counts at the third. `cmd/cogmer/selfwatch_test.go`,
  `TestOneFailedCheckIsConfirmedBeforeItCounts` and `TestAMissingRelayIsNoticedAfterSeveralChecks`.
- A message repeated at every check is not a signal, so a failure is said when it begins, at the slow repeat that an
  unreachable machine's state uses, and when it ends. `cmd/cogmer/ui.go`, `downRepeat`, `cmd/cogmer/selfwatch_test.go`,
  `TestAConfirmedFailureIsReportedOnceAndItsEndIsSaid`.
- Machines pull, so a machine nobody can reach hands its turns to nobody, and the room status leads with that
  before it lists colleagues. D-194 (a colleague this machine cannot reach is named, with how long, in the room
  status and the view), `cmd/cogmer/main.go`, `renderGuests`,
  `cmd/cogmer/selfwatch_test.go`, `TestAnUnreachableMachineIsSaidFirstInTheRoomStatusAndInSelfStatus`.
- Breaking the confirmation, the repeat, the three-read threshold, the timer, the recovery line and the status
  line each failed a test. `cmd/cogmer/selfwatch_test.go`.

**Rejected.**
- *Restarting the overlay when the check fails.* Starting a whole new daemon is what recovered the failure
  seen, and nothing shows that restarting only the overlay would, so a person decides, with
  `/cogmer:self-restart`.
- *Checking only when a person asks.* That is how a failure goes unseen until somebody is missing
  turns.

**Limits.** The connection check runs every twelve minutes, so a failure that leaves the relay attached can go
unnoticed for about that long and half a minute more. What the watcher knows lives in memory, so a restart
begins with no history. The check dials this machine's own address, and a network that blocks that route could
fail it while colleagues connect, as D-196 records.

**Revisit when** an overlay restart is shown to recover a daemon that fails the check, since the daemon could
then do it without being asked.
