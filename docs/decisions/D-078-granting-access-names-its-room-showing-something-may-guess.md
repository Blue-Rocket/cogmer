# D-078 — Granting access names its room; showing something may guess

**Date:** 2026-09-20 · **Status:** the showing/changing distinction stands; the
`--room` flag it introduced is **superseded by D-079**, which found that a terminal
has no standing to change a room at all, so there is nothing for it to name.

**Context.** Asked why a terminal command should be able to act without knowing
which room it applies to. Following the question properly turns out to split what I
had been treating as one decision.

**`invite` grants access.** Granting it in the wrong room means a person reads a
conversation nobody invited them to, and **`revoke` cannot undo it** — it stops
future reading and does not un-read. The answer to a question with that consequence
was a stored pointer, set possibly weeks earlier, never displayed before the command
acted on it. That is not good enough, and the command with the worst failure mode
was the one leaning hardest on the weakest answer.

**`log` and `guests` show you something.** Guess wrong there and you see a room you
did not mean, on your own screen, and every one of those commands names the room it
is showing. Visible and free.

**Decision.** `roomToChange` does not guess: inside a session it uses that session's
room, and at a terminal it requires `--room <name>`. `roomToShow` may still fall
back, because being wrong is self-announcing. `invite` and `revoke` take the first;
`guests`, `log` and `conflicts` take the second.

The refusal says what to do rather than only what is missing:

    name the room: this changes who can read it, so it will not guess.
      cogmer <command> --room <name>
    `cogmer rooms` lists them. Run it from inside a session and it uses
    that session's room.

**The slash commands need no flag and that is not an oversight.** They run inside a
session, so the session's room answers. The command file says so, so that nobody
later adds a flag to make it "consistent" with the terminal form — the two differ
because one has a session to ask and the other does not.

**What this leaves of the fallback.** Only read-only commands at a terminal. That is
a small enough surface to defend on its merits: it saves typing where the cost of
being wrong is noticing and retyping.

**Revisit when** a read-only command's output is used as the basis for something
consequential — piped into a script that invites people, say. The distinction drawn
here is between *showing* and *changing*, and it stops holding the moment something
shown is fed into something that changes.
