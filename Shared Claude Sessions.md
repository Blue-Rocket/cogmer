# Peer-to-Peer Shared Real-Time Collaboration for Claude Code

cogmer is a peer-to-peer daemon that replicates one Claude Code conversation between
people working separately, with no central collaboration server and no account
anywhere. Claude Code talks only to a daemon on the same machine. Section 3.1 governs
everything else: a user's session is never worse for having cogmer installed.

## 1\. Objective

cogmer lets several people, each using their own Claude Code subscription and their
own Claude Code session, work in one shared conversation without a central
conversation server. In a room:

- each user sees the prompts their colleagues send to Claude, and Claude's responses;
- those turns appear in near real time;
- each user keeps working in their own session, on their own subscription, and
  inference is ordinary Claude Code usage;
- each user's Claude receives the recent turns of their colleagues;
- the room's history is replicated between the machines in it, and no cloud service
  holds an authoritative copy.

---

# 2\. Core Concept

Each user's machine runs:

1. Claude Code
2. the collaboration hooks
3. a `cogmer` daemon
4. a local event store
5. a local view of the room

The daemons talk directly to one another.

```
             NO CENTRAL CONVERSATION SERVICE

 David                 Alice                 Carlos
   │                      │                     │
Claude                  Claude                 Claude
   │                      │                     │
hooks                   hooks                  hooks
   │                      │                     │
daemon ◄───────────────► daemon ◄────────────► daemon
   │                      │                     │
SQLite                  SQLite                 SQLite
   │                      │                     │
local UI                local UI               local UI
```

Every machine in a room holds its own replica of the room, and no server holds a
canonical copy.

---

# 3\. Architecture Principles

## 3.1 First, do no harm

A user's Claude Code session belongs to them, and cogmer is a guest in it. cogmer
never makes that session worse, and every other requirement in this document is
subordinate to this one, because collaboration is never worth degrading the session
that hosts it.

This is a duty rather than a list. The cases below are its known cases, not its
extent, so a harm they do not name is still forbidden.

Claude Code keeps working normally when the daemon is not running or dies during a
session, when a hook cannot reach the daemon or fails for any other reason, when
every other peer disappears, when the network is unavailable, and when
synchronization fails.

Every path by which a hook fails exits 0 with empty output, so a dead daemon means
no collaboration and never a broken session.

Starting the daemon never delays the first prompt. It is the session-start hook's
job, and waiting on a daemon the user did not ask for is worse than not having one.

Failure is silent to the user. The daemon records its own troubles in its own log,
where anybody who looks will find them, and says nothing to a user who did not ask.

Injected context is bounded (section 21), because it spends the context window the
user relies on for their own work.

A remote event never makes an interactive session take a turn (section 3.7).

cogmer draws nothing inside the session. A view of the room is a separate program,
so a busy room never costs a user the pane they are working in.

Claude Code communicates only with the daemon on its own machine, so nothing remote
sits on a path where Claude Code could notice it failing. Each user's daemon owns
that user's collaboration.

```
Claude Code
     │
     │ localhost
     ▼
cogmer daemon
     │
     ├── local event store
     ├── local browser UI
     ├── context synchronization
     └── peer synchronization
```

---

## 3.2 Replicated rooms

A room is a replicated event stream shared by a set of linked Claude Code sessions.

A room has two identifiers, and they are not interchangeable: an identifier such as
0f7a4e6c-2b91-4d0a-9c3e-7f1d8a5b2c44, and a name such as misty-canyon.

A room's identifier is its identity for synchronization. It is generated, globally
unique, never reused and never changed. Every event carries it, and replication,
deduplication and storage all key on it.

A room's name is for people. It is generated at the same moment, so that a room can
be spoken aloud, typed without copying, and recognized in a list.

A name is never authoritative. Nothing synchronizes, routes, deduplicates or stores
by name, and two unrelated rooms may carry the same name, because treating a name as
a key would eventually merge two unrelated conversations.

Neither identifier is named after, derived from or bound to a project, a repository
or a working directory, and nothing resolves a room from a directory.

Every peer in a room converges on the same history. David may hold:

```
A B C D E
```

while Alice, disconnected for a while, holds:

```
A B C D F
```

After they synchronize, both hold:

```
A B C D E F
```

---

## 3.3 No authoritative peer

No user's machine is a room's permanent host. Any peer in a room can create events,
receive them, go offline, return later and fetch what it missed, and no peer needs
another to be online in order to keep working.

---

## 3.4 Preserve actual conversation

cogmer never replaces a conversation with a summary. A room keeps each user's actual
prompts, Claude's actual responses, who said each, in what order, and in which
session, because part of what a colleague gains is seeing how another user
approached a problem and how Claude responded.

---

## 3.5 Independent Claude sessions

Each user keeps their own Claude Code process, subscription, working directory,
context window and session. Synchronization happens around those sessions, and
nothing inside a Claude Code session is shared between machines.

---

## 3.6 Session-scoped rooms

A room lasts as long as the sessions that joined it. There is no standing room that
people drift into and out of over weeks.

Membership is held by a Claude Code session, identified by its session ID. One peer
creates a room at the moment they invite someone into it, others join through that
invitation, and the room closes when its members have left it.

A session that becomes inactive has not left. Sessions can be resumed, so a process
that exits makes a member absent rather than gone, and a resumed session returns to
its room because its session ID survives resumption.

A session belongs to one room at most, and never joins a second. A user who starts a new Claude Code
session creates a new room or is invited to one, and does not resume an old room.

Joining a room is always a deliberate act by every participant, and nothing about a
room is inferred from a repository, a directory or a project, so a user cannot
publish one project's conversation into a room opened for another without meaning to.

A closed room is kept as an archive that can be read but never rejoined.

A room's history is bounded by the work that produced it, not by the age of a
project. So unseen conversation cannot build up beyond the pairing that produced it,
and a session that joins late can be given the room from its beginning, within the
injection limits, rather than a truncated tail.

---

## 3.7 A remote event never drives an interactive session

A session a user is working in takes a turn when that user asks it to, and at no
other time.

The rule is about what starts a turn, never about what a turn reads. An injected
event is part of the context and bears on what the model produces, since that is
what injecting it is for. It is never the reason the model runs.

Remote events are stored, displayed and queued. They enter an interactive session's
context at its next turn the user starts, and never before. A peer may place
something in front of a user's Claude, and only that user can cause it to be read.

A remote event never prompts, resumes or otherwise drives an interactive session,
never makes it consume queued context ahead of a turn the user starts, and any
scheduled or triggered work that affects it originates on the user's own machine.
The host offers ways to start Claude, so a daemon could drive a session, and this
rule forbids it.

An interactive session is a working state, not just a process. A turn that arrives
unbidden spends the context window the user relies on, may act on their working
tree in the middle of their thought, and takes away their ability to reason about
what their own session has seen and when.

Section 3.1 requires a user's session to survive every peer disappearing, and this
section is its converse: a user's session is undisturbed by every peer arriving.

Any Claude run a peer event causes outside the interactive session neither borrows
that session's context nor interrupts it.

---

## 3.8 The host is launched and used unchanged

The host is the application a session runs in. Claude Code is the host this
specification describes. The rule in this section is stated for hosts in general,
because a system that can only be installed into one product has chosen that
product permanently.

A user starts the host the same way they would without cogmer, and uses it the same
way. cogmer is installed into the host and never around it.

A user installs only artifacts of a type the host documents and demonstrates as an
extension mechanism: in Claude Code, hooks, skills, MCP servers and the plugin that
carries them. The rule is about the type, not about what the artifact does, so a hook
that replicates a conversation to a peer is a hook. Nothing replaces the command,
wraps the process, interposes a terminal or patches the application, and a runtime
that loads a shim on the host's behalf is loading it for cogmer, not for the host.

This is a boundary on the product before it is a technical one. An alternate way to
launch the host would make terminal emulation, pseudo-terminals on each operating
system, editor terminals, shell integration and every surface the host adds later
into cogmer's responsibility.

It also reaches every form the host takes. Claude Code is a terminal program, a
desktop application and an editor extension, and a system that extends it through
its own mechanisms works in all three without knowing any of them exist.

The rule can be applied without argument:

- a proposal that requires a user to start the host differently is out;
- a proposal that requires installing an artifact of a type the host does not
  document as an extension mechanism is out;
- a proposal that requires understanding how the host renders is out.

A host that offers no way in fails this rule rather than relaxing it. If a host has
no extension point cogmer can use, that may put the host out of reach, and it never
licenses an alternate way to launch it.

### Consequence: not everything is equally achievable

A host's extension points deliver to the model. In Claude Code, hooks supply
context, skills supply instructions and MCP servers supply capability, and each
reaches a user only through what the model then says.

So the two halves of cogmer are served unequally. Capture, synchronization and
context injection work everywhere, because they face the model. Showing a user a
colleague's turn as it arrives is best effort, because nothing in the extension
surface displays anything.

