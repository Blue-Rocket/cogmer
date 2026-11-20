# D-291 — A command gives the model its shell lines to run when the host did not run them

**Date:** 2026-11-20 · **Status:** active · **Areas:** commands, hosts

**Decision.** Each file in `plugin/commands/` repeats its shell lines, in order, in a section before its task, and tells the model to run them with the Bash tool, once, when the section that should hold cogmer's output holds a note or the line itself instead, and to run nothing when the output is there. A Cowork cloud session showed, in place of a typed command's shell line, "[a shell command here was not run: typing a command's name runs none]", and the model then found `cli.sh` and ran it, which answered as the command would have (`578833b`, `ffe2f33`). With shell lines disabled, Claude Code 2.1.328 filled in the plugin's root and the arguments in the rest of a command's text, and the model ran the repeated line once; with them enabled it ran nothing (`578833b`). A command keeps `disable-model-invocation: true` (D-119, no slash command is reachable by the model), so the model reads these lines only after the user typed the command, and running them carries out the user's request.

**Rejected.**
- *Commands as tools of an MCP server that the plugin bundles.* It needs the server, a test of each tool and a check that the server loads, and a tool the model can call on its own is the route D-119 closes.
- *Matching the host's note word for word.* The note differs between hosts and versions: a local Cowork session showed "[shell command execution disabled by policy]" (`e7ddab7`).

**Limits.** The model runs the lines in its own shell, so the command works only where that shell reaches the daemon: in a Cowork cloud session both are in the sandbox, and in a Cowork session on the user's computer the model's shell ran in a sandbox with no cogmer that could not reach the computer's daemon (`e7ddab7`). A command binds the session named by `CLAUDE_CODE_SESSION_ID` in the shell's environment, which no run has read in a Cowork cloud session's Bash tool. The user reads the model's account of the output rather than output the command put in the prompt.
