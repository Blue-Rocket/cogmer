# Specification review after Phases 0 and 0a

Evaluated against what the spike actually established (`phase0-findings.md`,
`phase0a-findings.md`) and the choices made since (`decisions.md`).

**Overall: the architecture held up. The event model, the no-CRDT call, the
local-first invariants, and the attributed-injection format all survived contact
with a real implementation.** What did not hold up is mostly at the seams —
places where two sections are individually reasonable but jointly ambiguous, and
one place where the specification still poses a question that has been answered.

**Scope.** This reviews the *specification*. A finding is resolved when the
specification says the right thing — never because the implementation does. Where
the code has run ahead of the text, that is noted as context and does not close
anything.

Divergence between the code and the specification is a different kind of problem
with a different remedy, and is collected under *Implementation conformance* at the
end rather than given a finding number.

Findings are ordered by consequence, not by section number.

**Status, as of the last revision.** Seven of the original fourteen are resolved, two
dissolved by a change of model rather than fixed, and five remain open — A1 among
them, reopened after being closed on implementation work rather than specification
work. Two further findings were raised afterwards: B4, resolved, and A5, open. Each carries its own
status line; the open ones are collected at the end so they are not lost among the
closed.

Resolution came from three directions worth distinguishing. Some findings were
defects and were fixed (A1). Some were questions the specification had not answered,
and answering them removed the finding (A2, C7). And some were answered by deciding
the design differently, which is not the same as the finding having been wrong —
A2's proposed fix was sound for the model it assumed, and that model was replaced.

---

## A. Defects

### A1 — §19 never says *when* delivery state advances, nor what counts as delivered

**REOPENED — was wrongly closed.** This was marked resolved on the strength of
D-014 being implemented and verified. The implementation is sound: delivery is
derived from evidence, confirmed only when the injected block is observed in the
transcript, with a set rather than a watermark and a fallback gated on B20 failing.

But §19 was never changed. It still says only "update the session's
incorporated-event state," without saying when, or what counts as incorporated —
which is the finding, verbatim. A second implementation reading the specification
would make the same mistake the first one did.

The remedy is a specification edit, not more code: state that delivery state must
not advance on offering an injection, and that it advances on evidence the injection
was received.

**High.** §19 lists five ordered steps, ending "update the session's
incorporated-event state," but binds that update to nothing. It also never says
what "incorporated" means: offered to the hook, or observably present in the
session.

Both matter, and the second is the one with a real bug behind it.

**What is not the problem.** The intuitive worry — an interrupted or failed turn
strands context that was marked delivered — does not hold. Injected content is
recorded in the transcript as its own record:

```
type: "attachment"
attachment: { type: "hook_success", hookName: "UserPromptSubmit",
              content: "<team-conversation>..." }
```

That record is part of session history, so the next prompt re-sends it. A turn
dying does not lose the context.

**What is the problem.** `daemon.go:78` commits delivery *before* the hook has
received the response. The hook has a 3s timeout; if it expires, or the daemon
dies mid-reply, the daemon has recorded delivery while the hook printed nothing
and Claude saw nothing. At-most-once delivery on a channel that needs at-least-once.
Narrower than a failed turn, but permanent and silent when it happens.

**Three designs, in increasing strength.**

- *Advance at injection* (current). Simplest; loses the race above.
- *Provisional at injection, committed at `Stop`.* `Stop` carries the same
  `prompt_id` as `UserPromptSubmit`, so correlation is already available and
  already verified (B01, B03). Survives the lost response. Still records intent:
  `Stop` proves a turn ended, not that context arrived.
- *Derive delivery from transcript evidence.* The daemon already reads the
  transcript at `Stop` for turn reassembly. The `hook_success` attachments in it
  are proof of what Claude actually received. Advance the watermark to match.
  This is the only option that verifies rather than assumes, and it is nearly
  free.