A view outside the session is permitted, since it is a separate program a user may
run and not a change to how they start the host. Taking ownership of the host in
order to draw inside it is not.

---

# 4\. Networking

## No network provider is required

No network provider is part of a room's identity, its membership or the replication
protocol. A room is a set of linked sessions and an event stream, and how bytes
reach another machine sits beneath that and is interchangeable.

cogmer works with no virtual private network of any kind when peers can already
reach one another. Two users on the same network are the simplest case, and need no
account, no external service and no configuration.

A private network such as Tailscale can connect peers that could not otherwise reach
each other. It is one provider among several, and never a prerequisite.

Anything that makes a particular provider necessary, in identity, in discovery, in
the invitation format or in replication, is a defect.

## Preference order

```
same network, direct
      │
      ▼
private network provider (for example Tailscale)
      │
      ▼
internet peer-to-peer
      │
      ▼
relay
```

A connection tries these in order and uses the first that succeeds. Which one
connected appears in no room's identity, in no event, and in no ordinary output a
user sees.

What synchronization means is independent of the transport that carries it.

## How an address is obtained

A daemon discovers the address it is reachable at, and never derives one from its
own hostname. A hostname is often not a routable address: it may be a purely local
name, may resolve only by mDNS, may resolve to a private address no remote peer can
reach, and is often not the name the transport knows the machine by.

A daemon that cannot determine a reachable address says so, rather than offering an
invitation that cannot be used.

## What kind of address it is

Addresses differ in what they name, and that decides how long each stays true and
who may be told it.

An overlay address names a node and where to find it. It carries the node's public
keys and a relay through which an introduction can be made, after which the path
upgrades to a direct one if the two machines can reach each other. The relay is a
rendezvous, where a node can be found and not where it is, so moving a machine does
not change its overlay address.

Everything an overlay address is built from, the node key and the relay once chosen,
is persisted with the identity, so the address survives both moving and restarting.
An address assembled afresh at each start would silently invalidate every pairing
string and invitation the machine had issued.

An address contains no secret. A pairing string is pasted into chat and read aloud,
and who may enter a room is decided by who is on its guest list, never by who holds a
string.

A public address names a place reachable from anywhere. It is true of one position
on the network, and a machine that moves has left it.

A private address names a place reachable only from one network. Elsewhere it names
a different machine, so reaching it from another network does not fail: it succeeds
against a stranger.

A loopback address names a place reachable only from the machine that holds it. In
anything offered to a colleague, it means no address was learned.

## The lifecycle of a peer address

An address belongs to a peer and is recorded once, against that peer's identity and
nowhere else, so that it is always known whose address to try, to replace and to
blame. Where a room needs to know whom to poll, the answer is its guest list and the
address recorded for each guest.

An address enters in one of four ways, and each says something different about how
far it can be trusted:

- from a pairing string a user typed in, which is a colleague's claim about where
  they listened when they sent it;
- from an invitation, which carries the room and where its host can be reached;
- from a synchronization request, which carries the address the caller advertises,
  so a member that has moved says so by continuing to poll;
- from discovery on the network in use, the only source that reports where a peer is
  rather than where it was.

Only an address that identifies a machine is advertised. An overlay address may
appear in a pairing string, an invitation or a stored record. A private address may
not appear in any of them, because an advertisement outlives the network it was true
on. A private address enters only from discovery, where it is a fact about the
network in use.

An address is a candidate, not a fact. Which address works is a property of the pair
of machines, so only the side attempting the connection can find out. Candidates are
ordered by kind, same host, same network, public, then relayed, and the first few
are attempted together, so that one that no longer answers costs a round trip
rather than a timeout.

The address that answered is remembered, with the time it answered, and tried first
next time. The record is a hint about where a machine was, and is discarded when it
stops working.

### How an address is invalidated

Nothing announces that a machine has moved, and the evidence that arrives instead
comes in kinds that are treated differently.

Silence is weak evidence. A refused connection, a timeout and a missing route are
what a wrong address produces, and equally what a sleeping machine, a closed laptop
and a hotel's captive portal produce. So silence moves an address down the order of
candidates, and never removes it.

A wrong key is strong evidence. Every connection is pinned to a key already verified
(section 25), so an address that answers with a different key is no longer that
peer's, and it is invalidated at once.

An address that is not refreshed grows less likely to be true, at a rate that depends
on what it names. A private address does not outlive the network it was learned on.
A public address may be tried for longer. The keys in an overlay address do not age,
and the relay network keeps its rendezvous current.

A peer that advertises a different address has moved, and the new address replaces
the old one. The old one is superseded rather than disproved, since the peer may
return to it, but keeping both would grow a peer's addresses with every network
they visit.

Forgetting a peer removes every address recorded for them, as it removes their
admissions (section 12), so a later meeting is a first meeting.

### What a failed key check means

Pairing is pinned to the key the pairing string carries, so a stale address makes a
pairing fail and never connects it to the wrong machine.

Verifying a peer again uses the address on file, which may be months old, and is
pinned just as strictly.

Synchronization accepts any peer this machine knows at the connection, and the signed
request inside the connection says which peer it is, because a room records where its
members listen rather than which member listens where.

During synchronization, an address that answers with an unexpected key has changed
hands, which is routine. While verifying a named peer, the same event is what
substitution looks like, and the user is told rather than the attempt being retried.

A connection is abandoned when the key on the other side is not one this machine
knows, before this machine presents anything of its own, so whoever holds a
reassigned address learns that something tried to connect and nothing about who.

### Whether an address is ever rehabilitated

An address is rehabilitated by evidence, and never by time. Time can cast doubt on
an address, but cannot restore confidence in one, because nothing has happened.

An address demoted for silence returns to ordinary standing the moment it answers,
which is why silence demotes rather than removes. An address set aside for age is
current again when the peer advertises it afresh, or when it answers.

An address invalidated by a wrong key returns only by presenting the right key. A
reassigned address can be reassigned back, and a connection that succeeds against
the expected identity is proof of that.

---

# 5\. Local Daemon

The daemon is one lightweight process, `cogmer`. It serves an interface for the Claude
Code hooks and the local view on the machine's loopback address, such as
`127.0.0.1:4782`, and a separate interface for synchronizing with peers.

```
                    cogmer
                         │
        ┌────────────────┼─────────────────┐
        │                │                 │
   Hook API          Local UI API       Peer API
 localhost only      localhost           P2P
        │                │                 │
   Claude Code        Browser          teammates
```

The session-start hook starts the daemon, not the user, and the daemon outlives any
one session. Section 29 says how cogmer is installed.

The daemon:

- captures local events;
- stores events;
- sends events to the local view;
- connects to the peers an invitation names;
- exchanges the events each side lacks, without duplicates;
- supplies each session with the colleagues' turns it has not seen;
- creates, joins and closes rooms;
- tracks which sessions are members of which room;
- tells a member that is absent from one that has left;
- archives a room once it has closed.

One daemon serves several rooms at once. Every request identifies the room it
concerns, directly or through the session it comes from, and the daemon never guesses
one.

---

# 6\. Identity

Every peer has a stable identity: a key pair, whose public key is the peer's
identifier; a name derived from that key; the name its user goes by; an identifier
for the user; and a label for the machine. Every Claude Code session also has its own
identifier.

## Peer names

A peer has two identifiers, for the same reason a room does: an identifier derived
from its public key, which is verified, signed with and keyed on, and a name, such as
quiet-otter, which people can read, say and recognize in a transcript. The name is a
word pair, an adjective and an animal.

A peer's name is derived from its identifier, never chosen, so that a peer cannot
simply declare itself to be someone else. A name a peer picks for itself is a claim
about who it is, and the name attribution rests on is never a claim.

A name short enough to say aloud has little entropy, and an identity can be generated
freely until its derived name matches a chosen target. So a peer's name is a
mnemonic for an identity already known, never an introduction to a stranger. A name
never admits, routes, deduplicates or establishes trust. Two peers may derive the
same name, and an interface shows that rather than hiding it.

A name is shown as itself only for a peer whose identity has been verified. Any other
peer is shown as unverified, with its identifier. Verification gates what is
exchanged (section 25), so an unverified peer's turns do not arrive at all, and the
marker is a backstop.

## Three names, and what each is for

A peer is referred to by three names, and confusing them is how a colleague ends up
trusting the wrong one:

- the derived name, such as quiet-otter, which is computed from the key and chosen by
  nobody;
- the display name, such as David, which is what the peer calls itself, and is a
  claim;
- the label, such as alice, which is what you call the peer, recorded when you
  paired.

The derived name cannot be chosen, so attribution rests on it. Its work is momentary:
it separates two peers asserting the same display name, and gives a changed key
something to be noticed against. A word pair computed from a key means nothing to a
user weeks later, so it is never relied on as a way to recognize a colleague.

