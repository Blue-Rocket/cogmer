# D-114 — Section 3.1 is a duty not to harm the session, and says nothing about where data lives

**Date:** 2026-09-21 · **Status:** active · **Areas:** project, daemon

**Decision.** §3.1 is "First, do no harm". A user's Claude Code session belongs to them and this
system is a guest in it, so before anything else the system must not make that session worse, and
every other requirement is subordinate to that. It is stated as a duty and not as a list.

**Support.**
- A list invites the reading that an unlisted harm is permitted. The known cases are that the
  system must not fail the session, including with a dead daemon or a failed hook, must not slow
  it, must not be noisy, must not spend the context window carelessly, must not take its turn and
  must not draw inside it. §3.1 (first, do no harm), §21 (context window management), §3.7 (a
  remote event never drives an interactive session).
- Claude Code communicating only with the local daemon is the mechanism that makes most of those
  achievable, and it is not the principle. D-150 (the session-start hook starts the daemon, and
  never delays the session), D-038 (separating the room from the session is correct on its merits).
- A title that said "local first" promised more than the system delivers, since rendezvous across
  a NAT needs a relay that somebody else operates, recovery from losing a room is a refetch from
  peers, and nothing before the first invitation is captured at all. D-068 (tailcat is the
  cross-network transport, behind our own dialer), D-029 (losing a room database does not end
  membership), D-022 (a room begins when someone is invited into it).

**Rejected.**
- *Keeping "local first" and completing the list.* The gaps would close and the title would still
  name the half nobody cites, and a reader looking for the hook contract has no reason to open a
  section about where data lives.
- *"A session degrades to solo, never to broken."* It covers failure alone, and slowing the first
  prompt, spending the context budget and interrupting the working pane are harms with nothing
  broken, which are the ones a well-meaning change causes.
- *"Nothing here can break a session."* Its negative framing says nothing about what a user does
  get.

**Revisit when** a harm is found that none of the cases anticipates, which is expected and is why
the duty is general. The case is added and the duty is not narrowed.
