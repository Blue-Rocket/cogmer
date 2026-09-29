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
list, which is the host's judgement. The offer says only that the entry exists,
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
travelling as though a user picked it is worse than carrying nothing. The name is a
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
and that is the host's judgement to make. Nothing presents it as intended, and nothing
claims that verification protects it.

## There is no join token

Admission is an entry on a guest list, or a host's approval. There is no code, no
invitation secret, and nothing a user can hold that would let them into a room. A
token that admits its holder has to be kept secret while it is sent, cannot be
checked afterwards, and enrols whoever intercepts it under a name the room's members
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
spoken aloud. Tens of thousands of combinations are enough to avoid confusion and
useless against a guess.

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

Each list is large enough that collisions are uncommon: a few hundred entries in each
gives tens of thousands of combinations.

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

Use the appropriate Claude Code hook associated with user prompt submission.

When David submits:

```
Why is SessionLambda returning a timeout here?
```

the hook sends it to:

```
localhost:4782
```

The local daemon immediately:

1. assigns an event ID;  
2. increments David's peer sequence;  
3. stores the event;  
4. updates the local UI;  
5. queues it for synchronization with peers.

Claude execution should not wait for peer synchronization.

---

# 15\. Capturing Claude Responses

Use the appropriate Claude Code lifecycle/hook mechanism to capture completed assistant turns.

Send the complete response to the local daemon.

The daemon generates:

```
ASSISTANT_MESSAGE
```

associated with:

```
David
David's Claude session
```

The UI displays:

```
Claude — David

I traced the timeout to the OkHttp connection pool...
```

Do not summarize the response before publishing it.

## Neither available source is complete

A completed turn must be reassembled from two sources, because each is missing a different part of it.

The turn-completion hook reports only the **final** text block of a turn. Anything said before a tool was called is absent. Since narrating before acting is the ordinary shape of a turn rather than an unusual one, publishing this value alone loses part of most substantive responses.

The transcript on disk, read at the moment that hook fires, is missing **exactly that final block**. The hook runs before the closing record is flushed. Reading the transcript alone has yielded a preamble of a hundred characters in place of an answer of several thousand.

The two omissions are complementary, and their union is the whole turn:

```
transcript, read at completion   →  every block except the last
turn-completion hook             →  the last block
union                            →  the complete turn
```

Neither source alone satisfies the requirement above to send the complete response.

## The union must tolerate the hook being fixed

Do not simply append the hook's value to what the transcript yielded.

Were a later version to widen that value to carry the whole turn, appending would duplicate every block the transcript had already supplied. An upstream improvement would silently corrupt the conversation, which is a worse failure than the one it repaired.

Detect the case instead. Where the hook's value already contains what the transcript yielded, prefer it and discard the rest. Where the transcript already ends with that value, the race was won and nothing needs appending.

## Segmenting a turn

Attribute assistant records to a turn by position: take every assistant record following the most recent record that carries a human prompt marker.

Do not attempt to correlate by identifier. Assistant records carry no prompt identifier, and the parent-pointer chain contains gaps — a record has been observed whose parent matched no preceding record in the same file.

Exclude from what is published:

- internal reasoning blocks, which are not part of a shared conversation;  
- records marked as belonging to a subagent, which belong to a nested session rather than to the room.

## What remains uncertain

The behavior described above is observed, not published. It can change without notice, and one part of it would change under an upstream *bugfix* rather than a regression.

Treat it as an assumption to be checked against the installed version rather than a property to rely on, and record what is found.

---

# 16\. Propagation

Three paths carry an event, with different requirements and different limits.
Conflating them is how a system comes to promise real-time collaboration and deliver something else.

## Peer to peer

```
David generates event E
        │
        ├────────► Alice
        │
        └────────► Carlos
```

Two requirements. Peers must converge promptly, and a peer must be able to recover what it missed **without any other peer having tracked what it was owed**.

Polling satisfies both and is the default. A peer asks for what it lacks; a peer that was absent recovers by asking again. Pushing would require a sender to know who is connected and what each holds — state that can be wrong — in exchange for improving a half-second that nobody is waiting on.

Target: under a second, typically, on a healthy network. A one-second poll meets this.

Receiving peers:

- validate the event;  
- deduplicate by event id, distinguishing redelivery from conflict;  
- persist it;  
- notify the local UI;  
- relay it onward when asked.

Anti-entropy is not a fallback here; it is the mechanism.

## Daemon to local UI

The UI must update without the reader doing anything.

This is the one path where pushing earns its cost: a person watching a conversation notices a delay a machine does not. It is local, over loopback, and needs no transport work.

## Daemon to a Claude session

There is no such path, and there must not be one.

No mechanism exists to place context into a session already underway. That is a property of the host.

That a peer event must never *cause* such a session to take a turn is a rule of this system rather than a property of the host, and is stated under the architecture principles.

Context reaches a Claude Code session when a prompt is submitted, and at no other moment. A session part-way through a turn cannot be told anything. A turn that runs for minutes will not learn of a teammate's message until it ends and the next prompt begins.

