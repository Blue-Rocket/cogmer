# D-275 — On macOS and Linux the dock is a shell script and a test requires its execute bit

**Date:** 2026-10-28 · **Status:** not built · **Areas:** release, daemon

**Decision.** On macOS and Linux `plugin/scripts/cogmer-dock`, the dock of D-272 (a dock in the plugin's scripts folder passes hook calls to the installed program), is a `#!/bin/sh` script that picks a handler by its first argument and runs the existing scripts by `bash`, so those systems run what they run today. On Windows with Git the same script hands every call to `cogmer-dock.exe`, so Windows has one implementation whether or not Git is installed. A test requires the repository's mode for `plugin/scripts/cogmer-dock` to be 100755, because an exec-form hook fails without the execute bit.

**Rejected.**
- *A native dock for each system under one file name.* A hook line names one path, and four builds cannot share it.
- *Changing the macOS and Linux hooks to call the program's subcommands.* It changes hooks that work for every current user to give the release one implementation it does not need.
- *`bash` on the handler script in each hook line.* Windows without Git has no `bash`.

**Limits.** An exec-form hook whose script lacked the execute bit failed with `EACCES` on macOS and Linux and was reported as a hook error that did not block the turn (`5baeaea`). The cached copies of three cogmer releases on one Mac carry the bit (`5baeaea`), and other install routes, Codex's among them, were not examined. A script run through the Bash tool on Windows with Git ran in the `claude -p` run recorded in `12b435d`.

**Revisit when** an install route is found that drops the execute bit.
