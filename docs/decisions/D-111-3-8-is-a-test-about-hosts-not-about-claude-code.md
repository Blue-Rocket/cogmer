# D-111 — §3.8 is a test about hosts, not about Claude Code

**Date:** 2026-09-21 · **Status:** active (amends §3.8; resolves the question D-110 left open)

**Context.** D-110 (host order) recorded that §3.8 — the single test applied to
every proposal before anything else — is written about one named product, and left
the resolution open rather than letting it be decided by drift. With three hosts
named, §3.8 either becomes host-general or stands as a Claude Code rule at the head
of a project that has committed to more.

**Decision.** §3.8 now reads *The host is launched and used unchanged*. It defines
**host** as the application a session runs in, names Claude Code as the host today,
and states every clause of the test about the host.

**The test itself did not change.** The same three clauses put a proposal out: a
different launch command, installing something the host does not already load, or
depending on how the host renders. It still rules out the PTY wrapper (D-034) and
still permits a view outside the session, on the same grounds as before.

**Generalizing exposed two things that had been resting on the single host.**

*First, "things the host already loads" was ambiguous, and had been carrying an
unstated word.* The original said "already loads **on its own**", and dropping it in
summary produced a phrase that could not be read at all. The sense is **something the host would load
anyway** — a sort of extension it loads already, for its own reasons, with nobody
having changed how it starts
— not a file that happens to be loaded, and not something that could be made to
load. Against Claude Code alone the distinction never had to be drawn: hooks, skills
and MCP servers are plainly in, a PTY wrapper is plainly out. Against a host nobody
has examined, "could it be made to load something?" has a yes answer for almost any
program, and the test collapses. §3.8 now says which question it is asking.

*Second, what the test says about a host offering no way in.* It says the host is
out of reach. That is the intended reading rather than a new rule — the constraint
is a product boundary, so a host that cannot be extended is a host this system does
not serve, and is never an argument for wrapping one. This does not conflict with
D-110's note that a host lacking a hook equivalent needs a different design rather
than a smaller adapter: that concerns hosts with some other way in.

**The rule generalizes; the evidence for it does not.** The reach argument — Claude
Code is a terminal program, a desktop application and an editor extension, and
extending it through its own mechanisms reaches all three — is a fact about one
product and stays stated as one. A different host has different surfaces, which
changes which mechanisms exist and changes nothing about the reasoning.

**Rejected.**

- *Leave §3.8 naming Claude Code, and add a note that it applies to other hosts.* It
  is consulted as a test, and a test carrying a footnote about which product it
  governs is a test people apply inconsistently.
- *Generalize the whole specification.* §3.8 is the framing test; the rest of the
  document describes Claude Code's actual mechanisms and is correct as written.
  Host-general language throughout would claim a generality nothing else in the
  document has earned, and would make every section quietly harder to check.
- *Wait for CoWork.* The rewrite costs the same now as later, and the ambiguity in
  "already loads" is a defect today, with one host and no prospect of a second
  mattering to it.

**Revisit when** a second host is actually supported — at which point §3.8's Claude
Code examples want a companion rather than a replacement — or if a host appears
whose extension points reach a person directly, which would make §3.8's consequence
subsection an understatement rather than a description.