The display name is asserted by the peer. Until a user chooses one it is inferred
from their operating-system account, which is David on a laptop and Ec2-user in a
container, so two peers asserting the same display name is ordinary, not an attack.

The label is the name this machine's user gave the peer when they paired, and it is
bound to one key, and only one, from then on. It is the only one of the three that is both
memorable and unforgeable, because a user chose it about somebody they had just
verified. Interfaces lead with the label where there is one, with the derived name
beside it.

A label means one key. Recording a second key under a label already in use is
refused, and the refusal says so in full, because it is the one moment at which a
changed key and a substituted key look the same.

## Choosing the name others see

A user may set the display name other people see for them. Until they do, it is
inferred, and a name that was inferred never travels as though somebody chose it,
because a guess presented as a choice is a claim about a user that nobody made.

Whether a name was chosen is recorded, not inferred. Keeping the inferred name counts
as choosing it.

A user is offered the choice the first time their name is about to leave the machine,
which is when they hand a colleague a pairing string. Only other people see the name,
so its owner has no other moment at which to notice it is wrong.

Changing the name later is safe. Turns already sent keep the name they carried,
because events are immutable (section 7), and a label a colleague chose is theirs,
so a rename cannot change what anybody else calls you.

## Naming people

The word lists for peer names hold nothing unkind to a person. An adjective that
would be unkind applied to a person is left out, however neutral it seems applied to
an animal, and so is any animal used as an insult. The lists are reviewed as they
grow, because the combinations are generated and nobody approves them one by one.

## When an identity is created

A peer identity is created once, on a machine's first use of cogmer, and persists
from then on. It is not created per room, per session or per invitation, and it
outlives all of them, so a peer can be recognized on a later occasion rather than met
afresh each time.

An identity belongs to a machine, not to a person. A user who works from a laptop and
a desktop is two peers, and appears as two wherever peers are listed or admitted, so
that losing one machine never loses the key the other holds.

## An identifier must be safe to know

A peer's identifier is its Ed25519 public key, and knowing it grants nothing. An
identifier appears in every event its peer creates, in every interface, and in every
exchange by which peers recognize one another, so an identifier that gave its holder
any power could not be protected.

An identifier is safe to know only because signatures are checked. Every event is
signed by the peer that created it, and an event whose signature does not verify is
rejected on receipt, so attribution is enforced rather than trusted.

An identifier says nothing about whose key it is. Section 25's verification is what
establishes that.

The machine label is for attribution and display. It is never an address, and nothing
routes by it, because where a peer can be reached is a property of the transport and
changes independently of who the peer is.

A Claude response is always attributable to the user and the session it came from,
and is shown as

```
Claude — David
```

rather than as

```
Claude
```

---

# 7\. Event Model

Events are immutable once created, and a room's store only appends.

A room records two kinds of event: a user's prompt, and Claude's response to it.

Each event carries its own identifier, the peer that created it and that peer's
sequence number, the room, the time it was created, the user who wrote it and the
name they went by, the machine it came from, the session it came from, its kind, its
content, and the signature of the peer that created it.

---

# 8\. Event Identity and Ordering

Neither the identity of an event nor synchronization rests on wall-clock time alone.

Each peer numbers the events it creates with a sequence that only increases:

```
David:

1825
1826
1827
1828
```

So an event has both a globally unique identifier and a position, its peer and that
peer's sequence number. The position lets peers work out cheaply which events they
are missing. Timestamps are for showing the conversation and for approximate
ordering between peers.

## The lifecycle of a sequence

A sequence belongs to a peer within a room. A peer in two rooms keeps two independent
sequences, because rooms synchronize independently, and a shared counter would leave
each room gaps that could never be closed.

Within a room, a peer's sequence begins at its first event, advances by one for each
event that peer creates, and is never reset, reused or rewritten. It is frozen when
the room is archived.

A peer that cannot establish where its sequence had reached publishes nothing into
that room until it can, because resuming at a lower number reissues numbers other
peers already hold against different events.

## Losing a room's local state

Losing a room's local database does not end a peer's membership in that room. The
user's own turns, and the colleagues' turns injected into their session, are already
in that session's context, and a session's room is fixed for its lifetime, so ending
membership would cost the user the session in exchange for a record that other
members can supply again.

A peer's highest issued sequence for each room is stored outside that room's
database, with the peer's identity. Everything else can be fetched again from another
member, because events are immutable and replicated. So a peer that loses a room's
events keeps its identity and its sequence, resumes above the recorded number, fetches
the events again and stays a member, and a peer that loses everything, identity
included, is a new peer with a new sequence space and nothing to collide with.

A sequence number is reserved before the event that uses it is published. A peer that
fails between the two has recorded a number it did not use, which is harmless, while
publishing first would issue a number the peer has no record of.

## Noticing that state is gone

A peer consults its record of membership every time it opens a room. State has been
lost if that record names a room whose database is absent, or if the database's
highest sequence for this peer is below the number the record holds. A peer does not
know it has lost state, so the check runs on every open, not only when a fault is
suspected.

## Saying so

A peer that has lost a room's state tells its user, in terms they can act on: that
local state for the room was lost, that membership and sequence position are intact,
that the room's history is being fetched again from other members and depends on one
being reachable, and that some colleagues' turns may be injected a second time. A
recovery nobody is told about looks the same as nothing having gone wrong.

A room's events return only from peers that still hold them. A member that recovers
while alone holds a correct sequence position and an empty history, publishes safely,
and receives the history as others reconnect.

Delivery state is lost with the database, so colleagues' turns a session has already
seen may be injected again. That is redundant rather than harmful, since the session
holds them already, and it is the direction to err in, because the alternative is
silence about turns never delivered at all.

## Redelivery and conflict are not the same thing

A receiving peer tells redelivery from conflict, because the obvious implementation
treats them alike and only one of them is harmless.

An event whose peer and sequence number the receiver already holds, with the same
event identifier, is ordinary redelivery and is ignored. That is what makes
anti-entropy and relaying safe to repeat.

The same peer and sequence number with a different event identifier is not a
duplicate. It means a peer has lost its state or an event has been forged, and it is
reported and kept. Absorbing it would lose it for good, since the sender believes it
shared the event and the receiver reports holding a higher sequence than the sender
has. The rejected event is kept, because without it nobody can later tell lost state
from forgery.

---

# 9\. Synchronization State

Each daemon knows, for every peer in a room, the highest contiguous sequence number it
has received from that peer:

```
Room: misty-canyon  (0f7a4e6c-2b91-4d0a-9c3e-7f1d8a5b2c44)

Known state:

David     1827
Alice      943
Carlos     611
```

When Alice reconnects to David, she says what she holds:

```
I have:

David  1802
Alice   943
Carlos  611
```

and David sends what she lacks:

```
David events 1803–1827
```

Alice stores them. No transcript is compared.

---

# 10\. Anti-Entropy Synchronization

Peers exchange what they hold periodically. Alice sends David what she holds:

```
Alice → David

{
  david: 1802,
  alice: 943,
  carlos: 611
}
```

David works out which events Alice lacks and sends them, and Alice does the same for
David.

So a room converges by itself after a temporary network loss, a laptop sleeping, a
daemon restarting, a machine switching networks, or a member's session being
resumed.

A room lasts only as long as its sessions, so synchronization reconciles
interruptions within an active pairing, and is not built to reconcile weeks of
divergence.

---

# 11\. CRDT Requirement

The conversation is an append-only collection of immutable events, so cogmer uses no
CRDT library. Synchronization rests on immutable events, unique identifiers, a
sequence per peer, anti-entropy synchronization and a deterministic order for
display.

---

# 12\. Forming a Room

A room is formed by invitation, not by configuration. One peer creates a room and
invites the others, and each of them joins by accepting.

Colleagues usually already know one another, so an invitation names a guest rather
than issuing a token:

```
David:   /cogmer:room-invite alice
         → invited alice to misty-canyon

Alice:   → misty-canyon is offered to you by david
         /cogmer:room-join misty-canyon
```

David's daemon records that the peer he knows as alice may enter misty-canyon, and
tells her daemon so over the channel they established when they paired. Her daemon
holds the offer until she accepts it. Nothing secret is typed, spoken or sent, and
the room's name, which is guessable, grants nothing to whoever guesses it.

Delivering an offer admits nobody. The admission is the entry on the host's guest
list, which is the host's judgment. The offer says only that the entry exists,
joining is the guest's own act, and reading the room waits on verification
(section 25).

When the host cannot reach the guest, inviting says so at once and produces a line
the guest can be sent by any means, which they present in place of a room name.

When the guest is not yet verified, the admission is recorded and nothing is sent,
because a guest who joined before verification would find the room silent
(section 25). The host is told to finish verifying, and the invitation is delivered
when they do. The output presents this as one step remaining, never as a refusal,
because trying to start a room is the moment a user is willing to verify.

