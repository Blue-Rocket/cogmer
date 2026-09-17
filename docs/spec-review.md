# Specification review after Phases 0 and 0a

Evaluated against what the spike actually established (`phase0-findings.md`,
`phase0a-findings.md`) and the choices made since (`decisions.md`).

**Overall: the architecture held up. The event model, the no-CRDT call, the
local-first invariants, and the attributed-injection format all survived contact
with a real implementation.** What did not hold up is mostly at the seams —
places where two sections are individually reasonable but jointly ambiguous, and
one place where the specification still poses a question that has been answered.

Findings are ordered by consequence, not by section number.

---

## A. Defects

### A1 — §19 never says *when* delivery state advances, and the obvious reading loses data

**High.** §19 lists five ordered steps, ending "update the session's
incorporated-event state," but does not bind that update to the turn completing.

The natural implementation — and the one now in `daemon.go:78` — advances the
watermark at prompt submission, because that is when the hook runs. If the turn
then fails, is interrupted with Ctrl-C, or hits an API error, **those teammate
events are marked incorporated and are never injected again.** The context is
lost permanently and silently, which is exactly the failure Phase 0a went looking
for in compaction and did not find. It was in our own code the whole time.

This is a live bug, not only a specification gap.

**Recommend.** State that delivery state must not advance until the turn in which
the context was injected completes. Since `Stop` carries the same `prompt_id` as
`UserPromptSubmit`, the natural design is provisional delivery at injection,
committed at `Stop`. The specification should say which, because both are
defensible and they differ under failure.

### A2 — Room resolution is undefined, and §5, §22, and §28 disagree

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

## B. Under-specification that will make peers diverge

### B1 — §24 lists ordering fields but defines no comparator

**Medium.** §24 offers "timestamp, origin peer, peer sequence, event ID" as fields
to order by, without a precedence or a tie-break rule. Two correct
implementations can therefore produce different orderings of the same event set,
which breaks §17's promise that each developer sees approximately the same room —
and quietly, since it only shows up when events are near-simultaneous.

**Recommend.** Specify the exact tuple and direction. `(timestamp, peerId,
peerSequence)` with `eventId` as final tie-break is sufficient and total.

### B2 — Injection order is unspecified, and arrival order is not chronological

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

**Low.** "Exclude Alice's own Claude conversation where it would duplicate
existing context." A developer may run two Claude sessions on one machine, in the
same room. Excluding by *peer* would blind each session to the other; excluding by
*session* is correct, and is what we implemented. The text supports either.

**Recommend.** Say `claudeSessionId`.

---

## C. Gaps the discoveries opened

### C1 — Nothing in the specification acknowledges that it depends on undocumented behavior

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

**Medium.** Phase 0a (which I added) is an investigation. §21 "Context Window
Management" governs injection limits and never mentions that Claude Code compacts
sessions on its own. So the *findings* — session ID survives, the transcript is
append-only, the summary record carries no `promptSource`, compaction never fires
mid-turn — live only in a findings document, while §21 reads as though injection
limits were the whole of context management.

**Recommend.** Fold the durable results into §21, and keep Phase 0a as the
historical investigation.

### C3 — Automatic compaction is unresolved and unowned

**Low–medium.** Phase 0a could not trigger it: the threshold floor is 100k, and a
378k-token session with a 100k threshold did not compact at a turn boundary. Every
compaction finding therefore describes *manual* compaction only.

No phase owns the question. It will first appear under real context pressure in
Phase 4, by which time cross-Claude context is the thing being debugged.

**Recommend.** Name it explicitly as a Phase 4 risk, or give Phase 0a a deferred
item. Do not leave it only in a findings appendix.

### C4 — "Real time" means two different things, and one of them is slow

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

**Low.** §3.4 says do not replace conversations with summaries; §21 says do not
introduce AI summarization. Claude Code's compaction summarizes injected teammate
context inside the session regardless.

There is no contradiction in practice — the room's stored history stays verbatim
in SQLite, which is what those sections protect — but the text currently reads as
an absolute guarantee that the system cannot make.

**Recommend.** Scope the prohibition to the room and its replication, and note
that what a given Claude retains post-compaction is outside the system's control.

### C6 — `COMPACTION` is a "later" event type that is already needed

**Low.** §7 lists `COMPACTION` under types to design for. D-007 commits to
recording it in Phase 1, as the observability backstop for D-006's decision not
to rewind the watermark. Minor drift between the specification and the decision.

### C7 — §25 asks for signable peer identity; identity is currently a random string

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
