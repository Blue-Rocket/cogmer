# D-107 — Pairing again with a peer who is paired does nothing, loudly

**Date:** 2026-09-21 · **Status:** active · **Areas:** pairing, commands

**Decision.** `/peer-pair` answers by the peer's state. Given a string for a peer not recorded it
records the peer and runs the ceremony. Given a name for a peer recorded and unfinished it resumes
the ceremony and asks for no label. For a peer who is paired it says so, says since when, and
stops, before asking for any name, and `--again` re-verifies and is offered only for the case that
needs it.

**Support.**
- Repeating the ceremony is not a harmless no-op. It needs the other person at their machine at
  the same moment, so an unrequested one leaves somebody watching a page count down 90 seconds for
  a colleague who was never asked. `cmd/cogmer/main.go`, `runPair`.
- Re-verification is for somebody who read two words that did not match or was told to check, and
  a person who re-runs a ceremony without cause learns that it is routine. `cmd/cogmer/main.go`,
  `reportAlreadyPaired`.
- The question of what to call them is not asked of a peer named when recorded, since asking
  implies that the answer might change something. `cmd/cogmer/main.go`, `runPair`.
- The command takes a name because resuming is the second half of an act. A name given to the
  command once hashed into an identifier and invented a peer nobody had met, so it resolves a
  recorded peer. `cmd/cogmer/main.go`, `runPair`.

**Rejected.**
- *Repeating the ceremony for a finished pairing without being asked.* It demands coordination
  from a third party while looking idempotent.

**Revisit when** a pairing can expire, since "paired" then has to say until when and not
since when.
