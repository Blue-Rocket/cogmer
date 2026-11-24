# D-293 — A test gives a build its version by overwriting the stamped bytes, and the program trims the padding

**Date:** 2026-11-24 · **Status:** active · **Areas:** release

**Decision.** The tests build the program once, with a version made of a fixed text followed by `~` padding, and make each version a test asks for by overwriting those bytes in a copy of that build with the version and enough `~` to fill the same length; the program trims `~` from the end of its version when it starts. A build takes about 12 seconds on four cores and the suite asked for about ten versions, so building once removes most of the suite's building, and the version is a property of the file, as it is in a release, so a test in which programs of two versions start one another sees the version each file carries. A release version never contains `~`, so the trim leaves it unchanged, and nothing outside the file can change a program's version.

**Rejected.**
- *A version read from an environment variable in a test build.* A process passes its environment to the programs it starts, so in a test where one version starts another, both would report the version the variable names.

**Limits.** On macOS the copy is signed again with `codesign`, since the changed bytes do not match the signature the linker made, and macOS on Apple silicon does not run a program whose signature does not match.
