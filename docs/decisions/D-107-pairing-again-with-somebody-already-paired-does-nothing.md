# D-107 — Pairing again with somebody already paired does nothing, loudly

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** Asked whether the pairing command is idempotent for a peer who has
completed all or part of it. Partly: resuming an unfinished pairing worked, and
running it on a finished one silently began the whole ceremony again.

**Repeating the ceremony is not a harmless no-op.** It needs the other person at
their machine at the same moment, so an unrequested one leaves somebody watching a
page count down ninety seconds for a colleague who was never asked. A command that
looks idempotent and quietly demands coordination from a third party is worse than
one that refuses.

**So the three states answer differently.**

Not recorded, given a string: record and run the ceremony. Recorded but unfinished,
given a name: resume — this is the second half of an act, and the reason the
command accepts a name at all. Already paired: say so, say since when, and stop.

**A string for somebody already paired changes nothing, including the address.**
This entry first took the address out of it, on the reasoning that a colleague who
moved has no other way to tell you. That was wrong twice over, and is superseded by
D-108.

**Re-verification stays available and must be asked for.** `--again` exists for the
case it is actually needed in: somebody read two words that did not match, or was
told to check. The message names that case rather than offering the flag as a
general option, because a person who re-runs a ceremony without cause learns that
the ceremony is routine, and it is the opposite.

**Checked before anything is named.** The question "what will you call them" must
not be asked of somebody who was named when they were recorded — asking it implies
the answer might change something.

**Revisit when:** a pairing can expire, at which point "already paired" needs to
say until when rather than since when.