A name locates a room, and a guest list admits a peer.

A peer that joins receives the room's identifier on admission, and uses it from then
on.

## Two acts, at two scopes, with two names

Pairing is between two machines. It happens once with each colleague, outlasts every
room the two will ever share, and is where a key is recorded and confirmed.

Inviting is into one room. It happens as often as rooms do, means nothing outside the
room it names, and is withdrawn by revoking, without touching the pairing.

Known peers belong to a machine and a room's guests belong to the room, and the
commands say which they act on: `pair` names the lasting act, and `invite` the one
scoped to a room.

Pairing is interactive, waits on another user, and ends with two words that must
reach the user's eyes unaltered, so it happens in the view the daemon serves and never
by way of the model (section 29). Inviting is a single act whose output only informs,
so it is a command inside a Claude Code session like any other.

A pairing string carries an address as well as the identifier, so that verification
can happen before any room exists. The address need only be correct once
(section 4).

A pairing string may also carry the name its sender goes by, so that the user who
receives it has a sensible default for the name they must supply. It carries the name
only when somebody chose it, because a name inferred from an operating-system account
traveling as though a user picked it is worse than carrying nothing. The name is a
claim by whoever sent the string, so it is recorded only after the two words match.

No part of a pairing string is a secret. An identifier is a public key, an address is
where a daemon listens, and a name is what somebody calls themselves, so an
interceptor learns that a peer exists and gains no way in. What interception
threatens is substitution, which the two-word comparison is for.

## Pairing, which happens once

Admission by guest list needs the host to hold the guest's key already, which takes one
exchange with each colleague, ever:

```
Each prints their own string and sends it to the other:

Alice:   /cogmer:self-status
         → ed25519:M7Kd…4Fq2@198.51.100.7:4783#Alice

David:   /cogmer:self-status
         → ed25519:GoR7…IWPU@203.0.113.9:4783#David

Then, on a call, both at the same time, each pasting the OTHER's:

David:   /cogmer:peer-pair ed25519:M7Kd…4Fq2@198.51.100.7:4783#Alice
Alice:   /cogmer:peer-pair ed25519:GoR7…IWPU@203.0.113.9:4783#David

         → a page opens on each machine: ribcage tambourine
         → each says whether the other read the same two words
```

Two strings cross, and both users run the command. What pairing establishes is
symmetric, that the key David holds for Alice is Alice's and the key Alice holds for
David is David's, and one direction alone would leave the other user trusting a key
nothing had checked.

A daemon refuses an incoming verification when its own side has not been run. It
refuses a peer whose identifier it has never recorded, and it refuses a verification
its user did not ask for, even from a known peer with a signature that checks, because
only the user at a machine may start a comparison on it.

A pairing string may be pasted into a chat, mailed, printed or read aloud.

Pairing takes the name for the peer before the comparison. It is the name this user
will call that colleague from then on, and it is held until the words match, and
written only then.

A pairing ends in one of three ways. Abandoned leaves the key recorded, unverified and
unnamed, so that the attempt can be resumed. Matched records the verification and
writes the name. Differed removes what the attempt created, so that a key nobody
established anything about does not stay on the list under the name meant for somebody
else.

Verifying an existing peer again is different. Different words there are an alarm
about a relationship, not grounds for discarding it and the admissions it holds.

Pairing needs integrity and no privacy. An interceptor who reads a pairing string
learns nothing, but one who replaces it is recorded as the guest, under the name the
host expected, with nothing appearing wrong, and the two-word comparison is what
catches that (section 25).

Only one of the two users needs an address that works. Both hold the other's
identifier and both run the command, and the exchange completes as soon as either side
reaches the other: a daemon looks for an exchange that has already arrived before it
dials, and derives the same two words from what it received. So a pairing string that
carries no address still lets its sender pair, and is never treated as a dead end.

## The guest list

Knowing someone and admitting them to a particular conversation are different
decisions, so they are two lists, at different scopes.

A machine's known peers are the identifiers it has learned and the names it knows
them by. The list is durable and outlasts every room, which is why a colleague is
recognized on a later occasion rather than met afresh.

A room's guests are the known peers who may enter it. The list is created with the
room and archived with it. A user may know six colleagues and admit two of them to a
room about customer data.

Both lists can be inspected and changed:

```
/cogmer:peer-list                    list known peers
/cogmer:peer-pair <string> [name]    record a peer and verify it
/cogmer:peer-forget <peer>           discard a peer

/cogmer:room-status                  this room, and who is in it
/cogmer:room-invite <peer>           admit a known peer
/cogmer:room-revoke <peer>           withdraw admission
```

A peer can be recorded without being verified, for scripts and tests, and such a peer
cannot collaborate until the comparison happens.

A room's creator is its first guest.

Revoking withdraws admission to one room. Forgetting discards the identity itself, so
a later meeting is a first meeting again.

## Proving admission

A peer that joins presents its identifier. The host finds that identifier among the
room's guests, then issues a fresh, unpredictable challenge, which the peer signs with
the private key its identifier corresponds to.

The challenge is new every time, because an exchange that can be replayed is a bearer
credential.

Nothing secret passes in either direction, so the exchange needs integrity, which the
signature supplies, and no confidential channel.

Proving possession of a key establishes who is asking, and never whether they may. A
peer can generate a key pair as easily as anyone, so admission begins at the guest
list, not at the signature.

## When someone not on the list asks to join

A peer that is not a guest is refused, and the refusal is more than silence.

The host is told that a peer asked to enter, and shown the identifier and derived name
it presented. The host may admit it, which makes that peer known and a guest in one
act. The case this serves is two colleagues whose machines have not met, which is an
absence of a recorded key, not of trust.

Approval is a person's act. What a host approves is an identifier, and the name beside
it is a claim made by whoever sent the request.

The refused peer is told it was refused, and shown its own identifier, so its user can
tell the host what to admit.

A request is not a queue. If the host is absent the request fails, and it never grants
entry later without the host's attention.

## Pairing with someone you do not know is not a supported case

Pairing with a stranger is not designed for. Section 25 requires verification over a
channel on which the other party can be recognized, and two people who have never met
have none, so the comparison would run and establish that two parties hold the same
key while saying nothing about whose. An appearance of assurance is worse than none,
because it is what people act on.

What a room carries is a working session, and admission sends a member's turns to
another user's model provider under that user's account (section 28). The situations
that would want pairing with a stranger almost always have a call or a message
available anyway.

A host who approves a request from someone unknown to them has paired with a stranger,
and that is the host's judgment to make. Nothing presents it as intended, and nothing
claims that verification protects it.

## There is no join token

Admission is an entry on a guest list, or a host's approval. There is no code, no
invitation secret, and nothing a user can hold that would let them into a room. A
token that admits its holder has to be kept secret while it is sent, cannot be
checked afterwards, and enrolls whoever intercepts it under a name the room's members
will treat as familiar.

Inviting someone in advance needs no token either. A host may admit a peer it knows
and then leave, and the guest joins whenever it likes. Only a host that is absent and
has never recorded the guest would need a token, and such a host has to act before
that guest can enter under any scheme.

So pairing with someone entirely unknown requires a host present to approve it, which
is the moment a user should be deciding.

## The endpoint is a bootstrap hint

An endpoint is opaque to the collaboration protocol. It is whatever the transport in
use can reach, such as a direct address or an overlay address (section 4), and
synchronization never depends on its form.

An endpoint needs to be correct only once. Having joined, a peer learns the room's
members and how to reach them, so the inviting peer's endpoint stops being special at
once, just as the inviting peer is not authoritative.

An invitation may carry more than one endpoint, because which one works depends on
where the joining peer is: a colleague on the same network and one across the internet
do not reach the same address. Endpoints change when a machine moves between networks,
so an invitation may be stale, and joining with a stale one fails clearly rather than
hanging.

## Discovery on a shared network

On a shared network, a room can be found by local service discovery rather than by
being told an address. The daemon advertises its live rooms, and a daemon that is
joining looks for the name it was given:

```
cogmer join misty-canyon
```

That needs no account, no external service, no configuration and nothing secret,
because the guest list decides who may enter. Discovery locates a room, and never
admits anyone to one.

## The room name is not a credential

A room name comes from a small space that is guessable on purpose, so that it can be
spoken aloud. A few thousand combinations are enough to avoid confusion and useless
against a guess.

Authorization is never the name. It is an entry on a guest list, proved by possession
of a key, or a host's explicit approval of a request, and no peer admits a session to a
room on the strength of a name. On a shared network, such as an office, a conference
or a cafe, any listener can list the advertised room names, and a room holds source
code, customer information and whatever a user has pasted into a prompt.

The guest list also tells rooms apart. Names are unique only among the rooms one peer
hosts, so local discovery may find two unrelated rooms with the same name, and a
joining peer is a guest of at most one of them.

## Room names