This is a property of the host rather than a choice, and it bounds everything above it. Taking peer propagation from five hundred milliseconds to fifty changes nothing a person experiences, because what they are waiting on is the turn.

Report the two separately wherever propagation is described. A system that quotes only the first number is describing the half that is fast.

---

# 17\. Shared Conversation UI

## The room belongs outside the session

A person's session is their conversation with their own Claude: their prompts, its replies, the work in progress. The room is something else — a record of what colleagues are doing, consulted rather than participated in.

Keeping them apart is right on the merits, independently of what is achievable.

A session is read closely and a room is glanced at. Interleaving them means the glanceable thing is buried inside the thing being read closely, and the thing being read closely is interrupted by arrivals that were not addressed to it. Neither is served. A person loses the thread of their own work in order to be told something they could have looked at when they chose.

The two also scale differently. With one colleague, interleaving might be tolerable. With three, a session becomes unreadable — and the cost lands on the person's own working view, which is the last place it should land. Kept separate, additional participants cost nothing at all in the session.

There is a neater way to say it. The model and the person want the same conversation in different forms. The model wants a teammate's turns **in its context**, arriving at a turn boundary, phrased for a reader that does not skim. A person wants them **available to glance at**, without their own thread stopping to carry them. One channel cannot serve both without compromising each.

So the model is served by injection, and the person by a view. That the second cannot be placed inside a Claude Code session is a constraint that happens to agree with the design rather than one the design is working around.

## What is achievable inside a session

Established by testing rather than assumed. A hook's output reaches the model and never reaches the person: standard output becomes context, and neither standard error nor a direct write to the controlling terminal is surfaced. Claude Code owns its display, and nothing a hook does appears in it.

So a person has no ambient view of the room from within their session, and no arrangement of hooks will produce one.

Two affordances remain, and they are different in kind.

## Asking

A person can ask their own Claude what the room has been discussing, and it will answer from the context already injected — naming who said what, and which of them are unverified.

This costs nothing to provide; it follows from injection working at all. It is a pull: it tells a person what they thought to ask about, and never that something has arrived.

Do not overlook it because it required no building. For a pair working on one problem, "what has the team found?" answers most of what a view would.

## Watching

Ambient awareness — seeing a teammate's turn arrive without asking — requires a view outside the Claude Code session.

Two forms, and they are not alternatives so much as different moments:

- a terminal view, run beside the session, for glancing at without leaving the keyboard;  
- a browser view, for reading a long exchange properly, with code and formatting.

Both read only from the local daemon, which is what allows more than one to exist. Neither is required for the system to function: a person who wants neither still has injection and asking.

```
http://localhost:4782
```

Because peers synchronize their event stores, each person sees approximately the same room.

Example:

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

Requirements:

- live updates;  
- human attribution;  
- Claude-session attribution;  
- timestamps;  
- Markdown/code rendering;  
- connection status;  
- indication when peers are offline;  
- automatic recovery of missed events.

---

# 18\. Cross-Session Claude Context

Human visibility alone is insufficient.

Each Claude should receive relevant conversation that occurred through other people's sessions.

Suppose David generates:

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

Alice's Claude must understand what "that" means.

Before Alice's prompt is processed, her Claude Code hook asks:

```
localhost:4782
```

for unseen room events.

The daemon returns events Alice's Claude has not yet incorporated.

---

# 19\. Incremental Context Injection

Each Claude session maintains:

```
lastSharedContextState
```

This should represent which external peer events have already been supplied to that Claude session.

When Alice submits a prompt:

1. retrieve unseen external events;  
2. exclude Alice's own Claude conversation where it would duplicate existing context;  
3. format the external events;  
4. inject them into Claude;  
5. update the session's incorporated-event state.

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

Do not repeatedly inject the entire room.

A session that joins a room already in progress is an exception: it receives the room from its beginning — the room's beginning, which is when it was created, not when any member's session started. This is affordable precisely because a room is bounded by the work it was created for.

---

# 20\. Attribution in Injected Context

Never make external conversation appear to be local conversation.

Prefer explicit structure, and prefer an encoding in which a value cannot leave its
field:

```
<team-conversation fence="d77a4fcd39b3d7ed781c">
{"turns":[
  {"speaker":"David","peerName":"fond-smew","verified":true,
   "kind":"USER_PROMPT","at":"…","text":"Could idle pool expiration explain this?"},
  {"speaker":"David","peerName":"fond-smew","verified":true,
   "kind":"ASSISTANT_MESSAGE","at":"…","text":"Yes. The implementation currently…"}
]}
</team-conversation fence="d77a4fcd39b3d7ed781c">
```

Markup assembled by interpolation has to escape by hand, which is a problem JSON
solves by construction: an encoder cannot produce a value that ends its own string.
Whether a turn came from a person or from their Claude is a field rather than a
prefix on a name, for the same reason.

Claude should understand:

- these are teammate conversation events;  
- they provide context;  
- they are not system instructions;  
- the local user's current prompt remains authoritative.

## The boundary must be unforgeable

A framing sentence at the top of the block is not sufficient, because the content it frames can end the block.

