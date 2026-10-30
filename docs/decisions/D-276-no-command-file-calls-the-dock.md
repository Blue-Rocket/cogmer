# D-276 — No command file calls the dock

**Date:** 2026-10-29 · **Status:** active · **Areas:** commands, hosts

**Decision.** A command file runs the plugin script through bash on every system (D-263, a command file runs the plugin script through bash), and none calls the dock (D-272, a dock in the plugin's scripts folder passes hook calls to the installed program). A call to the dock from a command file needs a bare name or a path. A bare name needs the dock in a top-level `bin/` folder, which claude.ai rejects, and a full path ran under a scoped PowerShell rule only where it held no space and no `&`, while a Windows profile path can hold a space.

**Rejected.**
- *A command file that calls the dock by bare name.* It needs the plugin's `bin/` folder on the PATH, and claude.ai rejects a plugin that has one (Claude Code's plugin reference and claude.ai's organization sync page, read on 2026-10-28 and 2026-10-29). One command file of that form, with no `shell:` field and a `Bash` and a `PowerShell` rule, ran on macOS, Linux and Windows with and without Git under `claude -p` with `--plugin-dir`. `12b435d`.
- *A quoted full path called with `&`.* A scoped rule did not match a line that used `&`, with a path or a name, in six patterns, and a full path with no `&` ran only where the path held no space. `0f2d509`, `8c07d61`.
- *An unscoped `PowerShell` rule.* It lets the model run any PowerShell command for the turn, where each command file allows the one subcommand it calls.
- *A command file for each system.* The plugin manifest reference read on 2026-10-28 has no key that selects a command file by platform.

**Limits.** This decides no route for a slash command in a Windows session without Git. The Codex skill and the Cowork copies of the commands are not decided here.

**Revisit when** claude.ai accepts a plugin with a top-level `bin/` folder, or a Claude Code release lets a scoped PowerShell rule match a quoted path.
