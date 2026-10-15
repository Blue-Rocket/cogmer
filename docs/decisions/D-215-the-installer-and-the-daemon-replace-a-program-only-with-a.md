# D-215 — The installer and the daemon replace a program only with a newer version

**Date:** 2026-10-15 · **Status:** active · **Areas:** daemon, release

**Decision.** The installer puts a program on a machine only when the program there is older than the one
its plugin expects, and a running daemon gives way only to a newer program. Two copies of the plugin on one
machine, such as one from a terminal and one from the desktop app's Plugins page, then leave the newest
program in place whichever starts a session. Versions are compared as dotted numbers, so 0.10.2 is newer
than 0.9.1.

**Rejected.**
- *Replacing the program or the daemon on any difference in version.* Two copies of the plugin moved the
  program back and forth at every session, as the install logs of two machines show.
- *Rolling back by installing an older version.* A bad release is withdrawn by releasing a newer version with
  the earlier code, since a version number only goes up.

**Revisit when** a version is released that is not dotted numbers.
