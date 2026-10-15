# D-158 — An older daemon gives way to the program the hook starts

**Date:** 2026-10-01 · **Status:** active · **Areas:** daemon, release

**Decision.** When the daemon answering on this installation's hooks address is older than the program, or
reports no version, the session-start hook starts the program, detached, and that program stops the running
daemon and takes its addresses. A daemon or program that reports `dev` never gives way and never takes over,
because a maintainer runs it by hand. The hook only compares versions, so a session waits for nothing.

**Rejected.**
- *The hook stopping the daemon itself.* Stopping waits up to 5 seconds for the old daemon to exit, and the
  hook runs before the session starts.
- *Replacing only when the installer has just fetched a program.* A daemon of an older version also
  outlives a program installed any other way, and the hook runs at every session.

**Limits.** Windows ends a daemon through the lookup `stop` uses, D-123 (`stop` finds a daemon by the addresses
it holds, and signals only cogmer), and only `stop` has been run there.
