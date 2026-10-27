# D-265 — A development or test build never opens the account's real state directory

**Date:** 2026-10-27 · **Status:** active · **Areas:** daemon, release

**Decision.** A program built from source, a program a test built, and a test binary refuse to open the account's state directory, `.cogmer` under the account's home as the operating system records it and not as `$HOME` says, or anything inside it, whether it is reached by default, by `COGMER_HOME` or through a link, and they stop before reading or writing there. A release build, which carries a version, has no such rule, and it is the program to use on real data. A command, the daemon and every subcommand of a build a test made print the reason and exit, a hook of a build somebody made from source exits 0 and says nothing as it does when cogmer is absent (D-011, hooks fail open: exit 0, empty stdout), and in a test `homeDir` panics. There is no override, because anything that sets one to make a command work makes the next mistake one assignment away, and a build from source can change the data in ways the released program does not expect (D-244, the data in the state directory carries one version, and a program that finds a newer one touches nothing).

**Rejected.**
- *An environment variable that allows it.* A script or an agent sets it to make a command work, and the check then protects nothing.
- *Each test naming a directory of its own.* The test suite has more than a hundred places that do, and a test that omits one runs against the real directory with nothing failing.
- *Refusing only when `COGMER_HOME` is unset.* A script that exports the real path reaches the same directory.

**Limits.** The check is in the program, so it does not stop a shell command or another program from changing the directory, which the account's file permissions and a copy kept elsewhere cover. A build from source cannot run on real data at all, so debugging a fault in real data means copying the data into a directory made for the run and setting `COGMER_HOME` to it.

**Revisit when** a need to run a build from source against the real directory is found.