A room name is generated, never chosen. It is made from two curated word lists, one
of weather or sky and one of landscape:

```
misty-canyon
thunder-ridge
clear-delta
frost-hollow
```

A name a user chose would be the name of a project, a client or a ticket, and rooms
named after projects become rooms scoped to projects, so a generated name keeps rooms
scoped to sessions without relying on anyone's discipline.

A room name is:

- speakable, because an invitation may be read aloud on a call;
- unambiguous when heard, with no homophones and no easily confused pairs;
- short enough to type without copying;
- drawn from a narrow, neutral domain, so that no combination is offensive or
  misleading.

Each list is large enough that the live rooms one peer hosts rarely collide.

Names are not globally unique and cannot be, since rooms are created independently on
machines that do not coordinate. A peer keeps only the live rooms it hosts distinctly
named, generating again on a collision, which is enough because a name is only ever
resolved against one peer.

A room's name is fixed for the life of the room, because renaming would invalidate
the invitations already sent and make an archived room harder to recognize.

The peer that issued an invitation does not become authoritative by doing so. It is
only the first member that can be reached, and it may leave while the room continues.

---

# 12a\. Room Membership

A Claude Code session is a member of one room at most, and never joins a second. Every captured
event belongs to one room and no other, and a session in two rooms would give no basis for
choosing which, while a session receiving turns from two unrelated conversations could
not keep them apart, and neither could the user reading the result.

## Creation

A room exists once a user creates one, and not before.

Creating a room is the act of inviting someone into it. The room's identifier and name
are generated at that moment, the creating session becomes its first member, and the
room's history begins there.

Nothing earlier belongs to the room. A session that has been working alone has
published nothing, and issuing an invitation never hands over that work afterwards,
because a room that began when the session did would give the first person invited
hours of work, disclosed by an act that looks like saying hello.

Throughout this specification, "from its beginning" means the beginning of the room,
never the beginning of a session that belongs to it.

Before a room exists, a session is an ordinary Claude Code session: nothing is
captured, injected or shared. Collaboration is something a user starts, not a state
they are in.

A new room does not contain the conversation that led a user to invite someone. Earlier
turns are never contributed to a room by default.

## Leaving

A session may leave a room. Leaving is always explicit, and is never inferred from a
Claude Code session becoming inactive.

A session that leaves stops publishing to the room and stops receiving injected
context from it. The events it published stay in the room, because published events
are immutable, and a user who leaves does not unsay what they said.

## Joining and rejoining

A session joins a room by accepting an invitation. A session that joins a room already
in progress receives the room from its beginning.

A session may rejoin a room it left, for as long as that room is live. It receives what
it missed, and only that, because delivery is recorded per event rather than as a
position in a stream.

A closed room is never rejoined, by a former member or by anyone else.

## Joining a different room

A session enters a room only by creating it or by joining it, and a session that has
been in one room never joins a second. It may leave its room and rejoin it. A user who
wants to work in a second room starts a second Claude Code session.

Injected context cannot be withdrawn. Once a colleague's conversation has entered a
session's context window, anything the session produces afterwards may be shaped by
it, so admitting that session to a second room would carry the first room's
conversation into the second through the model's own output, invisibly and without
either room's members knowing.

A session's own prompts and responses never restrict it, because that content came
from its own user.

## Presence is not membership

A Claude Code session does not end in any lasting sense. Its process exits, and the
session can be resumed later under the same session ID, so membership and presence are
separate.

Membership is durable. It is held by the session ID, begins at joining, and ends only
when the session explicitly leaves or the room closes.

Presence is transient. It says whether a member can be reached now, and it lapses
whenever a process exits, a machine sleeps or a network drops.

A member whose Claude Code process exits is absent, and has not left. Its membership
stands, its delivery state is kept, and events published while it was absent are still
owed to it.

Resuming that session restores presence, and is not a rejoining, because membership
never lapsed. The session receives what built up while it was absent, within the
injection limits.

A signal that a session ended is a signal about presence, never about departure, and
nothing depends on receiving one, because a process that is killed, or a machine that
loses power, sends nothing.

## Closing

A room closes when every member has explicitly left, or when it has been dormant, with
no member present, for long enough that resuming it is no longer plausible.

The dormancy threshold is generous. Nights, weekends and illness are ordinary, and a
room that dissolved because everyone went home would make resuming useless, and
closing early is the worse error, because a closed room can never be rejoined and its
sessions can join no other.

Closing ends the room, not the conversation: the room's events are archived.

---

# 13\. Transitive Synchronization

Synchronization does not need every peer connected at once.

```
David ↔ Alice ↔ Carlos
```

If David and Carlos cannot reach each other, David gives Alice his event 1827, Alice
stores it, and when Alice next connects to Carlos she passes it on:

```
David → Alice → Carlos
```

An event belongs to the peer that created it, whichever peer relays it, and a relaying
peer never changes an event's originating peer, its sequence number or its identifier.

Every event is signed by the peer that created it, over the event's own content, so a
relaying peer can carry an event and cannot author one: an event from Alice that claims
to come from David is distinguishable from one Alice composed, because Alice cannot
produce David's signature.

A receiving peer checks every event against the key its originating peer's identifier
names, and rejects any that does not verify. It rejects rather than keeps it aside,
because a failed signature has no harmless reading, unlike a sequence conflict, which
may be a peer that lost its state.

Signatures give integrity and attribution: nothing can be forged, altered on the way
or falsely attributed. They do not decide who may connect or read a room, which is
admission (section 12).

---

# 14\. Capturing User Prompts

The hook Claude Code runs when a prompt is submitted sends the prompt to the daemon on
the same machine. When David submits

```
Why is SessionLambda returning a timeout here?
```

the daemon at once gives the event its identifier, advances David's sequence, stores
the event, and shows it in the local view. The event reaches peers when they next
synchronize.

Claude never waits for synchronization with peers before answering a prompt.

---

# 15\. Capturing Claude Responses

The daemon captures each completed turn of Claude's, and records it as a response
attributed to David and to David's Claude session. The view shows it as:

```
Claude — David

I traced the timeout to the OkHttp connection pool...
```

A response is published whole, never summarized.

## Neither available source is complete

A completed turn is reassembled from two sources, because each lacks a different part
of it.

The hook that runs when a turn completes reports only the turn's final text block.
Anything said before a tool was called is missing, and narrating before acting is the
ordinary shape of a turn, so publishing that value alone would lose part of most
substantive responses.

The transcript on disk, read when that hook fires, lacks that final block, because
the hook runs before the closing record is written.

The two omissions are complementary, and together they make the whole turn:

```
transcript, read at completion   →  every block except the last
turn-completion hook             →  the last block
union                            →  the complete turn
```

## The union tolerates the hook being fixed

The hook's value is never simply appended to what the transcript supplied. If a later
version of Claude Code widened that value to carry the whole turn, appending would
repeat every block the transcript had already supplied.

So where the hook's value already contains what the transcript supplied, the hook's
value is used and the rest discarded, and where the transcript already ends with that
value, nothing is appended.

## Segmenting a turn

A turn's assistant records are found by position: every assistant record after the
most recent record that carries a human prompt's marker. They are never matched by
identifier, because assistant records carry no prompt identifier, and the chain of
parent pointers between records has gaps.

Neither a turn's internal reasoning nor the records of a subagent are published: the
reasoning is not part of a shared conversation, and a subagent's records belong to a
nested session, not to the room.

## These are observed behaviors

None of this is documented by Claude Code, and any of it can change without notice,
one part of it under a fix rather than a regression. Each behavior it rests on is
checked against the installed version of Claude Code, and what the check finds is
recorded.

---

# 16\. Propagation

Three paths carry an event, with different requirements and different limits, and a
description of propagation that treats them as one promises real-time collaboration
and delivers something else.

## Peer to peer

```
David generates event E
        │
        ├────────► Alice
        │
        └────────► Carlos
```

Peers converge promptly, and a peer recovers what it missed without any other peer
having tracked what it was owed.

Peers poll. A peer asks for what it lacks, and a peer that was absent recovers by
asking again, while pushing would need each sender to know who is connected and what
each holds, state that can be wrong, to improve half a second nobody is waiting on.

On a healthy network, an event typically reaches the other peers in under a second,
which a poll every second achieves.

A peer that receives an event checks it, tells redelivery from conflict, stores it,
tells the local view, and passes it on to any peer that asks.

Anti-entropy is not a fallback here. It is how events propagate.

## Daemon to local UI

The local view updates without the reader doing anything. The daemon pushes each
event to it as the event arrives, which is the one path where pushing earns its cost,
because a user watching a conversation notices a delay that a machine does not. It
runs over loopback and needs no transport, and a view that reconnects receives what
it missed.

## Daemon to a Claude session

There is no path from the daemon into a Claude session, and there is never one.
Nothing can place context into a session already underway, which is a property of
the host. That a peer event never makes such a session take a turn is a rule of
cogmer (section 3.7).

