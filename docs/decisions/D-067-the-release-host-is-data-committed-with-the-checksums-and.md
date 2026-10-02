# D-067 — The release host is data, committed with the checksums, and publishing verifies what it serves

**Date:** 2026-09-18 · **Status:** active · **Areas:** release

**Decision.** The address where release assets are served is the file
`plugin/release-url.txt`, committed with `plugin/checksums.txt` and `plugin/VERSION`.
`scripts/release.sh` builds and hashes and knows no host, and `scripts/publish.sh` puts
the files on the host and fetches each one back to hash it against the pinned value.

**Support.**
- The address and the hashes change in one commit, so a move cannot leave hashes that
  name bytes the new host does not serve, which would be a refusal nobody could explain.
  `plugin/release-url.txt`.
- Moving hosts changes `publish.sh` and leaves `release.sh` alone. `scripts/publish.sh`.
- A publish that succeeded while the host served something else is a failure nobody would
  look for, and a colleague's install would refuse the binary for no reason on their
  machine. `scripts/publish.sh`.

**Rejected.**
- *A host written into the installer.* A wrong host costs a failed download, and hashes
  that did not move with it cost a refusal.
- *Serving the assets from a machine the maintainer operates, over plain HTTP.* It is a
  component that fails D-115's test, an availability dependency serving the maintainer and
  not the person installing. D-121 (the repository is both the release host and the
  marketplace).

**Limits.** What authorizes a binary is the pinned hash and not the transport, which is
D-066 (the binary is fetched and verified, never shipped in the plugin).
