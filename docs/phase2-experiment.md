# Two-Peer Experiment: does synchronizing turns approximate a shared conversation?

**Verdict: yes, and more usefully than expected.** Two independent Claude Code
sessions, synchronizing complete turns between separate daemons, produced an
exchange in which the second session resolved a referent from the first, disagreed
with it on the merits, and advanced the investigation past where either had
reached alone.

This is the question §30 exists to ask. It is now answered affirmatively for two
peers on one machine.

| | |
|---|---|
| Date | 2026-09-16 |
| Peers | 2 daemons, ports 4801/4802, separate state directories |
| Transport | HTTP poll, 1s interval, localhost |
| Room | one, shared by name |

---

## What was run

David's session was given a file with two plausible defects — a retry loop with no
time budget, and an OkHttp connection pool holding idle sockets for 300s — and
asked for *one* change. It chose the retry budget, proposing a deadline-aware loop
driven by the Lambda `Context`.

That turn synchronized to Alice's daemon. Alice's session was then given a prompt
whose referent existed nowhere in its own context:

> "I don't think his fix addresses the root cause. What did he propose, and what is
> he missing?"

It answered:

> **What he proposed** (from the other session): thread the Lambda `Context` into
> `verify` and make the retry loop deadline-aware […]
>
> **You're right that it isn't the root cause.** That change controls the *blast
> radius* […] The request still fails; you just get a nicer failure, and your
> success rate doesn't move.
>
> **What he's missing — line 3, the connection pool** […] Between invocations
> Lambda freezes the execution environment, so OkHttp's pool-cleanup thread is
> frozen with it […] NAT Gateway drops idle flows at 350s and sends no RST/FIN;
> ALB defaults to 60s. Your 300s keep-alive is sitting right under the NAT limit
> and way over the ALB one.

Three things happened there, and only the first was being tested:

1. The referent resolved across a peer boundary.
2. The second session **disagreed on the merits** rather than deferring — §20's
   "context, not instructions" holding under real conditions rather than a probe.
3. The two sessions found **different real defects**. Together they covered the
   file; separately, neither did.

The third is §30's second question — whether parallel independent sessions beat one
shared conversation — and it is the first evidence for it.

## Mechanism

Anti-entropy reduced to what the experiment needed: a peer posts the highest
contiguous sequence it holds per peer, and receives what it lacks. Pull, not push,
so an absent peer recovers by asking rather than by anyone remembering.

Deliberately omitted, none of it affecting what was measured: invitations and guest
lists, signatures, real-time push, and any transport beyond localhost HTTP.

## Measurements

| | |
|---|---|
| Peer propagation, daemon to daemon | **570 ms** (1s poll interval) |
| Propagation into a teammate's Claude | not until their next prompt |

These are the two latencies review finding C4 says §16 conflates, and the
experiment makes the distinction concrete rather than theoretical. The first is a
property of the transport and can be driven to near zero with push. The second
cannot: a teammate mid-turn learns nothing until they submit again, and a long
agentic turn runs for minutes. **The wall between them is the turn, not the
network.**

Both peers produced **identical event ordering** — the deterministic comparator
from B1, verified in the only way that counts.

## What the experiment found that design review had not

**Display names collided, and attribution silently collapsed.** Both daemons ran
under one OS user, so both derived `userDisplayName` from `$USER` and asserted
"David". They were genuinely distinct peers — `dawn-crane` and `vivid-lemming` —
but every event in the room read as one person.

Comprehension survived it, which is the uncomfortable part: the prompt said "his
fix", the labels all said "David", and the model reasoned its way to the right
referent anyway. The failure would not have surfaced through use.

§20's injected attribution was anchored on that self-asserted name. It now anchors
on the **derived** peer name and marks the speaker unverified inside the injected
text:

```
<message speaker="David (dawn-crane, unverified)">
```

A later turn confirmed the difference. Asked who had corrected it, the session
distinguished the two peers by derived name, attributed each position correctly,
and noted that the material was "context from other rooms rather than my own
reading."

This is D-021's argument arriving empirically rather than by reasoning: a display
name is a claim, and the system had been repeating it as though it were a fact.

## Limitations

1. **One machine, one OS user.** No network, no NAT, no clock skew. Propagation
   measured here is a lower bound on nothing in particular.
2. **Two peers.** Transitive relay (§13) and three-peer ordering (Phase 6) are
   untouched. Review finding B2 — injection order under interleaved streams — does
   not arise with a single teammate and remains unverified.
3. **Polling, not push.** §16's sub-second target is met here by a 1s poll, which
   is not the same as meeting it.
4. **No admission, no identity.** Peers were named by environment variable. A peer
   is trusted to report its own identity, so nothing here exercises §12, §25, or
   D-025.
5. **One exchange.** A single successful pairing is evidence, not a result. Nothing
   is known about a long session, a busy room, or whether the effect survives
   compaction.

## What this changes

The prototype's central question has an affirmative answer, which reorders what
matters next. Cross-session context is worth building properly, because it demonstrably
works and demonstrably adds something. The remaining specification findings are now
about making it *safe* and *implementable by someone else*, rather than about
whether the idea holds.

The most valuable next evidence is a **three-peer** run: it is where transitive
relay, injection ordering, and forged attribution all become reachable, and where
the two-peer result might fail to generalize.
