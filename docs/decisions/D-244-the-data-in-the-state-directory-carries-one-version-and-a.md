# D-244 — The data in the state directory carries one version, and a program that finds a newer one touches nothing

**Date:** 2026-10-25 · **Status:** active · **Areas:** storage, daemon, release

**Decision.** The state directory, `~/.cogmer`, holds one data version for everything in it, in a file of its own, `~/.cogmer/data-version`, which a program reads before it opens any database. A release raises the version only when an older program would misread or damage what it writes, and data with no version is version 0. A program checks the version before it reads or writes anything, and a program that finds a version higher than it knows touches nothing: its hooks fail open (§3.1, first do no harm), its daemon does not start, and its commands say that the data was written by a newer cogmer and the plugin needs updating. A program writes the version only after every migration it runs has succeeded, so a migration that stops partway never claims the data is newer than it is. Those migrations include every room database under `rooms/`, each opened once for the purpose. A version file a program cannot read counts as higher than the program knows.

**Rejected.**
- *Repairing, at startup, the fields an older program is known to drop.* It guards only the damage somebody has found, one field at a time, while an older program can also misread every store it opens.
- *A version for each file or each database.* The identity, the membership index and the rooms change together in the release that needs a version, so separate numbers would only add states that never occur, such as an upgraded identity beside rooms that are not.
- *Writing the version once the membership index is migrated at start, and leaving each room database to migrate when something opens it.* The version would then claim a format that a room database nothing has opened does not have.
- *Supporting a return to an older program.* An older program cannot read what a newer one writes without being changed itself, and a machine's program moves only forward unless somebody runs an older one by hand (D-215, the installer and the daemon replace a program only with a newer version).

**Revisit when** a release needs an older program to keep working with data a newer one wrote.
