# D-082 — The daemon opens the view itself, from a detached process

**Date:** 2026-09-20 · **Status:** active · **Areas:** daemon, view

**Decision.** The daemon reaches a person by opening the view with the operating system's
own opener, `open` on macOS, from the detached process the session-start hook starts. It
does not write to the terminal and it does not ship an application bundle.

**Support.**
- A process started the way the session-start hook starts the daemon, in the background
  with no controlling terminal, can open a page and the browser comes to the front, and
  a fresh address opens a fresh tab. B23 (a process the daemon's shape keeps the GUI session,
  so it can open the view and notify), `cmd/cogmer/browser.go`, `openInBrowser`.
- The Mach bootstrap namespace is inherited apart from the POSIX session, so detaching the
  daemon into a new session costs nothing, and the `setsid` branch in
  `start_daemon_if_needed` is not a hazard. `f288913:docs/decisions/D-082-the-daemon-can-reach-a-person-directly-there-is-no-setsid.md`.

**Rejected.**
- *Writing to the terminal.* A hook can walk to the `claude` process and read its tty, but
  the title is contended with Claude Code, a write can splice into the middle of a sequence
  Claude Code is emitting and corrupt a live session, and nothing there can carry something
  clickable. A channel that can only say that something happened is not worth that risk.
- *An application bundle.* It would give a notification identity, a click that opens the
  view and a findable icon, none of which is needed while plain `open` works, and it would
  have to be created at install time and registered with the system.

**Revisit when** B23 fails, or a notification needs to carry an action.
