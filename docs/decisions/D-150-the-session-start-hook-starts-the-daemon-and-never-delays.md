# D-150 — The session-start hook starts the daemon, and never delays the session

**Date:** 2026-09-17 · **Status:** active

**Decision.** The session-start hook starts the daemon when it is not running. Starting it
never delays the session, finding it running is the ordinary case, and a failure
to start is silent to the user and recorded by the daemon.

**Support.**
- A user never has to start the daemon, notice that it has stopped, or know that it
  exists. §29 (the experience we want).
- Several sessions often begin at once on one machine, each tries to start the daemon,
  and at most one succeeds. §29.
- A session is never worse for having cogmer installed. §3.1 (first, do no harm).

**Rejected.**
- *The user starting the daemon by hand.* Installing would not be the whole of setup.

**Revisit when** starting the daemon is found to delay a session.
