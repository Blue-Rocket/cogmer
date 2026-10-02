# D-041 — Ship as a Claude Code plugin

**Date:** 2026-09-17 · **Status:** active · **Areas:** release, commands

**Decision.** cogmer is packaged as a Claude Code plugin that carries its hooks and its
commands, installed by adding the marketplace and then the plugin.

**Support.**
- A user installs only what Claude Code loads as an extension. §3.8 (the host is
  launched and used unchanged).
- The repository is both the marketplace and the release host, so the two install lines
  name the same repository. D-121 (the repository is both the release host and the
  marketplace).

**Rejected.**
- *A list of the surfaces cogmer reaches, such as the terminal, the desktop application
  and the editor extensions.* The surfaces change faster than a specification does, and
  whether a surface runs hooks locally is answered by testing that surface.

**Revisit when** Claude Code offers a way to install that needs fewer steps.
