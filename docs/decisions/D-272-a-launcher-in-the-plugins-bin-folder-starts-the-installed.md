# D-272 — A launcher in the plugin's bin folder starts the installed program

**Date:** 2026-10-28 · **Status:** not built · **Areas:** release, commands

**Decision.** On Windows the plugin carries a small launcher executable per architecture in its `bin/` folder, which finds the installed program in `~/.cogmer/bin` and starts it with the arguments it was given. A plugin file is the same text for every user. An exec-form hook has no shell to expand a home-folder variable, and Claude Code's reference lists only the plugin's paths, the project directory and plugin options as expanded. So no file can name a program whose path holds the user's name, and the launcher is reached by a path the host fills in. The launcher does nothing else, and the checksum the installer verifies (D-066, the binary is fetched and verified, never shipped in the plugin) authorizes the program it starts.

**Rejected.**
- *A `.cmd` launcher.* Claude Code's exec form takes an executable and refuses a `.cmd` shim, and `cmd.exe` parses the arguments it is given a second time, so an argument holding `&` could start another command. The second reason is reasoned from how `cmd.exe` parses and was not run.
- *Adding the program's folder to the user's PATH.* It is a lasting change outside cogmer's own folder, and a host that started before the change never sees it, so the first session after an install fails.
- *The program in `${CLAUDE_PLUGIN_DATA}`.* That folder belongs to one host's plugin, and one program serves every host (D-215, the installer and the daemon replace a program only with a newer version).

**Limits.** The launcher is an executable committed to the plugin, which D-066 does not cover for the program itself, and it holds no logic beyond starting the program. A program in a plugin's `bin/` folder, called by bare name, ran under a scoped rule with a space in the plugin's path (`8c07d61`), but only a stand-in was run, not a launcher, and only under `--plugin-dir`. Cowork installs no plugin that has a `bin/` folder (Claude Code's plugin reference, read on 2026-10-28), so Cowork does not use it.

**Revisit when** a host gives a hook line the user's home folder, or a Claude Code release leaves the plugin's `bin/` folder off the PowerShell tool's PATH.
