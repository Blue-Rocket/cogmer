# D-118 — The plugin manifest name is the command namespace, and Claude Code forces it

**Date:** 2026-09-22 · **Status:** active (implemented)

**Context.** D-070, as `f288913:docs/decisions/D-070-slash-commands-carry-a-distinctive-prefix-because.md` held it, chose `/room-*` over `/team-*` on an empirical finding: a
subdirectory changes how a command is **displayed** and not what a person types,
there was no `plugin:command` invocation form, and so two plugins defining the same
name collided at the only thing a user touches. It named its own revisit trigger —
"if Claude Code gains a `plugin:command` invocation form". That has happened, and
in the stronger form: the prefix is not offered, it is imposed.

**The manifest's `name` is documented as the namespace.** "Unique identifier and
skill namespace. Skills are prefixed with this." And: "Plugin skills are always
namespaced (like `/my-first-plugin:hello`) to prevent conflicts when multiple
plugins have skills with the same name. To change the namespace prefix, update the
`name` field in `plugin.json`."

**Confirmed rather than read.** A throwaway plugin named `nstest`, with one command
in `commands/` and one in `skills/`, surfaced as `nstest:legacyform` and
`nstest:skillform`. Both layouts, both namespaced, each exactly once.

**Decision.** The prefixes stay. `/cogmer:room-create`, not `/cogmer:create`.

D-070's protection becomes belt and braces, which is precisely what D-070, as
`f288913:docs/decisions/D-070-slash-commands-carry-a-distinctive-prefix-because.md` held it, said
would happen. What keeps the prefixes is D-096: a prefix names its target, the set
will grow, and `room-status` and `self-status` are two different questions that
would collapse into one word without it. Losing that distinction to save six
characters in a string the host already made long is a bad trade.

**The cost is stutter, and it is accepted.** `/cogmer:room-create` says the domain
twice. The alternative said it once and made `/cogmer:status` ambiguous between a
room and a person.

**The manifest name is now load-bearing in a way it was not.** It is what a person
types before every colon, so changing `plugin.json`'s `name` renames every command
at once. It is not a cosmetic field.

**D-096's overview command, as `f288913:docs/decisions/D-096-a-command-prefix-names-its-target-self-status-is-where-you.md` held it, cannot be what D-096 reserved.** It wanted
`/<product-name>` as an entry point saying what the tool is, held back because the
name was unsettled. There is no bare `/cogmer`; it would be `/cogmer:cogmer`. The
compensation is that D-096's stated blocker is gone — Claude Code owns `/help`, but
it does not own `/cogmer:help`, because the namespace is exactly what stops the
collision. Not built here.

**Test.** `TestEveryCommandReferenceCarriesTheManifestNamespace` reads the
namespace out of `plugin.json` rather than duplicating it as a Go constant, and
walks the Go, HTML, Markdown and shell sources for two defects: a command named
with no namespace, and one named with a namespace that is not the manifest's. The
second is the one worth having — renaming the plugin is a single-word edit that
would otherwise break every instruction string the binary prints, with
nothing failing until a person typed one. Both branches confirmed to fail when the
defect is introduced.

**Not in the behavior registry, and the reason is a limitation.** Registry checks
run against hook payloads inside a live session; this is a plugin load-time
property, which none of them can observe. The half of this finding that does belong
there — that `commands/*.md` is loaded identically to a skill, and so every command
is **model-invocable** — is in `open.md`, because which commands should be reachable
by the model is a decision and not a sweep.

**Revisit when** Claude Code offers an unprefixed alias deliberately rather than as
the behavior reported in `anthropics/claude-code` issue 15882, or when `commands/`
stops being loaded at all — the documentation already calls it legacy and prefers
`skills/<name>/SKILL.md`.
