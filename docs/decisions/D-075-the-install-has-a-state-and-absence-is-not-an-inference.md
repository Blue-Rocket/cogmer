# D-075 — The install has a state, and absence is not an inference

**Date:** 2026-09-20 · **Status:** active (implemented, v0.3.0)

**Context.** Asked what options exist for telling a session about a download in
progress, and then — better — whether a fresh install should be *marked* as not yet
downloaded, with commands behaving differently until the mark clears.

The second framing is the right one. Absence of the binary meant three different
things and every reader guessed the same hopeful one:

- nothing has started (the plugin is installed, no session has run yet);
- a download is in progress right now;
- an install **failed** and is waiting out an hour's cooldown.

The wrapper said "it may still be arriving" to all three. For the third that is
simply false, and it sends a person to wait for something that will not happen while
the reason sits in a log they have not been told about.

**Decision.** One state file, written by the installer, read by everything.
`installing` when the work begins, `failed` with the reason when it does not, and
removed on success. The cooldown reads the same mark, so a wait is explicable rather
than mysterious.

A fourth state is derived rather than written: an `installing` mark older than five
minutes reports as **stalled**. A crash between setting the mark and clearing it
would otherwise leave the file claiming a download is in progress forever, which is
the one answer worse than silence.

**What each reader does with it.** The command wrapper names which of the four it is
and what to do. The session-start hook passes it to the **model** as
`additionalContext` — the only channel that exists, since everything reaching a
person arrives by way of what the model says (D-033, D-036) — so a session asked
"why isn't this working" answers truthfully instead of speculating. It is emitted
only while the binary is absent, so an ordinary session carries nothing.

**What it does not do.** It does not notify, interrupt, or poll. A person who never
asks is never told, which is the same posture as the rest of this: §3.1 says a hook
must never break a session, and a progress bar nobody asked for is a small way of
breaking one.

**Released together, because they are not separable.** v0.3.0 ships the mark, the
`where` subcommand and per-session `leave`. v0.2.0's plugin already referenced
subcommands its own pinned binary did not have — `where` printed usage — because the
plugin was edited after the release it pins. **A plugin and the binary it pins are
one artefact**, and the release script writes `VERSION` precisely so they cannot
drift silently.

**Verified from nothing**, with no binary and nothing on PATH: the model is told what
is happening, the download and daemon are up in about three seconds, the mark clears
itself, and the command that printed usage under 0.2.0 now answers.

**Revisit when** Phase 13 says whether anybody reads any of it. The honest
possibility is that a colleague never sees these messages at all, because the install
finishes before they type anything — in which case the work was in making the
failure case truthful, which is the case that matters anyway.
