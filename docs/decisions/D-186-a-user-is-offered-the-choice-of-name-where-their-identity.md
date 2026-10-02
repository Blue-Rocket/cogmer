# D-186 — A user is offered the choice of name where their identity first travels

**Date:** 2026-09-20 · **Status:** active · **Areas:** identity, pairing

**Decision.** The offer to choose a name is made in `printPairingInvitation`, where a user's
identity first travels. `whoami` shows the name plainly and adds the line that it was guessed
only while it is.

**Support.**
- There is no earlier moment. Identity is created by whatever touches it first, which is a
  daemon started in the background by a hook, and a hook must not delay a session or speak to
  the person. D-150 (the session-start hook starts the daemon, and never delays the session).
- Every path that hands a user's identity to somebody else comes through that one function,
  so the offer belongs with the string and not with a command. D-089 (the reachability warning
  asks about the advertised address, and travels with the pairing string).
- The name is the literal answer to who the user is, so it is shown always, and saying that
  the user chose it is noise about something they know. `cmd/cogmer/main.go`, `nameLine`.

**Rejected.**
- *Offering the choice only through a command.* Nobody is watching when the name is invented,
  and a user would not know to run it.
- *Showing only the provenance line, or only the name.* One hides the answer and the other
  repeats what the user knows.
