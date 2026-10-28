# D-271 — A command file is the same on every system and calls the dock by bare name

**Date:** 2026-10-28 · **Status:** not built · **Areas:** commands

**Decision.** A command file has no `shell:` field and one line, `cogmer-dock <subcommand>`, with an `allowed-tools` entry for each shell that names that one subcommand: `Bash(cogmer-dock <subcommand> *)` and `PowerShell(cogmer-dock <subcommand> *)`. Claude Code runs the line through Bash where one is available and through PowerShell otherwise, and each shell finds the dock in the plugin's `bin/` folder by bare name. A scoped rule matched only a line that named the program without `&`, so no line uses a path or the call operator, and one file serves macOS, Linux and Windows with or without Git. D-272 (a dock in the plugin's bin folder passes calls to the installed program) says what the dock is.

**Rejected.**
- *`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" <subcommand>`, as D-263 has it.* Windows without Git has no `bash`, and the command prints nothing and says nothing. `0f2d509`.
- *A command file for each system.* The plugin manifest reference read on 2026-10-28 has no key that selects a command file by platform.
- *An unscoped `PowerShell` rule.* It lets the model run any PowerShell command for the turn, where each command file allows the one subcommand it calls.
- *A quoted full path called with `&`.* A scoped rule did not match a line that used `&`, with a path or a name, in six patterns, and a full path with no `&` ran only where the path held no space. `0f2d509`, `8c07d61`.

**Limits.** A file of this form ran on macOS, Linux and Windows, with and without Git, under `claude -p` with `--plugin-dir` (`12b435d`). Its line depends on the dock's execute bit on macOS and Linux (D-275), which D-263 (a command file runs the plugin script through bash) avoids by running `bash` on the script, so the two cannot govern one command file. Codex's skill and the Cowork copies of the commands are not decided here.

**Revisit when** a Claude Code release lets a scoped rule match a quoted path, or an install route is found that drops the execute bit.
