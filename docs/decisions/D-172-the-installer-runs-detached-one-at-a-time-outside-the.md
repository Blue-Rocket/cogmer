# D-172 — The installer runs detached, one at a time, outside the plugin directory, and waits an hour after a failure

**Date:** 2026-09-18 · **Status:** active · **Areas:** release, daemon

**Decision.** `plugin/hooks-handlers/install.sh` runs detached from the session-start hook
and installs into `~/.cogmer/bin`, outside the plugin directory. A `mkdir` lock lets one
installer run at a time, and the others exit without a message. After a failure it makes
no new attempt for an hour.

**Support.**
- A session is never delayed or made worse, so nothing waits on the download. A session that
  begins before the binary exists has nothing to inject yet, and the next has everything.
  §3.1 (first, do no harm), `plugin/hooks-handlers/install.sh`.
- An update of the plugin writes a new plugin directory, and the binary and the plugin
  change on different schedules, so a binary kept outside it survives an update.
  `plugin/hooks-handlers/install.sh`.
- Several sessions start at once on one machine, and two installers writing one path would
  leave a corrupt binary. `mkdir` is atomic on every platform, so it is the lock, and a
  loser leaves because being second is not an error. `plugin/hooks-handlers/install.sh`.
- A machine with no network and no Go would otherwise spend an attempt on every session it
  starts, and the recorded failure explains the wait. `plugin/hooks-handlers/install.sh`,
  `cooldown_seconds`.
- Five concurrent installs made one attempt, and a second run inside the cooldown
  added nothing to the log.
  `f288913:docs/decisions/D-066-the-binary-is-fetched-and-verified-never-shipped-in-the.md`.

**Rejected.**
- *Installing inside the plugin directory.* An update would discard a working binary and
  fetch it again.
- *Retrying on every session.* A machine that cannot install spends an attempt each time.
