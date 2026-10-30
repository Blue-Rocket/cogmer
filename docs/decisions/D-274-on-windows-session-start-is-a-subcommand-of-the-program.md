# D-274 — On Windows session-start is a subcommand of the program

**Date:** 2026-10-28 · **Status:** not built · **Areas:** daemon, trust

**Decision.** Once the program is installed, `cogmer-dock session-start` (D-272, a dock in the plugin's scripts folder passes hook calls to the installed program) runs `cogmer session-start`. That subcommand starts the daemon when it is not running (D-150, the session-start hook starts the daemon and never delays the session) and writes the standing policy for room content as the hook's `additionalContext` (D-176, the standing policy for room content is also stated once at session start). The dock writes neither, and while no program is installed it writes only the install state (D-075, the install has a state and absence is not an inference). A first Windows session therefore lacks the standing policy, which D-176 accepts for a session whose plugin was installed mid-session.

The standing policy's text exists in `session-start.sh` and in the program, and a test requires the two to be equal.

**Rejected.**
- *The session-start work in the dock.* The dock is a file committed to the plugin and changes only with the plugin, where the program changes with the installer.
- *`session-start.sh` run by Git Bash on Windows.* It needs Git for Windows (D-270, a Windows hook runs without a shell).

**Limits.** Whether the macOS and Linux handler later calls the subcommand is not decided here.

**Revisit when** the policy text changes in a way that makes the equality test a burden, or a hostile turn succeeds against a session without the policy (D-176).
