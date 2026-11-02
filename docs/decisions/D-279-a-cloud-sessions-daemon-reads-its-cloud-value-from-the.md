# D-279 — A cloud session's daemon reads its cloud value from the plugin files it finds, as well as from the variable

**Date:** 2026-11-02 · **Status:** not built · **Areas:** identity, daemon, hosts

**Decision.** A daemon whose environment has `CLAUDE_CODE_REMOTE` set to `true` and no `COGMER_CLOUD_KEY` reads every `cloud.value` under the account's synced plugins, discards each that is not a cloud value whose certificates verify, and acts on the one whose cloud certificate has the highest serial, since the user can copy, rename and change plugins and no count of them can be assumed (D-278, the machines page offers the cloud value as a plugin for the user to upload once). It reads those files again at each prompt and takes a newer value when one appears, because a session that began before an upload ran its first hook on the files from before the upload when the user returned to it (`419a2d6`). Two values with the same serial, which two of the user's machines that are apart can make (D-280, a user has one cloud key at a time), are chosen between by the order of their cloud keys' identifiers, so that every session of the user takes the same one.

**Rejected.**
- *Reading one plugin by its name.* The user can copy a plugin, rename it and alter it, so a name identifies nothing about its contents.
- *A hook in the plugin that hands the daemon the file's path.* A folder's name changes between hooks, with a suffix such as `~g2` after an upload (`419a2d6`), and a daemon that searches for the files needs no hook.

**Limits.** A daemon that has read a value holds it in memory, so removing the plugin does not end that daemon's authority and removing its cloud key does (D-250, a removal of a cloud key names a cutoff for each of its session keys in each room).

**Revisit when** Anthropic changes the folder a synced plugin's files are placed in, so that a daemon in a cloud session finds none.
