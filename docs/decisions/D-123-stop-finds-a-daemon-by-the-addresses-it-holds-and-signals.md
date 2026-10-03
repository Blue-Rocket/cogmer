# D-123 — `stop` finds a daemon by the addresses it holds, and signals only cogmer

**Date:** 2026-09-22 · **Status:** active · **Areas:** daemon

**Decision.** `cogmer stop` takes the addresses this installation uses, the hooks address and the
peer-sync address, finds the process listening on each, and stops it if and only if it is a cogmer
daemon, with SIGTERM, a few seconds' wait and then SIGKILL. It reports what it stopped with the pid,
and what it declined to stop with the process's name and pid. The daemon handles SIGTERM by closing its
listeners and stores. The blocked-address message names `stop` by its full path, from `invocation()`.

**Support.**
- The daemon in the way is by definition the one holding the address, and it need not be this
  installation's, so finding it by address is what clears a daemon left from another state directory.
  §29 (starting the daemon), `cmd/cogmer/main.go`, `runStop`.
- A process is identified as cogmer when its executable is named `cogmer`, so a build renamed by hand is
  not stopped, which is the safe way for the check to be wrong. `cmd/cogmer/stop_test.go`,
  `TestStopDeclinesAProcessNotNamedCogmer`.
- Being killed outright is survivable because durability precedes publication, so the handler buys a
  clean exit and not correctness. §23 (durability and recovery), `cmd/cogmer/main.go`.
- Signaling a process needs `syscall` and finding one needs `os/exec`, and neither may appear in the
  code that handles peer traffic, so `stop` and the handler live in `main.go`, and the calls that find
  a port's process and signal it live in `stop_unix.go`, with `stop_windows.go` reporting that finding a
  port's process is not supported there. D-146 (code that handles peer traffic cannot start a process),
  `cmd/cogmer/stop_unix.go`.
- The report of a blocked address finds its holder the same way, so a cogmer daemon left running
  from another state directory is called one and `stop` is named, where an address alone cannot
  say what holds it, and a holder that cannot be found is reported as unknown and never as a
  program that is not cogmer. `cmd/cogmer/main.go`, `describeBlock`, and
  `cmd/cogmer/stop_test.go`, `TestADaemonBlockedByAnotherCogmerSaysSoAndNamesStop`.

**Rejected.**
- *An HTTP shutdown route.* It reaches the daemon at this installation's hooks address, which may be a
  different daemon from the one in the way, and it is one more state-changing route reachable from this
  machine's browser, for an operation a terminal has. D-087 (the local API requires a header
  that a web page cannot send).
- *A pid file.* A daemon writes it into its own state directory, so it finds this installation's daemon
  and never the stray from another one.
- *Stopping every process named `cogmer`.* Two daemons with separate state directories and addresses can
  run on one machine without touching each other, and stopping one that holds none of our addresses
  would break sessions that were working.

**Revisit when** Windows can find the process listening on a port. A daemon of another version is
stopped through this path, by D-158 (a daemon of another version gives way to the one the hook starts).
