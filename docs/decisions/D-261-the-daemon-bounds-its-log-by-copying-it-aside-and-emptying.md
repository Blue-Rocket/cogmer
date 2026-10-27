# D-261 — The daemon bounds its log by copying it aside and emptying it in place

**Date:** 2026-10-27 · **Status:** active · **Areas:** daemon, transport

**Decision.** Once a minute the daemon looks at `daemon.log`, and when the file is over 10MB it copies the file to `daemon.log.1`, replacing any older copy, and empties `daemon.log` where it is. The copy is written to a file of its own and renamed over the older one, with mode 0600. The log therefore holds at most about 10MB and its older copy about 10MB, and the lines written in the moments between the copy and the emptying are lost. The daemon gives the relay library's clients a logger that drops the library's lines beginning `netstack:`, `wg:`, `[v1]`, `magicsock:` and `netcheck:` and passes every other line to the log. The daemon's own lines, including the diagnosis lines (D-210, the daemon says where a connection to a peer fails), do not pass through that logger and are written whole.

**Rejected.**
- *Renaming `daemon.log` and opening a new file.* The shell redirect in `plugin/hooks-handlers/common.sh` and `startDaemon` open the file for appending and hand the descriptor to the daemon as its output, so a rename moves the name and leaves the daemon writing to the renamed file.
- *Silencing the library's clients altogether.* A line from the library that is not routine is evidence when a connection fails, and the log is read for that.

**Revisit when** a log read after a failed connection is missing the lines that explained it, because the library says it under one of the dropped prefixes or because the log was emptied just before.
