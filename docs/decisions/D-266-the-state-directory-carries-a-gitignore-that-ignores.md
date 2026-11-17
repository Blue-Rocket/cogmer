# D-266 — The state directory carries a .gitignore that ignores everything

**Date:** 2026-10-27 · **Status:** active · **Areas:** storage, trust

**Decision.** The state directory holds a `.gitignore` that contains `*`, so that a bulk `git add` in a repository that holds the directory, such as a dotfiles repository that adds the home folder, skips the keys and the rooms, and adding one of them by name is refused unless the user forces it. `ensureStateDir` writes the file whenever the directory has none, so a directory made earlier gets one the next time a program opens it, and it leaves a `.gitignore` that exists as it is, since the user may have written it. A failure to write the file is ignored, because the file protects the user and the program does not depend on it.

**Rejected.**
- *A git repository inside the directory.* A `git add` of the directory then fails as a whole ("does not have a commit checked out"), which stops the user's other files from being added too, and the stray `.git` confuses other tools.
- *A line in the README telling users to keep the directory out of repositories.* It reaches only the users who read it, and the file protects the rest.

**Limits.** A forced add and a file that git tracks are not stopped, and a sync folder or a backup tool does not read a `.gitignore`.

**Revisit when** a key is found in a repository that held the state directory.