**Recommend.** Specify evidence-derived delivery, and require injected messages to
carry their `eventId` so the evidence maps to specific events instead of relying
on text matching — which also makes partial delivery recoverable. Note the cost
honestly: it adds a dependency on the `hook_success` attachment format, which
then needs a registry entry and check (C1). The provisional/commit design is the
conservative fallback, depending only on behaviors already verified.

### A2 — Room resolution is undefined, and §5, §22, and §28 disagree

**Resolved, by dissolution — see D-015.** The recommendation below assumed rooms
are durable project-owned things. They are not: rooms are now scoped to a set of
linked sessions and entered by invitation, so there is nothing to resolve from a
working directory. The `cwd` mapping, the config file, the walk-up rule, the
default-off guard and the room-name traversal check are all unnecessary. The
original finding is retained because the inconsistency it identified is what
prompted the model change.

**High.** Three sections describe mutually incompatible shapes:

- §5 — one daemon, one port (`127.0.0.1:4782`)
- §22 — `rooms/verantid-remote-idv.db`, `rooms/bohl-loyalty.db`: many rooms per machine
- §28 — the room is *project* configuration, so it varies by working directory

Nothing says how a hook call resolves to a room. The hook payload carries `cwd`,
and §28 puts the room in project configuration, so the intended mapping is
presumably cwd → project config → room — but that is inference, and it is the
central routing rule of the whole daemon.

Our implementation dodged it: one room per daemon process, selected by an
environment variable. That directly contradicts §22 and does not survive a
developer working in two repositories at once, which is normal.

**Recommend.** Specify that the daemon serves many rooms concurrently and
resolves each hook call by `cwd`, plus what to do when `cwd` matches no
configured room (most likely: do nothing, silently, per §3.1).

### A3 — §25 does not address what injection does with other people's data

**LARGELY ADDRESSED — D-015, D-019, D-022, D-024, D-025.** The disclosure surface
shrank from several directions rather than one. §28 states that injection carries a
teammate's conversation to another developer's model provider under their own
account, and gives `injectSharedContext` a stated purpose. Rooms are no longer
derived from a repository, so consent is per-pairing (D-015). A room begins when
someone is invited, so prior solo work is never handed over retroactively (D-022).
Admission is a guest list rather than a forwardable token, so there is nothing to
intercept (D-024, D-025).

What remains: §25 still does not enumerate the data-flow consequences in one place,
and no mechanism scopes what *may* be shared once a peer is admitted — admission is
all-or-nothing per room.

**High.** §25 correctly identifies that conversation history may contain
proprietary source, customer information, and pasted credentials, and requires
authenticated peer communication.

It stops at the network boundary. But §18–§19 take a teammate's conversation and
**send it to Anthropic as part of your prompt, under your subscription** — not
only across the network, but into a second inference provider relationship, and
into your session's transcript on your disk, where it is then subject to your
compaction and your retention.

For a room named `verantid-remote-idv` — identity verification, a domain that is
PII-dense by construction — that is a material data-flow question the
specification never raises. It is also the mechanism the whole project exists to
build, so it cannot be avoided; it needs stating.

**Recommend.** Add to §25: injected context crosses account and machine
boundaries, every participant's provider relationship is in scope, and a room's
membership is therefore a data-sharing decision. Consider whether some rooms need
injection disabled while capture stays on — §28 already has `injectSharedContext`
as a flag, which suggests this was anticipated but never argued.

### A4 — §15 still poses a question that has been answered, and the answer is counter-intuitive

**RESOLVED.** §15 no longer poses this as an investigation. It now states that
neither available source is complete, that they fail in opposite directions, and
that their union is the turn — with the additional requirement that the union
tolerate the turn-completion hook being *widened* upstream, since blind appending
would then duplicate every block the transcript supplied. Positional segmentation
and the exclusions (internal reasoning, subagent records) are stated as rules
rather than left to be rediscovered.

The investigative framing was kept only for what is genuinely unknown: a closing
section records that the behavior is observed rather than published, that it can
change without notice, and that one part of it would change under an upstream
*bugfix* rather than a regression.

