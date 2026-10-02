# D-079 — Changing a room requires standing in it, which only a session has

**Date:** 2026-09-20 · **Status:** active (implemented); supersedes the `--room`
flag added in D-078 hours earlier

**Context.** D-078 made `invite` and `revoke` refuse to guess a room, and gave them
`--room <name>` so a terminal could say which. That answered the wrong question.

The question is not *which* room. It is whether a terminal has any **standing** to
change one. §22 puts membership in a Claude Code session; a room's guest list is its
members' concern; at a terminal you are a member of nothing. Naming a room there is
reaching into a room that belongs to a session you are not in, which is a different
act from choosing among rooms you are in — and the flag made it look like the same
act, spelled more carefully.

**Decision.** `invite` and `revoke` require the invoking session to be in the room.
No flag, no fallback, no pointer. At a terminal they are refused and say where the
command belongs:

    this changes who can read a room, and only a session that is in one can do it.
    Run /room-invite or /room-revoke inside the Claude Code session that is in the room.

**What stays.** `log`, `guests` and `conflicts` still fall back at a terminal,
because showing you a room is not an exercise of standing — it is your own machine's
data on your own screen, and each names the room it shows.

**Why this took three attempts, which is the part worth keeping.** The same
question — *which room does this act on* — was answered three times by reaching for
whatever was nearest: an ambient environment variable, a stored pointer, then an
explicit flag. Each felt like an improvement on the last. None of them asked whether
the caller had any business acting on the room at all, and the flag was the most
misleading precisely because it looked most careful.

**Revisit when** something other than a session can hold membership. Nothing is
planned to; §22 is explicit, and every drift away from it so far has been an
accident rather than a proposal.
