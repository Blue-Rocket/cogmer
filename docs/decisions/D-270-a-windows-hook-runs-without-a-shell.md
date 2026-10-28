# D-270 — A Windows hook runs without a shell

**Date:** 2026-10-28 · **Status:** not built · **Areas:** daemon, hosts

**Decision.** On Windows a Claude Code hook starts the launcher in exec form, with no shell, by the path `${CLAUDE_PLUGIN_ROOT}/bin/cogmer-launch` with no extension, and Claude Code resolves that path to `cogmer-launch.exe`. A Codex hook names the launcher in `commandWindows`. No hook depends on `bash`. The Claude Code line is the same text on every system, since Claude Code's hooks file has no key that selects a line by platform in the part of the hooks reference read on 2026-10-28. Neither host requires Git for Windows. Claude Code runs a shell-form hook through PowerShell without it, and Codex runs hooks through `cmd.exe`. A hook line that needs `bash` on a machine without Git does not run, and Codex prints a failure line for it. D-272 (a launcher in the plugin's bin folder starts the installed program) says what the launcher is.

**Rejected.**
- *Requiring Git for Windows and saying so in the README.* No host requires it, and a user without it gets no collaboration, and Claude Code showed no message under `claude -p`. `f62a961`.
- *A hook line for each platform.* With no key that selects a line by platform, a Windows line and a line for the other systems would both run on a machine that can run either.
- *The launcher by bare name.* A plugin's `bin/` folder is not on the PATH of an exec-form hook. `1edc1a1`.
- *A `shell: powershell` hook line that calls the launcher.* It runs, with and without a space in the plugin's path, but it needs a quoted path and PowerShell where exec form needs neither. `f62a961`, `8c07d61`.

**Limits.** The no-extension path ran a stand-in program from an exec-form hook, with and without Git and with a space in the plugin's path, under `claude -p` (`1edc1a1`). With a script and an `.exe` of that name both in `bin/`, the Windows hook ran the `.exe` (`5baeaea`). On macOS and Linux a script with the execute bit ran from an exec-form hook, and one without it failed with `EACCES`, reported as a hook error that did not block the turn (`5baeaea`). Claude Code 2.1.303 accepted exec form. Which script the launcher is on those systems, how its execute bit is set and checked, and how the first install runs on Windows are not decided here. Cowork's hooks call the bridge and use no shell (D-205, Claude Cowork is a supported host through a bridge on the user's machine).

**Revisit when** Claude Code's hooks accept a command per platform, Claude Code stops resolving an extensionless path to an executable on Windows, or a host starts requiring Git for Windows.
