# D-120 — The manifest version is the release version, and `release.sh` writes it

**Date:** 2026-09-22 · **Status:** active (implemented)

**Context.** `plugin/.claude-plugin/plugin.json` carried `0.1.0` while
`plugin/VERSION` carried `0.7.0`. Noticed during the rename (D-117) and not caused
by it: `release.sh` had only ever written `VERSION` and `checksums.txt`, so the
manifest kept whatever was typed into it on the day it was written.

**What that field does is documented, and it is not display.** Setting `version` in
`plugin.json` pins the plugin to that string, and users receive an update only when
it is bumped — excepting a `command` source and a plugin loaded in place. If a
marketplace entry also sets one, `plugin.json` wins.

**So the stale number was a release nobody would receive.** Everything the plugin
carries rides on that delivery: the commands, the hooks, `plugin/VERSION` — which is
what `install.sh` reads to decide which asset to fetch — and `checksums.txt`, the
only thing authorizing a downloaded binary to run. A manifest frozen at `0.1.0`
leaves every installed copy fetching the binary an old `VERSION` names, verified
against the hashes an old `checksums.txt` pins, indefinitely. The symptom is not an
error: it is a person on an old build with nothing anywhere reporting it, which is
the same failure shape as the registry's behaviors, one layer further out.

**Decision.** One number for a release. `release.sh` writes the manifest version
from the same argument it writes `VERSION` and `checksums.txt` from, and refuses
rather than guessing if the manifest ever holds more than one `version` field. The
manifest is now `0.7.0`.

**The check.** `TestTheManifestVersionIsTheReleasedVersion` compares the manifest
against `plugin/VERSION`; confirmed to fail with the manifest returned to `0.1.0`.
`release.sh` is the writer, so what the test is for is the other path — a version
bumped by hand in one file. It catches a second thing for free: `VERSION` moved
without rerunning `release.sh`, which is the state in which `checksums.txt` pins
assets that were never built.

**Rejected.**
- *Deleting the field.* It is optional, and without it the version comes from the
  install source, so updates flow without anybody bumping anything — which would
  have fixed the drift by removing the gate. Rejected because the plugin people
  receive would then carry no version at all while the binaries it authorizes are
  pinned by number, and nothing in the repository could say which plugin goes with
  which binary.
- *Two numbers meaning different things* — one for commands and hooks, one for the
  daemon — on the argument that a commands-only change needs no new binary. They
  cannot move independently: `checksums.txt` ships inside the plugin and names
  binaries, so any change to either must ship a new plugin. The most this buys is a
  skipped build, at the price of a pair to reason about at every release.
- *Having `release.sh` refuse unless the manifest already names the version.* That
  makes a hand-edit mandatory at exactly the point where the hand-edit was forgotten.

**Not in the behavior registry**, for D-118's reason: registry checks observe a
live session through hook payloads, and update delivery happens in the plugin
manager before any session exists. Unlike the registry's entries, this one is also
documented.

**Revisit when** the plugin gains a marketplace entry that declares its own version
— `plugin.json` wins, so either they agree or the marketplace's number is decoration
— or when Claude Code changes where a plugin's version comes from.
