# D-270 — A Windows hook runs without a shell

**Date:** 2026-10-28 · **Status:** not built · **Areas:** daemon, hosts

**Decision.** On Windows a Claude Code hook starts the launcher in exec form, with no shell, and a Codex hook names it in `commandWindows`, so no hook depends on `bash`. Neither host requires Git for Windows. Claude Code runs a shell-form hook through PowerShell without it, and Codex runs hooks through `cmd.exe`. A hook line that needs `bash` on a machine without Git does not run, and Codex prints a failure line for it. D-272 (a launcher in the plugin's bin folder starts the installed program) says what the launcher is.

**Rejected.**
- *Requiring Git for Windows and saying so in the README.* No host requires it, and a user without it gets no collaboration, and Claude Code showed no message under `claude -p`. `f62a961`.
- *A `shell: powershell` hook line that calls the launcher.* It runs, with and without a space in the plugin's path, but it needs a quoted path and PowerShell where exec form needs neither. `f62a961`, `8c07d61`.

**Limits.** Claude Code's hooks file has no per-platform key in the part of the hooks reference read on 2026-10-28, so how one hooks file reaches Windows and the other systems is not decided here, and nor is how the first install runs on Windows. Cowork's hooks call the bridge and use no shell (D-205, Claude Cowork is a supported host through a bridge on the user's machine).

**Revisit when** Claude Code's hooks accept a command per platform, or a host starts requiring Git for Windows.
