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

- R0. Whether "peer" comes to mean a user as cogmer knows them, with "machine" for a device,
  or "peer" stays a machine and "colleague" and "user" carry the person.
- R1. Whether a user's own machines exchange a room's events with each other. D-222 says
  they are peers "for identity records and for nothing else", and D-103 leaves this user out
  of a room's sync targets, so a room's events reach the user's second machine only through
  a colleague.
- R2. Whether the colleague list, verification, labels, guest lists and the show-only choice
  belong to the user, carried between the user's machines, or to each machine.
- R3. How the view and the room status show a colleague with two machines when one of them
  cannot be reached.
- R4. How a user's own turn from another machine is shown in the view and framed in injected
  context, since the block says its content is other people's.
- R5. Whether a machine's name, which its user chose, is shown to colleagues, and if so as a
  field of its own marked as asserted.
- R6. When a user identity is created relative to a machine's first use and its first
  pairing string, and how a user's second machine learns the user's rooms.
- R7. Whether a room's name is unique among one machine's rooms or one user's.
- R8. Which keys the two pairing words are derived from.
- R9. Which command prefix holds the commands for a user's own machines.
