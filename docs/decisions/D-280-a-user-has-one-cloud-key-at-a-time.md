# D-280 — A user has one cloud key at a time

**Date:** 2026-11-02 · **Status:** not built · **Areas:** identity, view

**Decision.** The machines page adds a cloud key only when the user's records name none that is current, and a user who wants another removes the current one first (D-250, a removal of a cloud key names a cutoff for each of its session keys in each room). A plugin reaches every cloud session of the account that syncs the account's plugins (D-278, the machines page offers the cloud value as a plugin for the user to upload once), so a second key would sit beside the first with the same authority and no use, and a refusal makes the order of a rotation the only way to add one.

**Rejected.**
- *A cloud key for each environment.* Removing one would then mark only its sessions' turns, where one key's removal after a theft marks the turns of every cloud session of the user at each colleague (D-250). That cost is accepted, since a theft is the rare case and the key is readable in every session that holds it (D-216, a cloud session signs with a key of its own that the user's shared key vouches for).

**Limits.** Two of the user's machines that are apart can each add a key, and the user can copy a value anywhere, so a daemon that finds several chooses by the highest serial (D-279, a cloud session's daemon reads its cloud value from the plugin files it finds, as well as from the variable). A user whose records name several cloud keys keeps each until it is removed, and the page adds none until then.

**Revisit when** a user is found to need separate cloud keys for separate environments.
