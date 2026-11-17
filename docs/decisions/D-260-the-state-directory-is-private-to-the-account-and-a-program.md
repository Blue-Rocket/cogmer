# D-260 — The state directory is private to the account, and a program tightens one that is not

**Date:** 2026-10-27 · **Status:** active · **Areas:** storage, trust, daemon

**Decision.** The state directory and its `rooms` folder have mode 0700, so the databases, `daemon.log`, `install.log` and every other file in them are closed to other accounts on the machine whatever mode each file has. One Go function, `ensureStateDir`, creates the state directory and sets it to 0700 when it has any group or other bit, and every code path that writes into the state directory calls it, so a directory made earlier under a looser mode is tightened the next time a program opens it. `ensureRoomsDir` does the same for the folder of room databases. The shell scripts create the directory through `ct_state_dir` in `common.sh`, which makes it under a umask of 077 and runs `chmod 700` on it, and they create each log and state file under that umask. On Windows, where a directory has no such bits, the function only creates the directory.

**Rejected.**
- *Setting a umask of 077 for the whole of each script.* The daemon and every command inherit a script's umask, so a file a user asks for outside the state directory would be closed to everyone but the user.
- *Setting the mode on each file as it is created.* A new kind of file is easy to add without it, and the directory's mode closes every file at once, including those a library creates, such as the database's write-ahead log.
- *Tightening only a directory that a program creates.* A directory made by an older program or by a shell under a umask of 022 is the case that leaves the databases open.

**Limits.** A file that someone made readable is readable to anyone who can reach it by another path, such as a hard link made before the directory was tightened. A chmod runs each time a program opens the state directory with a loose mode, and does nothing when the mode is 0700.

**Revisit when** a host's sandbox needs another account to read part of the state directory.
