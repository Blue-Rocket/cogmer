# Inventory: decisions written with the machine as the participant

This is working material for the open item "A user's machines are separate identities to
colleagues, and a cloud session has no identity of its own to sign with" in `docs/open.md`,
and is deleted with it. The maintainer decided on 2026-10-23 that the user, and not the
machine, is the primary entity. A colleague pairs with, verifies, labels, admits and
attributes a user. A machine is one of a user's devices, with an address, a key certified
through the user's chain, its own sequence numbers and sessions, and it is reached and
removed by itself.

Three read-only sweeps of `docs/decisions/` on 2026-10-23 covered the areas identity,
pairing, trust, admission, rooms, sync, capture, view, storage and transport. No decision
is reversed. Each one listed below needs either new words or a partial rewrite (W-38), and
some wait on a root question in section 2. The quotes the sweeps found are in this session's
transcript, and each rewrite reads its file again before changing it.

## 1. What each decision needs

Wording only: the meaning carries over with "user" or "colleague" for "peer", and "machine"
where a device is meant.

D-018, D-021, D-023, D-026, D-035, D-044, D-045, D-050, D-051, D-054, D-055, D-059, D-062,
D-071, D-083, D-093, D-094, D-099, D-101, D-106, D-107, D-108, D-116, D-137, D-139, D-140,
D-142, D-143, D-144, D-152, D-153, D-154, D-156, D-157, D-162, D-163, D-164, D-165, D-169,
D-174, D-180, D-182, D-184, D-185, D-188, D-207, D-213, D-216, D-221.

Partial rewrite, once the root questions are answered:

| decision | what changes |
|---|---|
| D-024 (admission is a guest list) | a guest is a user, proved by a machine certified to that user |
| D-025 (a machine knows peers, and a room admits guests) | who holds the colleague list and guest lists, question R2; the title |
| D-027 (a sequence conflict is quarantined) | the conflict key is the machine, never the user |
| D-029 (the sequence lives with the identity) | the sequence lives with the machine; the title, and D-220's citation of it |
| D-042 (peer identity is an Ed25519 key pair) | its revisit condition has fired: a user's key is the master key, and a machine's key is certified to it |
| D-052 (verifying a peer is its own act) | the daemon dials a colleague's machines and keeps one certified to that user |
| D-053 (`pair` is the machine-scope act) | `pair` is the act with a colleague, once per user; the title |
| D-060 (`Store.Append` never derives a sequence) | its revisit condition, two machines under one identity, is answered by a counter per machine |
| D-073 (forgetting a peer discards their admissions) | forgetting a colleague covers all their machines; question R2 |
| D-074 (a name means one key) | a name means one master key, and a new machine is no key change |
| D-086 (the view is the surface) | creating and recovering a user and removing a machine are acts in the view |
| D-103 (an address belongs to a peer) | an address belongs to a machine; sync targets are the machines of guest users; question R1 |
| D-141 (a machine's key is created once) | recognition rests on the master key; a cloud session's key is per session |
| D-159 (joining admits the inviter) | the invitation carries the inviting user and the inviting machine's address; question R6 |
| D-198 (the view marks only other people's turns) | "mine" is any of this user's machines; question R4 |
| D-222 (a machine tells every machine it knows) | what passes between a user's machines besides identity records; questions R1 and R2 |

Waiting on a root question before the words can be chosen: D-048 (R8), D-090 and D-095 and
D-097 and D-099 (R5), D-096 (R9), D-105 (R6), D-112 and D-149 and D-204 (R4), D-135 (R7),
D-186 (R6), D-194 (R3), D-200 (R2), D-214 (R4), D-225 (R2).

The machine is the right subject in the decisions on listeners, transport, addresses, TLS,
the daemon, sequence counters, delivery state and sessions, and a rewrite leaves them about
the machine: among them D-014, D-016, D-027, D-029, D-030, D-031, D-056, D-060, D-061, D-064,
D-065, D-068, D-071, D-072, D-079, D-080, D-088, D-089, D-091, D-092, D-102, D-104, D-133,
D-145, D-146, D-160, D-167, D-168, D-169, D-183, D-187, D-192, D-196, D-197, D-202, D-206,
D-209, D-210.

## 2. Root questions

Answered by the maintainer:

- R1, on 2026-10-24: a user's sessions in one room share context with one another as they do
  with a colleague's, whichever of the user's machines they are on. A room's sync targets are
  the machines from which a session has joined the room, the user's own included, excluding
  only this machine, and D-222 loses "for nothing else". A machine where no session has
  joined a room holds nothing of it. A colleague's machine learns the address of a newly
  joined machine from the signed address in that machine's sync requests. This also settles
  most of R4: such a turn is injected, as a colleague's is.
- R2, on 2026-10-24: the colleague list, verification and labels belong to the user and are
  copied to every machine of the user. A room's guest list belongs to the user and is copied
  between the user's machines that have joined that room. Forgetting a colleague on one
  machine forgets them on all of them. The show-only choice belongs to each machine, made
  when a session there first joins the room.
- R6 in part, on 2026-10-24: a machine is added from a machine the user already has. The new
  machine's page shows its pairing string, the user enters it on the existing machine, both
  pages show the two words, and the existing machine certifies the new one and sends it the
  user's machine list with addresses, and the user's colleagues and labels. Typing the
  recovery phrase into a new machine's page is for recovery only. Each exchange of machine
  lists also repairs the addresses it carries, since a user's machines may share no room.
- R6, the rest, on 2026-10-24: a machine's key is made at its first use, as now, and the user
  identity is made in the daemon's page the first time the user does something that needs
  it: a first pairing string, pairing, creating or joining a room, or adding a machine. The
  page asks first whether the user has used cogmer on another machine. Without one, it
  creates the user: it shows the phrase, has the user confirm words from it, takes the name
  colleagues will see, certifies the shared key and this machine, and drops the master. With
  one at hand, it adds this machine from that one, and with none left, it recovers from the
  phrase. Until then, a command that needs the identity points to the page and does nothing
  else. A user's machines share the list of rooms the user is a guest of, each with its
  identifier, its name and the user's machines that have joined it, and not its guest list or
  events. A session on another of the user's machines joins such a room by name, and its
  machine then fetches the guest list and the events from a machine that has joined.
- R4, the rest, on 2026-10-24: a user's own turn from another machine is injected as a report,
  framed as a colleague's is, under the label "you", with a field the receiving machine sets
  saying it is this user's own turn from another session. It is never framed as words
  addressed to this session.
- R5, on 2026-10-24: every turn carries the name of the machine it came from, as a field of
  its own marked as asserted by its user, in the view and in injected context, and never
  joined to a name.
- R0, on 2026-10-24: a peer is a user as cogmer knows them, identified by the user's master
  key, and a machine is one of a peer's devices. Sentences that mean a device say "machine",
  and the `peer-` commands keep their names and act on users.
- R9, on 2026-10-24: one command, `/cogmer:self-machines`, opens the daemon's page, where the
  user adds, removes and renames machines and recovers, and `/cogmer:self-status` also lists
  the user's machines and whether each can be reached.
- R3, on 2026-10-24: a colleague is one line in the room status and the view, reachable when
  any of their machines that joined the room can be reached from this machine, and named as
  unreachable, with for how long, only when none can. The user's own machines are listed one
  by one, by name, with whether this machine can reach each: those that joined the room in
  the room status, and all of them in `/cogmer:self-status`.

Open:

- R7. Whether a room's name is unique among one machine's rooms or one user's.
- R8. Which keys the two pairing words are derived from.
