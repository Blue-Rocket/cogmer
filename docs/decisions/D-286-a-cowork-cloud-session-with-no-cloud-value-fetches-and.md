# D-286 — A Cowork cloud session with no cloud value fetches and starts nothing

**Date:** 2026-11-19 · **Status:** active · **Areas:** daemon, hosts

**Decision.** The session-start hook ends at once, printing nothing, when `CLAUDE_CODE_ENTRYPOINT` is `remote_cowork`, `COGMER_CLOUD_KEY` is unset, and no synced plugin folder of the name the daemon reads a value from holds a `cloud.value` (D-279, a cloud session's daemon reads its cloud value from the plugin files it finds). A daemon in that sandbox has no user to speak for and can join no room, so the download and the start it costs every session at its start buy the user nothing (`9c88af4`). A folder of that name with a `cloud.value` in it is enough to start the daemon, and the hook never reads the file, since the daemon checks the value itself and one reader of the secret is enough. It is an exception to D-150 (the session-start hook starts the daemon, and never delays the session).

**Rejected.**
- *Ending the hook in every cloud session with no value, as `CLAUDE_CODE_REMOTE` marks one.* A Claude Code cloud session sets it too, and what a Claude Code cloud session needs at its start was not part of the runs that showed the idle daemon.
- *Ending the hook in every Cowork cloud session.* A user who has uploaded a cloud value would then have no daemon in any cloud session, which ends the route D-279 makes.

**Limits.** A `/cogmer:` command typed in such a session runs through the command script once the model runs the command's line (D-291, a command gives the model its shell lines to run when the host did not run them), and the script installs the program and runs it as it does anywhere, so a daemon with no user can still start when the user asks for one, and the command says what D-281 (a command in a cloud session with no value says the user must first set cogmer up on one of their own computers) has it say.

**Revisit when** Cowork sets a different `CLAUDE_CODE_ENTRYPOINT` in its cloud sessions, so that the check matches none.
