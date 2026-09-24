# Phases

The phases the work was planned in, what each set out to do, and how each ended.
Phase numbers are referred to throughout the decision log, the findings documents and
the code, so they are never reused or reassigned.

## Status

Where a later phase supersedes an earlier one, the earlier is marked rather than rewritten.

| | |
|---|---|
| Phase 0 — Integration spike | **complete** |
| Phase 0a — Compaction probe | **complete** |
| Phase 1 — Single-machine daemon | **partial** — hooks, daemon and storage work; no local UI |
| Phase 2 — Two-peer synchronization | **complete**, ahead of Phase 1 and not over Tailscale |
| Phase 3 — Real-time push | **will not be built** as written; polling is the decided mechanism for peers |
| Phase 4 — Cross-Claude context | **complete**, in Phase 0 |
| Phase 5 — Offline/reconnection | **complete** — verified in-process and across two machines; see `docs/phase5-findings.md` |
| Phase 6 — Three-peer/transitive | deferred |
| Phase 7 — Hardening | **complete** — the outbound queue was dissolved by pull rather than built |
| Phase 11 — Installation | **complete** — plugin, hooks, commands, and a verified binary fetch (D-066, D-067) |
| Phase 12 — Discovery on a local network | deferred — see D-063; the first pair never share a network |
| Phase 13 — Somebody else uses it | ready — installable from the repository the module path names; the untested half of §30 |
| Phase 14 — First contact without a paste | conditional on Phase 13 |
| Phase 15 — Reaching a peer on another network | **built and working between two machines**; NAT-to-NAT awaits the real peer (D-068) |
| Phase 8 — The local UI | **complete** |
| Phase 9 — Peer identity | **complete**; verification added afterwards and gates synchronization |
| Phase 10 — Pairing | **partial** — pairing, rooms, invitation, joining and admission work; a host approving an unsolicited request does not exist, and whether it should is undecided (§12a) |

The original sequence assumed four things that have since been displaced: a room scoped to a project, Tailscale as the transport, push between peers, and admission by a shared secret. Work proceeded out of order while those assumptions were being tested, which was the right trade during an experiment and is the wrong one now.

**Execute the remaining phases in the order given below, not in numeric order.**

## Phase 0 — Claude Code Integration Spike

Before building networking:

Determine exactly how current Claude Code supports:

- capturing submitted prompts;  
- capturing completed assistant responses;  
- obtaining Claude session IDs;  
- injecting additional context before processing a prompt.

Build a local proof of concept.

Demonstrate:

```
prompt
   ↓
hook
   ↓
local daemon
   ↓
stored event
```

and:

```
local external event
   ↓
hook
   ↓
Claude context
   ↓
Claude correctly understands it
```

Do not proceed until both directions work reliably.

---

## Phase 0a — Compaction Probe

Phase 0 establishes that conversation capture and context injection work.

Compaction is the one mechanism that can silently invalidate both.

Run this probe before building any daemon beyond the spike.

Determine how the installed Claude Code behaves when a session compacts:

- whether a compaction hook fires, for automatic and for manual compaction;  
- what that hook receives;  
- whether the Claude session ID survives compaction;  
- whether the transcript path survives compaction;  
- whether the transcript is appended to, rewritten, or truncated;  
- whether a compaction boundary is marked in the transcript;  
- whether compaction can occur mid-turn, between prompt submission and turn completion.

### The critical question

Stored history and injected context are separate.

Delivery state is tracked separately again:

```
lastSharedContextState
```

Compaction may discard injected teammate conversation while that record still claims the session incorporated it.

The session is then marked as having received conversation it can no longer see.

Nothing reports this:

```
teammate turn injected
        │
        ▼
session compacts
        │
        ▼
injected turn discarded from context
        │
        ▼
delivery state still says "incorporated"
        │
        ▼
Claude silently loses the referent
```

Determine whether this occurs.

### Demonstrate

```
inject a teammate turn
        │
        ▼
drive the session past the compaction threshold
        │
        ▼
submit a prompt whose referent exists only in that injected turn
        │
        ▼
observe whether Claude still resolves it
```

This is the referential test from the Phase 0 spike, applied across a compaction boundary.

The failure is not an error condition.

It presents as Claude quietly misunderstanding a teammate reference it previously understood, so test it explicitly rather than waiting to encounter it.

### Remediation

If injected context does not survive compaction, evaluate:

- rewinding delivery state when compaction is detected, so unseen teammate turns are injected again on the next prompt;  
- re-injecting a bounded floor of recent teammate turns after any compaction boundary, regardless of delivery state;  
- recording a COMPACTION event, a type the event model already reserves;  
- relying on Claude Code's own compaction summary to carry the injected conversation forward.

Prefer the least invasive mechanism that restores the referent.

Do not introduce summarization of teammate conversation.

Duplicate injection is acceptable where loss is not.

### Turn Reassembly Across a Boundary

Verify that assistant turn reassembly still functions after compaction.

Reassembly locates the most recent human-submitted prompt record and collects the assistant records following it.

If compaction rewrites or truncates the transcript, that anchor may no longer be present.

Determine what reassembly returns when it is not.

### Document

As in Phase 0:

- the exact hooks and fields observed;  
- whether compaction is observable at all;  
- whether automatic and manual compaction behave identically;  
- limitations;  
- the chosen remediation, and why.

Do not begin Phase 1 until the behavior of injected context across compaction is known, and, if that context is lost, a remediation has been demonstrated.

---

## Phase 1 — Single-Machine Daemon

Build:

```
Claude hooks
     ↕
local daemon
     ↕
SQLite
     ↕
local UI
```

Demonstrate a complete local Claude conversation appearing in the UI.

---

## Phase 2 — Two-Peer Synchronization

Connect two machines over Tailscale.

Implement:

```
peer state exchange
missing event retrieval
event persistence
deduplication
```

Demonstrate:

```
David asks Claude
       ↓
David daemon
       ↓
Alice daemon
       ↓
Alice's browser
```

and vice versa.

---

## Phase 3 — Real-Time Push

Add immediate peer propagation.

Demonstrate subsecond conversational updates under normal network conditions.

Retain anti-entropy synchronization for recovery.

---

## Phase 4 — Cross-Claude Context

Implement incremental unseen-event injection.

Demonstrate:

```
David:
The problem appears to be the connection pool.

Claude-David:
Yes, specifically...

Alice:
I don't think that's the issue.
```

Alice's Claude should understand what Alice means without manually copying David's conversation.

This is the critical proof of concept.

---

## Phase 5 — Offline/Reconnection

Disconnect Alice.

Generate conversation independently on both machines.

Reconnect.

Verify that both event stores converge automatically and neither conversation is lost.

---

## Phase 6 — Three-Peer/Transitive Test

Use:

```
David ↔ Alice ↔ Carlos
```

Temporarily prevent direct David ↔ Carlos communication.

Verify that David-originated events reach Carlos through Alice while retaining David as the event origin.

---

## Phase 7 — Hardening

Add:

- authentication;  
- room authorization;  
- ~~local outbound queue~~ — **dissolved**, not deferred. It presupposed push:
  something must hold what a peer has not yet been told. Synchronization is a pull
  (§10), so an event is durable the moment it is committed locally, and "what did
  you miss" is a question asked rather than state anyone keeps. A queue would be a
  second record of what the event store already holds, and the two would drift.  
- peer health;  
- context-size controls;  
- better reconnection;  
- database recovery;  
- diagnostics.

Database recovery includes the membership index, the check that reads it when a room is opened, and the report when it finds state missing. That sits here rather than earlier because it guards against a loss, and a guard has nothing to protect until there is something worth losing.

Partially taken already, where doing so was cheaper than deferring: verification of relied-on host behaviour, quarantine of conflicting events, and separation of the local and peer network interfaces.

---

## Phase 8 — The local UI

A room, read by a person, updating without them acting.

Every experiment so far has measured whether *Claude* understands a teammate's conversation. Whether a *person* finds it useful to watch one has never been tested, and cannot be while the only way to read a room is a command-line dump. That is half of the question this prototype exists to answer, and it is the half with no evidence at all.

This phase carries nothing else. It depends on no identifier, no index, and no protocol, so it entrenches nothing and can be built immediately. Requirements are in the section on the shared conversation UI; the live-update path is the one place where pushing earns its cost.

## Phase 9 — Peer identity

- identifiers derived from a public key;  
- possession proved on connection;  
- events signed at origin, and signatures verified on receipt.

Before pairing rather than after. A guest list admits whoever claims a name until identity is verifiable, so an admission flow built first would be built twice — and transitive relay cannot be made safe at all without signing.

## Phase 10 — Pairing

- a room identifier and a generated room name;  
- invitations;  
- joining;  
- known peers, and a room's guests;  
- admission by proof of possession, or by a host approving a request.