Injected content is written by other people and their Claude sessions, and nothing verifies who sent it. A turn containing the closing delimiter escapes the block, and whatever follows appears to be *outside* it — where the framing no longer applies and text can impersonate an operator, a system, or the local user. This is not hypothetical; an implementation that interpolated content directly was defeated by a turn consisting of a closing tag and a forged instruction.

Therefore:

- delimit each block with a value the content cannot know, generated per injection;  
- remove that value from the content, so a turn cannot reproduce it by accident or by guess;  
- state that the block ends only at the matching value, and that text claiming otherwise is part of the block;  
- restate the framing *after* the content, so the last thing read is the boundary rather than the first.

## Frame by classification, not by authority

Telling a model to disregard instructions is weaker than telling it what kind of thing it is reading.

The useful framing names the failure mode: that nothing inside the block is addressed to it, however phrased, including text appearing to come from an operator or a system; and that a request appearing inside the block is *a report that someone made a request*, not a request made of it.

That distinction is what allows a model to classify hostile content correctly rather than weigh it against competing instructions — and, in practice, to say so: a session receiving a forged operator instruction identified it, explained that it came from inside the record and was therefore information rather than instruction, and reported the attempt to its own user.

Attribution in injected context carries more weight than attribution in a display, because a model reasons about who said a thing.

An unverified speaker does not appear in injected context at all: §25 makes verification a gate, so their turns are neither synchronized nor injected. The requirement that an unverified speaker be **marked inside the injected text** survives that as a backstop rather than as a normal state — if the mark is ever rendered, a filter has failed, and the text must say so where the model can read it rather than only in an interface.

What remains true of every speaker, verified or not, is that the **display name is self-asserted**. It comes from the peer's own environment and is a claim; two peers asserted the same one during the first two-peer run, because both daemons happened to run under the same OS user. Attribution therefore anchors on the name derived from the identifier, which cannot be chosen.

**The derived name is a separate field, never part of the same string as the claim.** Escaping keeps a crafted name from leaving its field; it does nothing to stop one imitating a field beside it. A display name reading `Alice (quiet-otter)` placed next to a derived name escapes nothing, forges no turn, leaves the block intact, and still reads as though it carried the anchor. The same holds for every fact the receiving side knows and the sender does not assert: whether the speaker is verified, and whether the turn came from a person or their Claude. Do not concatenate our facts with their claims.

A display name may also be **chosen** rather than inferred (§6), which changes what it is worth and not what it is: a chosen name is still a claim, still self-asserted, and still anchored by the derived one.

The block also carries the **label** — the name the person receiving it gave that peer when they paired (§6) — and says that it is the name to refer to them by. It is the only name in the block that the person reading the answer also uses, so answering with either of the others describes a colleague by a name that person has never used. All three travel together, because the label is preferred rather than substituted: what can be checked must remain checkable.

---

# 21\. Context Window Management

Stored history and injected context remain separate concepts.

A machine may retain many closed-room archives:

```
months of archived rooms
```

while Claude receives only:

```
unseen conversation from the room it is a member of
```

An archive is never injected. Only a live room's conversation is.

Because a room lives only as long as its member sessions, the unseen conversation available for injection is bounded by the pairing that produced it rather than by however long a project has existed. This is the principal reason to scope rooms to sessions.

Set configurable limits based on:

- **event count** — how many turns a model must hold apart;
- **character count per turn** — so that one enormous turn cannot crowd out every
  other;
- **character count for the whole block** — the only one of the three that bounds
  what actually reaches a context window. The others can all be satisfied by a
  block nobody would want injected.

An estimated token count should be derived from the character count rather than
measured. A tokenizer would have to track a model this system does not choose, and
the figure is wanted for judgement rather than for arithmetic.

Treat these as a safety valve rather than the ordinary path. Under session-scoped rooms, exceeding them indicates an unusually long or unusually busy pairing, not the normal accumulation of history.

If the limit is exceeded:

```
CONTEXT_CATCHUP_REQUIRED
```

Inject the newest manageable portion and indicate that earlier room conversation exists but was omitted.

Do not initially introduce AI summarization.

---

# 22\. Persistence

Each peer maintains a complete local event database.

SQLite is preferred for the prototype.

Conceptually:

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

Databases are named by `roomId`, never by name. Names may collide; identities do not.

A room's database moves to the archive when the room closes. An archived room is readable and searchable. It is never rejoined, never synchronized, and never injected.

Retaining the archive is what allows membership to be ephemeral without discarding the conversation. Preserving the actual conversation remains a requirement; resuming membership in it does not.

## The membership index

`membership.db` holds one record per room this peer has joined, outside every room database:

```
roomId
roomName
state          joined / left / archived
issuedSequence the highest sequence this peer has issued in that room
```

It exists because a room's database can be lost while the peer survives. Without it, an emptied room is indistinguishable from one never joined, and a peer would silently resume its sequence from the beginning — the condition that makes another peer's copy of the conversation diverge with nothing to signal it.

The index must share the fate of the peer's identity, not the fate of the rooms. A value stored only inside the thing whose loss it guards against is no guard at all.

