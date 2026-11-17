# D-186 — A user chooses the name colleagues see when the user identity is made in the daemon's page

**Date:** 2026-09-20 · **Status:** active · **Areas:** identity, pairing

**Decision.** The user chooses the name colleagues will see in the daemon's page, as a step
of making the user identity, the first time the user does something that needs it (D-230, a
user identity is made in the daemon's page the first time the user needs one). That comes
before the user's identity first travels, so no pairing string carries an identity whose
name was not offered. `whoami` shows the name plainly and adds the line that it was guessed
only while it is.

**Support.**
- A machine's key is created by whatever touches cogmer first, which is a daemon started in
  the background by a hook, and a hook must not delay a session or speak to the person.
  D-150 (the session-start hook starts the daemon, and never delays the session). The user
  identity waits for an act of the user's, so a person is at the page when it is made.
- Every path that hands a user's identity to somebody else comes through one function,
  `printPairingInvitation`, and each needs the user identity, so the name is chosen before
  any of them runs. D-089 (the reachability warning asks about the advertised address, and
  travels with the pairing string).
- The name is the literal answer to who the user is, so it is shown always, and saying that
  the user chose it is noise about something they know. `cmd/cogmer/main.go`, `nameLine`.

**Rejected.**
- *Offering the choice only through a command.* Nobody is watching when the name is invented,
  and a user would not know to run it.
- *Showing only the provenance line, or only the name.* One hides the answer and the other
  repeats what the user knows.