Context reaches a Claude Code session when a prompt is submitted, and at no other
moment. A session part-way through a turn cannot be told anything, so a turn that
runs for minutes does not learn of a colleague's message until it ends and the next
prompt begins.

This bounds everything above it: taking propagation between peers from five hundred
milliseconds to fifty changes nothing a user experiences, because what they are
waiting on is the turn. Wherever propagation is described, the two figures are given
separately.

---

# 17\. Shared Conversation UI

## The room belongs outside the session

A user's session is their conversation with their own Claude. The room is a record of
what colleagues are doing, consulted rather than taken part in, and it is kept apart
from the session.

A session is read closely and a room is glanced at. Interleaving them would bury the
thing glanced at inside the thing read closely, and interrupt the close reading with
arrivals not addressed to it. With three colleagues a session would become unreadable,
and the cost would fall on the user's own working view.

The model and the user want the same conversation in different forms. The model wants
a colleague's turns in its context, arriving at a turn boundary. The user wants them
available to glance at, without their own thread stopping to carry them. So the model
is served by injection, and the user by a view.

## What is achievable inside a session

A hook's output reaches the model and never the user. Its standard output becomes
context, and neither its standard error nor a direct write to the terminal appears
anywhere: Claude Code owns its display. So a user has no ambient view of the room from
inside their session, and no arrangement of hooks produces one.

## Asking

A user can ask their own Claude what the room has been discussing, and it answers
from the context already injected, naming who said what and which of them are
unverified. It tells a user only what they thought to ask about, and never that
something has arrived. For a pair working on one problem, asking what the team has
found answers most of what a view would.

## Watching

Seeing a colleague's turn arrive without asking takes a view outside the Claude Code
session, and that view is a page in the browser, for reading an exchange properly,
with code and formatting. It reads only from the daemon on the user's own machine.
cogmer works without it: a user who never opens it still has injection and asking.

Each room has its own address:

```
http://127.0.0.1:4782/room/misty-canyon
```

Peers synchronize their event stores, so every user sees roughly the same room:

```
MISTY CANYON
──────────────────────────────────────

David                         11:42

Why is SessionLambda returning a
timeout here?


Claude — David                11:43

I traced the timeout to the OkHttp
connection pool...


Alice                         11:44

That assumption may be wrong.
Check the retry path.


Claude — Alice                11:45

You're right. The retry path shows...
```

The view:

- updates live;
- shows who wrote each turn, and whose Claude wrote each response;
- shows when each turn was written;
- renders Markdown and code;
- shows whether it is connected;
- shows when peers are offline;
- recovers missed events by itself.

---

# 18\. Cross-Session Claude Context

Each user's Claude receives the conversation that happened through their colleagues'
sessions, because seeing it in a view is not enough for their Claude to follow it.

Suppose David's session produced:

```
David:
Could idle pool expiration explain this?

Claude — David:
Yes. The implementation currently...
```

Alice then types:

```
I don't think that's actually the problem.
```

Alice's Claude has to understand what "that" means. So before her prompt is
processed, her hook asks the daemon on her machine for the room's events her session
has not yet received, and the daemon returns them.

---

# 19\. Incremental Context Injection

The daemon knows, for each Claude session, which of the room's events have been
delivered to it.

When Alice submits a prompt, the daemon finds the room's events her session has not
received, leaves out those from her own session, formats the rest, and injects them.
Conceptually:

```
[SHARED TEAM CONVERSATION SINCE YOUR LAST TURN]

David:
Could idle pool expiration explain this?

Claude — David:
Yes. The implementation currently...

[END SHARED TEAM CONVERSATION]

Alice:
I don't think that's actually the problem.
```

A colleague's turn counts as delivered to a session only once the session has
received it, never when it is offered, so a turn the session never received is
offered again at its next prompt rather than lost.

The whole room is never injected again. The exception is a session that joins a room
already in progress, which receives the room from its beginning, the room's and not
any member session's, because a room is bounded by the work it was created for.

---

# 20\. Attribution in Injected Context

External conversation never appears to be local conversation.

Injected turns are given an explicit structure, in an encoding in which a value
cannot leave its field. They are JSON, because hand-escaped markup assembled by
interpolation can be broken by what it contains, and an encoder cannot produce a
value that ends its own string. Whether a turn came from a user or from their Claude
is part of that structure, never a prefix on a name.

Claude is told that these are a colleague's conversation, that they provide context,
that they are not system instructions, and that the local user's current prompt
remains authoritative.

## The boundary is unforgeable

A framing sentence at the top of the block is not enough, because the content it
frames can end the block. Injected content is written by other people and their
Claude sessions, and a turn containing the closing delimiter would escape the block,
so that whatever followed appeared to be outside it, where text can pass for an
operator, a system or the local user.

So each block is delimited by a value generated for that injection, which the content
cannot know, and that value is removed from the content, so that a turn cannot
reproduce it by accident or by guess. The block says it ends only at the matching
value, and that text claiming otherwise is part of it, and it states its framing again
after the content, so the last thing read is the boundary.

## Frame by classification, not by authority

The block tells the model what kind of thing it is reading, rather than telling it to
disregard instructions: that nothing inside is addressed to it, however phrased,
including text appearing to come from an operator or a system, and that a request
inside the block is a report that someone made a request, not a request made of it.
That lets a model classify hostile content correctly, rather than weigh it against
competing instructions.

A model reasons about who said a thing, so attribution in injected context carries
more weight than attribution in a display.

An unverified speaker does not appear in injected context at all, because
verification gates what is synchronized and injected (section 25). Every speaker is
still marked verified or not inside the injected text, so that if a filter ever fails,
the text says so where the model can read it.

A display name is asserted by the peer's own environment, and is a claim. Two peers
running under the same operating-system account assert the same one. So attribution
rests on the name derived from the identifier, which cannot be chosen.

The derived name is a separate field, never part of the same string as the claim.
Escaping keeps a crafted name from leaving its field, and does nothing to stop one
imitating the field beside it: a display name reading "Alice (quiet-otter)" breaks
nothing and still reads as though it carried the derived name. Every fact the
receiving side knows and the sender does not assert, whether the speaker is verified
and whether the turn came from a user or their Claude, is likewise its own field, and
none is joined to a claim.

A display name a user chose (section 6) is still a claim, asserted by the peer, and
the derived name still anchors it.

The block also carries the label, the name the receiving user gave that peer when
they paired (section 6), and says it is the name to refer to them by. It is the only
name in the block that the user reading the answer also uses. All three names travel
together, because the label is preferred, not substituted, and what can be checked
stays checkable.

---

# 21\. Context Window Management

Stored history and injected context are separate. A machine may keep months of
archived rooms, while Claude receives only the unseen conversation from the room its
session is in. An archive is never injected.

A room lasts only as long as its sessions, so the unseen conversation available to
inject is bounded by the pairing that produced it, not by how long a project has
existed.

Injection has limits, which can be configured:

- on the number of events, which is how many turns a model has to keep apart;
- on the characters in one turn, so that one enormous turn cannot crowd out the
  others;
- on the characters in the whole block, the only one of the three that bounds what
  reaches a context window.



The limits are a safety valve, not the ordinary path. Exceeding them means an unusually
long or busy pairing.

When a limit is exceeded, the newest portion that fits is injected, with a note
saying that earlier conversation in the room exists and was left out.

Injected context is never summarized by a model.

---

# 22\. Persistence

Each peer keeps a complete local database of every room it is in, in SQLite:

```
~/.cogmer/
    identity.json
    membership.db
    rooms/
        0f7a4e6c-2b91-4d0a-9c3e-7f1d8a5b2c44.db
    archive/
        3c1e05b9-7d42-4a88-b016-9e2f4c7a10d5.db
        a91f720d-4e63-4b15-8c77-05de3b8f6291.db
```

A room's database is named by the room's identifier, never by its name, because names
may collide and identifiers do not.

A room's database moves to the archive when the room closes. An archived room can be
read and searched, and is never rejoined, synchronized or injected. Keeping the
archive is what lets membership end without discarding the conversation.

## The membership index

The membership index is kept outside every room's database, and holds, for each room
this peer has joined, the room's identifier and name, whether the peer is in it, has
left it, or the room is archived, the highest sequence this peer has issued in it,
and the room's guests.

The index shares the fate of the peer's identity, not of the rooms, because a room's
database can be lost while the peer survives, and a record kept only inside the thing
whose loss it guards against guards nothing. Section 8 says how a peer uses it: it is
consulted every time a room opens, its sequence is advanced before the event that uses
it is published, and a peer that finds state lost says so.

A room's record is written when the room is created or joined, and kept once the room
is archived, so a peer can tell a room it once belonged to from one it has never seen.

## What a room's database holds

