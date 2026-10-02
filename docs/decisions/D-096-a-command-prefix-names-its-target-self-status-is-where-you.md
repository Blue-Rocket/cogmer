# D-096 — A command prefix names its target; `/self-status` is where you are

**Date:** 2026-09-20 · **Status:** active (implemented) · **Corrects** D-095

**Context.** D-095 wrote the prefix rule down as naming the **activity**, so that
`/peer-pair` with no arguments printing your own pairing string would not count as a
violation. Pointed out that the existing prefixes plainly name the **target**:
`peer-list`, `peer-forget`, `room-create`, `room-revoke`, `room-join` — every one of
them says what is being acted on.

**That was a rule bent to fit an exception, and the exception was the defect.** Two
commands printed your own string: `/peer-pair` with no arguments, and `/peer-list`
when there are no peers. Both are about other people and both printed you.

**`/self-status` now owns it**, mirroring `/room-status`: "status" already means the
current state of this thing in this set, and you read it for the same reason — what
am I, where am I, can anybody reach me. It replaces nothing, because until now there
was **no slash command for your own identity at all**; `whoami` existed only as a
terminal command, which D-086 says is not a surface to rely on.

`/peer-pair` with no arguments now explains the two halves of pairing and points at
`/self-status`. `/peer-list` with no peers does the same. Neither prints a string
that is not about its target, and `printPairingInvitation` has one caller again.

The cost is a hop for somebody who guessed `/peer-pair` first. Worth paying: the set
will grow, and a rule carrying a documented exception is one people stop trusting.

**No `/help`, and not for want of noticing.** Claude Code owns that name, and the
slash menu already lists all twelve commands with their descriptions, which is the
discovery surface we actually control. What is missing is an entry point that says
what the tool *is* — and that command would be `/<product-name>`, which is what
Phase 13 is blocked on. Do not invent a placeholder: a command name is harder to
change later than a directory is, and the whole point of the blocker is that the
name gets one chance.

**Revisit when:** the name is settled, at which point an overview command has
somewhere to live.
