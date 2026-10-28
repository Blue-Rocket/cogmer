# D-271 — A Windows command file calls the launcher by bare name under a scoped rule

**Date:** 2026-10-28 · **Status:** not built · **Areas:** commands

**Decision.** On Windows a Claude Code command file sets `shell: powershell`, calls the launcher by bare name, and carries an `allowed-tools` rule scoped to that name, `PowerShell(<name> *)`, so the line needs no `bash`, no quoted path and no change to the user's PATH. A scoped rule matched only a line that named the program without `&`, and the plugin's `bin/` folder is on the PowerShell tool's PATH, so the bare name resolves in a plugin path that holds a space. D-272 (a launcher in the plugin's bin folder starts the installed program) says what the launcher is.

**Rejected.**
- *An unscoped `PowerShell` rule.* It lets the model run any PowerShell command for the turn, where each command file allows the one program it calls.
- *A quoted full path called with `&`.* A scoped rule did not match a line that used `&`, with a path or a name, in six patterns, and a full path with no `&` ran only where the path held no space. `0f2d509`, `8c07d61`.

**Limits.** This was observed under `claude -p` with `--plugin-dir`, and not in an interactive session or for a plugin installed from a marketplace. It applies to Windows only. On other systems D-263 (a command file runs the plugin script through bash) governs, and a Cowork command file carries no shell line.

**Revisit when** a Claude Code release lets a scoped rule match a quoted path.
