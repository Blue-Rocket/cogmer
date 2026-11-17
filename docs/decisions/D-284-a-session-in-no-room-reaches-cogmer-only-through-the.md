# D-284 — A session in no room reaches cogmer only through the commands its user types

**Date:** 2026-11-16 · **Status:** active · **Areas:** daemon, rooms

**Decision.** The prompt and stop hooks of a session in no room end in the hook script, which neither starts the program nor sends the daemon anything, unless the prompt holds a cogmer command. The script tells such a session by the absence of a file named for it in `bound-sessions/` under the state directory, which a binding writes before it is committed and a leave removes after, so a missing file always means no room and a stray one costs a call to the daemon. A user who has not started collaborating has given cogmer nothing to handle (§12a, room membership), and a daemon that receives every turn of every session invites features that act on turns from outside a room.

**Rejected.**
- *Calling the daemon from every session and letting it answer that the session is in no room.* The daemon then holds every prompt of every session on the machine, and work done before its room check runs on sessions it has no business with.
- *Having the program read `membership.db` itself before calling the daemon.* The program still starts and still receives the prompt, and the session's identifier would be matched in two places, so a difference between them would stop capture without a sign.
- *Reading a cloud session's plugin files at every prompt, whether or not the session is in a room.* A session in no room uses the cloud value only through a command, and every command that reads the user identity first asks the daemon to read the files (D-279, a cloud session's daemon reads its cloud value from the plugin files it finds), so a cloud session in no room calls the daemon at no prompt.
- *Recording the room in an environment variable.* A command runs as a child of the host and cannot change the environment of the hooks the host starts later, and at session start no room has been chosen.

**Limits.** `session_rooms` in `membership.db` is the record, and removing a bound session's file by hand stops that session's capture until `membership.db` is next opened, which writes the file again. The session start hook runs in every session and is not governed by this decision. Cowork's hooks reach the daemon through the bridge and not through the script (D-205, Cowork through a bridge on the user's machine), so a Cowork session in no room still calls the daemon. The prompt hook of a Claude Code session in no room runs the program once, as `cogmer notice`, when it finds the `install-notice` mark, to take the line for a finished first install (D-282, the person is shown one line when the install finishes and when cogmer needs repair), and hands the program none of the prompt.

**Revisit when** a host lets a plugin register hooks for one session, so that a session in no room has none.
