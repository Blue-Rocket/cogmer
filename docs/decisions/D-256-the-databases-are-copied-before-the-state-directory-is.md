# D-256 — The databases are copied before the state directory is migrated, and a program that cannot migrate it stops

**Date:** 2026-10-27 · **Status:** active · **Areas:** storage, daemon, release

**Decision.** Before a program migrates the state directory (D-244, the data in the state directory carries one version, and a program that finds a newer one touches nothing), it copies `membership.db` and every room database as consistent snapshots into `backup-data-version-N` inside the state directory, where N is the version the data holds, and it migrates nothing unless the copy is complete. A copy that exists is the data from before the first attempt, and a retry leaves it as it is, since the databases a failed migration left are partly new; an unfinished copy is never taken for one, because its folder takes the name only when the last database is in. A program that cannot copy the databases or cannot finish the migration stops: the daemon does not start, and a command exits with an error that says where the copy is and how to put it back. A migration is a series of changes with nothing joining them, so one that stops partway leaves some databases new and some old, and a program that runs on them writes on top of the mixture. A hook fails open as it does without a daemon, and `stop` goes on, since the user is told to stop cogmer before putting the copy back.

**Rejected.**
- *One transaction around each database's migration.* It leaves each database whole after a failure but not the set of them, and it does nothing for a migration that finishes and is wrong.
- *Copying the key files with the databases.* A migration never changes a key file, and a copy of a private key is a second place for a secret to be taken from.
- *Deleting the copy when the migration succeeds.* A user who finds a database unusable after a successful migration, before cogmer has run again, would have nothing to put back, and the copies are the size of the databases.
- *A command that restores the copy.* The copy is right to put back only in the short time before cogmer runs again, so the message explains that and gives the steps, and a command would offer the restore at every other time.

**Limits.** A copy lacks every turn written after it was made, and the sequence counters in the membership database (D-029, the sequence lives with the machine) would reissue numbers that colleagues hold, so putting it back is right only before cogmer has run again. A program never deletes a copy, so a state directory holds one copy for each version it has been upgraded from.

**Revisit when** a user is found to have put a copy back after cogmer had run.