An index nothing reads guards nothing either. A peer is required to consult it whenever it opens a room, and to report what it finds; see the treatment of lost state under event identity.

`issuedSequence` is advanced before the event using it is published, never after, for the reason given under event identity.

A room's record is written when the room is created or joined, and retained once the room is archived, so a peer can still distinguish a room it once belonged to from one it has never seen.

## What a room's database holds

- events;  
- peer synchronization state;  
- Claude session context state;  
- the room's guest list and configuration.

It must **not** hold this peer's own sequence position for the room. That belongs to the membership index, for the reason above.

---

# 23\. Local Durability

A locally generated event should be committed to durable storage **before** being considered published.

Network transmission comes afterward.

Therefore:

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

A network failure must not lose local conversation history.

---

# 24\. Conflict Model

Conversation events are immutable, so conventional edit conflicts should not occur.

Two people may simultaneously generate:

```
David event E
Alice event F
```

Both are retained.

Neither overwrites the other.

Display ordering should be deterministic.

Use an ordering based on appropriate fields such as:

```
timestamp
origin peer
peer sequence
event ID
```

Do not pretend distributed wall clocks provide perfect causal ordering.

If necessary later, introduce logical clocks.

Do not do so until needed.

---

# 25\. Security

Conversation history may contain:

- proprietary source code;  
- customer information;  
- architecture;  
- logs;  
- credentials accidentally pasted by people.

Peer communication must therefore be authenticated.

For the initial Tailscale-based prototype:

- restrict connections to the private network;  
- identify expected peers;  
- validate room membership;  
- do not expose the daemon's peer API publicly;  
- bind Claude hook/UI APIs to localhost.

Nothing here is admitted by holding a value. An invitation names a guest; it does not grant entry, and an interceptor learns only that a room exists. There is deliberately no credential that can be forwarded, stolen, or replayed — given that peers can be known in advance, every property of such a credential is avoidable, and so none is accepted.

Do not assume network membership alone is sufficient for a production security model.

## Peer identity

Peer identity must eventually become cryptographic. Until it does, several rules in this specification are conventions rather than controls.

A peer identifier should be derived from a public key, a peer should prove possession of the corresponding private key on connecting, and events should be signed by the peer that originated them.

Until that exists:

- a peer identifier is self-asserted, and any peer may claim any identifier;  
- a list of permitted peers is a convenience rather than a control, because the names on it cannot be verified;  
- a relaying peer is trusted not to forge events attributed to others.

Build nothing that depends on those properties holding.

## Known peers

Once identity is cryptographic, admit known peers rather than holders of a secret.

A peer keeps a list of the peers it has met and the public keys it knows them by. A room admits members drawn from that list, and joining becomes a proof of possession rather than the presentation of a token. That is better in every respect that matters here: nothing is transmitted that an interceptor could reuse, nothing expires, and admission can be withdrawn.

This does not remove the first exchange. Two peers that have never met must still establish each other's keys over some channel they trust, exactly as a join secret must be sent over one.

What it removes is every exchange after the first: a key once verified is durable, whereas a secret is spent on use.

Verification is a comparison of **two words**, spoken aloud, derived from a live exchange between the two daemons. It is described in "The two-word comparison" below, and it is the **only** ceremony this system performs. A peer is verified or it is not, and there is one way to become so.

No second method is offered, and the omission is deliberate. Comparing a rendering of the whole identifier is the ceremony a standing value forces, and it would be available without a live exchange — but that advantage buys nothing here. Verification gates synchronization, so an unverified peer cannot collaborate; verification therefore matters only when collaboration is about to happen; and collaboration already requires both daemons running and mutually reachable, which is exactly what the live exchange needs. There is no state in which a peer needs verifying and cannot be verified this way.

What a second method would add is a **weaker way to satisfy the same gate**, which is what people reach for when the stronger one is inconvenient. A gate is only as strong as the weakest ceremony that satisfies it, and the weaker one here is the one people demonstrably perform badly: shown forty-three characters to check over a telephone, a reader takes the first group, the last group, and skims the middle, reducing the verified entropy to whatever was actually compared.

Rendering an identifier for a person to *read* remains useful — when a key has changed and somebody is looking at two of them, grouped output is kinder than an unbroken run of base64. That is a display, not a ceremony, and nothing about looking at it marks a peer verified.

The reason a short string is sound where a static one is not is worth keeping, because it is what makes the single method defensible. A long-term identifier sits still, so an attacker who has intercepted one can grind **offline**, at leisure, for a key of his own whose rendering matches, at a cost of exactly the entropy displayed: sixteen characters are beyond reach, three fall in under a second. A value derived from a fresh exchange cannot be aimed at in advance. The attacker must commit blind, gets one guess, and a wrong guess is a mismatch the two people hear.

This also settles what a peer name is for. A name derived from an identifier is a mnemonic for an identity already verified, never a verification in itself — it covers too few bits and, being derived, is grindable by anyone who wants a particular one.

