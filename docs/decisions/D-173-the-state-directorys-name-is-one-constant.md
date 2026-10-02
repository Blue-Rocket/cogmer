# D-173 — The state directory's name is one constant

**Date:** 2026-09-19 · **Status:** active · **Areas:** storage, release

**Decision.** The name of the state directory, `.cogmer`, is the constant `stateDirName`,
and `COGMER_HOME` overrides the whole path for the binary and the installer alike.

**Support.**
- A rename of the directory is one line and not a search. `cmd/cogmer/identity.go`,
  `stateDirName`.
- The installer and the binary read the same variable, so a user who sets it gets the
  binary and its state in one place. `cmd/cogmer/identity.go`, `homeDir`, and
  `plugin/hooks-handlers/install.sh`.

**Rejected.**
- *The literal repeated wherever the directory is named.* A rename becomes a search that
  misses places.
