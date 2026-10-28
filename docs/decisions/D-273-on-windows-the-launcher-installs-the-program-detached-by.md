# D-273 — On Windows the launcher installs the program, detached, by the shell installer's rules

**Date:** 2026-10-28 · **Status:** not built · **Areas:** release, daemon

**Decision.** When no program is installed in `~/.cogmer/bin`, or the installed one is older than the plugin's version (D-215), the Windows launcher fetches one as `install.sh` does. It takes the program from the address in `plugin/release-url.txt`, checks it against `plugin/checksums.txt` before anything runs it, and installs it detached from the hook, one installer at a time, with a wait of an hour after a failure and its progress in the install state file and `install.log` (D-172, D-075, D-084). While no program is installed, the launcher answers a session-start or a command with the install state and any other hook with nothing. The launcher is written in Go, since a hook line has no shell.

**Rejected.**
- *A PowerShell script that the launcher starts.* The machine's script policy can refuse it.
- *Git Bash running `install.sh`.* It needs Git for Windows (D-270, a Windows hook runs without a shell).

**Limits.** `release.sh` builds only a windows amd64 program, so Windows on ARM64 receives that build under emulation, as the install log of `0f2d509` shows. Choosing an arm64 build when one exists would read `PROCESSOR_ARCHITEW6432`, which was not run. A launcher committed to the plugin departs from D-066 (the binary is fetched and verified, never shipped in the plugin) for the launcher alone, and what makes the committed file equal its source is not decided here. Installing on macOS and Linux is `install.sh` (D-172, the installer runs detached, one at a time, outside the plugin directory, and waits an hour after a failure).

**Revisit when** a Windows machine is found that refuses to run the launcher, or the program is released for arm64.
