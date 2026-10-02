# D-114 — §3.1 is a duty not to harm the session, not a claim about data locality

**Date:** 2026-09-21 · **Status:** active (rewrites §3.1; retires "local first")

**Context.** §3.1 was titled "Local first" and said two things: a person's local
daemon owns their collaboration experience, and Claude Code keeps working when
peers disappear, the network is unavailable, or synchronization fails.

Two problems, found by testing the title against its use. **It named the wrong
half.** Twelve citations of §3.1 across the repository are about hooks (6), silence
toward the person (4), exiting 0 (2), a dead daemon (2) and a broken session (1);
none is about data locality. **And the section did not contain what they cite.**
§3.1 mentioned no hook, no exit code, and no dead daemon — `main.go` says
"Invariant from §3.1: Claude Code must keep working when the daemon is down" and
that failure was not on §3.1's list. Neither was silence toward the person, which
`common.sh` and D-107 both attribute to §3.1.

"Local first" also carried a borrowed claim. Rendezvous across NAT needs a relay
somebody else operates, D-029 (losing a room database) makes recovery a refetch
from peers, and nothing before the first invitation is captured at all — so the
phrase promised more than this system delivers, in its first line, to readers who
would never reach the definition.

**Decision.** §3.1 is **First, do no harm**. A person's Claude Code session belongs
to them and this system is a guest in it; before anything else it must not make
that session worse, and every other requirement is subordinate to that.

**Stated as a duty, not a list**, because a list invites the reading that an
unlisted harm is permitted. The known cases: it must not fail the session (now
including a dead daemon and a failed hook), must not slow it, must not be noisy,
must not spend the context window carelessly (§21), must not take its turn (§3.7),
and must not draw inside it. Claude Code communicating only with the local daemon
survives as the *mechanism* that makes most of those achievable, rather than as the
principle.

**What this unifies.** §3.7 (a remote event never drives a session), §21 (context
limits), D-150 (starting the daemon never delays the session) and D-038 (the view is a
separate program, so a busy room never costs somebody their pane) are all the same
obligation, and nothing said so. Each read as a local judgment; together they are
one duty with six known cases.

**Rejected.**

- *Keep "Local first" and complete the list.* The gaps would close and the title
  would still point at the half nobody cites. A reader looking for the hook
  contract has no reason to open a section about where data lives.
- *"A session degrades to solo, never to broken."* Vivid, and it covers only
  failure. Slowing the first prompt, spending the context budget and interrupting
  the working pane are harms with nothing broken, and they are the ones a
  well-meaning change actually causes.
- *"Nothing here can break a session."* Same defect, and negative framing says
  nothing about what a person does get.
- *Sweep "local-first" out of `decisions.md` too.* Entries there record reasoning
  as it stood and referring to a principle by the name it had then is accurate.
  The specification review likewise. The term survives once in the spec, at §11's
  "local-first CRDT such as Automerge", where it names a category of software and
  makes no claim about this one.

**Revisit when** a harm is found that none of the six cases anticipates — which is
expected, and is why the duty is general. Add the case; do not narrow the duty.