A room's database holds the room's events, the conflicting events it has kept aside,
and which of the room's events each session has received or been offered. It never
holds this peer's own sequence position for the room, which is in the membership
index.

---

# 23\. Local Durability

An event created on this machine is committed to durable storage before it counts as
published, and is sent to peers only afterwards:

```
Claude event
     │
     ▼
local durable store
     │
     ▼
local UI
     │
     ▼
peer propagation
```

So a failed network never loses the local conversation.

---

# 24\. Conflict Model

Events are immutable, so conventional conflicts between edits do not occur. When two
users create events at the same moment, such as David's E and Alice's F, both are
kept, and neither overwrites the other.

Every peer shows a room's events in the same order, by timestamp, then by originating
peer, then by that peer's sequence number.

Wall clocks on different machines do not give a causal order, and the order shown
does not claim one.

---

# 25\. Security

A room's history may hold proprietary source code, customer information, architecture,
logs and credentials pasted by accident, so every exchange between peers is
authenticated.

The interface for the Claude Code hooks and the local view binds only to the loopback
address. Being on the same network, or the same private network, never admits
anyone.

Nothing admits anyone by being held. An invitation names a guest and grants no entry,
and an interceptor learns only that a room exists. There is no credential that can be
forwarded, stolen or replayed.

## Known peers

A peer's identifier is its public key. A peer proves possession of the matching
private key when it connects, and every event is signed by the peer that created it,
so an identifier cannot be claimed by a peer that does not hold its key.

A room admits its members from the peers this machine knows, by proof of possession,
never by the presentation of a token. Nothing an interceptor could reuse is ever
sent, nothing expires, and admission can be withdrawn.

Two peers that have never met still have to establish each other's keys once, over a
channel they trust. Every exchange after that one needs nothing further, because a key
once verified is durable.

Verification is a comparison of two words, spoken aloud, derived from a live exchange
between the two daemons, as "The two-word comparison" below describes. It is the only
way a peer becomes verified.

No second method is offered, such as comparing a rendering of the whole identifier.
Verification gates synchronization, so it matters only when collaboration is about to
happen, and collaboration needs both daemons running and able to reach each other,
which is all the live exchange needs. A second method would only be a weaker way to
pass the same gate, and people check a long string badly: shown forty-three characters
on a call, a reader compares the first group and the last, and skims the middle.

Rendering an identifier for a user to read is still useful, such as when a key has
changed and somebody is looking at two of them. That is a display, and nothing about
looking at it marks a peer verified.

A short string is sound where a standing value is not. An attacker who has intercepted
a long-term identifier can search offline for a key of their own whose rendering
matches, at a cost set by how much of the rendering is compared: sixteen characters
are out of reach, and three fall in under a second. A value derived from a fresh
exchange cannot be aimed at in advance, so the attacker commits blind, gets one guess,
and a wrong guess is a mismatch the two users hear.

A peer's derived name is a mnemonic for an identity already verified, never a
verification in itself, because it covers too few bits and can be ground by anyone who
wants a particular one (section 6).

## Learning a peer's identity

A peer's identity is learned at pairing (section 12), from the pairing string a
colleague sends. There is no directory to consult. An identifier is a public key, so
learning one needs no confidentiality.

An identifier proves possession of a key, and never which person holds it. Until the
two words are compared, whoever's pairing string was received is the peer recorded,
and a string that was replaced on the way enrolls the wrong peer, durably, under a name
the user will treat as familiar. Verifying once, when pairing, is what establishes
whose key it is, and it never needs doing again.

## What the second channel actually provides

The comparison protects nothing by itself. What protects the user is the channel the
two words are compared over, through two properties of it:

- diversity: an attacker has to compromise two paths instead of one, so a comparison
  over the path the pairing string arrived on accomplishes nothing;
- recognition: the user comparing can tell who they are talking to.

Recognition matters most. It is why a voice call works and a second written channel
does not: two written channels are both text, and whoever controls the delivery of one
may control the other, while a voice adds a fact no interception supplies, that this
is the colleague and not somebody in their place.

So the channel is one the pairing string did not travel on, and one on which the other
party can be recognized. A comparison with a stranger, however careful, establishes
that two parties hold the same key and says nothing about whose it is.

## The two-word comparison

Verification is two words, said aloud on a call, by both users at once. It happens at
pairing, not at joining or admission, because pairing happens once between two
machines and outlasts every room they share (section 12).

People read forty-three characters of base64 over a telephone badly, or skip it, and
skipping leaves no trace. Two words are compared as units: somebody either says
"badger" or does not.

Two properties make the short form sound, and without either the ceremony looks the
same and protects nothing:

- commitment: each side sends a hash of its contribution before either reveals one,
  so an attacker relaying between them has to fix what they present to each side
  before learning what the other will present, and is reduced to one blind guess;
- freshness: the words derive from randomness chosen for this exchange together with
  both identities, so there is nothing to compute in advance.

Together they turn an offline search into a single guess made during the exchange, at
odds set by the length of the string.

A failed comparison is conspicuous, because nothing blocks a retry. A wrong guess
produces two different strings, and the two users hear it, so a mismatch refuses, says
plainly that something intercepted the exchange, and never presents itself as a
passing error worth repeating. Nothing about a failed comparison is recorded, because
a stored failure invites an interface that offers to try again.

The exchange meets these requirements:

- both sides commit before either reveals, and a reveal that arrives before a
  commitment, or does not open the commitment already held, is refused;
- the words derive from both identities and both nonces, ordered so that each side
  computes the same string without either being the initiator;
- the words are phonetically distinct and alternate between two lists, so that words
  said in the wrong order are not a valid rendering of anything;
- those lists are not the ones used for peer names or room names, so that the words
  cannot be mistaken for a derived peer name beside them;
- only a user records the result, because the exchange proves both sides hold the keys
  they named and cannot establish that the voice on the call is the colleague.

Both users ask for the verification. A daemon answers a step in the exchange only when
its own user has started one, so nobody can use the exchange to make something appear
on somebody else's screen.

A commitment step implemented incorrectly leaves a value that can be ground while
still looking like a ceremony, which is worse than none, because it gives the
confidence without the protection.

## Verification is a gate, not a label

No transcript crosses to or from a peer that has not been verified, and no unverified
peer's turn enters anyone's model context. Admission and verification are separate
requirements, and both must hold.

Admission says a key may enter a room, and is a decision about a key. Verification says
the key belongs to the person it is meant to, and is a decision about a person. Every
cryptographic check here passes as well for a key substituted on the way, because the
substituted key is a real key held by whoever substituted it, and only the comparison
over a channel where the other party can be recognized tells them apart.

Marking an unverified peer inside injected text is necessary, and not enough. A step
that is optional and costs nothing to skip will be skipped, and afterwards nothing
looks wrong, so verification is a gate.

An event relayed by another peer (section 13) is judged by the peer that wrote it,
never by the peer that carried it, so that a verified relay cannot pass on an
unverified author.

Events from an unverified peer are held, not discarded. Synchronization is a pull, so
refusing to store an event leaves it on offer, and once the two users verify, the next
poll brings everything that was waiting.

A room that is silent because of the gate says so. Both the refusal and the holding
back name the peer, and the command that clears it, because a room silenced by a gate
looks the same as a room where nobody is talking.

Inviting an unverified peer is permitted, and has no effect until they are verified.
Whom to admit is the host's judgment, and the gate is a different question asked of a
different party, not a second opinion about that judgment.

What waits is the delivery of the invitation, not the admission. The entry on the
guest list is written when the host makes it, and the guest is told about it once
verification makes the room usable, so completing a verification delivers whatever was
waiting. Otherwise the easiest room to create would be one that does nothing, while
both users believed they were collaborating.

## Room content is unreadable in transit

Signing gives integrity and origin, and conceals nothing. A room's content is a user's
prompts and whatever their Claude said back, so confidentiality is a requirement.

Every connection between peers is TLS 1.3, over every transport, including one that is
already encrypted, so that confidentiality never depends on which path a connection
took. Each daemon presents a self-signed certificate carrying its identity key, and
the check is that the key inside is a peer this machine knows, and, for a connection
aimed at one named peer, that it is that peer. Chains, authorities, hostnames and
expiry are never consulted, because none exist here. A client certificate is required,
not requested, because a peer with nothing to pin has no business completing a
handshake.

So the transport's identity and the event's identity are the same key, confirmed by the
same comparison.

Encryption in transit does not protect content at rest, and does not protect a room
from its own members. A member is entitled to read what the room carries, which is
what admission means.

## The local API is not reachable from a web page

A user acts through the local API: the view the daemon serves runs the comparison
above, and the API can mark a peer verified and publish a turn into a room, which
reaches colleagues' context windows. Binding to loopback keeps other machines out and
does nothing about a page open in this machine's own browser, which can send a request
with a simple content type to 127.0.0.1 without the browser asking first.