The two mechanisms therefore compose rather than compete. A single-use secret admits a peer that is not yet known, and being admitted is what makes it known. Between peers that already know each other, no secret is required and none should be demanded.

## Learning a peer's identity

A peer's identity is not obtained in advance. There is no directory to consult, and requiring one before two people could work together would defeat the purpose of an invitation.

Identities are exchanged on joining. The invitation admits a peer that is not yet known, and joining is where each side learns the other's identifier and records it. Because an identifier is a public key, learning one requires no confidentiality and creates no exposure: it is the one part of this exchange that can safely happen in the open.

A self-certifying identifier proves possession of a key. It does not prove which person holds it. Those are different claims, and only the first is established by joining.

The second rests on trust taken at first contact, and its weakness should be stated plainly: whoever presents a valid invitation becomes the peer that is recorded. An invitation that reached the wrong person enrolls the wrong person, durably, under a name every member will thereafter treat as familiar.

Verification is therefore worth doing once, shortly after a first meeting, by comparing rendered identifiers over a channel the invitation did not travel on. Two colleagues already on a call can do it in seconds. The value of doing it once is that it never needs doing again.

## What the second channel actually provides

A rendered identifier is no harder to intercept than the identifier itself: it is the same bytes, spaced for reading. Nothing about the rendering resists interception, and treating it as though it did is the mistake this section exists to prevent.

The protection lies entirely in the channel, and in two properties of it rather than one.

- **Diversity** — an attacker must now compromise two paths instead of one. Comparing a rendering over the same path the identifier arrived on accomplishes nothing at all.  
- **Recognition** — the person verifying can tell *who they are talking to*.

The second is the one that matters most, and the one most easily overlooked. It is why a voice call works and a second written channel does not: two written channels are both text, and whoever controls the delivery of one may control the other. A voice adds a fact no interception supplies — that this is the colleague, not somebody in their place.

State the requirement accordingly. The channel must be one the identifier did not travel on **and** one on which the other party can be recognised. A rendering compared with a stranger, however carefully, verifies that two parties hold the same key and says nothing about whose it is.

## The two-word comparison

Verification is two words, said aloud on a call, by both people at once. It is one act at **pairing** time — not at joining, and not at admission — because pairing is what happens once between two machines and outlasts every room they share (§12).

Reading forty-three characters of base64 over a telephone is a task people do badly or skip, and skipping leaves no trace, so the verified entropy is whatever was actually checked rather than whatever was displayed. Two words are compared as units: somebody either says "badger" or does not, where an eye slides over `ol5v` without stopping.

Two properties make that short form sound, and removing either leaves a ceremony that looks identical and protects nothing:

- **Commitment.** Each side sends a hash of its contribution before either reveals one. An attacker relaying between them must fix what he presents to each side before learning what the other will present, so he cannot search for a pair of substitutions whose words agree. He is reduced to one blind guess.
- **Freshness.** The words derive from per-exchange randomness together with both identities, not from the standing identifiers alone. There is nothing to precompute against, because the target does not exist until the exchange is under way.

Together these turn an offline search into a single online guess, at odds set by the length of the string.

**Failure must be conspicuous, because nothing blocks a retry.** A short string with silent retries is weak: the attacker simply tries again. What protects it is that a wrong guess produces two different strings and the two people *hear* it. So a mismatch must refuse, must say plainly that something intercepted the exchange, and must never present itself as a transient error worth repeating. Nothing about a failed comparison is recorded, because a stored failure invites an interface that offers to try again, and trying again is the one thing that must not be offered.

Requirements on the exchange:

- both sides commit before either reveals, and a reveal that arrives before a commitment, or that does not open the commitment already held, is refused;
- the words derive from both identities and both nonces, ordered so that each side computes the same string without either being the initiator;
- the vocabulary is phonetically distinct and alternates between two lists, so that words said in the wrong order are not a valid rendering of anything;
- that vocabulary is **not** the one used for peer names or room names, since a comparison string sitting beside a derived peer name must not be mistakable for it;
- only a person records the result. The exchange proves both sides hold the keys they named; it cannot establish that the voice on the call is the colleague rather than somebody in their place.

Both people must ask for the verification. A daemon answers a step in the exchange only when its own user has started one, which is what keeps this from becoming a channel by which anyone can cause something to appear on someone else's screen.

A commitment step implemented incorrectly degrades to a grindable value while still looking like a ceremony. That is worse than performing none, because it produces the confidence without the property.

## Verification is a gate, not a label

No transcript crosses to or from a peer that has not been verified, and no
unverified peer's turn enters anyone's model context. Admission and verification
are **two** requirements and both must hold.

They answer different questions and neither implies the other. Admission says this
key may enter this room; it is a decision about a key. Verification says the key is
the person's; it is a decision about a human being. Every cryptographic check in
this specification passes just as well for a key substituted in transit, because
the substituted key is a real key, held by whoever substituted it. Only the
comparison over a recognising channel distinguishes them.

Marking an unverified peer on injected text is necessary and insufficient. The whole
chain here is exact and automatic and rests on one human step, and a step that is
optional, skippable, and costs nothing to skip is a step that will be skipped —
after which nothing looks wrong, which is precisely the failure a marker warns
about. So verification is a gate.

