# D-292 — A release is tested from a staging repository whose plugin downloads from that repository's releases

**Date:** 2026-11-20 · **Status:** active · **Areas:** release

**Decision.** Before a release commit reaches the main branch of `Blue-Rocket/cogmer`, `scripts/stage.sh` pushes it, with one commit on top that sets `plugin/release-url.txt` to the releases of `Blue-Rocket/cogmer-staging`, as the main branch of that repository and publishes its binaries there, and the release is installed from that repository as `README.md` says. claude.ai reads a marketplace only from its repository's default branch, so a release can be installed in a Cowork cloud session as a user installs it only from the default branch of some repository, and the address committed beside the hashes (D-067, the release host is data committed with the checksums) lets that plugin download from the same repository with no change to the installer. `scripts/publish.sh` publishes to the repository that the address in the commit it is given names, so one script publishes to both.

**Rejected.**
- *A second plugin in the `blue-rocket` marketplace that names the release branch.* It needs a name other than `cogmer`, so its commands take another prefix (D-118, the plugin manifest name is the command namespace) and do not match `README.md`, and every user of the marketplace sees it.
- *A staging plugin that downloads from the releases of `Blue-Rocket/cogmer`.* The binaries would be on the public release host before the test, and a candidate rebuilt after a failed test would need that host's tag moved.

**Limits.** The staging repository is public, because the installer downloads without credentials and GitHub serves a private repository's release assets only to an account with access. A plugin installed from it downloads from it until the tester removes that marketplace and adds `Blue-Rocket/cogmer`. The address the released plugin names is not the one the test downloaded from, and `scripts/publish.sh` fetches each asset back from that address and hashes it before it reports success.

**Revisit when** claude.ai can add a marketplace from a branch other than the repository's default.