So a request that changes anything carries a header that cogmer's own code sends and a
page from another origin cannot. A browser will not send a custom header to another
origin without asking first, and that question is answered with nothing, so the
request is never made. The check is that the header is present, so anything that
cannot present it is refused.

None of this defends against a malicious program running as the same user, which can
send any header it likes and could edit the databases directly. The threat closed
here is a web page, which is the boundary the browser enforces.

## The case with no answer

Two people who have never met cannot recognize each other, so neither property of the
channel is fully available on first contact, and nothing in this design resolves that.
The first exchange is where trust is taken rather than established, and cogmer makes
that moment visible rather than claiming it is covered.

Removing a peer from a room prevents it from rejoining and from being present in
future, and never withdraws what that peer has already seen.

---

# 26\. Offline Operation

Offline operation is a first-class requirement. Both users keep working while their
machines cannot reach each other:

```
David       OFFLINE       Alice

E F G                    H I J
```

and after they reconnect, both hold the whole conversation:

```
       synchronize

David                  Alice

E F G H I J            E F G H I J
```

No user reconciles anything by hand.

Offline operation is bounded by the room's lifetime: a room that has closed does not
reopen when a member comes back online.

---

# 27\. Tool Activity

A response shows how many tool calls its turn made, rather than each call:

```
Claude — David

I found the problem in VerificationService.ts...

▸ 14 tool operations
```

The names of the tools a turn called travel with its response. Synchronizing prompts
and responses comes before anything about tools.

---

# 28\. Configuration

No project configures a room. Nothing in a repository, and nothing about a working
directory, determines which room a session joins, and rooms are entered only by
invitation.

A user's identity is kept apart from the state of their rooms. Identity is the user's
identifier, the name they go by, and their machine's label. A room's state is its
identifier, its name, its guests, and whether it injects its conversation into its
members' sessions, and it belongs to the daemon, is created when the room is created or
joined, and is never in a repository.

A room can be set to show its conversation to its members while injecting none of it
into any member's Claude session, because injection carries a colleague's
conversation into another user's session, and so to that user's model provider under
their own account.

Nothing a user configures for reaching peers is committed to a repository.

---

# 29\. The Experience We Want

A user sets up once, pairs once with each colleague, and then works in rooms from
inside a session:

```
# once, so colleagues have something to send you
/cogmer:self-name Alice

# once with each colleague, on a call with them, both at the same time
/cogmer:peer-pair ed25519:M7Kd…4Fq2@198.51.100.7:4783#David
→ a page opens: ribcage tambourine   → both say yes → verified

# thereafter, per room, from inside a session
/cogmer:room-create          → misty-canyon
/cogmer:room-invite david
```

## Installation

A user installs one thing, and it is a thing Claude Code already understands:

```
/plugin marketplace add Blue-Rocket/cogmer
/plugin install cogmer@blue-rocket
```

There are two lines because a plugin is installed from a marketplace, and a
marketplace is added by name. The repository carries its marketplace manifest beside
the plugin it lists, so both lines name the same repository. The first line is run once
per user, and the second once per plugin.

The plugin carries the hooks. Installing modifies no shell profile, needs no
configuration file edited by hand, registers no service with the operating system, and
changes nothing about how Claude Code starts.

Claude Code loads a plugin wherever it loads plugins, so cogmer reaches every surface
Claude Code runs on without knowing which they are (section 3.8).

## Where each command lives

Three homes, and the command decides which it belongs in.

Inside a Claude Code session, as commands it already loads, is everything a user does
in the ordinary course: listing rooms and peers, creating a room, inviting a known peer,
joining, leaving, reading the room, pairing with a colleague, and choosing the name
others see. Each either needs nothing beyond its arguments or hands off to a surface
that does, and its output is something a user reads once, so it can reach them by way
of the model, as everything from an extension point does (section 3.8).

Joining in particular belongs inside a session, because a command typed inside a
session can name the session it means, and a command outside one cannot. Nothing on the
machine picks a room on a session's behalf, so a room command typed at a terminal is
refused rather than answered with a guess.

The two-word comparison lives in the local view, a page the daemon serves, and never
passes through the model, for three reasons, any one of which would be enough:

- it is interactive, and waits on another user for as long as they take to start
  their side;
- its output must reach the user's eyes unaltered, and least of all by way of the model
  that reads room content from peers;
- recording its result is a person's act, and that person is on a call rather than at
  a prompt.

Those reasons rule out the model and not a browser. A terminal is a surface for
operating cogmer, and asking a user to open one in order to meet a colleague is not a
user experience.

So a slash command performs the comparison rather than pointing at it. It records the
peer, creates a page, opens it, and returns without waiting, and the waiting and the
comparison happen on the page. Every pairing gets its own address, so a second pairing
arrives as a new tab, and a page left from an earlier attempt can never become a
different pairing.

Only the user at the machine can reach the page. The daemon serves it on loopback, and
refuses any request that changes something unless it carries the header a page from
another origin cannot send (section 25).

At a terminal, as a separate program, is what operates cogmer rather than what a user
does: diagnostics that must work when the plugin path is broken, the daemon's own
lifecycle, this machine's identity, and testing. The comparison also runs at a terminal
on a machine that cannot open a browser, such as one reached over SSH or in a
container.

Pairing and inviting are separate verbs (section 12) because they live in different
homes.

## Command names

A command's prefix names what it acts on: `peer-` acts on somebody else, `room-` on a
room, and `self-` on you. The prefix names the target, not the activity, so the command
that prints your own pairing string is `self-`, however much it is about pairing.

Commands are prefixed at all because a plugin shares one namespace with every other
plugin a user has installed, and an unprefixed `create` or `status` would collide.

## Starting the daemon

A user never has to start the daemon, notice that it has stopped, or know that it
exists. The session-start hook starts it when it is not already running, so installing
is the whole of setup, and the first session after installing is like every session
after it.

Starting the daemon never delays the session. The hook returns at once, and the daemon
becomes useful when it is ready, so a session that begins before the daemon is
listening simply has nothing to inject yet.

A daemon already running is the ordinary case, not an error. Several sessions often
begin at once on one machine, each tries to start the daemon, at most one succeeds, and
none reports a problem.

A daemon that fails to start is silent to the user, and recorded by the daemon. It means
no collaboration, and the session continues as it would have without cogmer.

The daemon outlives the session that started it, because a room may have members in
several sessions, and starting it repeatedly is worse than leaving it running. So the
user whose machine it runs on can find it and stop it.

Stopping the daemon is a terminal command, because it is the daemon's own lifecycle. It
stops whichever cogmer daemon holds the addresses this installation uses, not only the
one this installation started, because a daemon left running from another state
directory or an earlier version blocks this one just as an unrelated program would.
It signals only a process it has identified as a cogmer daemon. Anything else holding
an address is named, with its process, and left alone.

After an update, the first session runs the new version. A daemon of a different
version holding this installation's addresses is stopped and replaced in the
background, so the session is not delayed, and a fix reaches the user without their
having to find and stop a process.

A user can ask which version of the plugin, the binary and the daemon they are
running, and is told when they disagree and what resolves it, so that whether an
update has landed is a question with an answer.

A daemon that cannot bind says which of these holds its address, and when it is a
cogmer daemon, gives the command that clears it.

A user never has to copy transcripts, summarize what Claude found, paste a colleague's
conversation, manage synchronization, find missing events, or resolve routine
conflicts. Collaboration is ambient.

---

# 32\. Explicit Non-Goals

cogmer has none of these:

- a central cloud conversation service;
- a cloud transcript database;
- a proxy for the Claude API;
- centralized model inference;
- retrieval-augmented generation, embeddings or a vector database;
- automatic summaries;
- an organizational knowledge base;
- a Slack integration;
- a shared Claude Code session;
- a CRDT framework;
- libp2p networking.

---

# 33\. Transports

Networking sits behind a transport, which finds peers and moves bytes, and does
nothing else. Replication sees only the transport. A transport never interprets an
event, never decides membership and never admits anyone: it finds candidates, and the
room's guest list decides who may enter.

The event model and synchronization are the same whichever transport carries them.

---

# 37\. The name

The product, the plugin and the repository are `cogmer`. The name reaches the module
path, the binary, the plugin manifest, the state directory and the release path, and
nowhere else.

Nothing cryptographic carries the name. Signing uses a namespace that is arbitrary on
purpose and never changes, because a namespace carrying a product name would make every
signature ever produced depend on a naming decision. A signature scheme is added beside
the current one and never edited, so a signature outlives every later change to what an
event carries, and a rename after real events existed would mean carrying the old
namespace for ever.

The state directory is `~/.cogmer`, the one place the name reaches the filesystem, and
nothing derives state from any other name.

The plugin manifest's name is the namespace of the slash commands. Claude Code prefixes
every command a plugin provides with it, and offers no form without it, so the manifest
name is what a user types before the colon, and changing it changes every command at
once.
