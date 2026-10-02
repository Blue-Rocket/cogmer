# D-118 — The plugin manifest name is the command namespace, and commands keep their prefixes under it

**Date:** 2026-09-22 · **Status:** active · **Areas:** commands

**Decision.** Commands keep their `room-`, `peer-` and `self-` prefixes under the namespace that
Claude Code adds from the manifest's `name`, as in `/cogmer:room-create` and not `/cogmer:create`. A
test requires every command reference to carry the manifest's namespace.

**Support.**
- Claude Code imposes the namespace. The manifest's `name` is documented as the plugin's identifier
  and skill namespace, and a throwaway plugin with one command in `commands/` and one in `skills/`
  surfaced both namespaced, each once. `f288913:docs/decisions/D-118-the-plugin-manifest-name-is-the-command-namespace-and.md`.
- A prefix names its target, the set will grow, and `room-status` and `self-status` are two
  questions that would collapse into one word without it, which costs stutter in a string the host
  made long and which is accepted. D-096 (a command prefix names its target, and
  `/self-status` is where you are).
- The manifest name is what a user types before every colon, so changing `name` in `plugin.json`
  renames every command at once. `plugin/.claude-plugin/plugin.json`.
- The test reads the namespace out of `plugin.json` and walks the Go, HTML, Markdown and shell
  sources for a command named with no namespace and one named with a namespace that is not the
  manifest's, and renaming the plugin is a one-word edit that would otherwise break every
  instruction the binary prints. `cmd/cogmer/commandname_test.go`,
  `TestEveryCommandReferenceCarriesTheManifestNamespace`.

**Rejected.**
- *Unprefixed command names under the namespace.* `/cogmer:status` would be ambiguous between a room
  and a person.

**Limits.** No registry check covers it, since registry checks run against hook payloads in a live
session and this is a property of plugin load time, which none of them can observe.

**Revisit when** Claude Code offers an unprefixed alias on purpose and not as the behavior reported
in `anthropics/claude-code` issue 15882, or `commands/` stops being loaded, which the documentation
calls legacy.