Room identity belongs here rather than earlier: a generated name exists to be spoken in an invitation, and has no work to do until there is one.

Nothing in this phase should be built before Phase 9. Its correctness rests entirely on identity being verifiable.

---

## Phase 11 — Installation

- ship as a Claude Code plugin, carrying the hooks;
- the session-start hook starts the daemon;
- slash commands for the room operations, each calling the corresponding CLI command;
- `create` and `join` bind the session that invoked them;
- delete the machine-level current room;
- accept an older protocol version for reads, so an upgrade is not a flag day.

Installation is the wall everything else is behind: nobody but the author has run this, and the reason is that running it takes a build, a hand-edited settings file, and a daemon started by hand. §29 says what the result must be — `/plugin marketplace add Blue-Rocket/cogmer` then `/plugin install cogmer@blue-rocket`, and nothing about how Claude Code starts changes.

A slash command is a **thin wrapper**, never a reimplementation (D-057). It shells out to the same binary a terminal would, so the CLI remains the surface that can be tested without a Claude session, and there is one implementation of each operation rather than two that drift. `CLAUDE_CODE_SESSION_ID` is in the environment of every tool call and equals the id the hooks report (B21), so a command run from a session knows which session it is in without being told.

That is what lets **`create` and `join` bind a session**, and that is what let the machine-level current room be deleted. It existed only because a terminal command cannot name a session, and it was a hazard while it existed: a room created and forgotten is silently joined weeks later by a session in an unrelated repository, because nothing derives a room from a directory and nothing expires the setting.

**Pairing and verification do not pass through the model** (§29). Each is interactive, each blocks on another person, and the two words must reach a person's eyes without passing through a model that reads room content from peers. The ceremony happens in the view the daemon serves, and at a terminal on a machine that cannot open a browser.

---

## Phase 12 — Discovery on a local network

- find peers on a shared network without an address being typed;
- a peer advertises an address it need not bind;
- an invitation may carry more than one endpoint.

**Deferred, and not because it is hard.** Discovery answers the same-network case, and the pair this is being built for work from home and will never share a network (D-063). It remains the right answer for a case that will arrive, and it is cheap once Phase 15 has built the seam a transport plugs into. It is simply not first, which is a reversal of D-019's priority and not of its requirement.

What it is for: a colleague who installs the plugin cannot reach anybody until somebody types an address at them, and the addresses are the part neither person can be expected to know.

The second item is not a refinement of the first, it is the defect that makes tunnels and NAT impossible to express: `peerAddr()` is both what the daemon binds and what it tells other peers to use, and those are the same string only when nothing sits in between. Observed in Phase 5 — an invitation advertised the host's own loopback, and the bad address then propagated to the other peer, which retried it once a second for the length of the run.

**Discovery locates; it never admits** (D-024). A peer found on a network is a peer whose identifier is now known, and knowing an identifier grants nothing. Admission remains the guest list, and no transport is ever a prerequisite (D-019).

---

## Phase 15 — Reaching a peer on another network

- two peers behind different NATs establish a direct path;
- the transport is one provider among several, and none is required;
- an endpoint is whatever that transport can reach, and is never assumed to be an address the daemon binds.

**This is the case that actually exists.** The first pair are colleagues at one company, both working from home, pairing daily. Local discovery (Phase 12) answers the same-network case and cannot answer this one. Every cross-network run so far has used an SSH tunnel — Phase 2's and Phase 5's both — which is a person doing NAT traversal by hand, and is not something a colleague can be asked to do twice a day.

The endpoint defect belongs here and has been taken: `peerAddr()` was both what the daemon binds and what it tells peers to use, and those are the same string only when nothing sits between them. Binding and advertising are now separate, and an endpoint carries the transport that understands it.

**Whatever is adopted must sit under everything** (D-019). A transport carries bytes. It does not decide who may speak: a sync request is still signed (D-044), still refused unless its peer is a guest (D-045), and still refused unless that peer has been verified by a person (D-054). A connection that reaches the door is not admission, and a peer identity is an Ed25519 key (D-020) whatever key a transport happens to use for its own tunnel.

That is made true rather than intended by serving **one set of routes over both listeners**. A request arriving through a tunnel reaches the same handler as one arriving over TCP, so it cannot pass a check the other would fail; there is no second path on which a rule might be forgotten.

**Connectivity and directness are different questions.** Any router permits outbound connections, which is all a relay needs, so a working path is near-certain and its floor is relay latency. Whether that path upgrades to direct is a question about latency rather than correctness, and for turns of about a kilobyte it is unlikely to matter. Do not report a relayed path as a failure.