**High.** §15 says "If Claude Code hooks do not directly expose the completed
assistant response, investigate session/transcript capabilities and implement the
least invasive reliable mechanism."

They do not, and the least invasive mechanism is **not** the obvious one. Both
available sources are incomplete and fail in opposite directions:
`last_assistant_message` holds only the final text block; the transcript at `Stop`
time is missing exactly that block. Anyone implementing §15 from the text alone
will use `last_assistant_message`, and will silently drop the opening of every
turn that speaks before calling a tool — which is most substantive turns.

Leaving §15 as an open investigation invites re-discovering this the hard way.

**Recommend.** Replace the hedge with the requirement: the complete turn is the
union of both sources, and the union must tolerate `last_assistant_message`
widening (D-005). Keep the investigative framing only for what is still unknown.

---

### A5 — Nothing in the specification requires a peer to notice that a room's state is gone

**RESOLVED.** §8 now requires the index be consulted on every room open, defines
the two conditions that mean state was lost, and requires the peer report it in
terms the user can act on — including that history is being refetched and may repeat
teammate context. §22 notes that an index nothing reads guards nothing.

The reasoning is recorded with the requirement: a peer recovering while no other
member is reachable holds a correct sequence position and an empty history, so it
publishes safely and behaves normally while the conversation it believes itself part
of is absent. Nothing about that is apparent from using it.

The implementation still does none of this — see *Implementation conformance*, C-1.

**Found 2026-09-16 by deleting a room database and watching what happened.**

**High.** §8 says a peer that has lost a room's local state must stop using its
identifier in that room, and D-028 says its membership ends. Nothing enforces
either. Reproduced end to end:

- **While the daemon runs, the loss is invisible.** It continued serving all six
  events from its open file handle after the file was deleted. The failure stays
  latent until a restart that may be hours later.
- **On restart the daemon silently creates an empty room** and logs an ordinary
  startup line. An emptied room is indistinguishable from a new one, because
  nothing durable outside the room records that the peer was a member.
- **The sequence counter restarts at 1**, which is precisely the condition D-027
  detects on the *receiving* side. The peer that caused it is never told.

The resulting experience is that nothing appears wrong. Teammate context stops
arriving because the local room is empty; the peer's own events stop reaching
anyone because they collide; each side sees the other fall quiet. The room reports
one event where it held six.

This is the same shape as A1 and B4: a failure with no error path, discovered only
by testing the failure rather than the success.

**Recommend.** Specify the two steps that currently fall between §22 and §8:

- **A peer must check the membership index when opening a room.** If the index
  claims membership and the room's database is absent, or its highest sequence is
  below what the index recorded, local state has been lost.
- **A peer must say so.** A room refetching its history, and possibly repeating some
  teammate context, is in a different state from one working normally, and the
  difference must be reported rather than inferred. Recovery that is silent is
  indistinguishable from nothing having gone wrong.

Detection at startup does not cover the open-handle case, where the file is
unlinked beneath a running daemon. That is the smaller half of the problem and can
follow.

## B. Under-specification that will make peers diverge

### B1 — §24 lists ordering fields but defines no comparator

**RESOLVED — implemented and verified across two peers.** Ordering is
`(timestamp, peer_id, peer_sequence)`, total without further tie-breaking because
§8 makes the last two unique. Local insert order, which two peers necessarily
disagree on, is no longer used for display or injection. The two-peer run produced
identical event ordering on both daemons.

**Medium.** §24 offers "timestamp, origin peer, peer sequence, event ID" as fields
to order by, without a precedence or a tie-break rule. Two correct
implementations can therefore produce different orderings of the same event set,
which breaks §17's promise that each developer sees approximately the same room —
and quietly, since it only shows up when events are near-simultaneous.

**Recommend.** Specify the exact tuple and direction: `(timestamp, peerId,
peerSequence)`, ascending.

*Revised since first written.* Two corrections, both from decisions taken after this
finding was raised:

- **The `eventId` tie-break is redundant.** §8 establishes that `peerId` plus
  `peerSequence` identifies an event, and the implementation enforces it as a
  uniqueness constraint. The three-field tuple is therefore already total, and
  appending `eventId` suggests a tie that cannot occur.
- **`peerId` must be compared over its canonical bytes, not its display form.**
  D-023 makes an identifier a public key, and an implementation is then free to
  render it as hex, base64, or anything else. Comparing rendered strings would
  reproduce exactly the divergence this finding is about, one level down and harder
  to see.

The reference to `peerId` itself remains correct, and is now less ambiguous than
when written: D-021 introduced a derived `peerName`, which collides by design and
must never order anything.

### B2 — Injection order is unspecified, and arrival order is not chronological

**PARTLY ADDRESSED; UNVERIFIED.** Injection now uses the same deterministic order
as display rather than arrival order, which is the fix. It could not be verified:
with a single teammate the injected block contains only that peer's turns, already
in sequence, so interleaving never arises. This needs three peers, or a peer
reconnecting with a backlog alongside a live one.

**Medium.** §19 says to inject unseen events; §24 governs *display* ordering.
Nothing says which order injected context uses.

These genuinely differ. A watermark over local arrival order is the right
mechanism for "has this session seen it," but anti-entropy (§10) and transitive
relay (§13) deliver old events late — so an event authored an hour ago can arrive
after one authored a minute ago. Injecting in arrival order hands Claude a
conversation that is out of sequence, which is precisely the kind of thing that
makes a referent resolve wrongly.

**Recommend.** Require that injected context be sorted by the §24 display
ordering, even though the delivery watermark tracks arrival.

### B3 — §19's exclusion rule is ambiguous between session and peer

**OPEN in the specification; settled in the implementation.** The code excludes by `claudeSessionId`, which is correct, and a test asserts a second local session still sees a peer's events. §19's wording is unchanged.

**Low.** "Exclude Alice's own Claude conversation where it would duplicate
existing context." A developer may run two Claude sessions on one machine, in the
same room. Excluding by *peer* would blind each session to the other; excluding by
*session* is correct, and is what we implemented. The text supports either.

**Recommend.** Say `claudeSessionId`.

---

### B4 — A restarted sequence counter is silently absorbed as a duplicate

**RESOLVED — specified and implemented, 2026-09-16.** §8 now states that a sequence
is scoped to a room, never reset or reused, that a peer unable to continue its
sequence must stop using its identifier in that room, and that redelivery and
conflict must be distinguished on receipt.

`Insert` now returns `stored` / `duplicate` / `conflict` instead of silently
ignoring both of the last two. A conflict is quarantined with both event
identifiers and the rejected event retained — discarding it would destroy the
evidence that distinguishes lost state from forgery — and `claude-team conflicts`
surfaces them, since a conflict is never routine and is invisible unless asked for.
Three tests cover the conflict, ordinary redelivery staying silent, and normal
sequential inserts.

**Medium, latent.** §8 makes `peerId` + `peerSequence` the pair that identifies an
event, and §9 builds anti-entropy on "the highest contiguous sequence received from
each peer." Both assume a peer's counter only ever moves forward. Nothing says what
happens if it does not.

It can. A peer that loses its room database — disk failure, a restored backup, a
deleted directory — restarts the counter at 1 while other peers still hold events
at higher numbers under the same identifier. Every event it then publishes carries a
`peerSequence` a receiving peer already has.

The receiving peer cannot tell this from an ordinary duplicate. In the current
implementation `INSERT OR IGNORE` with `UNIQUE(peer_id, peer_sequence)` drops it,
which is correct for a genuine redelivery and silently wrong here. The publishing
peer believes it has shared; the receiving peer never sees it; neither is told.
Anti-entropy cannot repair it either, since the sender's highest sequence is now
*below* what the receiver reports holding.

Session-scoped rooms (D-015) bound the damage to one pairing rather than months of
history, but do not prevent it.

**Recommend.** Two parts, one specification and one implementation.

