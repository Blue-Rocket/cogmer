# D-269 — The program serves the plugin of each of the last three releases

**Date:** 2026-10-28 · **Status:** active · **Areas:** release, commands, daemon

**Decision.** The program answers every subcommand that the command files and the Codex skill of each of the last three releases call, runs each of those releases' hook lines to an exit of 0 and no output when no daemon answers, and writes the daemon's mark of why it could not start and the version in its health answer in the forms those releases' scripts read. A test runs the program through the plugin of each of those releases, taken from git by its tag. One program serves every host and the installer and the daemon replace it only with a newer one (D-215, the installer and the daemon replace a program only with a newer version), so a host whose plugin has not updated runs its old plugin against the new program.

**Rejected.**
- *Checking only the plugin in the working tree.* The lagging host is the one that holds an older plugin, and the working tree's plugin is the one that matches the program.
- *Checking every release that was ever tagged.* The oldest tags carry layouts that no host still holds, and a check against them would keep every rename out of the program for good.

**Limits.** The test checks the names of the subcommands, the hook lines' exit and output, and two contracts the scripts read. It does not check what a subcommand prints, an argument or flag it takes, or what a hook sends to the daemon, and it skips the session-start line, which fetches a binary. It reads release tags, so a checkout without them skips it, as a shallow clone does.

**Revisit when** a host is found to hold a plugin older than three releases.