Three consequences follow, and all three are load-bearing:

- **Refuse at the origin, not the sender.** An event relayed under §13 is judged by
  the peer that wrote it, never by the peer that carried it. Otherwise a verified
  relay would launder an unverified author.
- **Hold, do not discard.** Synchronization is a pull against a watermark, so
  refusing to store simply leaves events on offer. When the two people verify, the
  next poll brings the whole backlog. Nothing is lost by waiting and nothing needs
  to be re-sent.
- **Say what is happening.** A room that is silent because of a gate is
  indistinguishable from a room that is silent because nobody is talking. Both the
  refusal and the holding back must name the peer and the command that clears it.

Inviting an unverified peer is permitted and has no effect until they are verified.
That is deliberate: whom to admit remains the host's judgement (D-051), and the
second gate is not a second opinion about that judgement — it is a different
question, asked of a different party.

What is withheld is the **delivery** of the invitation, not the admission. The
guest-list entry is written when the host makes it; the guest is told about it once
verification makes the room usable, and completing a verification delivers whatever
was waiting. Otherwise the easiest room to create would be one that does nothing:
the guest accepts, both believe they are collaborating, and the silence is explained
only by a sentence somebody may not read.

## Room content is unreadable in transit

Events are signed, which gives integrity and origin and conceals nothing. Room
content is a person's prompts and whatever their Claude said back, so
confidentiality is a requirement and not a preference.

Every connection between peers is TLS 1.3, over every transport — including one
that is already encrypted, so that confidentiality never depends on which path a
dial happened to take. The certificate each daemon presents is self-signed and
carries the identity key; the check is that the key inside it is a peer this machine
already knows, and for a dial aimed at one named peer, that it is that peer. Chains,
authorities, hostnames and expiry are never consulted, because none of them exist
here. A client certificate is required rather than requested: a peer with nothing to
pin has no business completing a handshake.

This places the transport's identity and the event's identity on the same key,
confirmed by the same comparison (§25).

It does not protect content at rest, and it does not protect a room from its own
members. A member is entitled to read what the room carries; that is what admission
means.

## The local API is not reachable from a web page

The daemon serves the view, and the ceremony above happens on it, so the local API
is now something a person acts through rather than only reads. It binds to loopback,
which keeps other machines out and does nothing whatever about a page open in this
machine's own browser — `127.0.0.1` is an address like any other to a page, and a
request with a simple content type is sent without the browser asking permission
first. The response cannot be read across origins; the effect has already happened
by then.

What is reachable that way is not incidental. The local API can mark a peer verified,
which is the gate every other guarantee in this section depends on, and it can
publish a turn into a room, which reaches teammates' context windows.

So a request that changes anything must **present** a header that our own code sends
and a page from another origin cannot. A browser will not send a custom header to
another origin without asking first, and that question is answered with nothing, so
the request is never made. The check is that the header is present, which is what makes
it fail closed: anything that cannot present it is refused, whatever it is.

None of this defends against a malicious program running as the same user, which can
send any header it likes and could edit the databases directly in any case. The
threat closed here is a web page, which is the one thing the browser enforces a
boundary for.

## The case with no answer

Two people who have never met cannot recognise each other, so neither property is fully available on first contact. Nothing in this design resolves that, and no rendering will.

What remains is to make the moment visible rather than to pretend it is covered: the first exchange is where trust is taken rather than established, and a system that hides that has moved a risk rather than removed it.

Admission is not retraction. Removing a peer from a list prevents it rejoining and prevents future presence. It does not withdraw what that peer has already seen.

---

# 26\. Offline Operation

Offline operation is a first-class requirement.

Example:

```
David       OFFLINE       Alice

E F G                    H I J
```

Both continue working.

After reconnection:

```
       synchronize

David                  Alice

E F G H I J            E F G H I J
```

Neither person should have to initiate manual reconciliation.

Offline operation is bounded by the room's lifetime. A peer whose session has ended has left the room, and does not rejoin by coming back online; reconciliation applies to members that are still members.

---

# 27\. Tool Activity

Do not initially place every tool invocation in the main transcript.

Prefer:

```
Claude — David

I found the problem in VerificationService.ts...

▸ 14 tool operations
```

Expandable later into:

```
Read SessionLambda.ts
Grep dldvVerification
Read VerificationService.ts
Edited VerificationService.ts
Ran tests
```

Tool synchronization is secondary to prompt/response synchronization.

---

# 28\. Configuration

A room is never configured by a project. Nothing in a repository, and nothing about a working directory, determines which room a session joins. Rooms are entered by invitation only.

Separate personal identity from per-room state.

Per-room state is created when a room is created or joined. It belongs to the daemon, not to any repository, and is not committed anywhere:

```json
{
  "roomId": "0f7a4e6c-2b91-4d0a-9c3e-7f1d8a5b2c44",
  "roomName": "misty-canyon",
  "injectSharedContext": true,
  "guests": ["ed25519:M7Kd…4Fq2", "ed25519:Rb91…7Twc"]
}
```

