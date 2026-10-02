# D-190 — The repository carries the marketplace manifest beside the plugin it lists

**Date:** 2026-09-22 · **Status:** active · **Areas:** release, commands

**Decision.** The repository carries `.claude-plugin/marketplace.json` beside the `plugin/` directory
it lists, so `/plugin marketplace add Blue-Rocket/cogmer` and `/plugin install cogmer@blue-rocket` name
the same repository.

**Support.**
- A plugin is installed from a marketplace and a marketplace is added by name, so installing takes two
  lines, and one repository gives a user one name to know and not two. §29 (installation),
  `.claude-plugin/marketplace.json`.
- The plugin still carries the hooks, and the session-start hook still starts the daemon. D-041 (ship
  as a Claude Code plugin), D-150 (the session-start hook starts the daemon, and never delays the
  session).

**Rejected.**
- *A separate marketplace repository.* One more name for a user to know and one more thing to keep in
  step with every release, whose only entry would be a plugin that lives somewhere else.
