# D-158 — A daemon of another version gives way to the one the hook starts

**Date:** 2026-10-01 · **Status:** active

**Decision.** When the daemon answering on this installation's hooks address
reports a version other than the binary's, the session-start hook starts the
binary, detached, and that daemon stops the running one and takes its addresses. A
daemon that reports `dev` never gives way, and a binary that is `dev` never takes over.

**Support.**
- After an update, the first session runs the new version, without delaying the
  session. §29 (the experience we want).
- `/healthz` reports the daemon's version, and a daemon built before it did reports
  none, which counts as another version. `health` in `cmd/cogmer/daemon.go`, and
  `daemonVersionAt` and `shouldReplace` in `cmd/cogmer/main.go`.
- The daemon in the way is found by the addresses it holds, and is stopped only if
  its executable is named `cogmer`. D-123 (`stop` finds a daemon by the addresses it
  holds, and signals only cogmer).
- The hook compares the two versions and returns when they match, and a start is
  detached, so a session waits only for the comparison, which asks the binary its
  version: 16ms on 10-01. `start_daemon_if_needed` in
  `plugin/hooks-handlers/common.sh`, and §29 (starting the daemon never delays the
  session).
- Several replacements started at once each stop whatever is in the way, one binds,
  and the rest find a daemon of their own version serving and leave it.
  `replaceDaemon` in `cmd/cogmer/main.go`.
- A replacement ends a running daemon of another version, and `stop` ends the
  replacement. `TestADaemonOfAnotherVersionIsReplaced` in `cmd/cogmer/stop_test.go`.

**Rejected.**
- *Replacing only an older daemon.* The plugin decides which version a machine
  runs, and moving it back is an update like any other, so any difference counts.
- *The hook stopping the daemon itself.* Stopping waits up to 5s for the old daemon
  to exit, and the hook runs before the session starts. §29 (starting the daemon never
  delays the session).
- *Replacing only when the installer has just fetched a binary.* A daemon of another
  version also outlives a binary installed by any other route, and the hook is what
  runs at every session.

**Limits.** Windows finds no process on a port, so a daemon of another version
keeps serving there until it is ended by hand. A maintainer's daemon built from
source reports `dev` and is left alone, as is an installed daemon while a `dev`
binary is the one the hook finds.

**Revisit when** two installations on one machine share addresses and run
different versions, since each would replace the other at every session.
