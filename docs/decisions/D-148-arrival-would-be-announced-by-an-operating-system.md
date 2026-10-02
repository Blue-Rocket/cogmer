# D-148 — Arrival would be announced by an operating-system notification

**Date:** 2026-09-17 · **Status:** not built · **Areas:** view

**Decision.** If a colleague's arrival is announced, the daemon raises an
operating-system notification itself, touching no Claude Code surface.

**Support.**
- Nothing Claude Code offers displays to a person. D-033 (the room cannot be displayed
  inside Claude Code), and D-036 (MCP logging notifications are not a display channel).
- A process the session-start hook starts in the background keeps the logged-in GUI
  session, so it can put something on screen. B23.

**Rejected.**
- *An undocumented display seam inside Claude Code.* It would be a worse dependency than
  a notification. D-037 (Claude Code is launched and used unchanged).

**Limits.** It does not decide whether arrival is announced at all.

**Revisit when** announcing a colleague's arrival is decided.
