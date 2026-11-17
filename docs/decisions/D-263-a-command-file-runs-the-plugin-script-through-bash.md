# D-263 — A command file runs the plugin script through bash

**Date:** 2026-10-27 · **Status:** active · **Areas:** commands, release

**Decision.** Every shell line in a command file under `plugin/commands/` starts with `bash`,
as `bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" <subcommand>`, and the command's `allowed-tools` entry
is `Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" <subcommand>:*)`, naming that one subcommand.

**Support.**
- A plugin copied without the execute bit runs its commands, since `bash` reads the
  script as a file. A script run directly fails with "Permission denied" on every command,
  while the hooks, which run `bash <script>`, go on working and show nothing.
  `plugin/codex-cli.sh`, `plugin/hooks-handlers/`.
- In the default permission mode the pattern lets the subcommand run with its arguments, and
  refuses arguments that add a `;` command or a `$(...)` substitution. `plugin/commands/room-create.md`.
- Each shell line and its pattern name the same subcommand, and no pattern is wider than its
  line. `cmd/cogmer/command_test.go`, `TestCommandFilesRunCliThroughBashAndAllowOnlyTheirOwnSubcommands`.
- The Cowork copies carry no shell line, so they do not change. `cmd/cogmer/cowork_setup_test.go`,
  `TestNoCoworkCommandIsModelInvocableOrRunsAShell`.

**Rejected.**
- *Keeping the direct call and relying on the mode bit.* A cache or install that drops the bit
  breaks every command at once, and nothing in cogmer can restore it.
- *`Bash(bash:*)` as the pattern.* It would let the model run any script through `bash` without
  a prompt, which is wider than the one script each command runs.

**Revisit when** a Claude Code release lets a command file name its script without a shell line.