Disabling `injectSharedContext` makes a room visible to its members as a conversation while placing none of it into any participant's Claude session. This matters because injection carries a teammate's conversation into another person's session, and therefore to that person's model provider under their own account.

Personal machine configuration:

```json
{
  "userId": "david",
  "userDisplayName": "David",
  "machineId": "davids-macbook"
}
```

Peer connectivity configuration should not require committing personal credentials to Git.

---

# 29\. The Experience We Want

Eventually:

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

A participant installs one thing, and it is a thing Claude Code already understands:

```
/plugin marketplace add Blue-Rocket/cogmer
/plugin install cogmer@blue-rocket
```

Two lines rather than one, because a plugin is installed from a marketplace and a
marketplace is added by name: the repository carries `.claude-plugin/marketplace.json`
alongside the plugin it lists, so both lines name the same repository. The first is
run once per person, the second once per plugin.

The plugin carries the hooks. No shell profile is modified, no configuration file is hand-edited, no service is registered with the operating system, and nothing about how Claude Code is started changes.

This follows from Claude Code being launched and used unchanged. A plugin is loaded by Claude Code wherever Claude Code loads plugins, so the system reaches those surfaces without needing to know they exist — which matters because that set will change, and a design that enumerates surfaces will be wrong within a release.

## Where each command lives

Three homes, and which one a command belongs in is decided by the command rather than by preference.

**Inside a Claude Code session**, as commands it already loads: everything a person does in the ordinary course. Listing rooms and peers, creating a room, inviting a known peer, joining, leaving, reading the room, pairing with a colleague, choosing the name others see. Either they have no interaction beyond their arguments, or they hand off to a surface that does — and their output is something a person reads once, so it is acceptable that it reaches them by way of the model, which is how everything from an extension point reaches a person (§3.8).

Joining in particular belongs here. A room is session-scoped, and a command typed inside a session can name the session it means; a command outside one cannot. Nothing machine-level picks a room on a session's behalf, so a room command typed at a terminal is refused rather than answered with a guess.

**In the local view**, as a page the daemon serves: the two-word comparison. Three properties keep that ceremony away from the model, and any one would be enough.

- It is **interactive**, and blocks on another person for as long as it takes them to start their side.
- Its output must reach a person's eyes **unaltered**. Two words whose purpose is that you compared exactly what your daemon computed must not pass through a model, and least of all through the model that is reading room content from peers — the content §20 exists because it once impersonated an operator instruction.
- Recording the result is a **human act**, and the human is on a telephone rather than at a prompt.

Those properties rule out the model. They do not rule out a browser, and the view is where the ceremony belongs: a terminal is an operator surface, and asking somebody to open one in order to meet a colleague is not a user experience.

A slash command therefore **performs** the ceremony rather than pointing at it. It records the peer, mints a page, opens it, and returns without waiting; the waiting and the comparison happen on the page. Every pairing gets its own address, so a second one arrives as a new tab rather than as a silent rewrite of one nobody is looking at, and a page left over from an earlier attempt can never quietly become a different pairing.

The page is reachable only by the person at the machine. The daemon serves it on loopback, and the requests that change anything — beginning an exchange, recording its result, publishing a turn — are refused unless they carry a header a page from any other origin cannot send. Loopback keeps other machines out; that header is what keeps out a page open in this machine's own browser.

**At a terminal**, as a separate program: the operator surface, which is not a user experience and is not meant to be. Diagnostics that must work when the plugin path is broken, the daemon's own lifecycle, this machine's identity, and testing. The terminal ceremony remains, as the fallback for a machine that cannot open a browser — one reached over SSH, or in a container — which makes it a second branch rather than a legacy.

This split is why the vocabulary separates pairing from inviting (§12). A single verb spanning both scopes would have forced both into the more restrictive home.

## Command names

A prefix names what the command acts on. `peer-` acts on somebody else, `room-` on a room, `self-` on you. The prefix names the target and not the activity, so a command that prints your own pairing string is `self-` however much it is about pairing.

Commands are prefixed at all because a plugin shares one namespace with every other plugin a person has installed, and an unprefixed `create` or `status` is a collision waiting to happen.

There is no prefix for the system itself, and so no single command that says what it is or lists the rest. That command would be named after the product, and the name is not settled. A placeholder is worse than the gap: a command name goes into habit and into whatever colleagues write down for each other, and is harder to withdraw later than a directory is.

## Starting the daemon

A person should not have to start the daemon, notice that it has stopped, or know that it exists.

The session-start hook starts it when it is not already running. Installation is then the whole of setup, and the first session after installing is indistinguishable from every session after that.

Three requirements follow, each easy to get wrong:

- **Starting must not delay the session.** Waiting on a daemon nobody asked for is worse than having no daemon. The hook returns immediately and the daemon becomes useful when it is ready; a session that begins before the daemon is listening simply has nothing to inject yet.  
- **Already running is the ordinary case, not an error.** Several sessions begin at once on one machine routinely. Each attempts to start the daemon, at most one succeeds, and none reports a problem. A failure to bind is the expected outcome, not a fault.  
- **Failure is silent to the person and recorded by the daemon.** A daemon that cannot start means no collaboration, which is degraded rather than broken: the session continues exactly as it would have without any of this, and says nothing about it.

