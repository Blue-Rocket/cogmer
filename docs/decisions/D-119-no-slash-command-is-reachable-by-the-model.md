# D-119 — No slash command is reachable by the model

**Date:** 2026-09-22 · **Status:** active (implemented)

**Context.** D-118 established that `plugin/commands/*.md` is loaded identically to
`skills/<name>/SKILL.md`, and left open what follows from it. What follows is that
every command was a model-invocable skill: confirmed with a throwaway plugin, where
both layouts surfaced in the model's own listing.

**Three harms, and the first is the one that does not depend on anything going
wrong.** Every command's name and description occupied the context of every session
with the plugin installed, invoked or not. §3.1 names being robbed of context budget
as harm in its own right, so this was a live violation of the duty every other
requirement is subordinate to, on every session, from the moment the plugin shipped.

The second is that the model could run one. `room-create` and `room-join` bind a
session to a room, and D-016 fixes that at the first prompt precisely because
injected context cannot be withdrawn from a context window — so a binding nobody
asked for is not undone by leaving. `room-invite` and `room-revoke` reach another
person's machine. `peer-forget` discards admissions that a pairing ceremony
established.

The third is that neither failure is visible from inside this repository. The
loading happens in somebody else's program, and nothing here would have reported
it.

**Decision.** `disable-model-invocation: true` on all fourteen, not on the eight
that change something.

**Why the read-only six are included, which is the whole of the argument.** The
question a split has to answer is what the model gains from a reachable
`room-status`, and the answer is nothing. A person who wants it types it. The model
cannot show them anything either way — D-033 and D-036 settled that every extension
point delivers to the model and nothing displays to a person — so the most it can do
is say what it read, which is what the command file already instructs it to do once
a person has run it. The six buy no capability and cost context on every session.

Keeping them would also make the rule carry an exception, which D-096 records as the
thing that makes people stop trusting a rule. A reader would have to know which
commands are reachable and why, to answer a question nobody has.

**None of these files was written to be model-invoked.** Each is a `!` pre-execution
followed by "report what the output says". They assume a person has just typed the
command and the model is reading the result back. There is no member of the set
whose body makes sense as something the model reached for on its own, which is the
strongest evidence that reachability was never intended — it was inherited from the
file layout.

**Test.** `TestNoCommandIsModelInvocable` requires the frontmatter on every file in
`plugin/commands/`, and fails if the directory is empty rather than passing
vacuously. The defect it guards is not the files that exist now, which could have
been a one-time fix: it is the command added later by somebody who does not know the
frontmatter is load-bearing. Confirmed to fail when the flag is removed from one
file.

**Not in the behaviour registry**, for D-118's reason: registry checks observe a
live session through hook payloads, and this is a plugin load-time property that
none of them can see.

**Not done here.** Migrating to the `skills/<name>/SKILL.md` layout, which the
documentation prefers and calls `commands/` legacy. It is worth doing and it is a
directory restructure with its own risk, which is a bad thing to bury a §3.1 fix
inside. It stays in `open.md`.

**Revisit when** a command appears that the model genuinely should reach for without
being asked — at which point the question to answer first is what it would do with
the result, given that it cannot show anybody anything.
