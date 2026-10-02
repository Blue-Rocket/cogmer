# D-037 — Claude Code is launched and used unchanged

**Date:** 2026-09-17 · **Status:** active · **Areas:** hosts, project

**Decision.** A user starts and uses Claude Code the same way they would without cogmer,
and installs only what Claude Code loads as an extension: hooks, skills, MCP servers and
the plugin that carries them. A proposal that needs Claude Code started differently, an
artifact it does not load, or knowledge of how it renders is out.

**Support.**
- Claude Code is a terminal program, a desktop application and an editor extension, and a
  system that extends it through its own mechanisms works in all of them. §3.8 (the host
  is launched and used unchanged).
- Every extension point Claude Code offers delivers to the model: hooks supply context,
  skills supply instructions and MCP servers supply capability. D-033 (the room cannot be
  displayed inside Claude Code), and D-036 (MCP logging notifications are not a display
  channel).

**Rejected.**
- *Searching Claude Code for an undocumented way to display something.* The installed
  artifact is a native binary with no source to read, and a seam found that way would be
  undocumented, unversioned, and beyond what the behavior registry can check, since
  whether something appears on screen needs somebody watching.

**Revisit when** Claude Code documents a way for an extension to show something to a
user.
