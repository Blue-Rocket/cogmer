# D-066 — The binary is fetched and verified, never shipped in the plugin

**Date:** 2026-09-18 · **Status:** active · **Areas:** release, trust

**Decision.** The installer fetches the binary into `~/.cogmer/bin` and runs nothing it
cannot verify. `plugin/checksums.txt` is committed to the plugin repository and is the
only thing that authorizes a downloaded binary to run. A download whose hash is not
listed is deleted and not run, and `scripts/release.sh` writes the file from the bytes it
just built.

**Support.**
- Five targets mean five binaries and an install uses one, so shipping them in the plugin
  would make every install download four fifths for platforms the machine is not.
  `scripts/release.sh`.
- A git repository keeps every version of every file, and binaries do not compress
  between versions, so binaries in the plugin repository grow its history with each
  release. `plugin/`.
- A hash typed and not computed authorizes something nobody has seen. `scripts/release.sh`.
- A download that does not match is deleted, and the installer exits 0 so the session is
  untouched. `plugin/hooks-handlers/install.sh`.
- When no pinned download is possible the installer builds from source if Go is present,
  and otherwise installs nothing, since having no binary is the correct outcome when
  neither path works. `plugin/hooks-handlers/install.sh`.
- Altering a committed file leaves a trace in history where a swapped release asset
  would not, so the check is weaker than a registry's provenance and still real.
  `plugin/checksums.txt`.
- Of 53 plugins in the official marketplace none ships a binary, and the one with a
  compiled server pulls a version-pinned image from a registry. A tampered asset was
  refused and deleted with the hashes named, and nothing was installed.
  `f288913:docs/decisions/D-066-the-binary-is-fetched-and-verified-never-shipped-in-the.md`.

**Rejected.**
- *Shipping the binaries in the plugin repository.* The download and the history above.
- *Building from source as the ordinary path.* It needs a Go toolchain on the user's
  machine and a module path that resolves, which most users of Claude Code do not have.
- *An `npx`-style runner or a registry.* A runner needs Node, which a user of Claude Code
  may not have, and it re-authorizes whatever the registry serves at every launch with no
  pin. D-001 (Go, not TypeScript/Node or Python).

**Limits.** Release assets and the plugin repository are both on GitHub, so the check
detects a swapped asset and not a compromised repository. How the installer runs is D-172.
