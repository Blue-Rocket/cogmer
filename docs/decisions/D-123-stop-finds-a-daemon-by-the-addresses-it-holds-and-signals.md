# D-123 — `stop` finds a daemon by the addresses it holds, and signals only cogmer

**Date:** 2026-09-22 · **Status:** active (implemented, except the blocked-address message)

**Context.** D-151 (the daemon outlives its session, and can be found and stopped)
required the daemon to be discoverable and stoppable, and D-080 (a terminal command
exists to be tested or to work when the plugin cannot) put its lifecycle at the
terminal. Nothing stops it. On 09-22 that
cost a first run: a daemon left from a test, with its own `COGMER_HOME` and its
hooks moved to 4799 but its peer sync on the default 4783, held that port when the
real daemon started. The real one exited with "held by something that is not a
cogmer daemon", and clearing it took finding the process and killing it at a
terminal.

**Decision.** `cogmer stop` takes the addresses this installation uses, the hooks
address and the peer-sync address, finds the process listening on each, and stops
it if and only if it is a cogmer daemon: SIGTERM, a few seconds' wait, then SIGKILL.
It reports what it stopped, with the pid, and what it declined to stop, with the
process's name and pid. The daemon handles SIGTERM by closing its listeners and
stores. Being killed outright is already survivable, because durability precedes
publication (§23), so the handler buys a clean exit, not correctness. The
blocked-address message names `stop` by its full path, from `invocation()`, in
place of the `lsof` line.

**Finding a daemon by address is the point.** The daemon in the way is, by
definition, the one holding the address. It need not be this installation's, and on
09-22 it was not.

**Identified as cogmer** means the listening process's executable is named
`cogmer`. A build renamed by hand is then not stopped, which is the safe way for
that check to be wrong.

**Where the code lives.** Signalling a process needs `syscall`, and finding one
needs `os/exec` for `lsof`. §3.7 (a remote event never drives an interactive
session) keeps both out of `daemon.go`, `store.go`, `sync.go` and `transcript.go`,
and a test enforces it. `stop` and the SIGTERM handler live in `main.go`, beside
`runDaemon`. The calls to the operating system that find a port's process and signal
it live in `stop_unix.go`, because `syscall.Kill` does not build for Windows, and
`stop_windows.go` reports that finding a port's process is not supported there.

**Rejected.**
- *An HTTP shutdown route.* It reaches the daemon at this installation's hooks
  address, which on 09-22 reached nothing: the real daemon had already exited, and
  the blocker's hooks were on 4799, so it would have kept 4783. It is also one more
  state-changing route reachable from this machine's browser (D-087, loopback is
  not a boundary), for an operation a terminal already has.
- *A pid file.* A daemon writes it into its own state directory, so it finds this
  installation's daemon and never the stray from another one, which is the case
  that happened.
- *Stopping every process named `cogmer`.* Two daemons with separate state
  directories and separate addresses can run on one machine without touching each
  other, and a second account on a shared machine runs its own. A daemon that is
  not holding one of our addresses is not in our way, and stopping it would break
  sessions that were working.

**Left open.** Whether `stop` then starts this installation's daemon, whether it
gets a slash command, and Windows, which has no `lsof`. All three are in `open.md`.

**Revisit when** Windows can find the process listening on a port. A daemon of
another version is stopped through this same path, by D-158 (a daemon of another
version gives way to the one the hook starts).
