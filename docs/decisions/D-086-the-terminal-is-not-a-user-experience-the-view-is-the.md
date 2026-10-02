# D-086 — The terminal is not a user experience; the view is the surface

**Date:** 2026-09-20 · **Status:** active (direction; amends D-080, does not alter D-055)

**Context.** Stated intent to stop treating "run it in a terminal" as a valid thing
to ask a user to do. Claude Code is the first host; Claude CoWork is wanted as a fast
follow.

**What actually depends on a terminal today: one bit.** Every room-scoped and
peer-scoped operation already has a slash command. `pair` and `verify` are the sole
exceptions, and the dependency is a single `fmt.Scanln` — the y/N answer to "did they
say the same two words?". The ceremony itself is already daemon-side and already
spoken over HTTP: `/verify/start` returns the words, `/verify/confirm` records the
answer, and the CLI is nothing but a client of those two endpoints.

So moving off the terminal is a **user-interface change, not an architectural one**.
There is no protocol to redesign and no logic to relocate.

**D-080's first reason, as `f288913:docs/decisions/D-080-there-is-no-current-room-a-terminal-command-exists-to-be.md` held it, is amended, not withdrawn.** It held that `pair` and `verify`
"cannot pass through a model" — interactive, blocking on another person, and the
words must reach a person's eyes unaltered. That is still true and still important.
What no longer follows is the conclusion that they must therefore be typed at a
terminal: *not passing through a model* and *being a terminal* are different
properties, and the browser view has the first without the second. D-080 could not
see that option because discovery was unsettled; D-082 settled it, and D-083 already
placed the first opening of the view at a first pairing. The three now agree.

**D-055 is untouched.** Two words, derived from a live commit/reveal exchange,
compared aloud on a call, by both people at once, with no fallback. Moving the
display from a terminal to a view changes the surface and nothing about the
ceremony. A view that offered a way to skip the call would violate D-055 no matter
how convenient; the gate is only as strong as the weakest ceremony that satisfies it.

**D-043 still holds: no adapter machinery, and naming a second host does not change
that.** CoWork is wanted, not present. Its extension model is not something to design
against on assumption. The argument in D-043 was that capture generalizes and
injection does not, and that every hard problem so far has been host-specific — none
of which a second host being *anticipated* falsifies.

**But this direction and that constraint point the same way.** The daemon and its
view are host-independent by construction: they are a local service and a web page,
not an extension of anything. Every piece of experience that lives there is a piece a
second host does not have to reimplement, and it gets there without a `source` column,
an adapter interface, or a speculative abstraction. Reducing what a second host must
supply is the opposite of building for one. That is the cheapest possible preparation
and it is justified on today's host alone.

**What the terminal keeps** (the other four reasons of D-080, as `f288913:docs/decisions/D-080-there-is-no-current-room-a-terminal-command-exists-to-be.md` held them, all intact): diagnostics that
must work when the plugin path is broken, the daemon's own lifecycle, machine-scope
identity, and testing. None of those is a user experience, which is the point — they
are an operator surface, and it is legitimate for an operator surface to be a CLI.

**Interim state, deliberately not marked in the files.** `/peer-pair` currently walks
a person through opening a terminal, and that is the truth today, so it stays correct
until the view can take the ceremony. It is not annotated as provisional in
`plugin/commands/peer-pair.md` because that file is a **prompt**, not documentation:
meta-commentary about the roadmap would be read by the model as instruction. The
record belongs here instead.

**Next, and not yet built:** the pairing ceremony in the view, driven by the two
endpoints that already exist. Open questions from D-083 remain open — whether later
pairings re-open the view, and whether the view drives or merely displays.

**Revisit when:** CoWork's extension model is known, or the view takes the ceremony
and `verifyWith`'s `Scanln` stops being the only interactive path.