The daemon outlives the session that started it, because a room may have members in several sessions and because starting it repeatedly is worse than leaving it running. It must therefore be discoverable and stoppable by the person whose machine it is on. A background process a person cannot find is not acceptable merely because it is useful.

Stopping it is a terminal command, because it is the daemon's own lifecycle. It stops whichever cogmer daemon holds the addresses this installation uses, not only the one this installation started. A daemon left running from another state directory, or from an earlier version, holds the same addresses and blocks this one exactly as an unrelated program would, and the person experiences the two the same way. It signals only a process it has identified as a cogmer daemon. Anything else holding an address is named, with its process, and left alone, because stopping a program this system did not start is not the person's request and is not this system's to do.

A daemon that cannot bind says which of those holds its address, and when it is a cogmer daemon, gives the command that clears it. Describing a blocker without identifying it leaves the person to do the identification.

The person should not have to:

- copy transcripts;  
- summarize what Claude discovered;  
- paste teammates' conversations;  
- manage synchronization;  
- identify missing events;  
- resolve routine synchronization conflicts.

Collaboration should be ambient.

---

# 32\. Explicit Non-Goals

Do not initially build:

- central cloud conversation service;  
- cloud transcript database;  
- Claude API proxy;  
- centralized model inference;  
- RAG;  
- embeddings;  
- vector database;  
- automatic summaries;  
- organizational knowledge base;  
- Slack integration;  
- shared Claude Code session;  
- CRDT framework;  
- libp2p networking;  
- sophisticated peer discovery.

These can be evaluated after the core experiment.

---

# 33\. Future Networking Evolution

Keep networking behind an abstraction.

A transport is responsible for finding peers and moving bytes, and for nothing else:

```
PeerSyncTransport

  discover(roomName)     → candidate rooms reachable by this transport
  connect(peer)
  send(peer, data)
  onMessage(handler)
  onPeerJoined(handler)
  onPeerLeft(handler)
```

Replication sees only this. A transport never interprets an event, never decides membership, and never authorizes a join — `discover` returns candidates, and admission remains a matter of the room's own secret.

Implement in this order:

```
PeerSyncTransport
       │
       ├── Local          ← no dependencies; same network
       ├── Tailscale      ← remote peers, no NAT traversal to solve
       └── WebRTC         ← remote peers with nothing installed
```

Local is first because it is the only one that requires nothing of the user. A project whose simplest case needs an account and a second daemon has narrowed its audience before anyone has tried it.

Later possibilities include QUIC and libp2p.

Internet peer-to-peer carries a cost worth stating plainly before it is chosen. Peers must be introduced to each other by some signalling service, and peers behind unfriendly network address translation need a relay. Signalling is small and never sees a conversation. Relaying is neither: it carries the traffic, and somebody has to pay for it. That is a commitment to be made deliberately, not discovered.

The event and synchronization models should not care which transport is being used.

---

# 34\. Possible Future CRDT Layer

If the application eventually introduces mutable shared state, consider a local-first CRDT such as Automerge.

Potential architecture:

```
Claude Code
     │
    hooks
     │
     ▼
cogmer daemon
     │
 ┌───┴────────────┐
 │                │
event store    shared state
 │                │
immutable       CRDT
 │                │
 └──────┬─────────┘
        ▼
  peer transport
```

Do not introduce this complexity for immutable conversation events.

---

# 35\. Future Capabilities

The architecture should permit later implementation of:

### Address another Claude

```
@David-Claude what led you to that conclusion?
```

### Ask another person to investigate

```
@Alice investigate the race condition David's Claude found.
```

### Presence

```
David
  Claude working...

Alice
  viewing room

Carlos
  offline — last seen 14 min ago
```

### Searchable history

Across archived rooms, since a live room holds only its own pairing:

```
What did we discover about DLDV last Tuesday?
```

### Durable team memory

Extract confirmed:

- architectural decisions;  
- requirements;  
- discoveries;  
- constraints;  
- unresolved questions.

This should be derived from the complete conversational record rather than replacing it.

---

# 37\. The name

The product, the plugin and the repository are `cogmer`. The name reaches the module
path, the binary, the plugin manifest, the state directory and the release path, and
nowhere else.

Nothing cryptographic carries it. Signing namespaces use the `protocolNamespace`
constant, which is arbitrary on purpose and must never change (D-069): a tag carrying
a product name would make every signature ever produced hostage to a naming decision,
and by D-058's rule a rename after real events exist would mean carrying the old
namespace forever. A scheme is added beside the current one and never edited, so a
signature outlives every later change to what an event carries.

The state directory is `~/.cogmer`, and it is the one place the name reaches the
filesystem. Nothing derives state from any other name.

The plugin manifest name is the slash command namespace. Claude Code prefixes every
plugin skill with it, so the manifest name is what a person types before the colon
and changing it changes every command at once (D-118).