- State that a peer's sequence is scoped to a room, monotonic within it, and never
  reset or reused — and that a peer which cannot continue its sequence must not
  reuse its identifier in that room.
- Distinguish redelivery from conflict on receipt. The same `peerId` and
  `peerSequence` arriving with a *different* `eventId` is not a duplicate; it is
  evidence of lost state or forgery, and must be surfaced rather than absorbed. This
  is one of the concrete things event signing (D-020) would let a receiver
  adjudicate rather than merely detect.

**Not yet reachable.** Nothing relays events between peers, so no conflict can
occur today. `Insert` is the method peer synchronization will call, which is why it
is worth fixing before Phase 2 rather than after.

## C. Gaps the discoveries opened

### C1 — Nothing in the specification acknowledges that it depends on undocumented behavior

**OPEN in the specification; built.** `claude-team doctor` verifies twenty behaviors against the installed version, keyed on `claude --version`, with negative tests. The specification still does not require any of it, so a second implementation would not know to.

**Medium.** The system rests on roughly nineteen behaviors of a third-party binary
— hook payload shapes, transcript record structure, what compaction preserves.
None are contractual, several fail silently, and one of them (§15's) would be
broken by an upstream *bugfix*.

The specification treats Claude Code as a stable platform. §36 tells the
implementer to inspect the installed version once, at the start, which was good
instinct — but a one-time inspection does not survive upgrades.

**Recommend.** Add a standing architectural requirement, alongside §33's transport
abstraction: relied-on behaviors are enumerated and verified against the installed
Claude Code, and a verification failure degrades the feature rather than breaking
the session. `claude-team doctor` already implements this; the specification
should require it rather than have it exist only as an accident of how we worked.

### C2 — Compaction has a phase but no standing requirement

**OPEN.** Phase 0a's findings still live only in `phase0a-findings.md`. §21 was rewritten for session-scoped rooms but still does not mention that Claude Code compacts sessions on its own.

**Medium.** Phase 0a (which I added) is an investigation. §21 "Context Window
Management" governs injection limits and never mentions that Claude Code compacts
sessions on its own. So the *findings* — session ID survives, the transcript is
append-only, the summary record carries no `promptSource`, compaction never fires
mid-turn — live only in a findings document, while §21 reads as though injection
limits were the whole of context management.

**Recommend.** Fold the durable results into §21, and keep Phase 0a as the
historical investigation.

### C3 — Automatic compaction is unresolved and unowned

**OPEN.** Still untriggerable in print mode, still owned by no phase.

**Low–medium.** Phase 0a could not trigger it: the threshold floor is 100k, and a
378k-token session with a 100k threshold did not compact at a turn boundary. Every
compaction finding therefore describes *manual* compaction only.

No phase owns the question. It will first appear under real context pressure in
Phase 4, by which time cross-Claude context is the thing being debugged.

**Recommend.** Name it explicitly as a Phase 4 risk, or give Phase 0a a deferred
item. Do not leave it only in a findings appendix.

### C4 — "Real time" means two different things, and one of them is slow

**RESOLVED.** §16 is rewritten around three paths rather than one. Peer to peer
states its two real requirements — prompt convergence, and recovery without any
peer tracking what another is owed — and records that polling satisfies both.
Daemon to UI is named as the one path where pushing earns its cost. Daemon to a
Claude session is stated as *not existing*: context arrives at prompt submission and
at no other moment, which is a property of the host and what bounds everything above
it. The section now requires both numbers be reported separately.

**Medium.** §16 targets sub-second peer propagation, and we measured 16.7 ms
locally, so the daemon-to-daemon claim is realistic.

But propagation *into a teammate's Claude* is bounded by their turn, because §18
injects at `UserPromptSubmit` and there is no other injection point. A teammate
mid-way through a long agentic turn — minutes, routinely — will not see your
message until they submit their next prompt. §16 does not distinguish these, and
§30's question ("approximating a single shared Claude conversation") depends
heavily on which one a reader assumes.

**Recommend.** State both latencies separately. This is a real property of the
design, not a defect, but it should be a stated expectation rather than a
surprise.

### C5 — §3.4 and §21 forbid summarization that compaction performs anyway

**OPEN.** The prohibition still reads as absolute.

**Low.** §3.4 says do not replace conversations with summaries; §21 says do not
introduce AI summarization. Claude Code's compaction summarizes injected teammate
context inside the session regardless.

There is no contradiction in practice — the room's stored history stays verbatim
in SQLite, which is what those sections protect — but the text currently reads as
an absolute guarantee that the system cannot make.

**Recommend.** Scope the prohibition to the room and its replication, and note
that what a given Claude retains post-compaction is outside the system's control.

### C6 — `COMPACTION` is a "later" event type that is already needed

**OPEN.** Still listed under types to design for, while D-007 commits to recording it in Phase 1.

**Low.** §7 lists `COMPACTION` under types to design for. D-007 commits to
recording it in Phase 1, as the observability backstop for D-006's decision not
to rewind the watermark. Minor drift between the specification and the decision.

### C7 — §25 asks for signable peer identity; identity is currently a random string

**RESOLVED for integrity; open for admission.** Identifiers are now Ed25519 public
keys, events are signed at origin, and a receiving peer rejects anything that does
not verify against the key its own identifier names. §13's relay rule is enforced
rather than stated, and attribution is no longer a claim a peer can make freely.

What remains is admission: nothing yet proves possession of a key *on connection*,
so the peer API still admits any host that can reach it to read a room. That is
Phase 10's work, and the startup warning now says exactly that rather than claiming
identity is not cryptographic.

**Previously — see D-020.** §25 now states
what cryptographic identity requires and which rules are conventions until it
exists, §13 records that the relay rule has no enforcement, and §6 warns that
attribution is not resistant to a peer that lies. The severity was understated
below: transitive relay cannot be made safe without it.

**Low.** §25 says to design peer identity so cryptographic signing can be added
later. `identity.json` holds a random hex `peerId` with no keypair. Nothing is
broken yet — there is no peer traffic — but "design for it later" becomes harder
once peers have exchanged unsigned events and a `peerId` format is entrenched.

**Recommend.** Decide the identity format before Phase 2, not before Phase 7.

---

## D. What held up

Worth recording, because the useful output of a review is not only a defect list:

- **The event model survived implementation unchanged.** `peerId + peerSequence`
  plus a globally unique `eventId` was sufficient; nothing needed adding.
- **§11's refusal to adopt a CRDT was right.** Immutable append-only events made
  the store trivial, and nothing encountered has argued for more.
- **§20's attributed-injection format worked better than expected.** Claude
  treated teammate turns as evidence rather than instruction — it resolved a
  referent from injected context and then disagreed with it on the merits, which
  is exactly the intent.
- **§3.1 is verifiable and verified.** Sessions survive a dead daemon.
- **§23's durability ordering** (commit locally, then transmit) needed no revision.
- **Phase 0's instruction to inspect the installed version rather than trust the
  specification** was the single most valuable line in the document. Both defects
  in §15 came from following it.


---

## Still open

Five findings and three notes, after the work of 2026-09-16:

| | finding | why it survives |
|---|---|---|
| A1 | §19 still does not say when delivery advances | reopened; was closed on implementation work |
| B1 | §24 has no ordering comparator | peers can diverge silently |
| B2 | injection order unspecified | late-arriving events injected out of sequence |
| C1 | behavior dependence unacknowledged | `doctor` exists; the specification does not require it |
| C2 | compaction has no standing section | findings live only in a findings document |
| C3 | automatic compaction unowned | will first appear under Phase 4 pressure |
| C4 | two latencies conflated | §16 promises one and delivers the other |
| C5 | summarization prohibition is absolute | compaction summarizes regardless |
| C6 | `COMPACTION` reserved but needed | D-007 commits to it in Phase 1 |

B3 and A3 are partly closed; the residue of each is noted in place.

A pattern across the open items is worth naming: every one of them is a case where
the implementation or the findings know something the specification does not. That
is survivable while one team holds both, and is exactly what stops being true if
this is released publicly.

## Findings that arose after this review

Recorded here so the review remains the single place to look:

- **The delivery fallback re-created the bug it fixed** (D-014). Committing on trust
  when no evidence is found is indistinguishable from the injection never arriving.
  Found by testing the failure path, not the success path.
- **`machineId` was never an address** (D-018). `os.Hostname()` returned
  `macbookpro.lan`, `scutil` returned `pushover`, and the resolvable name mapped to a
  LAN address no teammate could reach. The specification had used `alice-machine:4783`
  throughout without saying how it resolves.
- **A peer identifier was a broadcast secret** (D-023). Knowing one was sufficient to
  claim it, and the system puts it in every event. Fixed in the specification by
  making the identifier a public key; not yet enforceable, because it becomes safe to
  know when signatures are *checked*, not when keys are introduced.
- **The bearer token was never necessary** (D-024, D-025). A guest list admits a known
  peer with nothing typed, and a host present can approve a stranger's request.
  Tokens are now absent from the design entirely rather than deprecated within it.


---

## Implementation conformance

Where the code and the specification disagree. These are not findings about the
specification and carry no finding number; the remedy is code.

**C-1. A lost room database is not detected, and recovery does not happen.**
§22 requires a membership index and §8 requires resuming above a recorded sequence.
Neither exists. Reproduced by deleting a room database:

- while the daemon runs, the loss is invisible — it served all six events from its
  open file handle after the file was deleted, so the failure stays latent until a
  restart that may be hours away;
- on restart the daemon silently creates an empty room and logs an ordinary startup
  line;
- the sequence counter restarts at 1, the precise condition D-027 detects on the
  receiving side, while the peer that caused it is never told.

The experience is that nothing appears wrong. Teammate context stops arriving, the
peer's own events stop reaching anyone, each side sees the other fall quiet.

**C-5. Injected attribution used a self-asserted display name — fixed.** Both peers
in the two-peer run asserted `David`, derived from `$USER`, and every event in the
room read as one person despite being two. Attribution now anchors on the derived
peer name and marks the speaker unverified inside the injected text, per §20 and
D-021. Recorded here because it was a live divergence found by running the system,
not by reading it.

**C-6. One listener served hooks, UI and sync — fixed.** §5 diagrams a localhost
interface for Claude Code and the local UI and a separate peer interface, and §25
requires it. The implementation bound all three to one loopback address, so the
separation the specification treats as a security boundary did not exist. It was
also the blocker for two machines: exposing sync would have exposed the hook API,
which publishes into the room and reads the conversation back.

There are now two listeners. Hooks and UI refuse to bind anything but loopback.
Peer sync defaults to loopback and warns, when bound elsewhere, that it is reachable
and unauthenticated — which it is, until D-023 lands. Tests assert that neither
listener serves the other's routes.

**C-2. Rooms are selected by environment variable.** §12 and §28 describe rooms
entered by invitation with a guest list. The implementation takes a room name from
`CLAUDE_TEAM_ROOM` and has no invitation, membership, or guest list. Expected —
this is Phase 1 work — but it means no part of the admission design is exercised.

**C-3. Peer identity is cryptographic — resolved.** Identifiers are Ed25519 public
keys (`ed25519:…`), the private key lives in its own `0600` file and never reaches
`whoami`, events are signed at origin, and receipt rejects what does not verify.
Attribution and the relay rule are now controls rather than conventions.

Possession is now proved on connection too: sync requests are signed, with replay
and staleness rejected. **Admission remains a convention.** Verified by test — a
stranger generated a key, authenticated correctly, and read a private room, because
nothing yet decides *which* peers may ask. Authenticating every caller and admitting
every authenticated caller is not confidentiality. That is Phase 10.

**C-4. Room identifiers and names are not generated.** §3.2 requires a `roomId` UUID
and a generated `roomName`; the implementation uses a bare string. `PeerName` is
implemented; its room equivalent is not.