---

## Phase 13 — Somebody else uses it

- a person who did not write this installs it, pairs, joins a room, and works;
- record what they hit, in the order they hit it.

**Unblocked.** The repository the module path names exists, carries the marketplace
manifest beside the plugin, and serves the release assets the installer verifies
against its pins (D-121). What remains is a person: a plugin copied by hand would
measure a first five minutes that never happens again, and that is no longer what
anybody has to do.

**This is the untested half of §30.** Every experiment so far has measured whether Claude understands a teammate. None has measured whether a person finds a teammate's Claude worth having, because no person but the author has ever been in a room.

It is a phase rather than an afterthought because it is the only one that can fail in a way the others cannot detect. Everything above can be correct and this can still go badly, and the failure would look like somebody quietly not using it again.

Two things to watch for specifically, because both have been argued about without evidence: whether the two-word comparison is performed or skipped, and whether the browser view is consulted or forgotten.

**One question must be settled before this phase, not during it.** Reaching a peer
across NAT currently works by way of a public relay operated by a third party, used
with no account and no configuration of ours. Until now that has carried one
author's traffic between two of his own machines. Phase 13 is the point at which it
begins carrying somebody else's conversation, so the question of whether that relay
is an acceptable dependency stops being theoretical.

Three parts to it, and they have different answers. Whether depending on a relay
nobody here operates is compatible with §4's rule that
no transport is a prerequisite. What the relay observes — a relay cannot read what
it carries, and does see which nodes are talking, when, and how much. And whether
unconfigured use of somebody's free infrastructure is something to build a product
on at all.

---

## Phase 14 — First contact without a paste, if Phase 13 says so

- a host is told that a peer asked to enter, and may approve it;
- requests arrive only while the host has said they are expecting someone.

Conditional, and deliberately last. Its only value is removing one manual step — sending an identifier out of band — between people who already know each other. Whether that step is worth removing is a question about how it feels to do, and Phase 13 is where that is answered rather than guessed.

Two things are already settled about it and should not be reopened by building it. Pairing with someone unknown is **not a supported case** (D-051): §25's verification requires a channel on which the other party can be recognised, and strangers have none, so the ceremony would run and establish nothing about whose key it is. And whether an unsolicited request may arrive at all is **open** (§12a): a request that anyone reachable can cause is a prompt people learn to dismiss, which is a poor place for a decision that matters.

---

## Order of remaining work

Phases 0 through 10 are complete or dissolved. What remains is not more of the
prototype; it is everything between a prototype that works and a thing somebody
else can use.

```
Phase 11   installation                          done
   ↓
Phase 15   reaching a peer on another network    built; NAT-to-NAT untested
   ↓
Phase 13   somebody else uses it                 ← blocked on a permanent name
   ↓
Phase 14   first contact without a paste         only if 13 asks for it
   ↓
Phase 12   discovery on a local network          when a pair who share one appears
```

**The order is by dependency, and the dependency is a person's first five
minutes.** They install (11); they cannot reach anybody until the two machines can
find a path (15); and only then is there anything to observe (13). Phase 14 is last
because Phase 13 is what decides whether it should happen at all.

**Local discovery is last, and that reverses D-019.** It made local discovery the
first transport to build, reasoning that two people on the same network is the
simplest case. The reasoning is sound and the case is not the one that exists: the
first pair who need this are colleagues at one company who both work from home and
will never share a network. A zero-configuration path that serves nobody is not a
zero-configuration path. D-019's *requirement* is unchanged — no provider may be
required, and the same-network case must cost nothing when it arises — but it is no
longer what gets built first (D-063).

Every cross-network run so far has substituted an SSH tunnel, which is a person
performing NAT traversal by hand. For a pair who pair daily, that is the whole
product failing at the first step.

Installation comes before discovery even though discovery is what makes
installation true, because packaging decides where commands live and what a
session can name — and discovery has to expose itself through whatever that turns
out to be. Building discovery first would mean designing against a surface that is
about to change.

Phase 13 is deliberately a phase and not a milestone. It is the only remaining work
that can fail in a way none of the others detect, and its failure looks like
somebody quietly not using this again.

Each phase carries one purpose. Room identity and the membership index belong to the phases they serve; neither blocks the UI.

Pairs are the target throughout. A third peer adds noise to a working session and is unlikely to invalidate anything, so Phase 6 waits for evidence that anyone wants it.
