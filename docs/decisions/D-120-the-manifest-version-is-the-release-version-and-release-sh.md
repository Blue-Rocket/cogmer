# D-120 — The manifest version is the release version, and `release.sh` writes it

**Date:** 2026-09-22 · **Status:** active · **Areas:** release

**Decision.** A release has one version. `release.sh` writes it into the manifest from the same
argument it writes `plugin/VERSION` and `plugin/checksums.txt` from, and refuses and does not guess
if the manifest holds more than one `version` field. A test compares the manifest with
`plugin/VERSION`.

**Support.**
- A `version` in `plugin.json` pins the plugin to that string, users receive an update only when it
  is bumped, and the manifest's value wins over a marketplace entry's. It is not for display.
  `f288913:docs/decisions/D-120-the-manifest-version-is-the-release-version-and-release-sh.md`.
- Everything the plugin carries rides on that delivery, the commands, the hooks, `plugin/VERSION`,
  which `install.sh` reads to choose the asset, and `checksums.txt`, the only thing that authorizes a
  downloaded binary, so a stale manifest is a release nobody receives and its symptom is a user on an
  old build with nothing reporting it. `plugin/hooks-handlers/install.sh`, `scripts/release.sh`.
- The test catches a version bumped by hand in one file, and also `VERSION` moved without rerunning
  `release.sh`, which is the state in which `checksums.txt` pins assets that were never built.
  `cmd/cogmer/manifestversion_test.go`, `TestTheManifestVersionIsTheReleasedVersion`.

**Rejected.**
- *Deleting the field.* Updates would flow without anybody bumping anything, which removes the gate
  and not the drift, and the plugin people receive would carry no version while the binaries it
  authorizes are pinned by number, so nothing could say which plugin goes with which binary.
- *Two numbers, one for the commands and hooks and one for the daemon.* They cannot move
  independently, because `checksums.txt` ships inside the plugin and names binaries, so a change to
  either ships a new plugin, and the most a second number buys is a skipped build.
- *Having `release.sh` refuse unless the manifest names the version.* That makes a hand edit
  mandatory at the point where it was forgotten.

**Limits.** No registry check covers it, since update delivery happens in the plugin manager before
any session exists. Unlike the registry's entries, the behavior is documented.

**Revisit when** the plugin gains a marketplace entry that declares its own version, since the
manifest's wins and the marketplace's number is then decoration, or Claude Code changes where a
plugin's version comes from.
