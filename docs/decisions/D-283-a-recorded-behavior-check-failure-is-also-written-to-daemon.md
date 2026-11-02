# D-283 — A recorded behavior-check failure is also written to daemon.log by the process that ran the check

**Date:** 2026-11-02 · **Status:** not built · **Areas:** behaviors, daemon

**Decision.** When a behavior check records a failure for a host version, or records that its probe could not run (D-129, behavior checks run by themselves when a room is formed), the process that ran the check appends one line to `daemon.log`, holding the time, the host and its version, the behavior's identifier and title, and the error. The line is written when the result is recorded, and a later reminder that the version had failed writes none. The process opens the file for appending with mode 0600 and writes the line itself, since the check runs in a command's process, a daemon may not be running, and the log has other appending writers, which its bounding survives (D-261, the daemon bounds its log by copying it aside and emptying it in place). A failure recorded nowhere that anybody looks gives the person debugging no next question, which is why output goes to a log by default (D-084, output goes to the log by default, and never to stdout).

**Rejected.**
- *Writing the line through the daemon.* A check runs when a daemon may be absent or failing to bind, and a request to it would fail at the moment a failure most needs recording. Hooks and commands fail open around the daemon (D-011, hooks fail open: exit 0, empty stdout).
- *A line at each later room that reminds the user the version had failed.* The failure is recorded once, and a line for each reminder fills a bounded log with the same fact.

**Limits.** A line written between the daemon's copy of the log and its emptying is lost, which D-261 accepts for a few lines in a minute. `verified.json` is what later checks and delivery read. A probe that could not run is retried after an hour, so it writes at most one line an hour.

**Revisit when** a failed check is found that left no line in `daemon.log`.
