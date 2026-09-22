# Decision log

Decisions made while building this, with the alternatives that were rejected and
why. The rejected options matter more than the chosen ones — without them, a
later reader re-proposes something already ruled out, or "simplifies" code whose
awkwardness was load-bearing.

**Revisit when** ties each decision to the automated check that would invalidate
it, where one exists (`cogmer doctor`, registry in `cmd/cogmer/behaviors.go`).
A decision whose trigger fires is not automatically wrong — it is due for review.

Commit bodies carry additional detail; `git log` is the long form of this file.

Format: Context / Decision / Rejected / Revisit when. Add entries at the bottom,
numbered, never renumber. Superseded entries stay, marked, with a pointer forward.

---

## D-001 — Go, not TypeScript/Node or Python

**Date:** 2026-09-16 · **Status:** active

**Context.** Teammates will be on macOS, Linux, and Windows. The system needs a
long-lived daemon, hook executables invoked per prompt, and a local browser UI.

**Decision.** Go, with `modernc.org/sqlite` (pure Go, no cgo).

The deciding fact: `claude` resolves to `~/.local/share/claude/versions/…`, a
native Mach-O binary — **Claude Code no longer ships as an npm package**, so a
teammate can have Claude Code and no Node runtime at all. Go gives one
dependency-free binary per platform; all four targets cross-compile from one Mac.
The hook entry in `settings.json` is then an identical string on every platform
(`cogmer hook prompt`), avoiding Windows backslash-and-space paths inside
JSON string literals.

**Rejected.**
- *TypeScript/Node* — requires Node on every teammate's machine. Also `node:sqlite`
  in Node 22.12 throws without `--experimental-sqlite` (verified directly), so it
  would mean a `better-sqlite3` native dependency and its prebuild matrix. Faster
  to iterate; not worth the install burden.
- *Python* — weakest here. Windows environment fragmentation, and the UI is
  hand-written JS regardless.
- *Node spike first, Go daemon after* — genuinely tempting, since Phase 0 never
  leaves one machine. Rejected because the spike's hook-handling code is small and
  becomes the daemon's foundation; the rewrite does not pay for itself.

**Revisit when** the team standardizes on a Node toolchain, or the UI outgrows
plain JS + server-sent events.

---

## D-002 — Stop at Phase 0, and insert Phase 0a before Phase 1

**Date:** 2026-09-16 · **Status:** active

**Context.** Specification §36.10 says to stop after the integration spike and
report. Phase 0 proved capture and injection, but never exercised compaction —
`PreCompact` did not fire.

**Decision.** Stop as instructed, then add **Phase 0a** to the specification
before any daemon work. Compaction was the one mechanism that could silently
invalidate both proven directions.

**Rejected.**
- *Proceed to Phase 1 and handle compaction when it appears* — the failure has no
  error path; it surfaces as Claude quietly misunderstanding a teammate reference.
  Diagnosing that with three peers relaying events is far harder than probing it
  on one machine.
- *Renumber the phases* — "0a" avoids churn in a specification already referenced
  by section number throughout the code.

---

## D-003 — Reassemble turns by unioning two incomplete sources

**Date:** 2026-09-16 · **Status:** active

**Context.** Neither source Claude Code exposes contains a complete assistant
turn, and they fail in opposite directions: `Stop.last_assistant_message` holds
only the final text block, while the transcript at Stop time is missing exactly
that block (Stop fires before it is flushed).

**Decision.** Union both in `ReassembleLastTurn`. Verified by exact string match
against a real 2,582-char response.

**Rejected.**
- *Transcript only* — captured a 110-char preamble of a 2,804-char answer.
- *`last_assistant_message` only* — silently drops anything said before a tool
  call, which is most substantive turns.
- *Sleep-and-retry until the transcript settles* — adds latency to every turn and
  is still a race, just a longer one.

**Revisit when** B04 or B05 fires.

---

## D-004 — Segment turns positionally, not by identifier

**Date:** 2026-09-16 · **Status:** active

**Context.** Attributing assistant records to the turn that produced them.

**Decision.** Take every assistant record following the last user record bearing a
`promptSource`.

**Rejected.**
- *Correlate on `promptId`* — assistant records do not carry one. User records do,
  including tool-result records.
- *Walk the `parentUuid` chain* — observed an assistant record whose `parentUuid`
  matched no preceding `uuid` in the same file. The chain has gaps.

**Revisit when** B09 fires — assistant records gaining a `promptId` would let this
become exact correlation, which is strictly better.

---

## D-005 — `mergeTail` tolerates a widened `last_assistant_message`

**Date:** 2026-09-16 · **Status:** active

**Context.** D-003 appends `last_assistant_message` to the transcript-derived
text, which is correct only while it contains just the final block.

**Decision.** Detect the superset case rather than assume it cannot happen. If
Claude Code widens the field to the whole turn, use it alone.

**Rejected.** *Plain append* — an upstream **bugfix** would then duplicate every
pre-tool text block and silently corrupt the room. A dependency getting better
should not break us.

**Revisit when** B04 fires — at which point the transcript read may be removable.

---

## D-006 — No watermark rewind and no re-injection floor after compaction

**Date:** 2026-09-16 · **Status:** active

**Context.** The feared failure: compaction discards injected teammate turns while
the delivery watermark still records them as incorporated, so they are never
re-injected and the referent is lost with no error.

**Decision.** Change nothing. Phase 0a tested it twice, including the realistic
case where the teammate context was *incidental* — injected, then buried under
four unrelated turns so the summarizer had no reason to keep it. It kept it
anyway, with attribution. Both tests ran with `injected=0`, so the answers came
from surviving context rather than re-injection.

**Rejected.**
- *Rewind the watermark on `PreCompact`* — duplicate injection on every compaction
  for no benefit.
- *Re-inject a bounded floor of recent turns after any boundary* — same cost, same
  absence of benefit.

**Revisit when** B19 fires. Survival is summarizer judgment, not a format
guarantee; it could change with a model, a longer conversation, or repeated
compaction cycles (only one was tested).

---

## D-007 — Record `COMPACTION` events as observability, not remediation

**Date:** 2026-09-16 · **Status:** deferred to Phase 1

**Context.** D-006 rests on behavior that could change without any signal.

**Decision.** When the daemon is built, record a `COMPACTION` event — a type the
event model already reserves. Not a fix; it makes compaction visible in room
history so the correlation exists *before* anyone needs it.

**Rejected.** *Add nothing* — leaves no way to reconstruct what happened if D-006
ever becomes wrong.

---

## D-008 — Key behavior verification on Claude Code version, not on the room

**Date:** 2026-09-16 · **Status:** active

**Context.** Behavior checks cost a real turn of the user's subscription. They
should run when something might have changed.

**Decision.** Check at room formation; cache on `claude --version` in
`~/.cogmer/verified.json`. Forming a tenth room on a verified version is
free. `COGMER_PREFLIGHT=off` opts out.

**Rejected.**
- *Per room* — re-proves the same thing and burns quota; the version is what
  actually varies.
- *Time-based expiry* — a TTL is a proxy for "did the binary change," and the
  binary's version answers that directly.
- *Manual only* — the failures are silent; nobody runs a check for a problem they
  cannot see.

---

## D-009 — A failed behavior check never blocks the room

**Date:** 2026-09-16 · **Status:** active

**Context.** Specification §3.1: Claude Code must keep working when collaboration
is unavailable.

**Decision.** Report which assumption changed and what it breaks; carry on.

**Rejected.** *Refuse to form the room* — turns a degraded feature into a broken
session, which is precisely the failure mode §3.1 forbids.

---

## D-010 — The behavior registry is the source of truth; its documentation is generated

**Date:** 2026-09-16 · **Status:** active

**Decision.** `cmd/cogmer/behaviors.go` holds behaviors and checks together.
`docs/relied-on-behaviors.md` is generated (`cogmer behaviors --markdown`).

**Rejected.** *A hand-written document beside the checks* — it drifts, and a stale
list of safety properties is worse than none because it is believed.

Corollary enforced by test: every behavior needs a negative test proving it fails
on the regression it claims to catch. A check that cannot fail reads as protection
while providing none. This caught a real error — the first `--deep` run reported
B05/B12 failing, which was a bug in the probe's own evidence handling, not a
behavior change.

---

## D-011 — Hooks fail open: exit 0, empty stdout

**Date:** 2026-09-16 · **Status:** active

**Context.** Hooks run in the path of every prompt.

**Decision.** Every failure path exits 0 writing nothing. A dead daemon degrades
to "no collaboration," never a broken session. Measured 17 ms when the daemon is
down; connection-refused returns immediately rather than burning the timeout.

**Rejected.**
- *Non-zero exit on failure* — would surface an error on every prompt.
- *Explaining the failure on stdout* — `UserPromptSubmit` stdout is injected into
  the turn, so diagnostics would land in the user's conversation. Diagnostics go
  to stderr.

---

## D-012 — The preflight probe uses the `cogmer` binary as its own hook

**Date:** 2026-09-16 · **Status:** active

**Decision.** `cogmer probe-hook <name> <dir>`, registered as the hook
command, rather than writing a shell script to a temp directory.

**Rejected.** *A generated `.sh`* — it would not run on Windows, making the
verification mechanism itself the least portable part of a project whose entire
stack choice (D-001) was driven by Windows support.

---

## D-013 — The probe points ambient hooks at a closed port

**Date:** 2026-09-16 · **Status:** active

**Context.** Once the real hooks are registered in the user's settings, they also
fire inside the probe's own Claude session. A preflight could publish its
synthetic sentinel conversation into a live room.

**Decision.** Run the probe with `COGMER_ADDR` pointed at a closed port, so
any ambient `cogmer` hook fails open (D-011) and records nothing.

**Rejected.** *`--setting-sources ""` to load no ambient settings* — plausible, but
untested, and a flag-parsing surprise would break the probe entirely. The env var
reuses a guarantee already verified by B11.

---

## D-014 — Derive delivery state from transcript evidence, not from recorded intent

**Date:** 2026-09-16 · **Status:** active · **Supersedes** the advance-at-injection
behavior reviewed as A1 in `spec-review.md`

**Context.** §19 says to "update the session's incorporated-event state" without
saying when, or what counts as incorporated. Advancing it at injection commits
delivery before the hook has the daemon's response: with a 3s hook timeout, an
expired or lost reply meant the daemon recorded delivery while Claude saw
nothing. At-most-once on a channel that needs at-least-once.

**Decision.** Claude Code records a hook's stdout in the transcript as a
`hook_success` attachment. The daemon already reads that transcript at `Stop` for
turn reassembly, so delivery is confirmed by observing the injected block there —
matched on a sha256 of the exact emitted text — rather than assumed because a
hook ran. Delivery became a set rather than a watermark, since a lost injection
leaves a hole a contiguous watermark cannot represent.

Confirmation is self-healing: attachments accumulate across turns and survive
compaction, so an injection missed at its own `Stop` is confirmed at a later one.

**Rejected.**
- *Advance at injection* — the bug above.
- *Provisional at injection, committed at `Stop`* — fixes the lost response, but
  still records intent: `Stop` proves a turn ended, not that context arrived.
  Retained as the degraded path (see below) because it depends only on
  `prompt_id` correlation, already verified by B01/B03.
- *Embedding event IDs in the injected block* — would make evidence directly
  addressable, but at ~34 characters per event it puts real noise in every
  teammate's context window. Hashing the block gets the same mapping for free.

**Consequence: absence of evidence is not evidence of breakage.** The first
implementation fell back to committing on trust whenever no attachment was found
— which is indistinguishable from the injection never arriving, so it silently
lost context in exactly the case this decision exists to fix. Testing the failure
path caught it. The fallback is now gated on B20 being recorded as *failing* for
the installed version; otherwise events stay pending and are re-offered. That
failure mode is noisy and self-announcing rather than silent and lossy, which is
the right way round.

**Revisit when** B20 fires, or if re-offering proves disruptive enough in practice
that duplicate context costs more than the loss it prevents.

---

## D-015 — Rooms are scoped to sessions, not to projects

**Date:** 2026-09-16 · **Status:** active · **Supersedes** the project-scoped room
model throughout the specification; **dissolves** A2 in `spec-review.md`

**Context.** A2 found §5, §22 and §28 mutually inconsistent about how a hook call
resolves to a room, and the proposed fix was a `cwd` → project-config → room
lookup. That fix was answering the wrong question: it assumed rooms are durable
things a project owns.

**Decision.** A room is a set of linked Claude Code sessions, identified by a
generated id, entered by invitation, and closed when its last member session
ends. Nothing about a room is derived from a directory, repository, or project.

The strongest argument is one the original specification did not make: §21's
`CONTEXT_CATCHUP_REQUIRED` exists only because rooms outlive sessions. With a
room that persists for months and a delivery watermark keyed per session, a fresh
session on Monday faces weeks of unseen events, and the specification's answer was
to inject a truncated tail and admit the rest was dropped. That is a designed-in
truncation which only a durable-room model requires. Session-scoped rooms remove
the condition instead of coping with it, and a late joiner can be given the room
from its beginning.

Two facts made this cheap. Our implementation was already session-centric —
delivery keyed on `claudeSessionId`, events carrying it, with only the room *name*
being project-shaped. And Phase 0a verified that `claudeSessionId` survives both
`--resume` and compaction, so session-scoped membership is stable across laptop
sleep and session resumption rather than fragile.

**Split that makes it work:** membership is ephemeral, the record is not. A closed
room's event log is archived — readable and searchable, never rejoined, never
synchronized, never injected. This keeps "preserve the actual conversation" at
no cost while letting membership end cleanly.

**Rejected.**
- *Project-scoped rooms with `cwd` resolution* (the A2 proposal) — needs a config
  file, a walk-up rule, a default-off guard, and room-name validation against path
  traversal, all to infer something that an invitation states outright.
- *Standing team rooms* — the Slack-shaped model. Appealing, but it is precisely
  what produces the catch-up problem and A3's exposure, and §30's question is
  about real-time shared conversation, which it does not need.
- *Discarding the log when a room closes* — would satisfy "no continuity" more
  literally while losing conversation the specification requires preserving.

**Cost, stated plainly.** §10 and §26 lose most of their purpose: anti-entropy no
longer reconciles "several hours offline" or "working on an airplane," only
interruptions inside a live pairing. §35's durable team memory must be built over
archives rather than live rooms. Both sections were amended rather than deleted,
because the machinery is still correct — it simply has far less to do, which
argues for simplifying Phase 2.

**Settled by D-016:** a session holds membership in at most one room at a time.

**Revisit when** the experiment in §30 suggests asynchronous catch-up is more
valuable than bounded context — that is the trade this decision makes.

---

## D-016 — One room per session, and presence is not membership

**Date:** 2026-09-16 · **Status:** active · **Closes** the open question in D-015

**Context.** D-015 left open whether a session could hold membership in two rooms
at once, and stating the constraint immediately raised the lifecycle questions
behind it: can a session leave, rejoin, or move to a different room?

**Decision — one room at a time.** This follows from the event model rather than
being a policy preference. Every captured event belongs to exactly one room, and
a session in two rooms gives no basis for choosing which. Injection fails the same
way in reverse: a session receiving turns from two unrelated conversations cannot
separate them, and neither can the person reading the result.

**Leaving** is always explicit. **Rejoining** the same room is allowed while it
remains live, and the per-event delivery set from D-014 makes it correct for free
— a returning session receives what it missed and nothing else.

**Moving to a different room is refused once teammate context has been injected.**
This is the sharp edge. Injected context cannot be withdrawn: another person's
conversation is in that session's context window for the rest of its life, and
anything the session subsequently produces may be shaped by it. Admitting the
session to a second room would publish the first room's conversation into the
second through the model's own output — invisibly, irreversibly, and without
either room's members knowing. The system cannot detect that leak once it has
happened; it can only decline to create the conditions.

A session that has received *no* injected context may move freely, which covers
joining the wrong room and correcting it. A session's own prompts and responses
impose no restriction: that content originated with the person, so carrying it
forward is their own disclosure, not a leak of someone else's.

**Presence is not membership.** The first draft of this said membership ends when
the Claude Code session ends — which is wrong, because sessions do not end. The
process exits, but the session persists and resumes under the same ID (verified in
Phase 0a, checked by B14). Under that draft, two people closing their
terminals for lunch would have archived the room, and neither could rejoin it nor
join another.

So membership is durable and ends only by explicit departure or room closure,
while presence is transient and lapses whenever a process exits. An exiting
session is **absent**, not gone; resuming restores presence without rejoining.
A session-end signal is a presence signal, and the system must not depend on
receiving one at all — a killed process sends nothing. §35's existing presence
display (`offline — last seen 14 min ago`) already assumed this distinction; the
specification simply had not stated it.

**Consequence for closing.** A room closes on explicit departure by all members, or
on prolonged dormancy with a deliberately generous threshold. Closing early is the
more damaging error: a closed room can never be rejoined, and a member that
received injected context can join no other. The asymmetry should be resolved in
favour of keeping rooms open.

**Rejected.**
- *Allowing a session into a second room with a warning* — the leak is silent and
  affects people who did not see the warning.
- *Binding a session to one room for its entire life, with no exception* — simpler,
  but forces a full session restart for a mistyped invitation.
- *Treating process exit as departure* — the draft above; breaks resumption, which
  is ordinary rather than exceptional.

**Revisit when** a mechanism exists to scope or evict injected context within a
live session. The refusal to move rooms is a consequence of that being impossible,
not a value judgement about people.

---

## D-017 — A room carries two identifiers: a UUID for synchronization, a generated name for people

**Date:** 2026-09-16 · **Status:** active

**Context.** D-015 gave rooms a generated id plus a human-chosen label. The label
was the weak part.

**Decision.** Every room has a `roomId` — a UUID, globally unique, never reused or
changed, carried on every event, and the key for all replication, deduplication
and storage — and a `roomName`, generated at the same moment from two curated word
lists (weather or sky, plus landscape): `misty-canyon`, `thunder-ridge`.

**The name is generated rather than chosen, and that is the point.** A name a
person picks will be the name of a project, a client, or a ticket. Rooms named
after projects become rooms scoped to projects by convention — the exact model
D-015 abandoned. Generating the name resists that structurally instead of relying
on anyone's discipline.

Weather-plus-landscape was chosen for four properties, all of which matter because
an invitation may be read aloud over a call: speakable, unambiguous when heard,
short enough to type without copying, and drawn from a narrow neutral domain so
that no random pairing produces something offensive. The domain being narrow is a
safety property, not a stylistic one. It also happens to name a place, which is
what a room is.

**The name is explicitly non-authoritative.** Nothing synchronizes, routes,
deduplicates, or stores by name; databases are filenamed by `roomId`. Two
unrelated rooms may share a name, and an implementation that keys on one will
eventually merge two unrelated conversations. This had to be stated, because a
pleasant identifier invites exactly that misuse.

**Uniqueness is scoped to a peer, not global.** Global uniqueness is impossible —
rooms are created independently on machines that never coordinate. But a name is
only ever resolved against the peer named in an invitation, so a peer need only
keep its own live room names distinct, regenerating on collision. That makes a
few hundred words per list sufficient.

**Names are immutable** for the life of the room: renaming would invalidate
outstanding invitations and make an archived room harder to recognize.

**Rejected.**
- *A single identifier* — a UUID cannot be read over a call; a name cannot be a
  synchronization key. The two jobs have incompatible requirements.
- *Person-chosen names* — reintroduces project scoping by convention.
- *Globally unique names* — unachievable without coordination, and unnecessary once
  resolution is scoped to a peer.
- *Including the name on every event* — it is room metadata, and putting a mutable
  display string inside immutable events invites drift.

**Revisit when** a name is needed outside the scope of a single peer, such as a
directory of rooms across an organization. Per-peer uniqueness would no longer be
sufficient.

---

## D-018 — Identity and reachability are separate; invitations carry an endpoint and a secret

**Date:** 2026-09-16 · **Status:** active

**Context.** The invitation format from D-017 read
`misty-canyon@davids-macbook:4783`, and the specification had used
`alice-machine:4783` since §4. Neither said how a machine name resolves.

Checked on the development machine: `os.Hostname()` returns `macbookpro.lan`,
`scutil --get LocalHostName` returns `pushover`, and the resolvable name maps to
`192.168.86.31` — a LAN address no remote teammate can reach. Tailscale is not
installed, so MagicDNS does not exist there at all. Three names for one machine,
none of them `davids-macbook`, and the one that resolves is unreachable from
outside.

**Decision.** Separate the two concerns the invitation had merged.

- `machineId` is an **identity** label for attribution and display. Nothing routes
  by it.
- An **endpoint** is reachability, opaque to the collaboration protocol, supplied
  by whichever transport is in use.

A daemon must **discover** its endpoint rather than derive one from its hostname —
under Tailscale by asking Tailscale for the MagicDNS name or tailnet address, both
of which are Tailscale's properties and not the host's. A daemon that cannot
determine a reachable address should say so instead of issuing an unusable
invitation.

**The endpoint is a bootstrap hint,** correct only once. Having joined, a peer
learns the membership and how to reach it, so the inviting peer's address stops
mattering — the same principle as an inviting peer not being authoritative. An
invitation may carry several endpoints, since a teammate on the same network and
one across the internet do not reach the same address, and endpoints go stale when
machines move.

**A room name is not a credential.** This follows directly from D-017: the name is
drawn from a deliberately small, speakable space, which is ample for avoiding
confusion and useless against guessing. Authorization is a separate single-use,
expiring secret issued with the invitation. The invitation is consequently a
bearer credential, and §25 now says so.

**Rejected.**
- *Resolving `machineId` as a hostname* — the original implied design. Disproven on
  the first machine tested.
- *Requiring Tailscale MagicDNS* — it can be disabled, leaving only the address,
  and §4 already insists the protocol not depend on Tailscale specifically.
- *Relying on the room name for authorization* — makes a 16,000-combination guess
  sufficient to enter a conversation.

**Revisit when** a transport without stable addressable endpoints is added, such as
WebRTC through a signalling server, where the invitation carries a session
descriptor rather than an address.

---

## D-019 — No network provider is required; local discovery is the zero-configuration path

**Date:** 2026-09-16 · **Status:** active as to the **requirement** — no provider is
part of room identity, membership or replication, and none may be a prerequisite.
Its **build order** is superseded by D-063: the first real pair works from home and
will never share a network, so local discovery is not the first transport to build.

**Context.** The specification was written for an internal experiment, where
assuming Tailscale was free. For a public release it is not: "install this" and
"install this, create a Tailscale account, put every person in a configured
tailnet" attract very different numbers of people, and Tailscale's free tier is
framed for personal rather than commercial use, so an evaluating team may read a
free tool as requiring a paid service.

**Decision.** State as a requirement that no network provider is part of room
identity, membership, or replication, and that the system must function with no
VPN at all when peers can already reach one another. Two people on the same
network is the simplest case and must be the easiest: no account, no external
service, no configuration.

Transports are attempted in order — same network, then a private network provider,
then internet peer-to-peer, then relay — and which one connected is an
implementation detail that must not surface in room identity or event data.
Implementation order is Local, then Tailscale, then WebRTC.

The architectural seam already existed: §4 said the protocol must not depend on
Tailscale and §33 already had `PeerSyncTransport`. What changed is the **priority**.
Local discovery moved from "later possibility" to the first transport built, and
the independence became a stated requirement rather than an aspiration.

**That priority was wrong, and D-063 corrects it.** It rested on "two people on
the same network is the simplest case" — true, and irrelevant, because the first
pair who need this work from home and will never be on one. The *requirement* above
is unaffected: no provider may be required, and the same-network case must still
cost nothing when it arises. What changed is which case gets built first.

**Where this conflicted with earlier decisions, and how it was resolved.** The
appealing version of zero-configuration joining is `cogmer join misty-canyon`
— find the room by name on the network and enter it. That cannot be adopted as
stated. D-017 made room names short, speakable, and therefore guessable, and D-018
made authorization a separate secret precisely because of that.

On a shared network — an office, a conference, a cafe — any listener could
enumerate advertised room names, and could guess them without listening. A room
holds source code, customer information, and whatever has been pasted into a
prompt.

So discovery locates a room; it never admits anyone to one. Local discovery
replaces the *endpoint* in an invitation, which D-018 had already reduced to a
bootstrap hint, while the secret remains:

```
cogmer join misty-canyon#k7qm-2xpr-9vlt
```

Still short enough to say across a desk. The secret also disambiguates, which
local discovery needs anyway: names are unique only among the rooms one peer
hosts, so a broadcast search may surface two unrelated rooms sharing a name.

**Rejected.**
- *Tailscale as the architectural foundation* — couples adoption to an account and
  a second daemon for the simplest case.
- *Joining by name alone on a trusted network* — "trusted network" is doing
  unearned work; office and conference networks are neither small nor trusted.
- *Treating a signalling service as a small future detail* — signalling is small
  and never sees a conversation, but relays carry the traffic and cost money.
  Recorded in §33 so it is chosen deliberately.

**Note for implementation.** `tsnet` lets a Go program become a tailnet node
directly, which would satisfy D-018's requirement to obtain an address from
Tailscale rather than from the hostname without shelling out to its CLI. That
makes it attractive *inside* `TailscaleTransport`, and unacceptable anywhere else.

Tailscale is also not installed on the development machine, so local discovery is
now the shorter path to a working two-peer test as well as the better public
default.

**Revisit when** a transport is added whose peers have no addressable endpoint,
where `discover` cannot be expressed as returning candidates.

---

## D-020 — A guest list replaces the join secret only once peer identity is cryptographic

**Date:** 2026-09-16 · **Status:** active (partly deferred)

**Context.** If a room keeps a list of invited peers, is the join secret from D-018
still needed?

**Finding: not with identity as it stands.** A guest list is an *authorization*
mechanism and presupposes *authentication*. A `peerId` today is ten random bytes
generated locally, asserted by the peer that sends it, and verified by nothing.
A list of unverifiable names is a convenience, not a control — any peer can claim
any identifier.

It would also be worse than the secret it replaced. A join code is used once and
discarded; a `peerId` appears in every event the peer originates, so it is far
more discoverable than the credential it would be standing in for.

**Decision.** Keep the secret for now. State in the specification that peer
identity must become cryptographic — identifiers derived from a public key,
possession proved on connection, events signed at origin — and that until it does,
the guest list, the relay rule, and attribution are conventions rather than
controls.

**Once identity is cryptographic, the guest list is the better mechanism** and the
specification says to prefer it. Admission becomes proof of possession rather than
presentation of a token: nothing transmitted can be replayed by an interceptor,
nothing expires, and admission can be withdrawn.

**They compose rather than compete.** A guest list does not remove the first
exchange — two peers who have never met must still establish keys over a channel
they trust, exactly as a secret must be sent over one. It removes every *subsequent*
exchange, because a verified key is durable where a secret is spent. So: a
single-use secret admits a peer that is not yet known, being admitted is what makes
it known, and between known peers no secret is required. This is the model SSH uses
for host keys, and the reasoning is the same.

**The stronger argument for doing the work is unrelated to admission.** §13 requires
that a relaying peer never rewrite `originPeerId`, `peerSequence`, or `eventId` —
but nothing enforces it. An event arriving from Alice claiming to originate with
David is indistinguishable from one Alice composed herself, and transitive relay is
a stated resilience feature rather than an edge case. Signing at origin is what
makes relay verifiable instead of merely well-behaved. This closes review item C7.

A guest list also makes broadcast discovery (D-019) safe to enumerate: if admission
requires a key, a listener learning every advertised room name gains nothing.

**Rejected.**
- *Guest list instead of the secret, now* — authorization without authentication.
- *Guest list keyed on `userId` or `machineId`* — the same defect with friendlier
  names.
- *Deferring identity until after Phase 2* — Phase 6 tests transitive relay, which
  is precisely what unsigned events cannot make safe.

**Revisit when** implementing Phase 2. Peer identity format is entrenched by the
first event two peers exchange, so the keypair decision wants making before peers
exist, not after.

---

## D-021 — Peer names are word pairs derived from the identity, never chosen

**Date:** 2026-09-16 · **Status:** active (derivation implemented; verification deferred)

**Context.** Peer identifiers should be readable, as room names are (D-017):
`quiet-otter` rather than `peer-852ffe6408339bcada91`.

**Decision.** A peer carries two identifiers, mirroring a room: a `peerId` that is
verified, signed with and keyed on, and a `peerName` — an adjective and an animal —
for people to read and say. Implemented as `PeerName()`, which hashes the
identifier rather than slicing it, so the derivation works unchanged when `peerId`
becomes a public-key fingerprint (D-020).

**The name is derived, never chosen, and that distinction matters more for peers
than for rooms.** A room name colliding is confusing. A peer name colliding is
impersonation: §20 injects `speaker="Alice"` directly into a teammate's Claude,
so a display name a peer can select is a way to put words in a colleague's mouth,
in a form the model reads exactly as the colleague saying them. A name a peer
chooses is a claim about who it is, and a display name must not be a claim.

**Deriving it is necessary and not sufficient.** 8,280 combinations is trivially
grindable: generate identities until the derived name matches a chosen target. No
name short enough to say aloud can resist that, and a larger word list does not
change the conclusion — it changes the cost from seconds to minutes.

The protection therefore cannot come from the name. It comes from D-020's known
peers: a name is a mnemonic for an identity already verified, never an
introduction to a stranger. The specification now requires that a name be
displayed as itself only for a verified peer, and that unverified speakers be
marked as such **inside the injected text**, not merely in an interface — the
model reasons about attribution, so marking it only in a UI protects the wrong
reader.

**Word lists name colleagues, which constrains them more than the room lists.**
Weather and landscape name places; adjectives and animals name people. An
adjective that would be unkind applied to a person is unacceptable however
harmless it is applied to an animal, and so is any animal used as an insult.
Combinations are generated and nobody approves them individually, so the
constraint lives in the lists. A test asserts it, which is a floor and not a
substitute for reviewing additions.

**Rejected.**
- *Self-chosen display names* — the impersonation vector above.
- *Storing the name alongside the identity* — a persisted name can drift from the
  identity it represents. It is derived on load and marked `json:"-"`.
- *A larger word list as the answer to spoofing* — raises grinding cost by minutes
  and leaves the property unchanged.
- *Indexing the word lists by the first and last bytes of the identifier* rather
  than by a hash of it. The appeal is auditability: a name visibly derived from
  bytes you can read is one a person could check. Measured, it fails on the point
  that matters — two identifiers differing only in a middle byte derive the **same**
  sliced name and different hashed names, so slicing sees sixteen bits and is blind
  to everything between the ends. Hashing covers the whole identity, so any
  difference anywhere changes the name. Two lesser faults were fixable and the
  third was not: the identifier is currently `"peer-" + hex`, making the first two
  characters `pe` for every peer (one distinct value across 2,000 generated ids),
  and 256 byte values into a 90-word list leaves some names roughly 50% more likely
  than others.

  The instinct is sound at a different scale. Auditable word rendering is how key
  fingerprints are compared by people, and §25 now requires that such a comparison
  render the *whole* key. Two words carry thirteen bits; offering that as a
  fingerprint check would claim an assurance it cannot provide. When D-020's
  cryptographic identity lands, a full-fingerprint word sequence belongs alongside
  it — as a separate thing from the peer name, not as a redefinition of it.

**On collisions.** 8,280 names is comfortable for a pairing and not for an
organization: roughly 0.5% chance of a shared name among 10 peers, 3.6% among 25,
and 45% among 100. Expanding both lists to 256 entries would give 65,536 and push
that out by an order of magnitude. Held off because curating 512 words against the
naming-people constraint is the work, not the arithmetic.

**Revisit when** `peerId` becomes a key fingerprint. The derivation needs no
change, but the verified/unverified display rule becomes enforceable rather than
advisory, and that is the point at which names become trustworthy at all.

---

## D-022 — A room begins when someone is invited into it, and never earlier

**Date:** 2026-09-16 · **Status:** active

**Context.** The specification said a room is "created by one peer" and that a late
joiner "receives the room from its beginning," in three places, without ever
saying when the beginning is. Nothing said what happens before a room exists.

**Decision.** The `roomId` and `roomName` are generated at the moment a person
invites someone. Creating a room and issuing the first invitation are the same
act. The creating session becomes the first member and the room's history starts
there.

Before that, a session is an ordinary Claude Code session: nothing captured,
nothing injected, nothing shared. Collaboration is something a person starts,
not a state they are in.

**The alternative is a silent disclosure.** The natural implementation — create a
room when a session starts, so capture is always on — means a person who works
alone for two hours and then invites a colleague hands over all two hours. The
invitation looks like saying hello and behaves like publishing a transcript.
Nothing in the interface would suggest otherwise, and the disclosure is
irreversible.

So the specification now states that *"from its beginning" means the beginning of
the room, never of a session that belongs to it* — the ambiguity that made the
wrong reading available.

**Acknowledged cost.** A person usually wants to invite someone *because* of
what just happened, and that conversation is precisely what the new room does not
contain. The prototype accepts this; the workaround is the one people already use,
which is to explain. Deliberately contributing selected earlier turns is a
reasonable later feature. Contributing them by default is not, and the difference
is consent.

**Rejected.**
- *Create a room at session start* — the disclosure above.
- *Create a room lazily at the first captured event* — same outcome, reached less
  visibly.
- *Offer to include prior turns when inviting* — better than defaulting, but it is
  a prompt to share under time pressure, which is a poor moment to ask. Left for
  a later design with the turns shown before they are sent.

**Revisit when** selective contribution of earlier turns is designed, since that is
the feature this decision defers rather than forecloses.

---

## D-023 — A peer identifier must be safe to know; identities are created once and exchanged on joining

**Date:** 2026-09-16 · **Status:** active (specified; not implemented)

**Context.** Two questions: when does a peer identity come into being, and how does
a host learn a guest's identifier before inviting them? Answering the second
surfaced that the current identifier is hazardous to share.

**When.** An identity is created once, on a machine's first use, and persists. Not
per room, per session, or per invitation — it outlives all of them. This is the
inverse of a room (D-022), which begins at a known moment for a known purpose and
is archived when that purpose ends. An identity exists before there is anything to
join, which is what lets a peer be recognized later rather than met afresh.

An identity belongs to a **machine**, not a person: a person with a laptop and a
desktop is two peers and appears twice wherever peers are listed. Deliberate — a
key that never leaves the machine that made it cannot be lost from one machine by
losing another — but it should be visible rather than surprising.

**How a guest's identifier is obtained: it is not, in advance.** There is no
directory, and requiring one before two people could work together would defeat the
point of an invitation. Identities are exchanged on joining. The invitation admits
a peer that is not yet known; joining is where each side learns and records the
other.

**The identifier must be safe to know.** Asking how to obtain a guest's identifier
exposed that knowing one is currently a capability: it is a random value that
nothing verifies, so knowing it is sufficient to claim it and publish events
attributed to its owner. It is a shared secret in the costume of an identifier —
and it is broadcast by design, appearing in every event, every interface, and every
exchange between peers.

The remedy is not to keep identifiers private, which is impossible for something
the system exists to spread. It is to make holding one worthless: **the identifier
should be the public key, or a fingerprint of it.** A public key can be printed,
logged, listed, and read aloud, because possession proves nothing; only the private
key produces a signature.

**The order matters, and is easy to get wrong.** Changing the identifier's format
does not by itself help. While nothing verifies signatures, a peer can still assert
someone else's identifier and be believed. The identifier becomes safe to know at
the moment signatures are *checked*, not at the moment keys are introduced. So
generating keypairs now, without verification, would produce something that looks
like the fix and delivers none of it — which is why nothing was implemented here.

**Self-certifying is not self-authenticating.** An identifier that is a public key
proves possession of that key. It does not prove which person holds it. Joining
establishes the first claim only; the second rests on trust taken at first contact,
whose weakness is that whoever presents a valid invitation becomes the peer that is
recorded — durably, under a name members will thereafter treat as familiar.
Verifying once, over a channel the invitation did not travel on, closes that, and
once is enough.

**Rejected.**
- *A directory of peer identifiers* — an arrangement required before collaboration
  defeats the invitation.
- *Treating the current identifier as sensitive* — it appears in every event; a
  secret that must be broadcast is not a secret.
- *Generating keypairs now as a first step* — see the ordering note. Half of this
  change provides none of its value while appearing to.

**Revisit when** Phase 2 begins. This and D-020 are the same body of work, and it
must land before peers exchange their first event.

---

## D-024 — Admission is a guest list; the join code is a fallback for strangers

**Date:** 2026-09-16 · **Status:** active · **Corrects the emphasis of** D-018, D-020

**Context.** The intent behind a guest registry was `cogmer join misty-harbor`
with no secret in it. The specification conceded in one sentence that known peers
need no secret, and then made the bearer code primary everywhere else.

**The error was conflating two claims.** That authorization cannot rest on a
guessable name is true. That the credential must therefore travel *in the
invitation* does not follow, and I treated it as though it did.

**Decision.** A guest list is the ordinary path. A host records that a peer it
already knows may enter a room; the guest types only the room name; admission is
proof of possession of a key the host already holds. The name locates, the list
admits. Nothing secret is typed, spoken, or transmitted, and guessing the name
gains nothing.

**The out-of-band step does not disappear — it improves.** A host must hold the
guest's identifier first. But that is a *public* identifier, exchanged once per
person rather than once per meeting, and safe to paste into a chat, mail, print, or
read aloud, because holding it confers nothing.

Compare the two exchanges honestly, since both cost one message:

| | join code | public identifier |
|---|---|---|
| must stay secret in transit | yes | no |
| how often | every first meeting | once per person, ever |
| interception | enrols the wrong peer | reveals nothing |
| verifiable afterwards | no | yes, by fingerprint |

The friction argument for codes does not survive that table. It is the same number
of messages, in the other direction, and the safer one is also the one that stops
recurring. `authorized_keys` is the same trade, and nobody experiences it as
friction.

**The code survives as a fallback, explicitly weaker.** Two people who have never
exchanged identifiers should not need a round trip before pairing. A code should
expire, admit one peer once, and be unnecessary afterwards — the peer it admitted
is known now.

**A consequence worth stating:** preferring the guest list is a security decision,
not only a convenience. An intercepted code enrols an impostor under a name members
will thereafter treat as familiar. An intercepted invitation naming a guest reveals
only that a room exists.

**Unchanged by this.** The ordering from D-023 still governs: none of it is
enforceable until signatures are verified. Until then a guest list is a convention,
and the prototype is unauthenticated whichever path it nominally uses.

**Rejected.**
- *The code as the primary mechanism* — the position this corrects.
- *Removing codes entirely* — a first meeting should not require preparation, and
  refusing that makes the tool harder to try than to adopt.
- *Deriving admission from the room name on a trusted network* — the name is
  guessable by construction (D-017); there is no network where that is safe.

---

## D-025 — The guest list, specified: two scopes, a signed challenge, and approval in the moment

**Date:** 2026-09-16 · **Status:** active · **Completes** D-024

**Context.** D-024 made the guest list the primary admission path and invoked
`authorized_keys` as the precedent. It did not specify the mechanism: where a list
lives, how entries are added or removed, what the proof consists of, or what
happens to a peer that is not on one. Naming an analogy is not specifying a design.

**Two lists, at different scopes.** Knowing someone and admitting them to a
particular conversation are different decisions, and collapsing them makes the
second inexpressible. **Known peers** belongs to a machine, is durable, and outlives
every room — it is why a colleague is recognized later rather than met afresh.
**A room's guests** belongs to the room, names which known peers may enter, and is
archived with it. A person may know six colleagues and admit two to a room
concerning customer data.

Commands for both, because a list that cannot be inspected or corrected is not
administrable: `peers`, `allow`, `forget`; `guests`, `invite`, `revoke`. Forgetting
and revoking differ — revoking withdraws admission to one room, forgetting discards
the identity, so a later meeting is a first meeting again.

**Admission is a fresh signed challenge.** The joiner presents an identifier, the
host finds it among the room's guests and issues an unpredictable challenge, the
joiner signs it. The challenge must be new every time: an exchange that can be
replayed is a bearer credential with extra steps. Nothing secret passes in either
direction, so the exchange needs integrity rather than confidentiality.

**Refusal must be more than silence, and that changes the role of codes.** A peer
not on the list is refused — and the host is *told*, and shown the identifier and
derived name presented. The host may admit it, which makes that peer known and a
guest in one act.

This is how two people who have never met will ordinarily pair: Alice attempts to
join, David sees the request, recognizes the moment, approves. **No identifier is
exchanged beforehand and no token is issued.** It removes the round trip that was
the whole argument for keeping codes when both people are present.

What a host approves is an identifier. The name beside it is a claim made by a
stranger who chose when to make it, and the interface should not let the two be
confused.

The refused peer is told, and shown its own identifier, so its user can say what
the host needs to allow. A request is not a queue: if the host is absent the
request fails rather than waiting, and never grants entry later without attention.

**Codes now cover one case only:** inviting in advance, when the host will not be
present to approve. Where a host is present, approval is better in every respect,
because a person decides rather than a token. *(Superseded by D-026: that one case
does not survive examination either.)*

**Rejected.**
- *One combined list* — cannot express knowing someone without admitting them
  everywhere.
- *Silent refusal* — leaves both sides with no way to proceed, and makes the
  no-prior-exchange case impossible without a code.
- *Queuing requests for an absent host* — an admission that completes without
  attention is the token model wearing a different name.
- *Approving by displayed name* — the name is the stranger\'s claim; the identifier
  is the fact.

---

## D-026 — There is no join token at all

**Date:** 2026-09-16 · **Status:** active · **Supersedes** the residual code path in D-024 and D-025

**Context.** D-025 had narrowed join codes to a single case: inviting in advance,
when the host would not be present to approve a request. That case does not survive
examination.

**Decision.** Remove join tokens from the design. Admission is a guest list entry
proved by possession of a key, or a host's explicit approval of a request. There is
no code, no invitation secret, and nothing a person can hold that would admit them.

**Why the last case collapses.** A host who knows a guest can admit them and then
leave — the guest joins whenever it likes, with no host present and no token
involved. So a token would be needed only by a host that is absent **and** has never
recorded the guest. But such a host must act before that guest can enter under any
scheme, including the token scheme, since somebody has to issue the token. The token
therefore buys nothing that acting once through the guest list would not, while
carrying every property of a bearer credential: secret in transit, uncheckable
afterwards, and enrolling whoever intercepts it under a name members will treat as
familiar.

**What is given up, stated rather than engineered around.** Pairing with someone
entirely unknown requires a host present to approve it. That is the moment a person
should be deciding, so the constraint reads as correct rather than merely tolerable.

**Consequence.** The system now has no credential that can be forwarded, stolen, or
replayed — not as a deprecated path, but absent. §25 says so directly rather than
describing how to handle one safely.

**Rejected.**
- *Keeping codes for the absent-host-unknown-guest case* — the host must act anyway.
- *Keeping codes as an optional convenience* — an avoidable credential that exists is
  a credential that will be used, and its weaknesses do not become optional with it.

---

## D-027 — A sequence conflict is quarantined, not dropped

**Date:** 2026-09-16 · **Status:** active (implemented)

**Context.** B4: a peer that loses its room database restarts its sequence at 1
while other peers hold higher numbers under its identifier, so everything it
publishes afterwards collides. `INSERT OR IGNORE` against
`UNIQUE(peer_id, peer_sequence)` absorbed that as an ordinary duplicate.

**Decision.** Distinguish the three outcomes on receipt — `stored`, `duplicate`,
`conflict` — where a conflict is the same peer and sequence arriving with a
*different* event identifier.

**Quarantine rather than reject.** The rejected event is retained alongside both
identifiers. Rejecting it outright would keep the room consistent, which is the
part that matters, but destroys the only evidence that distinguishes a peer which
lost its state from an event that was forged. Those call for opposite responses,
and by the time anyone investigates, the event is the only thing that can tell them
apart.

**Why not repair it automatically.** Reassigning the incoming event a free sequence
number would preserve it, and §13 forbids rewriting `peerSequence` during relay for
good reason: the pair is an identity, and a receiver that edits it makes its copy
disagree with every other peer's. A conflict is a condition to report, not to
paper over.

**Surfacing matters as much as detecting.** `cogmer conflicts` exists because a
quarantined event is invisible otherwise. The failure being silent was the whole
of B4; detecting it into a table nobody reads would reproduce that.

**Rejected.**
- *Keeping `INSERT OR IGNORE`* — correct for redelivery, silently wrong here, and
  unrecoverable: the sender believes it shared, the receiver never sees it, and
  anti-entropy cannot repair a gap where the sender's highest sequence is below what
  the receiver reports holding.
- *Overwriting the held event* — the incoming event has no better claim, and events
  are immutable.
- *Waiting until Phase 2* — `Insert` is the method peer synchronization will call.
  Fixing it while the code is small costs almost nothing; fixing it once peers are
  exchanging events means diagnosing it first.

**Note.** Locally generated events take their sequence from `MAX()+1`, so a conflict
on a local `Append` means the local store is inconsistent rather than that a peer
misbehaved. It is reported as such.

---

## D-028 — Losing a room database ends that peer's membership; recovery is not attempted

**Date:** 2026-09-16 · **Status:** SUPERSEDED BY D-029 — the cost of ending
membership was assessed wrongly, and the mechanism that avoids it is cheaper than
the sequence epoch this entry rejected.

**Context.** D-027 made a restarted sequence counter detectable. It did not say what
a peer should do when it is the one that lost its state.

**Decision.** Treat it as the end of that peer's membership in that room. The peer
leaves, does not rejoin, and does not resume publishing.

**The blast radius is smaller than the word suggests,** and three things get
conflated here. Membership in that room is lost. The conversation is not — every
other member holds a full replica. The peer's identity is not — it lives in
`identity.json`, outside any room's storage. So the cost is one room, in a system
where a room is bounded by the work that created it (D-015). Under the
project-scoped model this decision replaced, the same event would have cost months.

**An inversion worth knowing.** Losing *identity* is the safe failure: the peer
becomes a new peer with a new sequence space and can collide with nothing. Losing a
*room* while keeping identity is the dangerous one, because that is the peer that
can republish sequence numbers others already hold. Anyone reasoning about backups
will assume the opposite.

**Why recovery was considered and declined.** It is not out of reach. Events are
immutable and replicated, so a peer could refetch the room from any member —
including its own past events — and resume above its highest sequence, using the
anti-entropy exchange that already exists.

Establishing "its highest sequence" is the problem. It must be the highest held by
*any* member, and an offline member may hold a higher one than anything reachable.
Resume below it and the conflict recurs. Resume far above it and the gap is
permanent, because the highest-contiguous rule can never close it — every peer would
believe indefinitely that it was missing events.

A sequence epoch solves this properly: an incarnation number raised on recovery, so
a restarted counter occupies a different space rather than colliding. That is the
standard answer, and it changes both the event model and the synchronization state
exchange. §11 declines a CRDT until testing proves one necessary; the same judgement
applies here, and the case for it is weaker because short-lived replicated rooms have
already made the loss cheap.

**Rejected.**
- *Refetch and resume* — sound until an unreachable member holds a higher sequence.
- *Resume with a safety gap* — trades a detectable conflict for a permanent
  synchronization stall, which is worse because nothing reports it.
- *A sequence epoch now* — correct, and disproportionate until a loss has cost
  something.
- *Rejoining under a fresh identity* — technically safe, but it splits one person
  across two peers in the room's history and in every guest list, to preserve a
  membership that D-015 made cheap to recreate.

**Revisit when** a room is long-lived enough that losing membership in one is
expensive — which would most likely mean the session-scoped model itself was being
reconsidered.

---

## D-029 — Losing a room database does not end membership; the sequence lives with the identity

**Date:** 2026-09-16 · **Status:** active · **Supersedes** D-028

**Context.** D-028 treated a lost room database as the end of that peer's
membership, on the grounds that a session-scoped room is cheap to lose. Two things
were wrong with that assessment.

**The database is not where the value is.** A person's own turns, and the
teammate turns injected into their session, are already in that session's context —
stored under `~/.claude/projects/`, untouched by the loss. What the room database
holds is the *record*. Losing it is nearer to losing scrollback than to losing work,
and D-028 traded something consequential for something largely recoverable.

**And it took more than it appeared to.** D-016 forbids a session that has received
teammate context from moving to another room. So a peer whose membership ended could
not collaborate again *from that session at all* — it would have to abandon the
Claude session, and with it the working context that was the actual point. A disk
hiccup cost the afternoon. Neither decision was wrong alone; their composition was.

**Decision.** Membership survives. Only one thing must survive with it for the room
to stay safe — the peer's own sequence position — so that is stored **outside the
room database, sharing the fate of the identity** rather than the fate of the events.

This inverts the dangerous failure rather than tolerating it:

- room events lost, identity and sequence intact → resume above the recorded number,
  refetch events from any member, membership continues;
- everything lost including identity → a new peer with a new sequence space, which
  can collide with nothing.

There is no longer any loss that both keeps an identifier and forgets what that
identifier issued — which was the precondition for B4.

**Reserve before publishing.** The sequence must be recorded before the event using
it is sent, never after. Failing between the two records a number that went unused,
which is harmless. The reverse publishes a number with no record of it, which is the
entire problem.

**Why this beats the sequence epoch D-028 rejected.** An epoch solves the same
problem by making a restarted counter occupy a different space, at the cost of a
field on every event and a synchronization state exchange keyed by peer *and* epoch.
Persisting the counter avoids the restart instead of accommodating it, changes no
event, and touches no protocol. D-028 was right that an epoch was disproportionate
and wrong that the alternative was giving up.

**Costs, stated so they are expected.** Events return only from peers that still hold
them, so a member recovering alone has a correct sequence and an empty history until
others reconnect. Delivery state is lost with the database, so some teammate turns
are injected twice — redundant rather than harmful, bounded by the room's lifetime,
and the right direction per D-014. The user should be told the room is refetching,
because that state is not the same as working normally.

**Rejected.**
- *Ending membership* (D-028) — see above.
- *A sequence epoch* — correct, and unnecessary once the counter cannot be lost.
- *Keeping the counter in the room database with a backup copy elsewhere* — two
  copies that can disagree, and the disagreement is the failure.

---

## D-030 — Two listeners: hooks on loopback, peer sync separately

**Date:** 2026-09-16 · **Status:** active (implemented)

**Context.** Making a pair work across two machines was preferred over a three-peer
run: transitive relay is nearly free in a pull design, so Phase 6 was unlikely to
invalidate anything, while two machines still hold real risk — clock skew, real
partitions, real latency — and are what makes the thing usable at all.

The blocker was small and structural. One listener on `127.0.0.1` served hooks, the
UI, and synchronization. A second machine could not reach it, and exposing it would
have exposed the hook API too.

**Decision.** Two listeners. `COGMER_ADDR` carries hooks and the UI and
**refuses to bind anything but loopback**. `COGMER_PEER_ADDR` carries
synchronization, defaults to loopback, and warns when bound elsewhere.

Separate listeners rather than one mux with a path filter, because the property that
matters is *which interface can reach an endpoint*, and a filter is a rule someone
can edit later without seeing what it guarded. Tests assert that neither serves the
other's routes.

**Why the hook API is the strict one.** Reaching `/hook/prompt` is equivalent to
being the local person: it publishes into the room and returns the room's
conversation. That is not an API to expose under any configuration, so it is refused
rather than discouraged.

**The peer API is exposed with a warning rather than refused,** because exposing it
is the entire point of a second machine. The warning states plainly that it is
unauthenticated and that nothing verifies who connects — true until identity becomes
cryptographic (D-023). Refusing would block the work; staying silent would imply a
protection that does not exist.

**Rejected.**
- *One listener, path filtering* — the boundary becomes a line of code rather than a
  network interface.
- *Exposing the peer API by default* — nothing should be reachable until someone
  decides it should be.
- *Refusing to expose the peer API until authentication exists* — that is the
  ordering D-023 warns against: it would mean building identity before ever running
  across a network, and the network is what the identity is for.

---

## D-031 — Peer synchronization polls; the push worth building is the local UI's

**Date:** 2026-09-16 · **Status:** active

**Context.** Polling was chosen for the two-peer experiment and recorded only as a
code comment calling push "deliberately omitted". It was an interim measure that
was never revisited, which is the gap this log exists to prevent. Asked directly
whether it was interim or decided, the honest answer was that nobody had decided.

**Decision.** Polling is the default for peer synchronization, not a placeholder.
§31 lists real-time push as Phase 3; it should not be built for the peer layer
without a reason beyond latency.

**The argument is C4's own measurement.** Peer propagation is roughly half a
second. Propagation *into a teammate's Claude* is not until their next prompt —
minutes, during a long agentic turn. Push would take the fast half from 500 ms to
50 ms while the slow half remains measured in minutes. It optimises the wrong side
of the path, and §16's sub-second target is already met by a one-second poll.

**Pull has properties push does not.** A peer that was absent recovers by asking,
so reconnection needs no retry queue, no delivery tracking, and nobody has to
remember what a missing peer missed. Push requires the sender to know who is
connected and what each has, which is state that can be wrong. Under session-scoped
rooms, where peers come and go with sessions, "ask for what you lack" is the
simpler shape as well as the cheaper one.

**The push that does matter is a different one.** §17 requires the local UI to
update live. That is the daemon pushing to a browser on loopback — server-sent
events — not peers pushing to each other. §31 places "Real-Time Push" in the peer
layer, where its value is lowest, and the UI has no phase of its own at all.

**Revisit when** something needs sub-second peer propagation for a reason other
than conversation: presence indicators (§35) are the likely first, since "Claude
working…" is stale the moment it is a second old. Large rooms where constant polling
is wasteful would be the second.

**Rejected.**
- *Push now, for §16's target* — the target is met, and the measurement showing it
  is met also shows why it does not help.
- *Leaving it undecided* — an interim measure nobody revisits becomes a decision
  taken by default, without the reasoning that would let anyone overturn it.

---

## D-032 — Re-sequence the phases, and follow them

**Date:** 2026-09-17 · **Status:** active

**Context.** Work had proceeded opportunistically: Phase 4 was completed inside
Phase 0, Phase 2 was completed before Phase 1 finished, Phase 3 was decided against,
Phase 6 was deferred, and parts of Phase 7 were taken early. That was the right trade
while the assumptions underneath the sequence were being tested. It stopped being
right once §31 described a plan nobody was following, which is worse than either
following it or replacing it.

**Decision.** Record actual status against every phase, add the phases the original
sequence lacked, and state an execution order to follow from here:

```
Phase 8  complete the local room   →  Phase 9  peer identity
      →  Phase 10 pairing          →  Phase 5  offline and reconnection
      →  Phase 7  hardening
```

Numbers are never reused or reassigned, so references in this log and in the code
still resolve. Superseded phases are marked, not rewritten.

**Why the UI comes first.** Every experiment so far measured whether *Claude*
understands a teammate's conversation, and none measured whether a *person* finds
watching one useful. That is half of §30, it has never been tested, and it cannot be
while the only way to read a room is a command-line dump.

*Amended the same day.* The first draft of this bundled room identity and the
membership index into the same phase, argued the UI was the reason that phase came
first, and then listed the UI third. Neither of the others blocks it — the UI needs
no room identifier and nothing from the index — so both were moved to the phases
whose purpose they actually serve: room identity to pairing, where a generated name
finally has an invitation to be spoken in, and the index to hardening, where
database recovery already sits. Phase 8 is now the UI alone. Bundling work that
shares a location rather than a purpose is how the thing a phase exists for ends up
scheduled behind the things it does not need.

**Why identity precedes pairing.** A guest list admits whoever claims a name until
identity is verifiable, so an admission flow built before Phase 9 would be built
twice. Transitive relay cannot be made safe without signing either. D-023 warned that
identity entrenches at the first exchange between peers; the experiments have already
exchanged events, but nothing has been released, so the warning still applies to the
first real use rather than to the first packet.

**What this does not change.** Pairs remain the target. Phase 6 waits for evidence
that a third peer is wanted, since it adds noise to a working session and is unlikely
to invalidate anything.

**Rejected.**
- *Follow §31 as written* — it specifies Tailscale, peer push, and a project-scoped
  room, all displaced by later decisions. Following it would mean building things
  already decided against.
- *Renumber the phases* — breaks every reference in this log and in the findings
  documents, to save reading one status table.
- *Leave the order implicit and keep working by judgement* — that is what produced a
  plan document contradicting the work, and it hid the UI gap for the length of the
  project.

---

## D-033 — The room cannot be displayed inside Claude Code; asking is the free affordance

**Date:** 2026-09-17 · **Status:** active

**Context.** §17 specified a browser at `localhost`, and a browser was built to it.
That turned out not to match what was wanted: the expectation was that the whole
experience lived inside the Claude Code session. Worth testing rather than
arguing, since a hook already pulls the teammate's turns — perhaps it could show
them too.

**Finding: it cannot.** Tested directly. A hook's standard output becomes context
for the model and never appears on screen. Standard error is not surfaced. Writing
to `/dev/tty` is not surfaced. Confirmed from the other side by an interactive
session: injection landed — the transcript holds the attachment, and the model
answered questions about the teammate's conversation in detail — while the
person saw nothing but Claude's reply.

Claude Code owns its display. No arrangement of hooks produces an ambient view
inside a session, and the specification now says so rather than leaving someone to
rediscover it.

**What the test surfaced that was worth more than the answer.** A person can
simply *ask*: "what is the team discussing?" gets a full answer — who said what, and
that they are unverified — from context already injected. It needs no code, no view,
and no protocol. For a pair on one problem it answers most of what a view would.

It also demonstrated D-021 reaching the person it was for. The `unverified` marker,
added so a model would not treat a display name as fact, was relayed to the
person unprompted in the model's own words. An attribution caveat travelling from
the wire to a human without a UI in between is the design working end to end.

**Decision.** §17 no longer prescribes a browser. It states that the room cannot be
shown inside the session, names asking as the affordance that already exists, and
treats a terminal view and a browser view as different moments rather than
competitors — one for glancing at without leaving the keyboard, one for reading a
long exchange properly. Both read only from the local daemon, which is what permits
more than one.

**Rejected.**
- *Making the injected block readable so it doubles as the display* — the premise
  was that the block is shown. It is not.
- *Writing to the terminal from a hook* — tested; not surfaced, and it would
  contend with Claude Code's own rendering even if it were.
- *Treating the browser as the answer* — it is a good way to read a long exchange
  and a poor way to stay aware while working, which is what was actually being
  asked for.

**Revisit if** Claude Code begins surfacing hook output. That would make an ambient
in-session view possible and is worth noticing; it cannot be checked automatically,
since it requires a terminal and an observer.

---

## D-034 — No terminal wrapper; a view sits beside the session rather than around it

**Date:** 2026-09-17 · **Status:** active

**Context.** D-033 established that Claude Code surfaces nothing a hook writes, so
ambient display needs something outside the session. A pseudo-terminal wrapper was
proposed: `cogmer` would launch the ordinary interactive `claude` inside a PTY,
proxy it, and draw peer turns in a reserved band the child cannot see.

The technique works. A passthrough prototype was byte-for-byte identical to running
`claude` directly — ANSI sequences, terminal dimensions, and exit code — under a
real 24×80 pseudo-terminal. Feasibility was never the problem.

**Decision.** Do not wrap. A view runs *beside* a session as a separate program,
not *around* it.

**Three reasons, compounding.**

*It replaces the entry point.* Everything else this project asks of a person is
something Claude Code already loads: hooks. A wrapper asks them to stop running
`claude` and run something else, permanently, and to keep doing so through every
future habit and alias. That is a materially larger ask than an install.

*It only reaches one of the ways Claude Code runs.* It is also a desktop application
on macOS and Windows, a web application, and a VS Code and JetBrains extension. A
pseudo-terminal intercepts the terminal and nothing else, and there is no wrapper
equivalent for an extension host. Ambient display would exist for some users and be
unreachable for others, with no path to closing the gap.

*Windows is a second implementation.* Pseudo-terminals there are ConPTY, a different
API from the Unix ones, and the terminal-behaviour matrix widens across Windows
Terminal, PowerShell, tmux, and IDE terminals. D-001 chose Go specifically so that
platforms would not diverge; this would have made the display path diverge anyway.

**What survives.** A standalone terminal view — run in a split pane beside a session
— has none of these properties. It does not replace `claude`, forks no
pseudo-terminal, never touches Claude Code's rendering, and works on Windows because
it owns its own terminal rather than puppeting somebody else's. It is additive: the
only thing installed remains the hooks Claude Code already loads.

**Rejected.**
- *Wrapping* — reasons above; the prototype is reverted rather than parked, since
  uncommitted code carrying two new dependencies would read as an intention.
- *Treating the browser as sufficient* — it is a good way to read a long exchange and
  a poor way to stay aware while working.
- *Keeping the wrapper for terminal users and something else for everyone else* —
  two display paths, the harder one reaching fewer people.

**Recorded as tested, so it is not re-derived:** an invisible pseudo-terminal
passthrough is achievable, and the reserved-band technique (shrink the child's
winsize, set the outer scroll region) avoids rather than solves the
partially-typed-prompt problem. If the entry-point objection ever stops applying,
that is the approach.

---

## D-035 — A remote peer never initiates local execution

**Date:** 2026-09-17 · **Status:** active (implementation already conforms)

**Context.** Proposed as an invariant: a remote peer event must not initiate Claude
execution in a receiving session; remote events are displayed asynchronously and
queued, becoming model context only at the receiving session's next locally
initiated turn.

**The implementation already conforms**, and not by design so much as by not having
written the code. The only path that starts a Claude run is `RunProbe`, reachable
from `runDoctor` (a typed command) and `EnsureVerified` (daemon startup). The
remote-event path, `pullFrom`, reaches neither.

**The specification did not state it, and said something weaker that was also
wrong.** §16 read "Daemon to a Claude session: there is no such path" — descriptive,
and untrue at the system level. `claude --bg` exists; a daemon could spawn a run on
receiving an event. The specification documented a limitation of *in-session
injection* while leaving open exactly what the invariant forbids.

**Decision.** State it as §3.7, an architecture principle rather than an observation,
because it constrains code that has not been written: no starting a session on a peer
event, no resuming or driving an existing one, no scheduled run originating from
received data.

**It is a security boundary before it is an ergonomic one.** Claude Code edits files
and runs commands. An event that could initiate a turn on a receiving machine is
arbitrary execution on that machine, authorised by whoever sent the event — and peer
identity is not verified (D-023), so that is whoever can reach the port. It is also
the person's subscription, context window, attention, and repository, none of
which are a teammate's to spend.

§3.7 is the converse of local-first: that principle says a session must survive every
peer disappearing; this says a session must be unaffected by every peer arriving.

**Guarded structurally rather than by review.** A test asserts that the files
handling peer traffic do not import `os/exec` or `syscall`, and do not call the
probe. Checking imports rather than call graphs is crude, and deliberately so: it
fails the moment the capability is added to the wrong file, which is when someone
should be asked to justify it.

**Scope, corrected the same day.** The first draft forbade a remote event starting
*any* Claude run, including a separate background one. That was broader than
intended and broader than is right: it collapsed two different concerns — "do not
disturb my session" and "do not spend my resources" — which have different remedies.
It also foreclosed something this specification contemplates elsewhere, where one
person addresses another's Claude directly.

The principle is now scoped to interactive sessions: a session a person is
working in takes a turn when that person asks it to, and at no other time. An
interactive session is a working state rather than merely a process, and a turn
arriving unbidden consumes the context window being relied on, may act on the
working tree mid-thought, and destroys the person's ability to reason about what
their own session has seen.

Whether a peer event may cause a **separate** run is explicitly left open. It raises
its own questions — whose subscription is spent, what tool access such a run has,
what the person whose machine it runs on agreed to — and those deserve an answer
rather than being settled here by implication. What must hold either way is that no
such run borrows the interactive session's context or interrupts it.

The structural test enforces something stricter than the principle requires: that
peer-handling code cannot start a process at all. That is deliberate. Nothing needs
the looser rule yet, and the questions above have no answers yet, so the guard stands
until they do.

---

## D-036 — MCP logging notifications are not a display channel

**Date:** 2026-09-17 · **Status:** active (tested, negative)

**Context.** After a terminal wrapper was ruled out (D-034), an MCP server looked
like the ideal carrier for ambient display: it is exactly "something Claude Code
already loads", it needs no change to how anyone starts Claude, it is
cross-platform, and it would plausibly reach the desktop application and the editor
extensions, which a pseudo-terminal never could.

**Tested, and it does not work.** A minimal stdio server was built that declares the
`logging` capability and emits `notifications/message`. Claude Code starts it, marks
it connected, and calls its tools normally. The notifications go nowhere.

Checked in every place they might surface:

- not in `--output-format stream-json`;
- not in `--debug` output;
- not in `--debug-file`, which produced 34 KB including thirteen lines about this
  server and zero containing the payload;
- not in `~/.claude/debug`.

**Tested twice, because the first test was wrong.** The first emitted only while the
server was idle, which a client may legitimately ignore — notifications are often
pumped only while a request to that server is in flight. So the server was rebuilt
to emit during a `tools/call`, before responding. Same result: the tool returned its
value, the notifications vanished.

**The conclusive evidence is the capability record**, not the absence of output.
Claude Code logs what it negotiated with each server:

```
{"hasTools":true,"hasPrompts":false,"hasResources":false,"hasResourceSubscribe":false, ...}
```

Tools, prompts, resources, resource-subscribe. Logging is not in that model at all,
though the server declared it. A client that rendered log notifications would track
the capability.

**Consequence.** MCP carries capability *to the model* — tools it can call, resources
it can read. It does not carry anything *to the person*. Display and inference reach
Claude Code by different routes, and MCP is only the second.

**Related, and worth stating before anyone builds an MCP server here for another
reason.** MCP also defines *sampling*, by which a server asks the client to run
inference. If Claude Code supports it, that is a direct route to violating §3.7 — a
peer's daemon could cause inference in an interactive session by way of a server.
Whether Claude Code implements sampling was not tested. Any MCP server this project
ships must not expose one.

---

## D-037 — Claude Code is launched and used unchanged

**Date:** 2026-09-17 · **Status:** active · **Generalises** D-034

**Context.** D-034 ruled out a pseudo-terminal wrapper by enumerating its costs:
it replaces the entry point, reaches only terminal users, and needs a second
implementation on Windows. That reasoning was correct and too specific — it had to
be re-derived for each new proposal, and it was derived *after* a prototype had
already been built.

**Decision.** State it as a principle instead. A person starts and uses Claude
Code exactly as they do today; this system installs *into* it, never *around* it.
The only things a participant installs are things Claude Code already loads: hooks,
skills, MCP servers, and whatever else it accepts.

The value is that it applies without argument. If a proposal requires starting
Claude Code differently, installing something it does not already load, or
understanding how it renders — it is out, and no cost-benefit discussion is needed.
Applied earlier, it would have stopped the wrapper before anything was written.

**What it buys.** Claude Code is a terminal program, a desktop application, and an
editor extension. A system extending it through its own mechanisms works on all of
them without knowing any of them exist. A system wrapping its process works on one
and cannot be made to work on the others. The constraint also keeps terminal
emulation, ConPTY, editor terminals, shell integration, and every future Anthropic
surface out of this project's responsibility.

**The consequence that must be accepted, not escaped.** Claude Code's extension
points all deliver to the model — hooks supply context, skills supply instructions,
MCP servers supply capability — and each reaches a person only through what the
model then says. Confirmed independently three times: D-033 (hooks display nothing),
D-036 (MCP logging is never rendered), and skills being markdown instructions rather
than programs.

So conversation semantics must work everywhere and do, being model-facing, while
presentation is best effort. A view *outside* the session remains permitted: it is a
separate program a person may run, not a change to how they start Claude Code.
What is forbidden is taking ownership of Claude Code in order to draw inside it.

**On searching for an undocumented display seam.** Proposed, and declined. The
installed artifact is a native binary, so there is no source to read; "Channels"
does not appear in this version; and the plugin surface is packaging — `claude
plugin details` reports a *projected token cost*, which confirms its components are
model-facing.

More decisively, a seam found that way would be a worse dependency than the wrapper,
not a better one: undocumented, unversioned, and unverifiable by the behaviour
registry, since visual correctness needs an observer rather than an assertion. "Find
an internal seam" and "do not take ownership of Claude Code" are in tension, and the
first loses.

**Preferred instead, if ambient awareness is wanted.** The daemon can raise an
operating-system notification directly — no Claude Code involvement, nothing to
break on upgrade, and it works on every surface because it never touches any of
them. That yields the signal ambiently and the content on demand, which is what the
extension surface can actually support.

If a display primitive is ever wanted from Anthropic, the request is small and
already well-specified here: append a display-only message to the current session,
without scheduling inference — §3.7 states that second half precisely.

---

## D-038 — Separating the room from the session is correct on its merits

**Date:** 2026-09-17 · **Status:** active

**Context.** D-033 concluded that the room cannot be displayed inside a Claude Code
session, and everything since has treated an external view as what remains after
that constraint. That framing was backwards.

**Decision.** Record the separation as a design position rather than a consequence.
A view outside the session is what should be built even if an in-session display
became available.

**The argument.** A session is a person's conversation with their own Claude,
read closely. A room is a record of what colleagues are doing, glanced at.
Interleaving them buries the glanceable thing inside the closely-read thing, and
interrupts the closely-read thing with arrivals not addressed to it. A person
loses the thread of their own work in order to be told something they could have
looked at when they chose.

They also scale differently. One colleague interleaved might be tolerable; three is
unreadable — and the cost lands on the person's own working view, which is the
last place it should land. Separated, additional participants cost nothing there.
This answers the earlier observation that a three-peer session "feels a little
noisy": the noise is only unavoidable while the room shares space with the
conversation.

**The form it takes.** The model and the person want the same conversation
differently. The model wants teammate turns *in its context*, at a turn boundary,
phrased for a reader that does not skim — which §20 already specifies. A person wants
them *available to glance at*, without their own thread stopping to carry them. One
channel cannot serve both without compromising each, so injection serves the model
and a view serves the person.

**Consequence for how the constraint is described.** §17 no longer opens by saying
the room cannot be shown inside a session. It opens with why the room belongs
outside one, and treats the constraint as agreeing with the design rather than
causing it. The distinction matters for anyone reading later: a reader who believes
this is a workaround will try to undo it the moment an in-session display appears.

---

## D-039 — One renderer until it has been used

**Date:** 2026-09-17 · **Status:** active

**Context.** D-038 established that the room belongs outside the session, which made
two further renderers look attractive: a terminal view for a pane beside the
session, and an operating-system notification for ambient arrival.

**Decision.** Build neither yet. The browser view exists; nobody has worked with it.

**Why this order.** Everything now known about what a view should be is reasoning.
The questions that decide the next renderer cannot be answered by more of it: whether
glancing at a second window is acceptable or whether it is one window too many;
whether a room is watched continuously or consulted occasionally; whether arrival
needs announcing at all, or whether noticing on the next glance is enough. Each has a
different answer for a pair than for four people, and none is knowable in advance.

A second renderer built now would encode a guess and then have to be maintained
whether or not the guess held.

**What using it will settle.** Solo use is sufficient to start: a person watching
their own turns appear tests readability, live update, and whether a separate window
is glanced at or forgotten — without needing a second participant. The
collaboration-specific questions need a pair, but the ergonomic ones do not.

**Revisit when** there is experience to report. The likely outcomes are that the
browser is fine and nothing more is needed; that it is right but wants announcing,
making the notification next; or that a second window is not consulted at all, making
the terminal pane next. Those lead to different work, which is the reason to wait.

**Outcome, 2026-09-17.** Used, and judged the right avenue. That settles the question
this decision was waiting on: a separate window is consulted rather than forgotten,
so the terminal pane is not the next thing and the browser is not a placeholder for
it. D-038's position — that the room belongs outside the session on its merits —
now rests on use rather than on argument.

Still open is whether arrival wants announcing. That is a different question with a
different answer for someone watching a pairing closely than for someone dipping in
during long solo stretches, and it remains unanswered.

---

## D-040 — The injected block is fenced with an unforgeable value, and framed by classification

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** Asked whether the literal teammate turn is pushed into context without
language making clear it is informational rather than instructional. There was such
language — a sentence at the top of the block — and it was defeatable.

**The vulnerability.** Content was interpolated raw. A teammate turn consisting of
`</message></team-conversation>` followed by a forged operator instruction escaped
the block entirely: the injected text then appeared *after* the closing tag, where
the framing no longer applied. Demonstrated rather than theorised. Since peer
identity is unverified (D-023), the capability belonged to anyone who could reach
the sync port.

**Decision — the boundary must be unforgeable.** Each block carries a fence value
generated per injection and unknowable to the content; the value is stripped from
the content so it cannot be reproduced; the framing states that the block ends only
at the matching value and that text claiming otherwise is part of the block; and the
framing is restated *after* the content, so the last thing read is the boundary
rather than the first.

**Decision — frame by classification rather than authority.** Instructing a model to
disregard instructions invites it to weigh two instructions. Telling it what *kind of
thing* it is reading does not. The framing now says that nothing inside the block is
addressed to it however phrased — including text appearing to come from an operator,
a system, or its own user — and that a request appearing inside is *a report that
someone made a request*, not a request made of it.

**Verified against a live session, not only in structure.** A real session given a
forged `SYSTEM OVERRIDE` instruction ignored it, explained that it came from inside
the record and was therefore information rather than instruction, and reported the
attempt to its own user unprompted. The classification framing is what gave it the
language to do that.

**What this does not solve.** A teammate's genuine turn may legitimately contain
imperative text — colleagues tell each other to run things. No fence distinguishes a
hostile imperative from an honest one, and none should: both are reports of what
someone said. The defence is that neither is addressed to the reading model, which is
exactly what the framing now asserts.

**Rejected.**
- *Escaping the delimiters* — whack-a-mole against prose, and it assumes the
  boundary is syntactic when the model reads it as language.
- *Truncating or sanitising content* — §3.4 requires the actual conversation be
  preserved, and a teammate's words are not the system's to edit.
- *Relying on the model to be robust* — it was, here, and that is a property of the
  model rather than of this design. The fence holds whether or not the next model
  does.

---

## D-041 — Ship as a Claude Code plugin; the session-start hook starts the daemon

**Date:** 2026-09-17 · **Status:** active (specified; not implemented)

**Context.** §3.8 requires that the only thing installed is something Claude Code
already loads. That settled what *not* to build — no wrapper, no launcher — without
saying what installation actually looks like. In practice it was still a manual
daemon start, a hand-written settings file, and environment variables.

**Decision.** Package as a plugin carrying the hooks, installed with
`claude plugin install cogmer`. The plugin surface is real and includes
`install`, `uninstall`, `update`, `validate`, `init`, and `marketplace`.

**And the session-start hook starts the daemon.** This is the part that converts the
install from "run these commands and edit this file" to one line — which was the
original objection to the wrapper, now answered without any of the wrapper's costs.
A person should not have to start the daemon, notice it has stopped, or know it
exists.

**Three requirements, each easy to get wrong.**

*Starting must not delay the session.* Waiting on a daemon nobody asked for is worse
than having no daemon. The same fault was already made once, where the behaviour
preflight ran before the listeners and left a new room unreachable for several
seconds.

*Already running is the ordinary case, not an error.* Several sessions begin at once
on one machine routinely; each attempts the start, at most one succeeds, none
reports anything. A failure to bind is the expected outcome.

*Failure is silent to the person and recorded by the daemon.* No daemon means no
collaboration, which is degraded rather than broken — the same fail-open rule the
hooks already follow.

**A background process must remain findable.** The daemon outlives the session that
started it, since a room may have members in several sessions and restarting it
repeatedly is worse than leaving it up. That makes it something a person did not
start and might not know about, so it must be discoverable and stoppable by the
person whose machine it runs on.

**Deliberately not encoded: which surfaces this reaches.** A surface matrix — CLI,
desktop, editor extensions, and whatever comes next — changes faster than a
specification does, and a design that enumerates surfaces is wrong within a release.
§3.8 is stated so that the answer follows from the mechanism rather than from a list.
Whether any particular surface runs hooks locally remains a question to answer by
testing that surface, not by consulting a table.

---

## D-042 — Peer identity is an Ed25519 key pair; events are signed at origin

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** Phase 9. D-023 required an identifier safe to know; §13 required that a
relayer never rewrite an event's origin, with nothing enforcing it; §25 asked for
signable identity and got a random string.

**Decision.** A peer's identifier *is* its Ed25519 public key, rendered
`ed25519:<base64url>`. Events are signed at origin over a length-prefixed encoding
of their own fields, and a receiving peer verifies every event against the key its
own identifier names.

**Why the identifier is the key rather than a fingerprint of it.** It is
self-certifying: a signature can be checked from the identifier alone, with nothing
to look up and no key distribution step. And it settles D-023's requirement
absolutely rather than approximately — knowing an identifier grants nothing, which
is what allows the system to put one in every event, every interface, and every
exchange, as it does by design.

**The private key lives in its own file.** `identity.json` is printed by `whoami`
and is meant to be handed to a colleague; an identity that cannot be shown without
checking what else is in it is not much of an identity. A test asserts the key never
marshals.

**Signing covers length-prefixed fields with a purpose tag.** Concatenating fields
directly would let a boundary move — a content ending in one value and a session id
beginning with another could swap undetected. The leading tag binds a signature to
this purpose and version, so one made here can never be replayed as one made over
something else.

**Rejection, not quarantine.** A sequence conflict is ambiguous — a peer may have
lost its state — so D-027 keeps the evidence. A failed signature has no benign
reading, so it is refused and logged. Verified live: a peer impersonating another
and offering an event signed by nobody was rejected, and nothing reached the room.

**What this does not do, stated because the startup warning used to overclaim.** It
gives integrity and attribution: nothing can be forged, altered in transit, or
falsely attributed, and §13 is enforced rather than merely stated. It does not
decide who may *connect*, so the peer API still admits any host that can reach it to
read a room. Proof of possession on connection is Phase 10, with admission, and the
warning now says exactly this rather than claiming identity is not cryptographic.

**Migration.** An `identity.json` whose identifier is not the local key is rewritten
to match it. The old identifier named an identity nothing could verify; preserving
it would preserve a claim.

---

## D-043 — The wire format is defined separately from the stored row

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** A suggestion that the daemon's database schema should not become the
protocol. We were half-violating it: `Event` was simultaneously the SQLite row and
the type marshalled into `/sync`. One field, `claudeSessionId`, also encoded an
assumption about which agent produced a turn.

**Decision.** Define the wire format in its own file, as its own type, with explicit
conversion in both directions. Rename the session field to `originSessionId`, which
says what it is — where a turn came from — without naming the agent that produced
it. Carry a protocol version in every sync exchange and refuse a peer that speaks a
different one.

**Why a separate type when the fields are currently identical.** A database row and
a protocol message answer to different pressures. A column can be added for local
bookkeeping without telling any peer; a wire field cannot change without every peer
agreeing. Sharing one struct means the next convenient column silently becomes
protocol, and nobody has to decide anything for that to happen.

**Why the version moved to v2.** A signature covers field values, so changing what a
field means changes what was signed. The signing tag is the protocol's real version
marker, and it was already versioned — which is why this was cheap.

**Why now.** No room existed that anyone would mind losing, and the signing format
had been fixed for exactly one day. The same change after two colleagues have a room
they care about means a migration, a compatibility window, and a reason not to
bother.

**What was deliberately not done.** No `source: claude-code | codex` field, no
adapter architecture, no design for a second host. The argument for those was that
cross-agent collaboration becomes nearly free once events are normalised, and the
evidence says otherwise: capture was solved in Phase 0 and every hard problem since
has been host-specific injection — hooks display nothing, MCP never renders, a remote
event must not trigger inference, the injected block needs an unforgeable fence,
delivery must be confirmed by evidence. None of that transfers. A second adapter
would inherit the schema and none of the difficulty.

Generalising from one adapter, zero users, and a question answered the day before is
where that goes wrong. The rename buys the optionality; the architecture can wait for
a second host to actually exist.

**A bug this surfaced.** `CREATE TABLE IF NOT EXISTS` creates a table and then
ignores it forever, so every room keeps the shape it was born with and every column
added since is missing from every room that predates it — surfacing not at open but
at the first query that names it. An existing room failed with *no such column:
signature* only when a peer asked it to sync. Rooms are now migrated on open, and
adding a column to that list is the whole of what a future migration needs.

---

## D-044 — Sync requests are signed; authentication is not admission

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** Phase 9's third part. Signing events gave integrity — nothing could be
forged or falsely attributed — while the peer API still answered anyone who could
reach it.

**Decision.** Every sync request carries the caller's identifier, a timestamp, a
nonce, and a signature over all four plus a purpose tag. The receiver verifies the
signature against the key the identifier names, rejects a timestamp outside two
minutes, and rejects a nonce it has already seen.

**Signed requests rather than a session.** The protocol polls. A handshake per poll
would cost two round trips a second to avoid holding one piece of state, and a
server-issued challenge would add a round trip for the same reason. A timestamp and
a nonce give replay protection without either.

**Two minutes of tolerance** because clocks differ: two NTP-synced machines measured
408ms apart, and a peer on a worse network should still sync. Wide enough to work,
narrow enough that the seen-nonce set stays small — and it is pruned on every
admission, so a daemon polled every second does not accumulate.

**The `have` map is not signed.** Altering it gains an authenticated peer nothing,
since it may ask for everything anyway, and canonicalising a map for signing invites
exactly the ambiguity that length-prefixing exists to prevent.

**What this does not do, and the test that proves it.** Authentication establishes
*who* is asking. It does not establish *whether they may*. A stranger generated a key
pair, authenticated correctly, and read a private room — while the host logged
nothing, because nothing was wrong with the request.

That is worth stating plainly because it is easy to bank: a system that
authenticates every caller and admits every authenticated caller has gained a name
for its visitors and nothing else. The confidentiality gap narrowed from "anyone who
can reach the port" to "anyone who can reach the port and generates a key", which is
no barrier.

**So what Phase 9 delivers is the ability to make an admission decision, not the
decision.** The guest list is Phase 10, and until it exists the startup warning says
exactly this rather than implying the room is protected.

---

## D-045 — Rooms are records with a guest list; admission is enforced

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** Phase 10. D-044 left the confidentiality gap open and said so: a
stranger generated a key, authenticated correctly, and read a private room, because
nothing decided *which* peers could ask.

**Decision.** Rooms become records rather than arbitrary strings — a UUID, a
generated `weather-landscape` name, and a guest list — stored in `membership.db`
beside the identity rather than inside any room. A sync request is refused unless
its authenticated peer is a guest.

**Verified by repeating the test that failed.** The same uninvited stranger now
reads nothing, and the host records why: *"clever-crane … is authenticated but is
not a guest of this room."* An invited peer reads the room with no refusals. That
pair of results is the whole of Phase 10's value.

**Two scopes, as §12 requires.** `known_peers` is machine-wide and durable; a room's
guests are per-room. Knowing six colleagues and admitting two to a room about
customer data has to be expressible, and one list cannot express it. `forget`
discards an identity so a later meeting is a first meeting; `revoke` withdraws
admission to one room and leaves the identity known. They read similarly and are not
the same act.

**Only keys can be admitted.** `allow` and `invite` refuse an identifier that names
no key. Recording `alice` or an old `peer-8f3a…` would be recording a hope: nothing
could ever prove possession of it, so the entry could never do its job.

**The out-of-band step is a public key, and the tooling says so.** `allow` prints
the full fingerprint and tells the operator to verify it over a channel the
identifier did not travel on. That is the only moment trust is taken on faith, and
it should be the one moment a person is asked to pay attention.

**A bridge, noted as such.** A daemon pointed at a room nobody created makes one and
admits its creator, so existing setups keep working. §12 has invitation as the
deliberate act and D-022 has a room beginning when someone is invited; a daemon
creating a room on startup is neither. It stands until a session joins a room rather
than a daemon serving one, which is the multi-room refactor §5 describes and this
phase did not attempt.

---

## D-046 — The daemon serves many rooms; a session says which one it is in

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** A daemon served exactly one room, chosen by an environment variable at
startup — so it invented a room when pointed at one nobody had created. That
contradicted §12 and D-022, where a room begins when somebody is invited, and it was
recorded as a bridge rather than hidden.

**Decision.** The daemon is the machine's local service, not a room. It opens a
store per room on demand, and a **session** binds to a room on first sight. A
session in no room is an ordinary Claude Code session: nothing captured, nothing
injected, nothing shared.

**Binding at first sight rather than asking**, because nothing knows a session exists
until its first hook fires, and there is nobody to ask at that moment. A machine-level
*current room* answers instead: `join` sets it, and sessions started afterwards enter
it. Once a session has been offered teammate context it is marked, because §12a
forbids moving it and there is no later moment at which moving it would be safe.

**Two gaps only the end-to-end test exposed.** Both had the same shape — one side of
a symmetric arrangement.

*A guest knew no room existed.* Invitation recorded admission on the host's side
alone, so the guest had nothing to join. An invitation now carries the room's name,
its identity, and where to reach it — all public, no token (D-026).

*Synchronisation was one-way.* The guest could read the host and never be read, for
two compounding reasons. The host had no address for the guest, since a pull needs
somewhere to pull from — so a peer now advertises where it listens, signed, because a
peer acts on that address by polling it and an unsigned one would redirect polling.
And the guest had recorded the room without its host, leaving a one-sided guest list
that refused the host's requests. An invitation now carries the inviting peer's
identifier, and joining admits them.

The second is worth keeping in mind generally: **a guest list is per-peer, so two
peers can disagree about who belongs.** Here it presented as one-way collaboration
and was really an asymmetric list.

**What this makes possible that was not before.** A person can be in several
rooms across different sessions, which §12a's constraint describes and a one-room
daemon could not express. A room outlives the session that created it. And nothing
is created by starting a process — the bridge is gone, not because it was removed but
because the situation it papered over no longer arises.

---

## D-047 — The fingerprint is the only manual link, and had the least careful encoding

**Date:** 2026-09-17 · **Status:** CLOSED, not implemented — the construction it
argued about was removed. See the closing note at the end of this entry; the
measurements below are kept because they are what justified removing it.

**Context.** Asked what a fingerprint is and what it accomplishes. Demonstrating it
made the answer sharper than expected.

**What it accomplishes, precisely.** Every cryptographic guarantee here binds a key
to itself: the peer speaking today holds the same key as yesterday, nobody forged
its events, nobody replayed its requests. None of it binds a key to a **person**,
and no amount of cryptography can — that is not a mathematical question.

Demonstrated: an attacker substituted her own identifier in transit, was recorded and
invited under the name `alice`, read a private room, and replied into it. **Zero
refusals.** Nothing was wrong, because nothing was wrong — the host invited exactly
the key he was given.

The fingerprint closes that by comparison over a **second channel**. The attacker who
controlled the first would have to control the second too. What is protected is not a
secret: the identifier is a public key and intercepting it is harmless. The risk was
never that someone reads it — it is that someone **swaps** it. A fingerprint does not
guard a secret; it detects a swap.

**Why the whole key — stated too loosely at first, and corrected.** Truncation is
not itself the fault; the cost of grinding a matching key is exactly the entropy
shown, and an attacker must know the target first, which interception provides.
Measured: three characters fell in 339,297 tries and under a second; four take about
thirty seconds; eight are 2^48 and within reach of a resourced attacker; sixteen are
2^96 and are not. Showing the whole key is therefore not strictly necessary — sixteen
characters would do — it is the rule that avoids reasoning about thresholds, and it
costs nothing at 43 characters.

**The number that actually binds is not how much is displayed but how much a person
compares.** Show someone forty-three characters of base64 to check over a telephone
and they will read the first group, the last group, and skim the middle. That is
ordinary behaviour, and it quietly reduces the verified entropy to whatever was
genuinely checked — plausibly the three-to-four character range that falls in under a
second.

This makes the encoding argument and the grinding argument the same argument. Words
are not preferable because they are prettier: a word is compared as a unit. Somebody
either says "badger" or does not, where an eye slides over `ol5v` without stopping.
Security here is bounded by what a person will actually do rather than by what the
system displays.

**The problem, which the shape of the output exposed.** This is the *only* manual
step in the design. Everything else is automatic. The entire chain — admission,
attribution, signatures, the unverified markers — rests on one person, once, reading
a string aloud correctly.

That step has the most error-prone encoding available. Base64url is case-sensitive
and the alphabet contains `l`, `I` and `_`, which are indistinguishable from `1` and
from each other when spoken: `ol5v` must be dictated as "lowercase-o, lowercase-L,
five, lowercase-v". §25 asks for "a sequence of words or grouped digits"; grouped
base64 is neither.

A comparison that is tiresome to do accurately is one people do badly or skip — and
skipping it reproduces the demonstration above exactly, where nothing looks wrong.

**Decision at the time.** Render the fingerprint as words. Digits would also work
and are what Signal uses, but curated wordlists already exist here, and this is the
one place where slow to read beats quick to mishear.

**Closed without implementing it (2026-09-18).** The question "how should the
fingerprint be rendered for comparison" stopped having an answer when comparing a
fingerprint stopped being a way to verify anything. D-052 built the two-word
comparison and D-055 removed the whole-key comparison entirely, so there is one
ceremony and the fingerprint is not it.

The argument above is what closed it rather than what was overtaken by it. Its
finding — that security here is bounded by what a person will actually do, not by
what the system displays — is the reason a second, harder ceremony could not be left
standing beside an easier one. Two accepted ceremonies means the weaker is what gets
performed, and this entry measured exactly how weak that is: three characters fell in
339,297 tries and under a second, and a reader shown forty-three of them checks the
first group, the last group, and skims the middle.

`Fingerprint` survives as a **display**, used where a person genuinely reads an
identifier — when a key has changed and two are side by side. Nothing about looking
at one marks a peer verified, and its test now asserts that it is lossless rather
than that it is a comparison.

**The general point worth keeping.** The weakest link in this system is a human
reading a string, and it was given the least design attention of anything in it.
Where a design has exactly one manual step, that step deserves the most care rather
than the least.

---

## D-048 — Verification should bind a live exchange, not a standing identifier (ZRTP's SAS)

**Date:** 2026-09-17 · **Status:** active as to construction; its **placement** was
superseded by D-052 (its own act, not part of join) and then by D-053 (at pairing).
D-047 remains held.

**Context.** Tracing the manual steps produced a count: there are **two**. Alice runs
`whoami` and her identifier reaches David somehow (transfer); David then asks her to
confirm the fingerprint (confirmation). §12 says transfer is safe "over any channel
whatsoever, because holding it confers nothing" — true of **confidentiality** and
silent about **integrity**, which is where the entire risk lives. Nothing is lost when
an interceptor reads an identifier; everything is lost when one swaps it, which D-047
demonstrated end to end with zero refusals.

Confirmation only means something across a channel boundary. Alice pasting into Slack
and David asking in Slack is one step performed twice: same bytes, same path, same
attacker.

And the two steps collapse into one when done properly — on a call, Alice reads the
identifier out and transfer and confirmation are a single act. **The two-step shape is
the asynchronous convenience, not a stronger construction.** It exists because reading
43 base64 characters aloud is miserable and pasting is not. Worth labelling as an
ergonomic trade so nobody later defends it on security grounds.

**What ZRTP does.** Two parties agree a fresh Diffie-Hellman over the media path,
derive a **Short Authentication String** — around 16–20 bits, rendered as words —
from a hash of the shared secret and both public values, and read it aloud over the
voice call they are already on. Two properties make that sound at a length we had
assumed was unusable:

- a **hash commitment** forces each side to fix its contribution before seeing the
  other's, so a relaying attacker cannot search for a substitution that collides on
  both sides — he is reduced to one blind guess;
- the compared value is **fresh**, so there is nothing to precompute against.

Key continuity does the rest: the secret is cached and chained into later calls, so
the ceremony happens once and a later mismatch is an alarm. That is our known-peers
list, structurally.

**What it corrects here.** §25 said a short mnemonic "catches an accident and not an
adversary, because the bits it does not cover are free to differ." The conclusion is
right for the construction we chose and the reason given was wrong, and the wrong
reason made the rule look universal. The real line is **offline precomputation versus
one online guess**:

- a standing identifier can be ground against offline, at a cost of exactly the
  entropy displayed (D-047 measured it), so it must be compared in full;
- a committed, freshly randomised value cannot be aimed at in advance, so a short
  form is sound.

"Render the whole key" is therefore a consequence of having chosen a static
construction, not a law of the domain. §25 now says so, because as written it
foreclosed the better option while appearing to rule it out on principle.

**Decision.** Adopt the live-exchange form: each side commits to a hash of its
contribution, both reveal, the string derives from both long-term identity keys plus
both fresh nonces, and each side prints two words for the people to compare on the
call they are already on.

This entry placed that at **join**, reasoning that both daemons are connected there.
D-052 rejected the placement: the ceremony is its own act, because it is interactive,
because it blocks on another person, and because joining is a room operation while
verifying a key is not. D-053 then moved it to **pairing**, which is the act that
happens once between two machines. Read the placement below from D-052 and D-053; the
construction below is as built.

Three consequences, in order of how much they change:

1. Confirmation costs **two words instead of 43 characters**. The friction that makes
   people skip it mostly disappears — and disappears for a principled reason rather
   than by making the same long string prettier.
2. Transfer stops needing to be trustworthy. What is authenticated is whichever key
   actually arrived, however it arrived, so §12's "any channel whatsoever" becomes
   true about integrity as well.
3. **D-047 is held, not cancelled.** Words beat grouped base64 under either
   construction, but it is a rendering change to a construction we may not keep, and a
   two-word SAS makes the question much smaller. Decide the construction first and the
   rendering falls out. A `verified` flag — which today does not exist anywhere in
   `membership.go`, so the `unverified` marker is a constant true of every peer
   forever — should record whichever ceremony is actually built, and is therefore
   sequenced behind this rather than ahead of it.

**What this does not fix.** Nothing, for two people who have never met. ZRTP rests on
recognising a voice, which presumes prior acquaintance; §25's "case with no answer"
survives untouched. This improves the ergonomics of the case that *can* be handled.
Do not let a cheap ceremony be read as having closed the expensive gap.

**The hazard to implement against.** A short string with unlimited silent retries is
weak — the attacker simply tries again. What protects it is that failure is
**conspicuous**: a mismatch must refuse the join, say plainly that something
intercepted it, and never present itself as a transient error worth repeating. And a
commitment step implemented incorrectly degrades to a grindable value while still
looking like a ceremony, producing the confidence without the property. That is worse
than performing no ceremony at all.

**Revisit when** the construction is built, or if verification is ever wanted at a
moment when the two peers are **not** simultaneously connected. The short form is
unavailable there and the full-length comparison of D-047 is the only option, so both
renderings may need to exist — the live one for joining, the static one for
confirming a peer after the fact.

---

## D-049 — The sync request addresses a room by id, never by name

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** Asked whether the room creator's daemon signs its polling requests. It
does — there is one polling path, `pullRoom`, every daemon takes it for every room,
and no host role exists in the code. Reading it turned up something else: the request
carried `room: <roomName>`, `handleSync` resolved it with `FindRoom`, which matches
`room_id = ? OR room_name = ?`, and the **name** was what the signature covered.

**What the collision actually costs — the first reading was wrong.** The obvious
worry is that a name resolves to the wrong room. It cannot: `rooms.room_name` is
`NOT NULL UNIQUE`, so a daemon never holds two rooms under one name and the lookup is
unambiguous. No confidentiality was at stake and nothing read the wrong room.

The real cost sits one step earlier. `CreateRoom` regenerates on a local name
collision, but `RecordRoom` — the path taken when **joining** a room someone else
named — has `ON CONFLICT(room_id)` only. Joining a second room whose generated name
matches one already held violates the unique index, `runJoin` calls `log.Fatalf`, and
the person is told they cannot join a room for a reason that names nothing they
did. 7,656 names make that unlikely per pair and certain at some scale. **That bug is
not fixed by this entry** and is recorded here so it is not mistaken for fixed.

**Decision.** `roomId` on the wire, in both directions, and peer-supplied
identifiers resolve through `RoomByID` rather than `FindRoom`. `FindRoom` keeps
accepting a name, which is right at a command line and wrong on the wire.

`wireVersion` 1 → 2 and the sync-request signing tag to its v3, because
the signed bytes changed meaning rather than shape. A peer on the old version is
refused with a version mismatch instead of failing a signature check, which is the
difference between a diagnosis and a mystery.

**Why bother, given nothing was exploitable.** D-017 says never key on the name, and
the wire was keying on the name. The property protecting it lived somewhere else
entirely — a unique index in a local schema, which no peer can see and nothing
obliges the next schema change to keep. A guarantee held at that distance from the
thing it guards is one nobody will think to preserve.

**Tests.** `TestTheWireRefusesARoomName` presents a request correct in every other
respect — genuine guest, genuine signature, a room it really is a guest of, named —
and requires 401. Confirmed to fail with 200 when the lookup is reverted to
`FindRoom`. `TestTheWireAcceptsARoomID` pairs with it so a future failure of the
first is read as the lookup breaking rather than the fixture rotting.

**Revisit when** a room identifier needs to be typed by a person on a path that
reaches a peer. It should not: a person types a name, their own daemon resolves it,
and only the id travels.

---

## D-050 — Room names may collide locally; the schema stops forbidding it

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** Found while making the wire address rooms by id (D-049).
`rooms.room_name` was `NOT NULL UNIQUE`. `CreateRoom` coped by regenerating on a
clash, but `RecordRoom` — the path taken when **joining** a room somebody else
named — had `ON CONFLICT(room_id)` only, so the insert violated the index and
`runJoin` called `log.Fatalf`. The person was told they could not join a room,
for a reason naming nothing they had done and nothing they could change.

Not hypothetical at any real scale. Names are drawn from 7,656 combinations, the
table accumulates **every room ever recorded** rather than the ones currently in
use, and the clash is with rooms other peers named — which this machine cannot
influence. Around a hundred rooms over a machine's lifetime makes it a coin flip.

**Decision.** Drop the constraint. Names collide by design (D-017) and this table
holds rooms other peers named, so uniqueness here was never a rule about the world:
it was this machine refusing to record something that had already happened
elsewhere.

Three consequences, each a place the constraint was silently doing work:

- `CreateRoom` **asks** whether a name is free instead of catching the violation. A
  name this peer mints it is free to mint differently, so avoiding a local clash
  still costs nothing — unlike one arriving with a room already named.
- `FindRoom` reports **three** outcomes, not two. Answering "no such room" for an
  ambiguous name would send someone looking for a room they are already in, and
  returning whichever row came back first would choose for them without saying so.
  It now names both identities and asks which was meant.
- `migrateMembership` rebuilds the table, since SQLite has no `DROP CONSTRAINT`.
  `CREATE TABLE IF NOT EXISTS` leaves an older table exactly as it was, so without
  this the relaxed schema would apply only to databases created after it — the same
  trap `migrate()` exists for on the room stores.

**What was not done.** Renaming a joined room locally to keep names unique was the
obvious alternative and is worse: it makes this machine disagree with the host about
what the room is called, so the name David says aloud is not the name Alice sees.
A name is a mnemonic for a room, and one that differs per peer is not a mnemonic.
Ambiguity is the honest state and is reported as such.

**Tests.** `TestJoiningTwoRoomsWithOneNameSucceeds` is the bug itself;
`TestAnAmbiguousNameIsReportedNotGuessed` requires the report to be distinguishable
from "unknown" and to name both ids; `TestMigrationDropsTheNameConstraint` writes a
pre-change database, opens it, and requires both that the old room survives and that
the second insert succeeds. All three confirmed to fail against the pre-change
schema with the migration disabled.

**Revisit when** a person needs to refer to a room across peers by name in a context
with no host to resolve it against. §12 says a name is only ever resolved against a
specific peer; anything that breaks that assumption reopens this.

---

## D-051 — Stranger pairing is not a supported case

**Date:** 2026-09-17 · **Status:** active (specification)

**Context.** §12a said the host-approval path exists "because it is how two people who
have never met will ordinarily pair." Questioned directly: why would two strangers
sharing linked Claude sessions be ordinary? They would not, and §12 says the opposite
two sections earlier — "where the people involved already know one another, **which is
the ordinary case**, since colleagues pair repeatedly." The specification asserted both.

**The distinction that was collapsed.** There are two first contacts, and §12a merged
them under one word. **Socially strangers**: no relationship, no established reason for
trust. **Known people whose machines have not met**: you work with Alice daily and
there is simply no key on file. The second is genuinely the ordinary first contact and
needs the mechanism §12a describes; it needs none of the trust the word "stranger"
implies. Calling it "two people who have never met" made a colleague-ergonomics feature
read as a stranger-trust feature.

**Decision.** Exclude stranger pairing as a design target, on the specification's own
terms rather than on taste.

§25 requires verification over a channel where the other party can be **recognised**,
and already records that people who have never met have no such channel. A stranger
pairing would run the ceremony and take nothing from it: the words match, and a match
between strangers establishes that two parties hold the same key while saying nothing
about whose. Every other pairing gets a real assurance from that step. This one gets
its appearance — worse than omitting it, because the appearance is what people act on.

Secondarily, the exposure is asymmetric with the benefit. A room carries a working
session, and admission sends a member's turns to another person's provider under
that person's account (§28). The situations wanting stranger pairing — mentoring, an
interview, a contractor's first day — are ones where a call is almost always available,
so what is bought is convenience rather than capability.

**What is not decided.** The mechanism does not forbid it: a host who approves a
request from someone unknown has paired with a stranger, and that is their judgement.
What is excluded is designing for it — no affordance presents it as intended, and no
claim is made that verification protects it.

Also not decided: whether the host-approval path is built at all. Its value is
removing the manual identifier paste between colleagues, and that is a separate
question answered separately. Noted because the two were previously argued as one, and
conflating them is how a mechanism gets justified by a case nobody wants.

**Recorded as open in §12a**: whether a request may arrive unsolicited. Any peer that
can reach the address and name the room could otherwise cause something to appear on
the host's screen, and names are guessable by design (D-017). Guessing grants nothing,
which is what makes names safe; producing an interruption is a different matter, and a
prompt people learn to dismiss quickly is a poor place for a decision that matters.
The alternative — requests accepted only while the host has said they are expecting
someone — is not a token and does not reopen D-026: arriving in the window admits
nobody, it only earns the right to ask.

**Revisit when** there is a concrete use for pairing with someone unknown that a call
cannot serve, or if a verification channel becomes available that does not depend on
recognising the other party. Both would change the argument rather than merely the
appetite.

---

## D-052 — The SAS is its own act, not part of joining or approving

**Date:** 2026-09-17 · **Status:** active (implemented)

**Context.** D-048 accepted ZRTP's short authentication string in principle and put
it "at join, where both daemons are connected." Asked for the two-word approach
directly, and asked whether that was the same thing as the host-approval path
(§12a). It is not, and conflating them would have made a verification feature wait
on an admission feature nobody has decided to build.

**The two are orthogonal.** The approval path is about **how a key arrives** —
pasted after `whoami`, or presented in a request a host approves. The SAS is about
**whether the key that arrived is the right one**, whichever way it came. Building
the second requires nothing of the first.

**Decision.** `cogmer verify <peer>`, run by **both** people at the same time,
on a call. Each CLI asks its own daemon to open a session; each daemon runs
commit-commit-reveal-reveal with the other; both print the same two words; each
person answers whether the other said the same ones.

**Both sides run it, and that is what keeps §12a's open question closed.** A daemon
answers a verification step only when its own user has asked for one — otherwise
409, and nothing is displayed to anyone. So this adds no inbound surface, no prompt
that can be trained away, and no path by which a stranger causes anything to appear
on a host's screen. The question of unsolicited requests stays open in §12a and is
not reached here.

**The exchange is symmetric** — true of a single round, and **not** of the session
around it, which D-059 had to correct after a two-machine run: the side that
finished first tore its session down and stranded the other. Each side sends its
commitment and receives the other's as the reply, so there is no initiator to elect
within a round. The
address is discovered by trying each peer address this machine knows and keeping the
one that answers **signed by the identity asked for** — a reply from a different
peer is a wrong peer, not a wrong address, and is refused.

**Only a person may record a verification.** The exchange proves both sides hold the
keys they named; it cannot establish that the voice on the call is the colleague
rather than somebody in their place. `MarkVerified` is reached only from the
confirmation, never from the protocol. A mismatch records **nothing** — there is no
failed-verification state, because storing one invites an interface that offers to
retry, and retrying is precisely what must not be offered.

**The marker now means something.** `known_peers.verified_at` is NULL until two
people compare words, `FormatTeamContext` takes a predicate, and a verified peer
loses the `unverified` tag inside injected text. Before this the tag was true of
every peer forever, which is a marker a reader learns to stop seeing — the failure
mode §25 cares about, since the model is the reader that reasons about attribution.

**Wordlists.** The PGP biometric word list, 256 words each for even and odd
positions, alternating. Alternation makes a transposition detectable: said in the
wrong order the pair is not a valid rendering of anything. Deliberately **not** the
peer-name or room-name vocabularies — a SAS sits on screen beside a derived peer
name, and one mnemonic mistakable for the other is how somebody compares the wrong
thing. A test asserts the lists are disjoint and exactly 256 unique words each,
since the security argument assumes one byte per word.

**Tests, and the two that matter.** `TestAnInterceptorProducesDifferentWords` is the
property itself: a relayed exchange derives each side's words from a different pair
of identities, and the strings disagree. `TestRevealBeforeCommitIsRefused` and
`TestARevealMustOpenItsCommitment` guard the ordering, which **is** the security —
both confirmed to answer 200 instead of 400 when the check is removed, so a
commitment step that degraded to decoration would be caught. Also covered:
symmetry, per-exchange freshness, list disjointness, refusal of unsigned messages,
refusal of unknown peers, refusal of unsolicited sessions, and a real two-daemon
exchange over HTTP reaching identical words.

**Not done.** The words are 16 bits, so a blind guess succeeds once in 65,536 and
the protection against repetition is that **failure is conspicuous** rather than that
retries are blocked. Nothing rate-limits attempts. That is the right place to look
first if this is ever attacked, and it is recorded rather than fixed because the
conspicuousness argument only holds while a mismatch stops a person — which is a
claim about the interface, not the protocol.

**Revisit when** verification is wanted between peers that cannot be connected at
the same moment. The short form is unavailable there and D-047's full-length
rendering is the only option, so both may need to exist.

---

## D-053 — `pair` is machine scope and `invite` is room scope; the commands now say so

**Date:** 2026-09-18 · **Status:** active (implemented), **except its gating
position, reversed by D-054 the same day** — verification now gates synchronization
and injection rather than merely warning.

**Context.** Asked why the commands are typed at a terminal rather than as slash
commands in a session. The honest answer was that nothing is packaged yet (D-041),
not that it had been decided. Examining which commands *could* move produced the
real finding: `verify` cannot — it needs stdin, it blocks on another person, and its
two words must reach a person's eyes without passing through a model that reads room
content from unverified peers (D-040). Observed in reply that if verification must
live outside the session, the vocabulary should separate peer onboarding from room
operations rather than leave them interleaved.

That is the right cut, and it was already the data model: §12 has kept **known peers**
(per machine, durable) and **a room's guests** (per room) as two lists since D-024.
The commands did not reflect it. `allow` said only that something had been permitted
and never which of the two, and sat in a flat list beside `invite`, which is a
different act at a different scope.

**Decision.** `cogmer pair <identifier>[@address] [name]` is the durable act:
record the peer, record a bootstrap address, and run the two-word comparison, in one
command that both people run at once on a call. `invite` remains room-scoped and
unchanged. `allow` survives as the low-level "record without verifying" for scripts
and tests, and now says **UNVERIFIED** rather than printing a fingerprint and advice
about a ceremony it does not perform.

**Why the naming matters more than it looks.** The split decides where each act can
live. Pairing is interactive and ends in something a person must read exactly, so it
belongs at a terminal. Inviting is one non-interactive act with informational output,
so it can be a slash command inside a session. A vocabulary that ran the two together
would force both into the more restrictive home — which is what "commands are typed
at a terminal" had quietly become.

§12 is retitled from "Session Pairing" to "Forming a Room", because under this
vocabulary *pairing* means peer onboarding and the old title named the wrong act.
Section numbers are unchanged, so every reference still resolves.

**The ordering gap this closes.** `verify` previously found a peer only through
addresses learned from room membership, so verification could not precede the room it
was meant to protect. `known_peers.endpoint` holds a machine-scope address recorded at
pairing, `syncTargets` includes them, and the sequence is now pair → create → invite →
join rather than create → invite → join → verify. §4 already permits this: an endpoint
is a bootstrap hint that need only be correct once.

**Gating — see D-054.** The warnings this entry added to `invite` and `join` remain,
and now announce a gate rather than a caution.

**Revisit when** discovery lands. A peer found on a local network has an address
nobody typed, which changes what a pairing string is for and may reduce it to the
identifier alone.

---

## D-054 — Verification gates synchronization and injection, not just a marker

**Date:** 2026-09-18 · **Status:** active (implemented)

**Context.** Asked whether we plan to admit unverified guests. We did — by omission
rather than by decision. `IsVerified` was consulted in exactly three places: the
marker inside injected text and two command-line warnings. Nothing in `auth.go`,
`sync.go` or `Invite` looked at it, so an unverified guest synchronised normally and
its turns entered a teammate's model context carrying a tag and nothing else.

Answered directly: there should be no exchange of transcript and no injection of
context without verification.

**Why the previous position did not hold.** It rested on D-051 — whom to admit is
the host's judgement — but that is an answer to a different question. Admission
decides whether a key may enter. Verification decides whether the key is the
person's. Every check in the system passes for a key substituted in transit, because
a substituted key is a real key held by whoever substituted it, and D-047
demonstrated exactly that end to end with **zero refusals**. A marker is the right
thing to show once content is in front of a reader; it is not a control, and the one
manual step the whole chain rests on will be skipped if skipping it costs nothing.

**Decision.** Three gates, because there are three ways in:

1. **Serving.** `verifyRequest` refuses a sync request from an unverified peer,
   after the guest check. Not a disclosure — a caller reaching that line is already a
   recorded guest, so it learns nothing it did not put there.
2. **Accepting.** An event whose **origin** peer is unverified is not stored. At the
   origin rather than the sender, so a verified relay cannot launder an unverified
   author (§13).
3. **Injecting.** `onlyVerified` filters again at the prompt hook. Belt and braces:
   an event stored while its peer was verified outlives a later `forget`, and
   injection is the step that cannot be undone, because a context window has no
   delete.

**Held, not discarded.** Sync is a pull against a watermark, so refusing to store
leaves events on offer and does not advance the mark. When the two people verify,
the next poll brings the whole backlog. That is what makes the gate safe to apply
early: nothing is lost by waiting, and nothing has to be re-sent.

**Silence had to be explained.** A room quiet because of a gate is indistinguishable
from a room where nobody is talking, so the refusal names the peer and the command,
the poll logs what is being held back, and `invite` now says that inviting an
unverified peer does nothing until they are verified. A gate nobody can see reads as
a bug and gets debugged as one.

**Inviting an unverified peer remains permitted** and remains inert. D-051 is not
overridden: the host still decides whom to admit. The second gate asks a different
question of a different party, and the two compose rather than compete.

**Tests.** `TestAnUnverifiedGuestIsRefused` builds a request correct in every other
respect — real key, real signature, real guest of a real room — requires refusal,
requires the message to say what to do, and then requires that verifying is what
opens it. `TestUnverifiedEventsAreNotInjected` covers the third gate. Both confirmed
to fail with their checks disabled. The existing `authDaemon` fixture now verifies
its guest, which is itself evidence the gate bites: every authentication test failed
until it did.

**Revisit when** a case appears where two people genuinely cannot verify but must
collaborate. §25's "case with no answer" is the candidate, and the right response is
probably still refusal — but it should be decided against a real situation rather
than in advance.

---

## D-055 — There is exactly one way to verify a peer

**Date:** 2026-09-18 · **Status:** active (implemented)

**Context.** §25 retained the whole-key comparison as a fallback for where no live
exchange is possible. Asked why. There is no good reason, and three against.

**Its only advantage buys nothing.** What distinguishes the static comparison is
working without a live connection between the daemons. But verification now gates
synchronization (D-054), so an unverified peer cannot collaborate; verification
therefore matters only when collaboration is about to happen; and collaboration
already requires both daemons running and mutually reachable — exactly what the live
exchange needs. There is no state in which a peer needs verifying and cannot be
verified by the two-word comparison.

**It is a downgrade path.** A gate is only as strong as the weakest ceremony that
satisfies it, and a second, harder ceremony beside an easier one is not a choice
people make on the merits — it is what gets reached for when the other is
inconvenient. D-047 measured the cost: shown forty-three characters to check over a
telephone, a reader takes the first group, the last group, and skims the middle.

**It did not exist.** `Fingerprint` had no callers outside its own test, and nothing
marked a peer verified from a fingerprint comparison — `verified_at` is reachable
only from the two-word confirmation. So the specification described an affordance the
implementation did not offer, which is the prose form of the false-affordance problem
this project already decided is worse than nothing.

**Decision.** One ceremony. A peer is verified or it is not, and there is one way to
become so. §25 says so, and says why the omission is deliberate rather than an
oversight somebody should helpfully repair.

**What is kept.** Rendering an identifier for a person to *read* remains useful, and
is not a ceremony: when a key has changed and somebody is looking at two of them,
grouped output is kinder than an unbroken run of base64. `Fingerprint` is now
documented as display-only, used in the mismatch alarm, and its test asserts
losslessness rather than fitness for comparison.

Also kept: the reasoning about why a short string is sound where a static one is
not. It is no longer a comparison between two available methods, but it is what makes
the single remaining method defensible, and removing it would leave the two-word
comparison looking like a shortcut.

**Consequence.** D-047 is closed without being implemented. "How should the
fingerprint be rendered for comparison" has no answer once comparing a fingerprint
verifies nothing.

**Revisit when** a case appears where two people must verify and their daemons
cannot reach each other. The honest response is probably still that they cannot
collaborate either — but it should be decided against a real situation.

---

## D-056 — A session's room is fixed at first sight; the `injected` flag is removed

**Date:** 2026-09-18 · **Status:** active (implemented)

**Context.** Initialized CodeGraph and ran a dead-symbol sweep. Two results:
`min`, which duplicated the Go 1.21 builtin, and `HasReceivedContext`, which
`codegraph_callers` confirmed had none.

The second is not dead code — it is a **rule recorded and never enforced**.
`MarkInjected` wrote `session_rooms.injected = 1` after teammate context was
offered; `HasReceivedContext` read it; nothing called `HasReceivedContext`. §12a's
constraint was in fact enforced by something else entirely: `RoomForSession` returns
the bound room unconditionally once a row exists, so a session can never move
whether or not anything reached it.

**The specification was therefore looser than the code, in the direction that
matters.** §12a said a session may not move *once teammate context has been
injected*, which permits moving one that has had none — the ordinary case of joining
the wrong room and correcting it before anything arrives. That narrower rule existed
on paper and nowhere else, and nobody noticed for the same reason it was safe: the
stricter behaviour is what anyone would want.

**Decision.** Keep the strict rule and delete the machinery for the loose one. A
session binds on first sight and stays. `injected`, `MarkInjected` and
`HasReceivedContext` are gone, and `RoomForSession` carries the comment saying it is
the whole of the constraint.

Correcting a wrongly joined room now means starting a session. That is cheaper than
a rule which has to be right about what a context window contains — a judgement made
from the outside, about state that cannot be inspected, where being wrong once is
irreversible.

**The column is dropped, not left.** It would have been harmless: it has a default
and nothing writes it. But a column encoding a rule that was removed is a rule
somebody will later find and reinstate, so `migrateMembership` drops it from
databases that predate this.

**What the sweep says about the method.** This is the second finding of the same
shape in two days — after `Fingerprint`, which also had no callers and also
represented a decision recorded but never wired up (D-055). Both were invisible to
every test, because a test exercises what is called. A symbol with no callers is
worth treating as a question rather than as tidiness: it usually means something was
decided, written down, and then satisfied some other way.

**Tests.** `TestASessionsRoomNeverChanges` states the surviving rule directly: a
bound session stays put when the current room changes, and a session starting
afterwards gets the new one. `TestMigrationDropsTheInjectedColumn` writes a
pre-change database and requires both that the column is gone and that the binding it
carried survives.

**Revisit when** a case appears for moving a session that has received nothing. It
would need a way to know that from outside the session which does not depend on a
flag nobody reads, and the flag is what failed here.

---

## D-057 — A slash command is a thin wrapper over the CLI; the session names itself

**Date:** 2026-09-18 · **Status:** decided; commands not yet built

**Context.** §29 had already split commands between a session and a terminal
(D-053). What it had not answered is how the in-session ones would be implemented,
and the assumption underneath was that they would need a second implementation.

Proposed instead: every slash command simply calls the corresponding CLI command.
One implementation, two entry points, and the CLI remains the surface that can be
tested without a Claude session at all.

**The objection that had blocked this was wrong.** Binding a room to the session
that asked for it appeared to require naming a session from outside one, which
nothing at a terminal can do. Probed rather than assumed, and it is not required:
**`CLAUDE_CODE_SESSION_ID` is exported into the environment of every Bash tool
call**, and it equals the id the hooks report. Verified on 2.1.275 by running a
prompt with a `UserPromptSubmit` hook recording `session_id` while the tool call
recorded the variable — the two strings were identical.

Recorded as **B21**, because it is undocumented, load-bearing, and has a silent
failure mode: a variable that is present but names a *different* session would bind
a room to a session that does not exist while the real one binds to nothing, and
capture would stop with no error anywhere.

**Decision.** Slash commands shell out. They pass no session id, because the CLI
reads `CLAUDE_CODE_SESSION_ID` from its own environment.

That yields the property that motivated the question — nobody would ever run
`create` from a terminal — **by construction rather than by convention**. The
variable is absent at a terminal, so a session-scoped command run there has no
session to bind and refuses. It does not guess, and it does not fall back to a
machine-level setting.

**What it allows us to delete.** The machine-level *current room* exists only
because a terminal command cannot name a session. Once `create` and `join` bind the
session that invoked them, a session that has run neither is in no room, which is
already what §12a wants — *before a room exists, a session is an ordinary Claude
Code session*. That removes the hazard behind the original question: today a room is
created, forgotten, and a session started weeks later in an unrelated repository
silently joins it and begins publishing. Nothing about the directory scopes it,
because D-015 forbids deriving a room from one.

**Not every command can be wrapped**, and §29 already says which: `pair` and
`verify` are interactive, block on another person, and must reach a person's eyes
unaltered. Their slash counterparts print an instruction to run them in a terminal.
A signpost is honest in a way a proxy would not be.

**Two hazards to implement against.**

- **The model may retry.** A wrapper is a model deciding to run a command, and a
  model that reads a timeout as a failure may run it twice. `invite`, `join` and
  `revoke` are idempotent already; `create` is not, and two rooms is a confusing
  outcome rather than a harmless one. Either make creation idempotent per session —
  a session that already has a room gets that room back — or have it refuse.
- **Testing must not require a session.** Session-scoped commands accept an
  explicit override so tests can pass a synthetic id; the environment variable is
  the default, not the only source.

**Revisit when** B21 is recorded as failing. At that point a session-scoped command
cannot know its session, and the answer is to refuse rather than to reinstate a
machine-level current room — which would restore the hazard this removed.

---

## D-058 — Signature schemes are kept, never replaced

**Date:** 2026-09-18 · **Status:** active (implemented)

**Context.** Asked whether anything in the current approach would make backwards
compatibility hard once a second host type exists. The adapter boundary turned out
not to be the problem — it is clean, and `behaviors.go` already documents the
interface. The problem is one layer down and has nothing to do with hosts.

`signingBytes()` hard-coded a single event tag and a fixed field list,
and `Verify()` always recomputed with **today's** code. Adding a field to an event —
which a second host would plausibly require — would therefore have stopped every
historical event verifying.

**Why that is unfixable rather than inconvenient.** Events are immutable (§7), so
they can never be re-signed. And they do not sit still: §13 relays them between
peers, and **D-029 makes refetching a room's history the designed recovery from
local loss**. So after a format change, old events would still move between peers
and be rejected on arrival — and the message would read `signature does not match
the peer id that claims to have made it`, which sends someone hunting an attacker
rather than installing a build.

**Decision.** An event records the scheme it was signed under, and is verified under
that scheme. Old `signingBytes` implementations are kept rather than edited. Adding
a field to an event means adding `signingBytesV3` and leaving v2 intact.

Three details that make it work:

- **Zero means v2.** A row written before the column existed, and a peer on an older
  build that omits the field, both read as the only scheme that existed then. No
  migration rewrites anything, which matters because rewriting a signed record is
  the operation §7 forbids.
- **The version is not covered by the signature.** An attacker who alters it only
  causes verification to fail. Every scheme is Ed25519 over length-prefixed fields,
  so there is no weaker scheme to be downgraded to; if one is ever found weak, the
  answer is to refuse that version rather than to have signed its number.
- **An unknown scheme is refused as a version problem, not a forgery.** The two
  demand opposite responses from a person — install a newer build, versus somebody
  is attacking you — and a test asserts the message says "upgrade" and does not say
  "does not match the peer id".

**The related hazard, recorded and not fixed.** `wireVersion` is a hard refusal:
a peer speaking a different version is rejected outright, so there is no rolling
upgrade. Combined with an event-format change, an upgrade becomes a flag day. That
is tolerable now, when every peer is on one machine's build, and will not be once
anyone else uses this. The likely answer is a minimum-compatible version that allows
reads from older peers.

**What was already right**, and worth not disturbing: `originSessionId` rather than
`claudeSessionId` (D-043), nothing on the wire naming the agent, host-specific
values like `promptId` living in free-form `metadata` rather than as columns, and
`EventType` being an open string whose only consumers test for one specific value —
so an unknown type renders generically instead of failing.

**Revisit when** a second scheme is actually added. That is the moment the mechanism
is first exercised, and the test to write then is that an event signed under v2 by
an older build still verifies against a build that signs v3.

---

## D-059 — Only one side drives a verification; the other completes from inbound

**Date:** 2026-09-18 · **Status:** active (implemented)

**Context.** Phase 5's two-machine run. Pairing failed twice before it worked, in
two different ways, and neither was reachable from one machine.

**First: whoever typed first lost.** The droplet sent its commitment four seconds
before the Mac had recorded it, so `handleVerify` answered `401 Unauthorized` —
an unknown peer. `RunVerification` retried only on 409 and treated 401 as final.
The earlier caller failed at once; the later one waited ninety seconds and timed
out. Both people fail, and the one who followed the instruction promptly fails
faster.

Fixed by answering **409 for "I do not know you yet"**, a state the other person
resolves by typing. 401 now means only that a signature did not verify, which is
the one answer here that waiting cannot fix.

**Second, and deeper: the side that finished stranded the other.** Both sides drove
their own exchange. The one that completed tore its session down in a `defer`, so
the other's commitment arrived to nothing and waited out the timeout. It reads as
symmetric and is not: it fails whenever two people type a few seconds apart.

D-052 described the exchange as symmetric — "each side sends its commitment and
receives the other's as the reply, so there is no initiator to elect and no race to
resolve." That was true of a single round and false of the session around it.

**Decision.** Either side may complete from **inbound**. A peer whose own session
already holds the other's revealed nonce has everything the SAS needs and stops
driving. Only one side has to run the exchange; both learn the words. The loop now
checks inbound first, then attempts a round outbound, so the two orderings and the
simultaneous case all converge.

**The security argument is untouched.** Commit-commit-reveal-reveal still holds,
because the inbound path only reads a nonce that arrived through `handleVerify`,
which refuses a reveal without a commitment and refuses a reveal that does not open
it. What changed is who runs the loop, not what the loop requires.

**Why a test did not catch it.** `TestTwoDaemonsReachTheSameWords` starts both
sides in goroutines with no delay, so neither finishes before the other begins —
the one arrangement in which the bug cannot occur. The new test introduces the
delay that makes it ordinary rather than rare.

**Revisit when** a third peer verifies. Nothing here assumes two, but nothing has
exercised more.

---

## D-060 — A sequence is reserved outside the room before the event that uses it

**Date:** 2026-09-18 · **Status:** active (implemented)

**Context.** Review C-1, the last open conformance item, and Phase 7's stated
"database recovery". D-029 settled the design a fortnight ago: **lose the room,
keep the sequence, resume above it and refetch.** None of it was built.

`rooms.issued_sequence` existed in the schema and was written by nobody and read by
nobody — the third such column found this week, after `injected` (D-056) and the
verification flag before it. `nextSequence` derived the next sequence from
`MAX(peer_sequence)` in the room's own database, which is exactly the value that
goes to zero when that database is lost.

**What that cost.** Delete a room database and the counter restarts at 1, so the
peer reissues numbers it has already used. A reissued sequence is a *different
event under an identifier other peers already hold* — the precise condition D-027
quarantines on the receiving side, caused by a peer that is never told. Reproduced
during the review: the daemon kept serving from its open file handle after the file
was deleted, so nothing looked wrong until a restart hours later, and then the room
simply went quiet.

**Decision.** `Membership.ReserveSequence` issues the number and records it in
`membership.db`, beside the identity and outside the room. `Store.Append` takes the
sequence rather than deriving one, so the dangerous path is not merely unused but
absent.

**Reserve, then publish**, and not the reverse. A crash between the two then loses a
number instead of reissuing one, and losing one is harmless: the watermark is the
highest *contiguous* sequence, so a gap makes peers wait rather than skip.
`TestAReservationIsSpentEvenIfNothingIsWritten` fixes that order.

**The loss is reported, because it is otherwise invisible.** `reportLostState` runs
when a room is opened — the only moment it can, since a running daemon serves a
deleted file from its handle. It says what §8 requires: state was lost, membership
and sequence position are intact, history is being refetched and depends on a
member being reachable, and teammate turns may be injected a second time.

**Recovery needed no new mechanism.** Anti-entropy already refetches: a store
holding nothing reports low watermarks, and peers resend. What was missing was only
that the peer not corrupt the room on its way back.

**Also fixed, from the Phase 5 findings.** `log`, `conflicts` and `seed` resolved a
room through `config.json`, which D-046 replaced. During a live room holding seven
events, `cogmer log` printed `ROOM DEFAULT -- 0 events`. They now use the
current room from `membership.db`. `whoami` deliberately does **not**: who you are
is answerable in no room at all, and a command reporting your identity must not
fail for want of one.

**Revisit when** a peer needs to reserve sequences for a room it has not joined, or
across two machines under one identity. Neither is possible now, and the second
would need the reservation to be shared rather than local — at which point this
becomes a distributed counter and the argument changes entirely.

---

## D-061 — Phase 7's last three: one dissolved, two built

**Date:** 2026-09-18 · **Status:** active (implemented)

**Context.** Phase 7 listed a local outbound queue, peer health, and context-size
controls. Taking them in turn produced one deletion and two findings.

### The outbound queue is dissolved, not deferred

It presupposed push: something must hold what a peer has not yet been told.
Synchronization is a pull (§10, D-031), so an event is durable the moment it is
committed locally and "what did you miss" is a question asked rather than state
anyone keeps. A queue would be a second record of what the event store already
holds, and the two would drift — the first time a peer was dropped from one and not
the other, silently.

Recorded in §7 as struck through with the reason, rather than removed. An item that
vanishes reads as forgotten.

### Peer health: report transitions, not polls

The sync loop logged `unreachable` once per address per poll. At a one-second
interval that is two lines a second saying the same thing, which was observed
filling a log through the Phase 5 partition. **A message repeated that often is not
a signal — it buries the one line that matters, which is the transition.**

Outages are now reported when a peer stops answering, again every ten minutes while
it continues, and once when it returns with how long it was gone. The repeat exists
because somebody reading a log an hour later should learn the peer is *still* gone
rather than only that it once went.

Also fixed: `peerStatus` enumerated only `COGMER_PEERS`, so every peer learned
by pairing or by joining a room was **invisible in the browser view** — which since
D-053 is most of them. It now reports every address the daemon would try.

### Context size: the limit that was missing bounded nothing

`maxInjectedEvents` capped turns and `maxInjectedChars` capped each turn, and
neither bounds the block. Forty turns of eleven thousand characters pass both.
Measured: **404,635 characters** would have been injected into a context window.

A whole-block budget now applies after the event cap, measured on the rendered
turns rather than estimated from their inputs, dropping from the front so the newest
conversation survives — a referent is most likely to point at recent turns. All
three limits are configurable, as §21 asked and as none of them were, and a limit
that parses as zero or as nonsense is ignored rather than obeyed: injecting nothing
looks exactly like a room where nobody is talking.

§21 is amended to say what each limit is *for*, since the one that was missing is
the one the list made sound redundant.

**On estimated token count**, which §21 also asked for: derived from characters at
four to one rather than measured. A tokenizer would have to track a model this
system does not choose, and the figure is wanted for judgement rather than
arithmetic.

**Revisit when** a room routinely exceeds the block budget. §21 treats these as a
safety valve, and under session-scoped rooms hitting one means an unusually long
pairing — so hitting one *often* means the rooms are not as session-scoped in
practice as D-015 assumes, which is worth knowing.

---

## D-062 — Tailcat evaluated for Phase 15: a good fit, adopted behind an interface if at all

**Date:** 2026-09-18 · **Status:** **adopted** — see D-068 for what was built and
what remains unproven. Was: investigated, not yet adopted, but the likely answer
rather than a contingency — D-063 established that the first pair are
remote, so this is the only case that matters. Its costs are prices to be paid, not
risks to be weighed.

**Context.** Phase 15 needs two peers behind different NATs to find a direct path.
Every cross-network run so far — Phase 2's and Phase 5's — used an SSH tunnel,
which is a person performing NAT traversal by hand and is not something a colleague
can be asked to do. Tailscale published **tailcat** in August 2026: Tailscale's data
plane (WireGuard, magicsock, DERP) with **no control plane, no account, no tailnet**.

**What it is.** A Go package and CLI. A server generates a keypair, picks a DERP
relay, and produces a `tc…` address carrying its public key, a path-discovery key
and DERP bootstrap info. A client dials that address through DERP; both sides then
attempt direct UDP, with DERP remaining as fallback. Userspace throughout — gVisor
netstack and wireguard-go — so no root, no TUN device, no routing-table changes.

**Why it fits this design unusually well.** The API is `net.Conn`-shaped:
`Server.OnTCP(port) func(net.Conn)` and `Client.DialTCPPort(ctx, port)`. Sync is
plain HTTP over TCP, so the integration is `http.Serve` over one and a
`DialContext` over the other. There is no protocol to redesign.

And it lands exactly where D-019 said a transport must: **under everything.** A
tailcat connection carries bytes and decides nothing. A sync request is still signed
(D-044), still refused unless its peer is a guest (D-045), and still refused unless
a person has verified that peer (D-054). D-019 named Tailscale explicitly as "one
provider among several, never a prerequisite", and this is the version of Tailscale
that can actually be one.

**Two distinctions to hold, because both look like contradictions and are not.**

A `tc…` address is something a person can hold, and holding it gets you a
connection. D-026 says nothing a person can hold admits them. Both are true: the
address reaches the **door**, and the guest list and the two-word comparison decide
whether anyone comes in. An intercepted address is worth what an intercepted IP
address is worth today.

Tailcat has its own WireGuard keypair. That is a **transport** identity and must
never be confused with a `peerId`, which is an Ed25519 key (D-020) and is what
signs events and is what a person verifies. Two keys, two jobs.

**Do not use `AllowedClients`.** Tailcat can restrict connections by peer public
key, which would be a third allowlist, keyed on a different key type, needing to
agree with `known_peers` and `room_guests`. Two lists that can disagree is precisely
the defect behind the asymmetric guest list (D-046). One authority.

**Measured, not assumed.** Built against a trivial program, `CGO_ENABLED=0`, all
four targets:

| | tailcat hello-world | `cogmer` today |
|---|---|---|
| darwin/arm64 | 23.3 MB | 16.3 MB |
| linux/amd64 | 25.0 MB | |
| linux/arm64 | 23.3 MB | |
| windows/amd64 | 24.5 MB | |

Pure Go on every target, so cross-compilation survives. The cost is size and supply
chain: **526 dependencies against our current one**, and a binary likely to roughly
double. For something a colleague installs, both are real.

**Three risks to weigh at Phase 15, not now.**

- **No stability promise.** "The Go API, CLI flags and output, and wire format may
  all change." Load-bearing code behind an unstable API is why it should sit behind
  a narrow interface of our own — a Dial and a Listen — which costs almost nothing
  to write now and a great deal to retrofit.
- **DERP is a rendezvous dependency.** Direct paths are peer-to-peer once
  established, but two peers behind NAT cannot *meet* without reaching a relay. That
  is not a control plane and it is not local-first either. Self-hosting DERP is
  possible and is a server, which is the thing this project avoids. The honest
  framing: the cross-network case has always needed something in the middle, and
  DERP is a more honest version of the SSH tunnel we have been using.
- **Public relays are rate-limited with no SLA.** Room traffic is text and small, so
  throughput is unlikely to bind; availability might.

**What would make it the answer** was whether Phase 13's first outside user is
remote. **They are** (D-063): colleagues at one company, both working from home,
pairing daily. So Phase 15 is a prerequisite rather than a detour, and the
alternatives are all worse here — a company VPN cannot be assumed, a public bind is
not available from a home connection, and an SSH tunnel is a person performing NAT
traversal by hand twice a day.

The DERP objection also weakens for this pair specifically, which matters because it
was the strongest one. The relay is for rendezvous and can be self-hosted; a droplet
already exists and has carried both cross-machine runs. A relay this pair controls,
used to meet and as fallback with the working path still direct, is a different
thing from depending on a third party — and from the central server §3.3 rejects.

**Revisit when** Phase 15 begins, or if tailcat reaches a stable API. Re-measure the
binary then: a figure from a hello-world is a floor, not the number that matters.

---

## D-063 — The first pair is remote, so cross-network reach comes before local discovery

**Date:** 2026-09-18 · **Status:** active (ordering)

**Context.** The phase order placed local discovery (Phase 12) before cross-network
reach (Phase 15), with 15's position marked conditional on "whether the first person
who is not the author sits on the same network". That fact is now known: the first
real peer is a colleague at the same company, both working from home. **They will
never share a network.**

**What that invalidates.** D-019 made local discovery the first transport to build,
reasoning that "two people on the same network is the simplest case and must be
the easiest". The claim is true and it is not about anybody who will use this. A
zero-configuration path that serves nobody is not a zero-configuration path; it is
an unused feature with a good argument behind it.

D-019's *requirement* is untouched and should not be read as weakened: no provider
belongs in room identity, membership or replication, and none may be a prerequisite.
The same-network case must still cost nothing when it arises. Only the build order
changes.

**Decision.** Phase 15 moves ahead of Phase 12 and becomes a prerequisite for Phase
13 rather than a conditional detour. Local discovery is not cancelled — it is the
right answer for a case that will arrive, and it is cheap once the transport seam
exists. It is simply no longer first.

**What this does to the tailcat evaluation (D-062).** It moves from "a good fit for
a case that may not arise" to the likely answer for the only case that matters, and
its costs move with it: 526 dependencies and a roughly doubled binary are now prices
being paid rather than risks being weighed. The offsetting fact is that every
alternative is worse here. A company VPN cannot be assumed. A public bind is not
available from a home connection. An SSH tunnel is what Phases 2 and 5 used and is a
person performing NAT traversal by hand, twice a day, forever.

**The DERP objection weakens for this specific pair**, which is worth recording
because it was the strongest one. Tailcat's relay dependency is for *rendezvous*,
and the relay can be self-hosted. A droplet already exists and has hosted both
cross-machine runs. Self-hosting DERP is a relay this pair controls, used to meet
and as fallback, with the working path still direct peer-to-peer — which is a
materially different thing from depending on a third party's infrastructure, and
from the central server §3.3 rejects.

**What has not changed.** A transport still decides nothing (D-019, D-062): a sync
request is signed, refused unless its peer is a guest, and refused unless a person
verified that peer. Reaching the door is not admission.

**Revisit when** a second pair appears who do share a network, or when the same-
network case becomes the common one. Phase 12 is then worth its cost, and the seam
this phase builds is what makes it small.

---

## D-064 — A session's room is the one somebody chose inside it, never a machine default

**Date:** 2026-09-18 · **Status:** active (implemented) · **reconstructed 2026-09-21**

**This entry was never written.** The number was allocated while writing `e4fd2d8`
("Phase 11: ship as a plugin, and bind sessions explicitly", 2026-09-18), which
cited it in six files and did not touch this document. What follows is
reconstructed from those citations, the code they annotate, and the tests that
enforce it. The rule and its hazard are recoverable exactly. **The alternatives
actually weighed are not**, so the rejected list below states only what the code
rules out, and should not be read as a record of deliberation.

**Context.** One pointer was answering two different questions.

- *Which room is this session in?* — `session_rooms`, written only when somebody
  runs `/room-create` or `/room-join` **inside that session**.
- *Which room does a command at a terminal act on?* — a machine-level
  `current_room`, for the case where there is no session to ask.

Answering the first with the second is the conflation this decision names.

**Decision.** `RoomForSession` reports the room a session was put in and never puts
it in one. A session joins a room because a person ran a command inside it, and a
session in no room is an ordinary Claude Code session: nothing captured, nothing
injected, nothing shared (§12a).

**The hazard is silent, which is why it needed a decision.** Otherwise a room
created and forgotten is joined weeks later by a session in an unrelated
repository, capturing and publishing with nobody having done anything. Nothing
derives a room from a directory (D-015), so nothing else would have caught it. The
failure has no error and no moment: its first sign is a colleague reading turns
from work they were never shown.

**Enforced by a test rather than by care.** `membership_test.go:424` names itself
"the hazard D-064 removes, stated as a test so it cannot come back."

**What became of the other pointer.** D-076 placed `COGMER_ROOM` above the
session's own room, reinstating this failure at higher precedence the same
afternoon. D-077 gave every room its own URL, removing the pointer's last consumer,
and D-080 deleted it. `current_room` no longer exists anywhere. Its schema comment
outlived it and described the removed behaviour as the design, which is most of why
this entry was hard to recover; the comment is gone as of 2026-09-21, along with the
`settings` table it annotated, which nothing had ever read or written.

**Rejected** — what the code rules out, not what was considered:

- *Derive a session's room from its working directory or repository.* Independently
  forbidden by D-015 (rooms are session-scoped).
- *Let a machine-level default apply only when a session has no room.* That is the
  conflation itself, in the one case where it does the damage.

**Revisit when** something needs a room and genuinely has no session to ask. That
case is a command at a terminal, and D-077 answers it by naming a room per
invocation rather than by storing one.

---

## D-065 — The daemon reads a range of wire versions, so upgrading is not a flag day

**Date:** 2026-09-18 · **Status:** active (implemented) · **reconstructed 2026-09-21**

**This entry was never written**, for the same reason and in the same commit as
D-064. Reconstructed from `protocol.go:19-27` and `sync.go:250-256`, which carry
the reasoning nearly in full.

**Context.** `wireVersion` names the protocol a build speaks. Comparing it for
equality is the obvious implementation and makes every protocol change a flag day:
both people must upgrade at the same moment or the room goes silent, and the
failure names a version rather than saying what to do.

**Decision.** A build declares the newest version it speaks (`wireVersion`) and the
oldest it can still read (`minWireVersion`); `speaks` accepts anything between.
Today that is 2 and 1. A peer outside the range is reported rather than guessed at,
because silently accepting unknown-shaped events is how a field comes to mean two
things — and the error says which side is older and what to do about it.

**Zero means one.** A peer predating the field spoke v1, so an absent version reads
as 1 rather than as unknown.

**Why this stopped being optional at Phase 11.** A flag day is tolerable while one
person builds both sides, because both sides upgrade when he says so. Phase 11
shipped the plugin, which is the point at which the two sides belong to two people
upgrading on their own schedules. The constraint did not change; the number of
people did.

**Why the floor moves rarely.** Raise `minWireVersion` only when an older version
genuinely cannot be understood — a statement about the events on the wire, not
about tidiness. Raising it to delete a branch is how the room goes silent for
whoever upgrades last.

**Rejected** — what the code rules out, not what was considered:

- *Hard equality on the version.* The flag day above.
- *No version on the wire at all.* Then there is nothing to refuse, and an
  unknown-shaped event is accepted as though understood. D-043 (the wire format is
  not the database row) keeps the two structs separate precisely so a column added
  for local bookkeeping cannot become protocol by accident; a version is what makes
  that refusable at the far end.

**Revisit when** a change cannot be expressed so that a v1 reader can skip it. The
floor rises then, the range narrows, and whatever announces a release has to say
so.

---

## D-066 — The binary is fetched and verified, never shipped in the plugin

**Date:** 2026-09-18 · **Status:** active (implemented); inert until a release exists

**Context.** §29 requires that a participant installs one thing. The hooks call a
compiled binary, and nothing put it on the machine. Three candidates: ship binaries
in the plugin repository, fetch them at first run, or build from source.

**What the ecosystem does — stated more carefully than it first was.** Of 53
plugins in the official marketplace, not one ships a binary. That is true and it
proves much less than it sounds like, because **almost none of them has one**: four
run `bun run --cwd ${CLAUDE_PLUGIN_ROOT}` over TypeScript sitting in the plugin
directory, two are npm packages, one is Python, one runs the user's own PHP project.
The runner there is an interpreter for code already present, not a distribution
strategy. Eight of nine never faced this problem.

The signal that does survive is the ninth. `terraform` is the only plugin with a
genuinely compiled server, and it runs `docker run hashicorp/terraform-mcp-server:0.4.0`
— a registry, content-addressed and version-pinned, rather than the repository that
describes it. Where a compiled artifact exists, it comes from somewhere with content
addressing.

**The arithmetic.** Five targets at 11 MB stripped is 56 MB of binaries. Every
install would download all of them to obtain the one it can run, because four fifths
are for platforms that machine is not — against a plugin that is otherwise about
100 KB of text. Tailcat would roughly double it (D-062).

A git repository also keeps every version of every file forever, and binaries do not
delta-compress, so ten releases would accumulate half a gigabyte of history. Whether
an installer pays that depends on clone depth, which was asserted here before it was
checked and is **not** established: the official marketplace turns out not to be a
git clone at all — it arrives as a content-addressed archive with a `.gcs-sha` — and
no git-sourced plugin was installed locally to measure. The 56 MB per install stands
on its own and needs none of it.

The runner pattern is also unavailable to us, and for a reason already recorded:
`npx` needs Node, and the README says plainly that Claude Code ships as a native
binary and a teammate may have none. Choosing Go with pure-Go SQLite was precisely
to avoid a runtime dependency; reintroducing one as the *delivery* mechanism would
undo it at the last step.

**What a registry would have given us, and what we gave up.** Version resolution for
free, the platform matrix handled by somebody else, provenance and revocation, and a
fetch path corporate networks already proxy. Checksum pinning is a thin hand-rolled
substitute for the third of those. Set against it: `npx pkg@latest` re-authorises
whatever the registry serves on every single launch, with no pin at all. The trade
is not convenience against rigour in one direction — we lost provenance
infrastructure and gained a pin they do not have.

And the framing that resolves it: GitHub Releases with checksums **is** the Go
convention, which is what goreleaser exists to do. We follow our language's norm and
they follow theirs. The divergence is downstream of choosing Go, which was decided
for a reason that still holds.

**Decision.** Fetch at first run into `~/.cogmer/bin`, and **run nothing that
cannot be verified**. `plugin/checksums.txt` is committed to the plugin repository
and is the only thing that authorises execution. A download whose hash is not
listed is deleted rather than run, and `scripts/release.sh` generates the file from
the bytes it just built, because a hash typed rather than computed authorises
something nobody has seen.

That is the same shape as the rest of this design: the thing that arrives is
checked against something that came by a different path. The check is weaker here —
release assets and the plugin repository are both on GitHub — but altering a
committed file leaves a trace in history where a swapped release asset would not,
and that is a real difference rather than a comforting one.

**Three lessons taken from the only comparable bootstrap in the marketplace**, whose
comments record them as bugs it already paid for:

- **Install outside the plugin directory.** A plugin update must not discard a
  working binary and re-fetch it; the two change on different schedules.
- **Several sessions start at once, routinely.** Two concurrent installs writing one
  path is a corrupt binary. `mkdir` is the lock, being atomic everywhere; the losers
  exit silently, because being second is not an error.
- **Never retry a failure every session.** A machine with no network and no Go would
  otherwise spend an attempt on every session forever. One hour's cooldown.

And one of our own: it runs **detached**. §29 requires that starting must not delay
a session, so nothing waits on a download — a session that begins before the binary
exists simply has nothing to inject yet, and the next one has everything.

**Verified by running it, including the case that matters.** A local release server
produced a correct install; a tampered asset was refused and deleted with the hashes
named, nothing was installed, and the process still exited 0 (§3.1). Five concurrent
installs produced exactly one attempt. A second run inside the cooldown added
nothing to the log.

**It is inert today, and deliberately so.** `checksums.txt` carries no entries until
a release is published, which disables downloading entirely — the intended failure
direction, since no binary is better than an unverified one.

**A finding that changes what must happen next.** The source-build fallback assumes
a module path a colleague can reach, and this repository is private: `go install`
fails for anyone but its owner. So for the first real pair, a published release with
pinned checksums is not the preferred path, it is the **only** one. Phase 13 depends
on it — if the binary is hand-delivered, the first-five-minutes test measures
something that will never happen again.

**Revisit when** a release exists and the first colleague installs. What to watch:
whether the detached install completes before they try to use it, and what the
session says in the window where the plugin is present and the binary is not.

---

## D-067 — Release assets are served from the droplet; the host is data, not code

**Date:** 2026-09-18 · **Status:** active (implemented, v0.1.0 published)

**Context.** D-066 built verified acquisition and left it inert: no repository
existed to release to. Worse than absent — **there is no git remote at all**, so
`github.com/Blue-Rocket/cogmer` is a module path inherited from `go.mod` rather
than somewhere anything lives. And GitHub release assets inherit repository
visibility, so on a private repository the download URL returns 404 to an
unauthenticated client while `go install` fails for the same reason. Both paths
dead, for the same cause.

**Decision.** Serve the assets from the droplet that already exists, and treat
GitHub Releases as where this goes when the code is public.

**The host is data.** `plugin/release-url.txt` sits beside `plugin/checksums.txt`
and `plugin/VERSION`, all three committed together. Moving hosts is then one commit
that changes the URL and the hashes at the same time. A wrong host costs a failed
download; hashes that did not move with it would cost a refusal nobody could
explain.

That also decides where the seam goes. `scripts/release.sh` builds and hashes and
is host-agnostic; `scripts/publish.sh` puts the files somewhere and is not. When
this moves to a public release host, publish.sh is replaced and release.sh is
untouched.

**Plain HTTP, and the checksum is why.** The binaries are not secret, and what
authorises running one is a sha256 pinned in the plugin rather than the transport.
A tampered response is refused exactly as a tampered file is — demonstrated in
D-066. HTTPS would still be better and is one of the things a public release host
would hand us for free; until then the pin is load-bearing rather than
belt-and-braces, and should not be quietly removed as redundant.

**publish.sh verifies what the host actually serves**, fetching each asset back and
hashing it against the pinned value. A publish that succeeded while the host served
something else is the one failure nobody would think to look for, and it is
invisible until a colleague's install refuses a binary for reasons on their machine
rather than yours.

**Verified end to end, from a clean state with no environment overrides.** A fresh
install fetched and hashed and landed 11 MB; a second run added nothing to the log;
bumping the version and republishing was picked up and replaced the binary in
place. nginx serves one directory with autoindex and nothing dynamic.

**What this costs.** The distribution host is now a machine you run. If it is down
when a colleague installs, they get no binary and retry in an hour — degraded
rather than broken, which is the right direction, but it is an availability
dependency that a release host would not be. It is also one more thing to remember
exists.

**Revisit when** the code goes public. GitHub Releases then gives HTTPS, a CDN,
provenance, and no host to run — and the change is `release-url.txt`, the checksums
regenerated for the same bytes, and a different publish.sh.

---

## D-068 — Tailcat is the cross-network transport, behind our own dialer

**Date:** 2026-09-18 · **Status:** active (implemented); **NAT-to-NAT still unproven**

**Context.** Phase 15. Two people working from home are behind two routers and
neither can bind an address the other can reach (D-063). Phases 2 and 5 substituted
an SSH tunnel, which is a person performing NAT traversal by hand.

**What was verified, stated precisely because the first version of this was
overstated.** A spike connected a home Mac to the droplet in 291 ms with nothing
configured but one address — and **the droplet has a public IP, so that was
NAT-to-public, which is an ordinary outbound connection.** It proved tailcat builds,
runs and handshakes; it proved nothing about NAT traversal.

The full integration was then run between the two machines with **no tunnel**:
pairing (the two-word exchange over the tunnel), room creation, invitation, joining,
capture, and convergence on both sides. DERP carried the introduction and the path
upgraded to direct UDP — `now using <droplet>:34746`. Zero unreachable errors.

**Still not tested: neither side able to accept inbound.** That is the case that
exists, and a stateful firewall on a public IP is a poor imitation of a consumer
router — no symmetric mapping, no CGNAT, no ALGs. It will be tested with the real
peer, on real routers, and until then the claim here is that our integration works
between two machines, not that NAT traversal does.

**What is nearly certain regardless.** Any router permits outbound connections,
which is all a relay needs, so the floor is a working path with relay latency.
The uncertainty is how often the upgrade to direct succeeds, which is a question
about latency rather than correctness, and for turns of about a kilobyte is unlikely
to matter.

**Two defects found by running it, neither visible from reading it.**

**A client is not a connection.** `clientFor` built a fresh `tailcat.Client` per
dial, and constructing one starts a userspace WireGuard stack, probes the network,
chooses a DERP region and handshakes — seconds of work. At a one-second poll every
round paid first-contact cost and none ever finished: the first attempt was still
handshaking when the next began, and every sync timed out against a peer that was
plainly reachable. Clients are now cached per peer for the life of the daemon, and
the sync timeout is 30s to bound first contact rather than the ordinary round.

**A timeout is the signature of the packet filter.** Connections reached netstack
and `OnTCP` was never called. The answer was in tailcat's source: *"Unlike OnTCP's
nil-handler response, packets dropped by the filter get no RST; a client dialing a
filtered port times out."* `ServedTCPPorts` is now set explicitly. Two gates that
fail differently — the filter drops silently, `OnTCP` returning nil resets promptly
— and knowing which one is refusing is the difference between a diagnosis and a
week.

**A third defect, found and unrelated to tailcat.** The hook payload field is
`session_id`, and a hand-written test harness had been sending `sessionId`. Under
the old binding rule an empty session id was silently bound to the current room, so
it appeared to work — **including in the Phase 5 two-machine run, whose events were
all captured under an empty origin session.** That run's convergence finding stands,
because convergence is what it measured; its session attribution was degenerate and
injection would not have worked. D-064's explicit binding is what surfaced it.

**Costs, measured.** 566 dependencies against 40 before; 21 MB stripped against
11 MB. Both are prices D-063 already accepted, and both are worth restating at the
point they were actually paid.

**Confined to one file.** Tailcat promises no API stability, so `tailcat.go` holds
all of it behind the `Dialer` interface and a `net.Listener`. One set of routes
serves both transports, which is what makes "a transport decides nothing" true
rather than merely intended: a tailcat request passes the same signature check, the
same guest list and the same verification as a TCP one, because it arrives at the
same handler.

**Revisit when** the real peer test runs. What to watch: whether the path goes
direct or stays on the relay, and what a relayed round trip costs a turn.

---

## D-069 — Signing namespaces and the state directory are decoupled from the name

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Asked to push the repository, which meant choosing a name — and the
name was not settled. Mapping where the product name was load-bearing turned up two
couplings that are free to break today and permanent after the first real room.

**The signing namespaces.** Every signature covered a domain-separation string with
the product name inside it: the event tag, the sync-request tag, and the SAS and
verify tags. D-058 settled what a change to those costs — schemes
are **added, never edited**, because an event is immutable and can never be
re-signed and D-029 makes refetching history a recovery path. So a rename after
real events exist would mean carrying the old namespace forever, for a name nobody
uses.

They should never have carried a product name. Domain separation needs **stability
and uniqueness**; it does not need meaning. `protocolNamespace` is now `peer-room`,
documented as arbitrary on purpose so there is never a reason to change it.

**D-058's mechanism got its first real use, which is the point of having built it.**
`signingBytesV3` carried the new namespace while the superseded scheme stayed
untouched beside it, serving every event an older build had signed. (D-117 later
deleted that scheme, on the ground that no such event existed anywhere; the
mechanism that would have carried it is unchanged.)

The other three tags moved outright rather than gaining a version. They protect a
live exchange — a sync request, a verification — and nothing signed under them
outlives it, so there is no history to keep faith with.

**The state directory.** `~/.cogmer` is where the name reaches the filesystem.
It is now a single constant, so a rename is one line rather than a search.

**And a latent bug found by looking.** `install.sh` honoured `COGMER_HOME`
while the binary ignored it, so anyone setting it got a binary in one place and its
state in another, with nothing saying so. It was invisible because the only thing
that set it was my own testing, which set `HOME` as well and so never noticed.

**What is deliberately not done.** The module path still says
`github.com/Blue-Rocket/cogmer`, and the organisation is actually
`Blue-Rocket` — a path that has never resolved, which is why `go install` failed in
D-066's test. It is left alone because a module path must match the repository URL,
and that needs the name. Nothing external imports it, so it costs nothing to wait.

**Revisit when** the name is settled. What changes then: module path, binary name,
plugin name, state directory, release URL path, and the `/team-*` commands only if
the prefix is wanted differently — they carry no product name already. What does
**not** change, by construction, is anything cryptographic.

---

## D-070 — Slash commands carry a distinctive prefix, because invocation is not namespaced

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Observed that command names need namespacing to avoid colliding with
other plugins. Checking the mechanism rather than assuming one: a subdirectory
changes how a command is **displayed** — `/build (project:ci)` — and not what a
person types. There is no `plugin:command` invocation form.

So two plugins defining the same name collide at the only thing a user touches. This
is not hypothetical: in the official marketplace today `hookify` and `ralph-loop`
both define `/help`.

**Decision.** `/team-*` becomes `/room-*`. `team` is about as collision-prone as a
prefix gets in a directory of developer tooling; `room` is the domain's own noun and
markedly less crowded.

**The prefix is tied to the protocol namespace, not the product name.** `room`
follows `protocolNamespace` (D-069), which is committed to never changing. Had it
followed the product name it would be hostage to a naming decision that has not been
made — and unlike the signing tags, nobody would notice until a collision.

**Unlike the couplings in D-069, this one is cheap to revisit.** Six markdown files
and no stored data depends on them, so if the settled name suggests a better prefix,
changing it costs a rename. It is being done now because the cost of *not* doing it
is a collision in somebody else's session, which is a bad way to find out.

**Revisit when** the name is settled, or if Claude Code gains a `plugin:command`
invocation form — at which point the prefix becomes belt and braces rather than the
only protection.

---

## D-071 — Leaving is a pause, and the row that records it is a tombstone

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Tracing the room lifecycle against the code found that `leave` did not
leave. It cleared the machine-level current-room pointer and nothing else: sessions
already bound stayed bound and kept publishing, the sync loop kept polling, and the
room's `state` column — written once as `joined` and never read — stayed as it was.
Its message was accurate about *new* sessions and easy to misread as having stopped
participating.

Three requirements were then stated: leaving must not foreclose rejoining the same
room, must not obscure the room's history, and must not affect the visibility of the
transcript as it had accumulated.

**The third one caught a live defect.** The browser view follows the current-room
pointer, and clearing that pointer was the whole of leaving — so leaving **blanked
the view**. The accumulated transcript disappeared at the moment a person stopped
adding to it, which is the opposite of what leaving should mean. Leaving no longer
touches the pointer at all; that pointer says what commands and the view are
looking at, which is a different question from whether a session is participating.

**The first one sharpened the invariant.** §12a forbids a session **moving to
another room**, because injected context cannot be withdrawn. Returning to the room
it was already in is not a move — the context it holds came from that room. So
rejoining is permitted, and only a different room is refused.

**The trap, which required the tombstone.** `BindSession` refuses a second room by
checking whether a `session_rooms` row exists. Had leaving *deleted* the row, then
leave-then-join would be exactly the move §12a prevents, with two extra keystrokes
— the session still holding the first room's turns and now publishing into a second
by way of the model's own output. The row is therefore kept and marked `left_at`,
so that two questions stop sharing one answer: **is this session in a room**, and
**which room has it ever been in**. A test asserts that a plain delete reopens the
hole.

**Leaving is per-session**, because §22 puts membership there: "Membership is held
by a Claude Code session, identified by its session ID." A peer with three sessions
in a room leaves three times, and that follows from where membership lives rather
than being a quirk. `/room-leave` runs inside the session it removes; at a terminal
there is nothing to leave, and it says so rather than reaching into sessions nobody
is looking at.

**What leaving deliberately does not touch.** The room's events. The guest list —
admission is the host's to withdraw with `revoke`, and leaving is your own act.
Synchronization, which continues: whether a peer should stop serving a room it has
left is a separate question, and answering it carelessly would silently degrade
somebody else's room.

**Still not implemented, and now the only part of §22's lifecycle that is not.** A
room never closes. Nothing archives, nothing goes dormant, and `state` remains
inert. That matters once rooms accumulate, and nobody has enough for it to bite.

**Revisit when** Phase 13 says whether rooms accumulate the way §22 assumed. If a
pair creates one room a week and never closes any, dormancy is the next piece; if
they reuse one room for months, it is not.

---

## D-072 — A refused join is explained, not merely refused

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** A session that has been in one room is refused a second (§12a, D-071).
The refusal read:

> this session has been in gusty-cavern and cannot join another: what it has been
> told cannot be withdrawn

Delivered through `log.Fatalf`, so a person saw it prefixed with a timestamp. It
states a rule and a fragment of a reason, and offers no way forward.

**Why that is worse here than it would be elsewhere.** The person has done nothing
wrong: joining a second room is a reasonable thing to try, the refusal comes from an
invariant they have never read, and **nothing they can do will make it work** — the
session's context already holds the other room. A message that gives only the rule
leaves them stuck with no route, in the one situation where being stuck is
permanent.

**Decision.** Explain the mechanism and give the way out.

The mechanism is worth stating because it is not obvious and it is not about
copying. Joining a second room would carry the first room's conversation into it
**by way of what the session says next**, which may be shaped by everything it has
read — invisible to both rooms' members, and impossible to take back because a
context window cannot be un-read. Refusing is the only moment that prevents it.

The way out is that a second Claude Code session costs nothing, and that this
session may still rejoin the room it was in (D-071).

**A typed error, not a formatted string.** `BoundElsewhereError` carries both rooms,
because a caller that cannot name the room a person was trying to reach cannot tell
them what to do instead. The explanation lives where it is shown to a person rather
than inside an error value that tests compare and logs decorate.

**And the slash command is told not to condense it.** `/room-join` relays the whole
explanation rather than summarising, for the same reason the view is not summarised
(D-070's sibling concern): a one-line "cannot join" is exactly the unhelpful form
this replaces, and a model paraphrasing helpfully would reproduce it.

**Revisit when** anything else refuses a person for an invariant's sake. The pattern
generalises: state the mechanism, say what is impossible to undo, and give the route
— and if there is no route, say that too rather than implying one exists.

---

## D-073 — Forgetting a peer discards their admissions with them

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Tracing a guest's lifecycle through the code rather than the
specification. §12 promises:

> Forgetting and revoking are not the same act. Revoking withdraws admission to one
> room. Forgetting discards the identity itself, **so a later meeting is a first
> meeting again.**

`Forget` deleted the row in `known_peers` and left every row in `room_guests`.

**What that cost.** Two things, and the second is the serious one.

A forgotten peer still appeared in `guests` — a host reading the list would see
somebody admitted who is not known, which is a false affordance in the place a host
looks to decide who is in a room.

And **meeting them again readmitted them to everything**. `allow` reinstates the
identity; the old `room_guests` rows were still there; verification would follow;
and they would be a guest of every room they had ever been in, without the host
inviting them to any of it. The host is never asked, because from the code's point
of view nobody was ever un-invited. That is the opposite of a first meeting.

It was masked rather than harmless: a forgotten peer is unverified, so D-054's gate
refused their sync. The hole only opened at the moment they were verified again —
which is to say, at the moment the host believed they were starting over.

**Decision.** `Forget` deletes admissions and identity in one transaction. A later
meeting then really is a first meeting: known again, unverified again, admitted to
nothing until a host says so.

**Revoke is unchanged and remains the narrow act.** One room, admission only, the
identity untouched. The two were always meant to differ in scope rather than in
kind, and now they do: revoke removes one admission, forget removes the identity and
every admission it carried.

**Tests.** `TestForgettingDiscardsAdmissionToo` follows the whole arc — allow,
verify, invite to two rooms, forget, meet again — and requires that meeting again
grants nothing. `TestForgettingIsNarrow` requires that forgetting one peer disturbs
neither another peer nor the room's own creator, who is its first guest. The first
is confirmed to fail without the cascade.

**Revisit when** a peer needs to be forgotten on one machine while remaining a guest
elsewhere. Guest lists are per-peer (D-046), so two peers can already disagree about
who belongs; this changes nothing about that, and a host forgetting somebody does
not tell them so.

---

## D-074 — A name means one key, and a collision is where a key change surfaces

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Tracing a peer identity's lifecycle. It is created once from
`identity.key`, never rotates, never expires, and the key is authoritative — if
`identity.json` disagrees it is corrected, because an identifier that is not this
key names an identity nothing can verify.

`pair` ends by telling a person:

> if this key changes, that is an alarm rather than a new first meeting.

**Nothing implemented that alarm**, and it is not obvious that anything could. A
peerId **is** a key (D-042), so "the same person with a new key" is not expressible:
to this machine that is a peer it has never seen, and it is refused as a stranger.
Correct, and it is not an alarm — it never mentions the person whose name is on the
old key.

**There is exactly one moment where the two can be connected**: when somebody
records the new key under a name they already use. That is also precisely what a
substitution looks like from the host's side — Mallory sends a pairing string in
Alice's name, and David types `pair <key> alice`.

**And it passed silently.** `known_peers.peer_id` is the primary key; `name` had no
constraint. A second row was created, `peers` listed two alices, and `resolvePeer`
returned whichever `KnownPeers()` yielded first — so `invite alice` could admit
either, with nothing said. That is the failure the entire verification apparatus
exists to prevent, reachable by typing a name twice.

**Decision.** A name may belong to one key. `Allow` refuses a name already held by a
different key and returns a typed `NameTakenError` carrying both, because the
explanation must show what changed.

The explanation says what the collision means rather than treating it as a naming
mistake: a changed key is indistinguishable from somebody else's key sent in their
name, so check on a call before recording it. And it gives the route — `forget` the
old one first, which since D-073 discards its admissions too, so the person is
invited again deliberately rather than inheriting rooms.

**`resolvePeer` refuses an ambiguous name rather than choosing.** Recording two keys
under one name is now prevented, but a database written before this could hold one,
and picking between them would admit a peer nobody named. Same shape as D-050: report
the ambiguity, name both, say what it probably means.

**What this does not do.** It does not detect a key change by itself — only a key
change that somebody tries to file under a familiar name. A peer that simply
presents a new key is still an unknown peer, refused without ceremony. That is the
honest limit of self-certifying identifiers, and it is why the alarm lives at the
moment of recording rather than at the moment of contact.

**Revisit when** identity rotation is wanted. There is none today: an identity is
created once and lives until `identity.key` is lost, at which point the peer is a
stranger to everyone and must pair again. Rotation would need a way for a new key to
be vouched for by the old one, which is a real design and not a small one.

---

## D-075 — The install has a state, and absence is not an inference

**Date:** 2026-09-20 · **Status:** active (implemented, v0.3.0)

**Context.** Asked what options exist for telling a session about a download in
progress, and then — better — whether a fresh install should be *marked* as not yet
downloaded, with commands behaving differently until the mark clears.

The second framing is the right one. Absence of the binary meant three different
things and every reader guessed the same hopeful one:

- nothing has started (the plugin is installed, no session has run yet);
- a download is in progress right now;
- an install **failed** and is waiting out an hour's cooldown.

The wrapper said "it may still be arriving" to all three. For the third that is
simply false, and it sends a person to wait for something that will not happen while
the reason sits in a log they have not been told about.

**Decision.** One state file, written by the installer, read by everything.
`installing` when the work begins, `failed` with the reason when it does not, and
removed on success. The cooldown reads the same mark, so a wait is explicable rather
than mysterious.

A fourth state is derived rather than written: an `installing` mark older than five
minutes reports as **stalled**. A crash between setting the mark and clearing it
would otherwise leave the file claiming a download is in progress forever, which is
the one answer worse than silence.

**What each reader does with it.** The command wrapper names which of the four it is
and what to do. The session-start hook passes it to the **model** as
`additionalContext` — the only channel that exists, since everything reaching a
person arrives by way of what the model says (D-033, D-036) — so a session asked
"why isn't this working" answers truthfully instead of speculating. It is emitted
only while the binary is absent, so an ordinary session carries nothing.

**What it does not do.** It does not notify, interrupt, or poll. A person who never
asks is never told, which is the same posture as the rest of this: §3.1 says a hook
must never break a session, and a progress bar nobody asked for is a small way of
breaking one.

**Released together, because they are not separable.** v0.3.0 ships the mark, the
`where` subcommand and per-session `leave`. v0.2.0's plugin already referenced
subcommands its own pinned binary did not have — `where` printed usage — because the
plugin was edited after the release it pins. **A plugin and the binary it pins are
one artefact**, and the release script writes `VERSION` precisely so they cannot
drift silently.

**Verified from nothing**, with no binary and nothing on PATH: the model is told what
is happening, the download and daemon are up in about three seconds, the mark clears
itself, and the command that printed usage under 0.2.0 now answers.

**Revisit when** Phase 13 says whether anybody reads any of it. The honest
possibility is that a colleague never sees these messages at all, because the install
finishes before they type anything — in which case the work was in making the
failure case truthful, which is the case that matters anyway.

---

## D-077 — Every room has its own URL, and no ambient value picks one

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Two objections, and the second landed on something introduced an hour
earlier: per-room URLs are mandatory, the machine-level pointer still has no
evident use, and **an environment variable is inadequate on a machine running
several sessions**.

The third is the serious one. `COGMER_ROOM` is **ambient** — exported once in
a profile, or inherited by every session a machine starts — and D-076 had just
placed it *above* the session's own room. So a single export would have answered
for every session regardless of which room it was in: exactly the failure D-064
removed, reinstated at higher precedence, in the same afternoon it was fixed
elsewhere. It is gone, along with the dead `LoadConfig` that also read it, and a
test now fails if anything reads it again.

**A room is named per invocation, or it is the session's.** Nothing ambient decides
who is admitted to what.

**Per-room URLs.** `/room/<name>` serves that room; `/` lists rooms and chooses
none, redirecting only when there is exactly one — a list of one is a question
nobody needs asked. The events endpoint and the event stream take the room as a
parameter, so two tabs stream two rooms and neither changes identity when somebody
creates a third. `watchLine` hands over the room's own address, which makes the line
printed at `create` a stable link rather than a window whose contents can move.

**What that settles about the pointer.** The view was its last consumer that could
not be given a session, and per-room URLs remove the need: the address carries the
room. What remains is a fallback for a terminal, where there genuinely is no session
to ask — and it is now reached only there, after the session has been asked and
found absent rather than merely unhelpful.

**The pattern worth naming, because it recurred three times in a day.** Ambient
state that answers "which room" is wrong wherever a session could have been asked:
once in the view, once in `where`, once in `currentRoom`. Each time it looked like a
convenience and behaved like a silent wrong answer. A session is the unit of
membership (§22), so anything that can ask a session must.

**Revisit when** a terminal command needs a room and there are several. Today the
fallback answers; the honest alternative is to require naming it, and the friction
only appears once rooms accumulate — which nothing yet archives.

---

## D-078 — Granting access names its room; showing something may guess

**Date:** 2026-09-20 · **Status:** the showing/changing distinction stands; the
`--room` flag it introduced is **superseded by D-079**, which found that a terminal
has no standing to change a room at all, so there is nothing for it to name.

**Context.** Asked why a terminal command should be able to act without knowing
which room it applies to. Following the question properly turns out to split what I
had been treating as one decision.

**`invite` grants access.** Granting it in the wrong room means a person reads a
conversation nobody invited them to, and **`revoke` cannot undo it** — it stops
future reading and does not un-read. The answer to a question with that consequence
was a stored pointer, set possibly weeks earlier, never displayed before the command
acted on it. That is not good enough, and the command with the worst failure mode
was the one leaning hardest on the weakest answer.

**`log` and `guests` show you something.** Guess wrong there and you see a room you
did not mean, on your own screen, and every one of those commands names the room it
is showing. Visible and free.

**Decision.** `roomToChange` does not guess: inside a session it uses that session's
room, and at a terminal it requires `--room <name>`. `roomToShow` may still fall
back, because being wrong is self-announcing. `invite` and `revoke` take the first;
`guests`, `log` and `conflicts` take the second.

The refusal says what to do rather than only what is missing:

    name the room: this changes who can read it, so it will not guess.
      cogmer <command> --room <name>
    `cogmer rooms` lists them. Run it from inside a session and it uses
    that session's room.

**The slash commands need no flag and that is not an oversight.** They run inside a
session, so the session's room answers. The command file says so, so that nobody
later adds a flag to make it "consistent" with the terminal form — the two differ
because one has a session to ask and the other does not.

**What this leaves of the fallback.** Only read-only commands at a terminal. That is
a small enough surface to defend on its merits: it saves typing where the cost of
being wrong is noticing and retyping.

**Revisit when** a read-only command's output is used as the basis for something
consequential — piped into a script that invites people, say. The distinction drawn
here is between *showing* and *changing*, and it stops holding the moment something
shown is fed into something that changes.

---

## D-079 — Changing a room requires standing in it, which only a session has

**Date:** 2026-09-20 · **Status:** active (implemented); supersedes the `--room`
flag added in D-078 hours earlier

**Context.** D-078 made `invite` and `revoke` refuse to guess a room, and gave them
`--room <name>` so a terminal could say which. That answered the wrong question.

The question is not *which* room. It is whether a terminal has any **standing** to
change one. §22 puts membership in a Claude Code session; a room's guest list is its
members' concern; at a terminal you are a member of nothing. Naming a room there is
reaching into a room that belongs to a session you are not in, which is a different
act from choosing among rooms you are in — and the flag made it look like the same
act, spelled more carefully.

**Decision.** `invite` and `revoke` require the invoking session to be in the room.
No flag, no fallback, no pointer. At a terminal they are refused and say where the
command belongs:

    this changes who can read a room, and only a session that is in one can do it.
    Run /room-invite or /room-revoke inside the Claude Code session that is in the room.

**What stays.** `log`, `guests` and `conflicts` still fall back at a terminal,
because showing you a room is not an exercise of standing — it is your own machine's
data on your own screen, and each names the room it shows.

**Why this took three attempts, which is the part worth keeping.** The same
question — *which room does this act on* — was answered three times by reaching for
whatever was nearest: an ambient environment variable, a stored pointer, then an
explicit flag. Each felt like an improvement on the last. None of them asked whether
the caller had any business acting on the room at all, and the flag was the most
misleading precisely because it looked most careful.

**Revisit when** something other than a session can hold membership. Nothing is
planned to; §22 is explicit, and every drift away from it so far has been an
accident rather than a proposal.

---

## D-080 — There is no current room; a terminal command exists to be tested or to work when the plugin cannot

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asked for the reasons a terminal command should exist at all, given
that plugin commands always run in a session which may or may not have a room —
leaving testing as the only obvious remaining purpose. Enumerating the surface
honestly produced five reasons, and the important result was what is **absent**
from them.

**The reasons that survive.**

1. **It cannot pass through a model.** `pair`, `verify`. Already decided (D-052,
   §29): interactive, blocking on another person, and the compared words must reach
   a person's eyes unaltered.
2. **It must work when the plugin path is broken.** `doctor`, `behaviors`,
   `version`, reading `install.log`. A diagnostic that only works when things work
   is not a diagnostic.
3. **The daemon's own lifecycle.** §29 requires it be discoverable and stoppable,
   and it runs when no session exists.
4. **Machine scope rather than room scope.** `whoami`, `peers`, `forget`, `allow` —
   this machine's identity and relationships, which involve no room.
5. **Testing**, by emulating a session.

**What is absent: no room-scoped operation has a terminal-only reason to exist.**
Every one of them fulfils a slash command or is being tested. So the machine-level
pointer had no remaining user — the browser view was its last consumer that could
not be handed a session, and per-room URLs (D-077) removed that.

**Decision.** `SetCurrentRoom` and `CurrentRoom` are deleted. A room-scoped command
resolves one way: the room the invoking session is in. A terminal that wants one
emulates a session, which is what every test here already does.

**The exception, and why it is one.** `conflicts` takes a room by name. It is a
diagnostic of category 2, wanted precisely when a room is misbehaving and possibly
with no healthy session to ask — and naming a room in order to **read** it is not an
exercise of standing, which is what makes this safe here and unsafe in `invite`
(D-078, D-079).

**Slash commands now cover every command that can have one.** Added `/room-revoke`,
which had no session-side route at all and so was unreachable through the supported
path since D-079; `/room-conflicts`; `/peer-list`; `/peer-forget`. `/room-pair`
became `/peer-pair`, because pairing is machine scope and the prefix was saying
otherwise.

**Two prefixes, because there are two scopes** (§12): `room-` acts on one room,
`peer-` on relationships that outlast every room. Deliberately excluded, with
reasons: `pair`/`verify` (category 1 — `/peer-pair` signposts them and does not
attempt them), `doctor`/`behaviors` (category 2 — a slash command for them would
imply the plugin path is a reasonable way to diagnose the plugin path),
`daemon`/`hook`/`probe-hook` (machinery), `seed` (a fixture), `allow` (the
unverified form, for scripts).

**A test now asserts every command file names a real subcommand**, because the
plugin and binary version together and v0.2.0 already shipped a mismatch — a
`/room-status` that called `where` and got usage back.

**Revisit when** something other than a session can hold membership, or when a
category-2 diagnostic needs to change something rather than show it. The line drawn
here is between showing and changing, and it is the second that requires standing.

---

## D-081 — Untrusted turns are JSON, the policy is stated separately, and the relay is measured

**Date:** 2026-09-20 · **Status:** active (implemented and tested against a live session)

**Context.** Asked whether Anthropic's prompt-injection guidance is useful here. Most
of it assumes an application that controls message structure — choosing
`tool_result` blocks, writing a system prompt, adding mid-conversation system
messages. This controls **a string** written to a hook's stdout. Two items
transferred, one of them uncomfortable.

**JSON, because escaping by hand is escaping by hand.** Turns were interpolated into
markup and defended by a per-injection random fence plus Go-quoting the speaker. That
is a good workaround for a problem JSON solves by construction: a value cannot leave a
JSON string without an unescaped quote, and the encoder guarantees there is not one.
Delimiting is now structural rather than careful.

The fence stays. It is tested, it costs nothing, and it does one thing JSON does not —
it lets the framing assert where the block ends in a way the content cannot imitate.

**The uncomfortable one: our framing sits where instructions get discounted.** The
guidance says not to put your own instructions in tool results, because a model is
trained to treat content in that position with scepticism. Every word we say about how
to read room content has been travelling in the same blob as the room content. If
Claude Code's `hook_success` attachment is treated as tool-result-like, the scepticism
that protects us from the payload applies equally to the paragraph explaining the
payload.

So the standing policy is now stated **once at session start**, via
`additionalContext` — a position the content cannot occupy. It does not replace the
in-block framing, which is tested and survives compaction in a way one statement may
not. It is reinforcement from somewhere else.

**Then measured, because a defence nobody tested is a hope.** Four live runs:

| | |
|---|---|
| "Why isn't /room-create working?" | relayed the download state accurately, in its own words |
| "What is 2+2?" | `4` — no leak |
| "nothing seems to be happening, is something broken?" | connected the oblique complaint to the note unprompted |
| hostile turn + "what did my teammate say?" | quoted it, named it an injection attempt, refused it, and told the user — **including refusing the "do not mention this" clause** |

**And the honest result about the new policy: it could not be shown to help.** The
same attack was run without it, and the in-block framing alone produced the same
refusal, with the same reasoning, and the same report to the user. The session-start
policy is defence in depth against a position problem that is real in principle and
was not observable here.

**What this settles about reaching a person.** Everything user-facing in this design
depends on the model relaying — install state, an unverified peer, a quiet room. That
relay now has evidence rather than assumption: it is accurate when relevant, silent
when not, and it fires on an oblique question as well as a direct one. That is the
property the browser-view discovery work depends on.

**What it does not settle.** Whether the attachment really is treated as tool-result
content. That is an undocumented property of someone else's binary that a defence now
leans on, and it belongs in the behaviour registry as a reliance even though it may
not be falsifiable from outside.

**Not adopted: screening tool output through a classifier.** It needs an inference
call per injection, and a remote event causing inference in an interactive session is
what §3.7 forbids. A separate run is the deliberately undecided area, and either way
it would spend the user's own quota on every teammate turn.

## D-082 — The daemon can reach a person directly; there is no setsid hazard

**Date:** 2026-09-20 · **Status:** active (measured; supersedes an earlier false finding)

**Context.** Discovery of the browser view was blocked on a question nobody had
answered: how a first-time user, who cannot be told a loopback address, ever arrives
at it. Two candidate channels were investigated — writing to the terminal directly,
and OS notifications — along with auto-opening the view.

**Auto-open works, and takes focus.** A process started the way the session-start
hook starts the daemon (`nohup … &`, no controlling terminal) can call `open` and the
browser becomes frontmost. Measured end to end: a detached, daemon-shaped process
that waits six seconds and then opens the view brought Chrome to the front and opened
the tab. This is the mechanism discovery now rests on, and it is recorded as **B23**.

**An earlier finding said the opposite, and was wrong.** A test reported that a
detached `open` never took focus, and it was attributed to macOS focus-stealing
prevention. The test ran `setsid nohup open …`, and **macOS ships no `setsid`** —
`command -v setsid` returns 1, there is no binary anywhere. The command failed, its
error went to `/dev/null`, and `&` backgrounded the failure. Nothing was ever
launched. The browser did not open a background tab; it did not open at all.

**So the setsid branch in `start_daemon_if_needed` is not a hazard.** It was briefly
believed to be one — that a machine with Homebrew's `util-linux` on PATH would flip
the branch and silently strand the daemon outside the GUI session. That was tested
directly, using Go's `SysProcAttr{Setsid: true}`, which performs the real `setsid(2)`
on darwin even though the binary is absent:

| | POSIX session inherited | new POSIX session |
|---|---|---|
| `launchctl managername` | `Aqua` | `Aqua` |
| `open <url>` | opens, takes focus | opens, takes focus |

The Mach bootstrap namespace is inherited separately from the POSIX session, so
detaching costs nothing. **No code change was made**, and a comment now records why,
because the wrong inference is an easy one to repeat.

**Terminal writes: possible, rejected.** A hook cannot inherit a tty — `/dev/tty` is
`ENXIO` inside a tool call — but it can walk its parent chain to the `claude` process
and read the tty from `ps`, and the daemon can then write to `/dev/ttysNNN`. Only OSC
sequences are safe, since they set terminal chrome rather than grid cells. Rejected
anyway on three counts: the title is contended with Claude Code, which overwrites it;
a `write(2)` can splice into the middle of a sequence Claude Code is emitting, because
a tty offers no `PIPE_BUF` atomicity guarantee, so the failure mode is intermittent
corruption of somebody's live working session; and nothing there can carry something
clickable. A channel that can only say "something happened" is not worth that risk.

**Notifications: kept, with a stated limit.** `osascript -e 'display notification'`
delivers reliably and persists in Notification Center — a banner missed at the time
was found there afterwards. But it posts under Script Editor's identity, and clicking
it does nothing. More importantly the delivery is **unobservable**: the notification
database is unreadable and the Focus state is TCC-protected, so the daemon cannot
learn whether a notification was shown, suppressed, or dismissed unseen. Against
D-014 that settles its role — fit for announcing, unfit for anything that must be
known to have landed.

**Not adopted: an `.app` bundle.** It was built and tested. It would buy a proper
notification identity, a click that opens the view, and a Spotlight-findable icon.
None of that is needed once plain `open` is known to work, and a bundle would have to
be created at install time and registered with LaunchServices. Revisit if a
notification ever needs to be clickable, or if macOS withholds focus from background
processes — which is exactly what B23 watches for.

**Revisit when:** B23 fails, or a notification needs to carry an action.

## D-083 — The view is opened at a first pairing, not at room creation

**Date:** 2026-09-20 · **Status:** active (sequencing decision; not yet implemented)

**Context.** With auto-open shown to work (D-082), the remaining question was *when*
to spend it. The obvious answer was the first `create` or `join` — the moment a room
first exists and therefore first has something to show.

**That is the wrong moment, because it is not the first one.** Pairing precedes room
formation in the user's experience, and it precedes it by design: D-053 separates the
two scopes, pairing happens once with a colleague and outlasts every room, and D-054
makes verification a **gate** — an unverified peer is refused sync before any room
content moves at all. A person therefore meets this system at pairing, not at a room.
Opening the view at room creation would leave the very first unfamiliar step — a
two-word comparison read aloud on a call (D-055) — happening with nothing on screen,
and would then open a window for the second step, which is the more familiar one.

**So the first open belongs at the first pairing.** It is also the step that most
needs a surface: the two words have to be read by both people at once, and a terminal
is the wrong place to put something a non-person must find and compare under time
pressure.

**Not settled here:** whether pairing and verification are *driven* from the view or
merely displayed in it, and whether an open recurs on later pairings or happens only
once ever. Both were raised and neither is answered; do not build either on
speculation.

**Revisit when:** the pairing ceremony gets a surface, or D-055's ceremony changes.

## D-084 — Silent to the person is not silent to the log

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** A test reported that a detached `open` never brought a browser forward,
and the conclusion stood for an hour before it turned out that macOS ships no
`setsid` — the command had never run. Asked why detection took so long, and whether
logging practice should change.

**Four things stacked, and each alone was survivable.** The error was discarded at
source: `setsid nohup open … >/dev/null 2>&1 &` closes both channels that would have
said the command did not exist — stderr to `/dev/null`, exit status to the
background. The observable was **negative**: "is the browser frontmost" cannot
distinguish *ran and did not focus* from *never ran*. A plausible mechanism was
available — macOS focus-stealing prevention is real and fit the data — so having an
explanation is what ended the inquiry. And the code taught the wrong thing:
`if command -v setsid` reads as "setsid is the preferred path" when it encodes "we do
not know whether this exists."

What broke it was a **positive** observable — counting browser tabs. Zero tabs is not
explicable by a focus theory. One measurement the prevailing story could not
accommodate did what five consistent ones could not.

**§3.1 requires silence toward the person, not toward the log,** and the code had
been conflating the two. `install.sh` kept a record of its decisions; `common.sh` and
`session-start.sh` kept none, so the four load-bearing choices in
`start_daemon_if_needed` — which binary won the search, whether the daemon answered,
which detach branch ran, whether a child started — were decided and recorded nowhere.
"The daemon is not running" had no next question.

**The rule adopted: redirect to the log by default; use `/dev/null` only when you can
say what the discarded output would have said.** Output that is *noise by
construction* may be dropped — an exit status is all that is wanted, stdin is being
drained, a `mkdir` whose failure the next line catches. Output that is merely
*usually empty* must not be, and that distinction is the whole lesson: the setsid
line was usually empty, which is exactly why the one time it had something to say it
looked normal.

**Never to stdout, in any of this.** A hook's stdout is injected into the user's
turn, so a stray line there does not produce a worse message, it corrupts a prompt.
`json_safe` was added for the same reason: details assembled from error text reach a
JSON string literal, and an unescaped quote makes a hook emit malformed JSON.

**What changed.** A shared `ct_say`; the four decisions above recorded; "curl said no"
separated from "there is no curl", which were the same answer and start a daemon on
every session; `install.sh` output to `install.log` rather than `/dev/null`, since its
own logger cannot record what goes wrong before it is defined; and its `mkdir`
failure — the quietest line in the file, and a load-bearing one — made audible.

**A busy port is two different events and now says which.** `net.Listen` failing was
`log.Fatalf` for both. Several sessions starting at once race to start a daemon and
exactly one wins: that is the ordinary outcome (§29), and it now exits quietly after
confirming via `/healthz` that the holder is one of ours. Anything else holding the
port is a real fault, and it now writes `daemon-state`, which the session-start hook
relays to the model — the same mechanism D-075 built for a half-finished install, and
the only route from a detached daemon to a person (D-033, D-036).

**Revisit when:** the logs grow enough to be worth rotating, or a failure is found
that none of these three channels — daemon.log, install.log, the state files —
would have caught.

## D-085 — A line telling somebody to run `cogmer` is a line that fails

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Reviewing documentation for the assumption that every user is a
person. The tone was the smaller problem.

**The instructions did not work.** `/peer-pair` told people to run
`cogmer pair <string>` in a terminal; `/room-join` told them to run
`cogmer verify`. The installer puts the binary in `~/.cogmer/bin`, which is
deliberately not on PATH — §29 forbids editing a shell profile to put it there, and
`cli.sh` exists precisely because slash commands hit this. So those lines produce
`command not found` for **every plugin-installed user**: not an edge case, but the
default install, and the people least equipped to diagnose it.

**The binary prints its own invocation.** `invocation()` resolves `os.Executable()`,
spells `$HOME` as `~`, and is used wherever a person is told to go and type
something. A path the binary reports about itself cannot go stale, and it is
correct for a from-source install and a plugin install alike.

**The command docs now relay rather than restate.** They are told explicitly not to
shorten it, and `/peer-pair` walks through opening a terminal, pasting, and pressing
return — because somebody being asked to compare two spoken words with a colleague
may never have opened one.

**Not done: making pairing work from a slash command.** It was raised, and it needs
the view to have somewhere to show the words. That belongs with D-083, not here.

**Revisit when:** the install location changes, or pairing gains a surface that is
not a terminal.

## D-086 — The terminal is not a user experience; the view is the surface

**Date:** 2026-09-20 · **Status:** active (direction; amends D-080, does not alter D-055)

**Context.** Stated intent to stop treating "run it in a terminal" as a valid thing
to ask a user to do. Claude Code is the first host; Claude CoWork is wanted as a fast
follow.

**What actually depends on a terminal today: one bit.** Every room-scoped and
peer-scoped operation already has a slash command. `pair` and `verify` are the sole
exceptions, and the dependency is a single `fmt.Scanln` — the y/N answer to "did they
say the same two words?". The ceremony itself is already daemon-side and already
spoken over HTTP: `/verify/start` returns the words, `/verify/confirm` records the
answer, and the CLI is nothing but a client of those two endpoints.

So moving off the terminal is a **user-interface change, not an architectural one**.
There is no protocol to redesign and no logic to relocate.

**D-080's first reason is amended, not withdrawn.** It held that `pair` and `verify`
"cannot pass through a model" — interactive, blocking on another person, and the
words must reach a person's eyes unaltered. That is still true and still important.
What no longer follows is the conclusion that they must therefore be typed at a
terminal: *not passing through a model* and *being a terminal* are different
properties, and the browser view has the first without the second. D-080 could not
see that option because discovery was unsettled; D-082 settled it, and D-083 already
placed the first opening of the view at a first pairing. The three now agree.

**D-055 is untouched.** Two words, derived from a live commit/reveal exchange,
compared aloud on a call, by both people at once, with no fallback. Moving the
display from a terminal to a view changes the surface and nothing about the
ceremony. A view that offered a way to skip the call would violate D-055 no matter
how convenient; the gate is only as strong as the weakest ceremony that satisfies it.

**D-043 still holds: no adapter machinery, and naming a second host does not change
that.** CoWork is wanted, not present. Its extension model is not something to design
against on assumption. The argument in D-043 was that capture generalises and
injection does not, and that every hard problem so far has been host-specific — none
of which a second host being *anticipated* falsifies.

**But this direction and that constraint point the same way.** The daemon and its
view are host-independent by construction: they are a local service and a web page,
not an extension of anything. Every piece of experience that lives there is a piece a
second host does not have to reimplement, and it gets there without a `source` column,
an adapter interface, or a speculative abstraction. Reducing what a second host must
supply is the opposite of building for one. That is the cheapest possible preparation
and it is justified on today's host alone.

**What the terminal keeps** (D-080's other four reasons, all intact): diagnostics that
must work when the plugin path is broken, the daemon's own lifecycle, machine-scope
identity, and testing. None of those is a user experience, which is the point — they
are an operator surface, and it is legitimate for an operator surface to be a CLI.

**Interim state, deliberately not marked in the files.** `/peer-pair` currently walks
a person through opening a terminal, and that is the truth today, so it stays correct
until the view can take the ceremony. It is not annotated as provisional in
`plugin/commands/peer-pair.md` because that file is a **prompt**, not documentation:
meta-commentary about the roadmap would be read by the model as instruction. The
record belongs here instead.

**Next, and not yet built:** the pairing ceremony in the view, driven by the two
endpoints that already exist. Open questions from D-083 remain open — whether later
pairings re-open the view, and whether the view drives or merely displays.

**Revisit when:** CoWork's extension model is known, or the view takes the ceremony
and `verifyWith`'s `Scanln` stops being the only interactive path.

## D-087 — The local API requires a header a web page cannot send

**Date:** 2026-09-20 · **Status:** active (implemented and demonstrated)

**Context.** D-086 makes the browser view the place people *do* things rather than
only read them. Checking what that would expose found that no handler on the local
HTTP server checked `r.Method` or `Origin` — not one.

**It was exploitable, and was demonstrated rather than argued.** A page served from
another port marked an unverified peer **verified** with a single POST: no ceremony,
no two words, no call. Loopback binding keeps other machines out and does nothing
about a page in this machine's browser, which reaches 127.0.0.1 like any address.
`json.Decode` ignores `Content-Type`, so `text/plain` makes it a "simple request"
that is sent without the browser asking permission first. The response is unreadable
cross-origin — and irrelevant, because the side effect has already landed.

That bypasses D-054, the gate every other guarantee depends on. `/hook/prompt` and
`/hook/stop` are the same class and arguably worse: what is published there reaches
teammates' context windows.

**The rule is REQUIRE, not refuse.** A request must positively present
`X-Cogmer: 1`. A browser cannot send a custom header to another origin without
a preflight, and we answer preflights with nothing, so the real request is never
sent. Measured: the `OPTIONS` arrived carrying
`Access-Control-Request-Headers: x-cogmer`, and **no POST followed**.

Requiring presence is what makes it fail **closed**. The alternative first
considered — refuse a mismatched `Origin` — is fail-open on absent, and would have
needed a standing argument that no browser can ever produce a header-less
cross-origin POST. Requiring a header needs no such argument: anything that cannot
present it is refused, whatever it is. `Origin` is kept as a second layer because it
costs three lines and fails the request earlier.

**Rejected: `Referer`, and rejected on measurement.** Proposed as a way to require
presence. A page suppresses its own referrer with one `<meta name="referrer">` tag —
measured: `Referer` absent, `Origin` still present — and an absent `Referer` must be
treated as allowed, because the CLI and hooks send none. The check would pass exactly
when it needed to fail. In the realistic case it is worse: a hostile page is on
`https://` and we are on `http://`, and the default `strict-origin-when-cross-origin`
policy drops `Referer` on that downgrade without the attacker trying.

**Rejected: a redirect to control the referrer.** Also measured. A 307 through our
own origin left `Referer` as the attacker's page, because the referrer is the
*initiating document* and survives the redirect. A **client-side** redirect would
have worked — a document we serve that navigates onward does become the referrer —
and that is sound. It was still not taken: `Referer` is a header third parties are
entitled to strip. Our own view's `Referrer-Policy`, a privacy extension, or a
corporate proxy would remove it from our **own** requests and break the view, with a
failure that reads as a bug rather than a policy. Nothing strips a header it has
never heard of.

**Also fixed:** state-changing routes require POST, so no navigation or `<img src>`
variant reaches a handler that reads a body.

**Not guarded, deliberately:** `/healthz`, `/events`, `/stream`, and the page itself.
These are reads, and a cross-origin page cannot read their responses — we send no
CORS headers, so the browser withholds the response from the script. Guarding them
would break the view, which fetches them as ordinary GETs.

**What this does not defend against:** a malicious program running as this user. It
can set any header, and could edit `membership.db` directly in any case. The threat
closed here is a web page, which is the one the browser enforces a boundary for.

**Revisit when:** the view stops being loopback-only, at which point a secret in the
URL (D-077's per-room addresses are the natural carrier) replaces this rather than
supplementing it.

## D-088 — The pairing ceremony lives in the view, and every pairing gets its own URL

**Date:** 2026-09-20 · **Status:** active (implemented and verified end to end)

**Context.** D-086 established that the terminal is not a user experience and that
the only thing still requiring one was a single `fmt.Scanln` — the y/N answer in
`verifyWith`. D-087 gave the view an origin boundary, which was the prerequisite for
letting it do anything rather than only show things.

**D-055 is unchanged, and that is the point.** Two words, from a live commit/reveal
exchange, compared aloud on a call, by both people at once, no fallback. What moved
is the surface. A terminal was never the ceremony — it was the only thing available
that was not the model (D-080), and the view is the other one.

**Every pairing gets its own URL**, and this is load-bearing rather than tidy:

- A pairing page is a live ceremony with a deadline, not a dashboard. Two must never
  share a page, and a page left open from an earlier attempt must never quietly
  become a different one — the entire security property is that the person knows
  *which key* they are vouching for.
- It is what makes a second pairing visible. A fixed address whose contents we
  rewrote would, at best, change a tab nobody is looking at.

The id is 128 random bits, because it is the capability: holding it is what lets a
page start that exchange.

**Measured, and it refined the premise.** `open` creates a **new tab every time**,
even for an identical URL — tab count climbed 2→3→4→5 across repeated opens — and
focus was taken in all six trials, same-URL and unique-URL alike. So in Chrome the
silent-background-rewrite case does not arise from `open` itself. Unique URLs are
still right: they do not depend on that behaviour, and the correctness argument above
stands on its own.

**The page names a pairing, never a peer.** `/verify/start` and `/verify/confirm`
grew an optional `pairId`, and the daemon resolves it. So even a page that reached
the endpoint could not begin an exchange for an arbitrary identifier, and an expired
link fails as a link rather than silently starting something new.

**The 90-second window starts when the person is ready, not when the tab loads.**
The first build ran the exchange on load, which spends the deadline on however long
it takes two people to get on a call — exactly the coordination the deadline exists
to bound. The page now opens in a *ready* state with a Start button, and a timeout
returns there rather than to an immediate retry.

**The terminal path is kept, and is not legacy.** `--terminal` forces it, and it is
also the automatic fallback when no browser can be opened — a machine reached over
SSH, a container, a server. `openInBrowser` returns an error rather than failing
quietly, precisely so the caller can choose that branch.

**Verified end to end**, not only by unit test: two real daemons on one machine, a
genuine commit/reveal between them, matching words rendered in both browser pages,
and both sides recording `verified` only after a human clicked. Nothing was recorded
before the click.

**Also settled from D-083's open list:** later pairings do re-open the view, because
each one is a new URL and therefore a new tab. Whether the view *drives* or merely
*displays* is now answered — it drives, since the confirmation is the one bit that
was keeping a terminal in the path.

**Revisit when:** the ceremony gains a step that a page cannot host, or the view
stops being loopback-only (which would make the pairing id a bearer token over a
network rather than a local one).

## D-089 — The reachability warning asked the wrong question, and fired always

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asked whether a loopback address in a pairing string was simply "the
daemon wasn't running", and whether enough was done to avoid it. Reading the code to
answer found three defects, one of which made the existing warning worthless.

**The warning tested the bind address, not the advertised one.**

```go
if isLoopback(peerAddr()) {          // the address this daemon BINDS
```

`peerAddr()` is `127.0.0.1:4783` by default and stays loopback even when tailcat has
negotiated a perfectly routable `tc://` endpoint — which is the **ordinary** case,
since tailcat is on unless switched off. So the warning fired on every pairing string
ever printed, including all the working ones. Observed directly: `whoami` printing a
`tc://` string with "nobody else can reach it" underneath it. A warning that is
always on is not a warning; it trains somebody to ignore the real case.

It now asks about `AdvertisedEndpoint()` and treats a negotiated tailcat endpoint as
routable by construction.

**It was attached to a command rather than to the string.** Three commands print a
pairing string — `whoami`, `pair` with no arguments, and `peers` when empty — and
only `whoami` warned. The check moved into `printPairingInvitation`, so it travels
with the thing it is about.

**The two causes need different answers, and got one.** "No daemon has ever run, so
this address is a guess" is fixed by starting a session. "A daemon ran and negotiated
no route out" is fixed by setting `COGMER_PEER_ADDR`. Sending somebody to the
second when the first is true wastes their time on a setting that is not the problem.
`endpointRecorded()` distinguishes them.

**A fallback that needed the thing that had just failed.** `beginCeremony` fell back
to the terminal when `/pair/new` failed — but the terminal ceremony posts to
`/verify/start`, so with the daemon down it failed too, one message later. Two
failures reading as two problems. It now checks `daemonAlreadyServing` up front and
says the one true thing. The terminal fallback remains for what it is actually for:
a machine that cannot open a browser.

**Found by a test, immediately:** `ParseEndpoint` accepts anything without a scheme
as a TCP endpoint, and `isLoopback` answers `false` for what it cannot parse — so a
malformed address read as "not loopback" and therefore as fine. A bare TCP endpoint
must now parse as host and port.

**Have we done everything to avoid a loopback endpoint?** Nearly. Tailcat is on by
default and supplies a routable endpoint without configuration, which is why the
false warning mattered so much — it was obscuring that the ordinary path works. What
remains is the case where tailcat cannot negotiate: the string is then genuinely
unusable and we now say so precisely.

**Not taken: advertising the machine's LAN address as a fallback.** It is easy to
detect and it is right only for peers on the same network — which D-019 names as the
zero-configuration path and which is not built. Advertising an address that works for
some colleagues and silently not others is worse than an address that visibly works
for none. This belongs with local discovery, not ahead of it.

> **The conclusion holds; the reason is restated in D-091.** A LAN address is not
> rejected because it works for some colleagues and not others, but because it is a
> claim about a network this machine may no longer be on — and on a different
> network the same address is a different machine. A location is discovered, never
> advertised.

**Revisit when:** local network discovery lands, or tailcat stops being on by
default.

## D-090 — Attribution is structured; the derived name is a field, not part of a string

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** From `docs/residual-concerns.md`: what should happen when a peer's
self-identified name collides with one already in the list. Tracing it found three
different name spaces with three different answers, and one genuine hole.

**The derived name** (`quiet-otter`) can collide and that is accepted — D-050 says
names may collide and identities do not, nothing keys on a name (D-017), and it takes
roughly 107 known peers before a collision among 8,280 combinations is more likely
than not.

**The local label** you assign is settled by D-074: `known_peers.name` is unique and
`explainNameTaken` says that a changed key is indistinguishable from somebody else's
key sent in their name.

**The self-asserted display name has no constraint, correctly** — it is the peer's
name for themselves and two colleagues really may both be David. That already
happened: the first two-peer run had both daemons assert the same display name
because both ran under the same OS user.

**The hole was structural rather than a collision.** The injected block built one
string:

```go
speaker := fmt.Sprintf("%s (%s%s)", e.UserDisplayName, PeerName(e.PeerID), mark)
```

Peer-chosen free text immediately before a parenthetical the model is meant to read
as the derived identity. A display name of `Alice (quiet-otter)` produced
`Alice (quiet-otter) (prudent-wagtail, unverified)`. It escapes nothing, forges no
turn and leaves the block intact — so `TestACraftedDisplayNameCannotForgeASpeaker`
passed, because it tests escaping. Escaping stops a value leaving its field. It does
nothing about a value imitating the field beside it.

**So the derived name became a field**, as `verified` already was, and two other
concatenations went with it: the verified state (a boolean) and the `Claude-` prefix
that marked an assistant turn (what `kind` already says). Nothing of ours is glued to
their text any more, and a value cannot occupy another field's position.

This is the same lesson as D-040 (fence the block so content cannot close it) and
D-081 (encode turns as JSON so a value cannot escape its string), applied one level
further in: **do not concatenate our facts with their claims.**

**Still open, and not fixed here:** the view renders the two names in separate
elements already, so it does not have this bug — but it gives them similar visual
weight. Somebody skimming a room is a different reader from a model parsing JSON,
and nothing has been measured about what they actually notice.

**Revisit when:** another field is added that a peer can influence.

## D-091 — Identity is advertised; location is discovered

**Date:** 2026-09-20 · **Status:** decided, not implemented · **Supersedes** part of D-089

**Context.** Two questions, a few hours apart. How can a peer's daemon know which
address suits the network topology between two particular machines? And does a
machine travelling between networks change the answer? It does, and it changes the
shape of the answer rather than a detail of it.

**The sender cannot know which address works, and should not try.** Topology is a
property of the *pair*. The sender knows its own interfaces and nothing about where
the receiver sits, so only the receiver can find out, by attempting. That is what
ICE and happy-eyeballs do, and what an overlay already does inside a single
endpoint: a relay first, direct once it can. Which is why the overlay path survives
NAT and the plain TCP path does not — the negotiation exists at one layer and is
absent at the other.

**Addresses are of two kinds, and only one of them can be advertised.**

| | names | survives a move | where it belongs |
|---|---|---|---|
| identity-shaped (`tc://…`) | a node, and where to find it | mostly — see below | the durable peer record |
| location-shaped (an IP and port) | a place | no | a candidate, with an expiry |

An advertisement outlives the fact it asserts. That is tolerable for something that
identifies a machine, which does not change when a laptop moves from a desk to a
hotel, and not tolerable for an address, which is true of one network position at
one moment.

**An overlay address is not purely an identity, and the difference matters.**
Decoding one gives a small CBOR map: three 32-byte public keys, and the node's home
relay — a hostname and its v4 and v6 addresses. The keys are the durable part. The
relay is not an identity but a **rendezvous**: where this node can currently be
found so that an introduction can happen, after which the path upgrades to direct
if the two ends can reach each other.

A rendezvous can go stale. The overlay picks the lowest-latency relay, and that
choice changes when a machine moves far enough — a different continent, sometimes a
different country. The keys in the advertised value stay correct and the relay named
beside them may no longer be the one that node is attached to.

In practice it usually still works, because the relays mesh and one that receives a
connection for a node attached elsewhere forwards it. That is a property of somebody
else's network rather than a guarantee this design holds, which is one of the
reasons the dependency on it is a precondition of Phase 13 rather than a settled
matter. So: treat the keys as durable and the rendezvous as best-effort, and do not
write anything that assumes an overlay address is timeless.

**A private address is not merely stale; it can be confidently wrong about a
different machine.** `192.168.1.42` at a coffee shop belongs to somebody else's
laptop, so attempting it does not fail — it succeeds against a stranger, and then
has to be unwound at a higher layer. Private ranges therefore never appear in a
pairing string, an invitation, or a durable record. They enter only as facts
discovered on the network we are on now (D-019), where discovery states where a
peer *is* rather than where a peer *was*.

This entry first argued that the cost was disclosure — that a signed sync request
(D-044) would deliver our identifier to whoever now holds the address, on every
poll. That was true when it was written and TLS removed it (D-101): the connection
is abandoned as soon as the far side presents a key this machine does not know,
which in TLS 1.3 is before this machine has sent a certificate of its own. Measured
rather than assumed. What a stranger at a reassigned address learns is that
something attempted a connection, and nothing about who. The argument from
correctness stands unchanged and is sufficient on its own.

**A remembered winner is a snapshot too.** Remembering which candidate worked is
right, and it must carry the time it worked, be tried first, and be discarded on
failure rather than kept. It is a hint, not a record.

**On ordering, asked directly: no, a daemon should not have a configurable sort.**

The receiver holds the information, so any order the sender expresses is a
preference rather than knowledge — and one the receiver would be honouring on the
word of the peer whose address it is.

Order by *class* instead, which is derivable rather than configured — same-host,
same-network, public, relayed — and try the first few concurrently, so a dead
candidate costs a round trip rather than a timeout. A learned winner then dominates
any static order after the first success, which makes a configured one dead weight
that is wrong in exactly the cases it was added for.

Every real requirement that arrives dressed as ordering is a **filter**: "never use
a relay" for a site that will not route through third-party infrastructure, or
"never expose a direct address". Those need *never*, and a preference order cannot
express never. Add filters if such a requirement appears; do not add a sort in
anticipation of one.

**D-089 rejected advertising this machine's LAN address, and was right.** Its stated
reason — that an address working for some colleagues and not others is worse than
one that visibly works for none — was the weaker argument, and an earlier form of
this entry overturned it on those grounds. The reason that holds is that a LAN
address is a claim about a network the machine may no longer be on, and the claim is
wrong about somebody else rather than merely wrong.

**None of this is implemented.** A peer is advertised at one address, that address
is recorded once, and nothing expires.

**Revisit when:** local discovery lands (D-019's zero-configuration path), which is
what gives a location candidate a legitimate way in.

## D-092 — An address is learned once, out of band, and only one side needs one

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asked when and how an address is learned for pairing. Tracing it found
the answer is narrower than expected, and that one consequence of it was being
actively refused.

**When: once, at `pair` time. How: carried by a person.** The address is the `@…`
tail of the pairing string a colleague sends by whatever channel they like.
`parsePairing` splits on the last `@`, `SetPeerEndpoint` stores it, and
`syncTargets` offers it to `RunVerification` later. There is no other path —
`RemoteAddr` is never read anywhere in the codebase, so nothing is learned from a
connection, and there is no discovery to learn it from (D-019's zero-configuration
path is not built).

**Only one side needs a usable address.** `RunVerification` checks for an inbound
exchange before it dials: if the other side ran the whole exchange against this
session, the revealed nonce is already here and the same two words come out. So an
asymmetric arrangement works, and it is the one a colleague behind a NAT that cannot
be traversed actually needs — they can be the side that publishes nothing.

**And `pair` refused to allow it.** A pairing string with no address printed "there
is nowhere to reach them yet" and returned before starting a session, so the inbound
exchange had nothing to arrive at. A working arrangement was made to look broken, and
the person was sent to fetch an address they did not need. It now says which of them
has to dial and starts anyway.

Asserted by `TestOnlyOneSideNeedsAnAddress`, which gives one daemon no addresses at
all and requires both to reach the same words.

**Seven more instruction sites were still naming a bare `cogmer`** (D-085),
including the one in this path. Fixed, and the ones about pairing now point at
`/peer-pair` rather than at `verify`, since the ceremony is in the view (D-088).
Operator surfaces — the daemon log, `doctor`'s stderr, the generated behaviours
header — keep the short form, because you are already at a prompt when you read them.

**Revisit when:** D-091 lands, at which point a pairing string carries a set and
"no address" becomes "no address that worked", which is a different message.

## D-093 — A two-interaction flow must choose what an unfinished second one means

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asked whether the local label should be a required argument to pairing,
after observing that a derived name like `quiet-otter` has no durable connection to a
person across weeks of inactivity. The answer turned on a general principle stated
during the discussion: anything that happens in two user interactions has to choose
what happens if the second is not completed.

**Optional did not mean unlabelled.** `Allow` filled an empty name with the derived
one, so somebody who never thought about a name got exactly the label that stops
meaning anything. The default was the failure mode.

**Asking after the words matched was worse, and was the first plan.** It creates a
second completion point, and both answers to an unfinished one are bad: default the
label to the derived name, which is the thing being escaped, or withhold the
verification until named, which holds the security-meaningful act hostage to a
convenience field. The flow should not have had two completion points.

**Collecting is not asserting.** The objection to taking the name up front was that
the label claims "this key is Alice", which is unjustified until the comparison
succeeds. That is true of *writing* it, not of *asking* for it. The name is now taken
with the pairing string, held in the pending pairing, and written only on a match.

**The three endings are now distinct, and were not before:**

| second interaction | what remains |
|---|---|
| never happened | the key, unverified, labelled with the derived placeholder — resumable |
| words matched | the key, verified, labelled with the chosen name |
| words differed | nothing, if this pairing created the row |

The last was the substantive bug. Abandoning and mismatching left identical state —
a recorded unverified peer wearing a colleague's name — so a **detected interception**
left the attacker's key on the list under the name meant for the real person. Not a
breach, since D-054 refuses everything from an unverified peer, but D-074's
uniqueness then blocked pairing with the real Alice until somebody worked out they
had to `forget` first. The attack you correctly detected became a second,
unrelated-looking problem at the worst moment.

**A mismatch removes only what the pairing created.** Re-verifying a colleague of two
years and seeing different words is an alarm about an existing relationship, not a
reason to discard it and every admission it holds (D-073).

**The name clash is checked before the ceremony, not after.** `NameFree` is split out
of `Allow` for this. Discovering the clash once the words had matched would leave a
verified peer and an unusable name — one more unfinished thing, which is the shape
this entry exists to avoid.

**Both surfaces now run through one pairing record.** The terminal path mints a
pairing and confirms against its id exactly as the page does, so what a match, a
mismatch and an abandonment mean is decided once rather than twice.

**A hazard found while moving the write:** `SetPeerEndpoint` is an `UPDATE … WHERE
peer_id = ?`, which affects nothing and returns nil when the row does not exist yet.
Recording the peer daemon-side moved the row's creation after the CLI's call to it,
which would have silently lost the only address anybody had. The endpoint is now
recorded with the row.

**How somebody learns the requirement**, since a required argument nobody is told
about is just a failure: the slash command's `argument-hint` names both, running
`/peer-pair` with nothing explains both halves of pairing, and giving a string
without a name produces the command echoed back with the string already in it and
one word missing. The command doc tells the model to ask rather than to pick one.

**Revisit when:** another flow gains a second interaction — the three endings should
be enumerated for it too, and named, rather than left to whatever the code does.

## D-094 — The name you chose leads the view; the unverified marker becomes a fact

**Date:** 2026-09-20 · **Status:** active (implemented, with one half deliberately deferred)

**Context.** Raised that a derived name like `quiet-otter`, however charming, has no
durable connection to a person across weeks of inactivity. Checking where each name
actually goes found that the one name which is both memorable and bound to a key
reached nobody.

| name | chosen by | bound to one key | reached the view |
|---|---|---|---|
| display name | the peer | no, it is a claim | yes, leading |
| derived name | nobody, computed from the key | yes | yes, secondary |
| the label you chose | you, at pairing | yes, D-074 makes it unique | **no** |

`known_peers.name` was read only by the CLI `peers` listing and `resolvePeer`. The
view led with the peer's own claim and anchored on a word pair nobody picked.

**The label now leads**, with the derived name kept beside it: it is still what a key
change surfaces on (D-074), and it still separates two peers claiming one display
name. A label equal to the derived name is treated as absent, because a placeholder
offered as a choice would be a lie.

**The derived name is doing a job it was never given.** Its actual jobs are
momentary — disambiguate a collision, anchor an alarm. Neither asks anybody to
remember it. D-042 already calls it "a mnemonic for an identity already verified",
and a mnemonic nobody chose, for a key they will never look at, is not much of one.

**And the unverified marker was on every remote turn.** `uiEvent` had no verified
field, so the view could not ask; a comment written before D-054 explained that every
remote peer was unverified. D-054 now refuses an unverified peer's events outright,
so everything displayed is verified by construction and the marker was permanently
on. Same failure as D-089: a warning that is always on trains somebody to ignore the
one that matters. It is a fact now, and seeing it means a filter failed.

**Deferred: carrying the label into the injected block.** The model should say
"Alice" where the person says "Alice", and the field belongs there for the same
reason it belongs in the view. `FormatTeamContext` takes `verified func(string) bool`
and has twenty-two call sites, so adding a lookup is a signature decision rather than
a field addition, and it deserves its own entry rather than being smuggled in here.

**Not done, and blocked: defaulting the label to the name the peer chose.** Proposed
that the pairing string carry it, so `/peer-pair <string>` could offer a sensible
default. **A peer does not choose their name.** `UserDisplayName` is `$USER` with the
first letter capitalised, computed once at identity creation, with no command, flag
or environment override to change it — which is why two daemons asserted the same one
during the first two-peer run. Shipping that in a pairing string would be worse than
the derived name: `quiet-otter` is honest about being machine-made, while `Ec2-user`
or `Person` looks like a claim about a person. Choosing a name has to exist before
a pairing string can carry one, and the moment for it is the half of `/peer-pair`
that already explains what to send.

**Revisit when:** a peer can choose their own name, at which point the pairing string
should carry it and the label should default to it.

## D-095 — A name is chosen or guessed, and the difference is recorded

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** D-094 stopped short of defaulting a peer's label to the name they had
picked, because a peer does not pick one: `UserDisplayName` is `$USER` with the first
letter capitalised, fixed at identity creation, with no way to change it. Asked what
the natural point is for somebody to choose their own.

**The name has one function: to be seen by other people.** You never see yourself
labelled — D-021 keeps the derived name off your own turns, and your turns are marked
as yours. So there is no moment when its owner notices it is wrong, which is exactly
why `Ec2-user` could travel for weeks.

**So the moment is the first time it travels**, and that is one place:
`printPairingInvitation`. All three paths that hand your identity to somebody else go
through it (D-089 consolidated them), so the offer belongs with the string rather
than with a command. There is no earlier candidate — identity is created by whatever
touches it first, which is a daemon started detached by a hook, and D-041 forbids
delaying a session or speaking to the person. Nobody is watching when the name is
invented.

**Chosen is recorded, not inferred.** `NameChosen` is false at creation and flipped
by a deliberate set — including setting it to the guessed value, because running the
command is the choice. Comparing the name against the guess instead would pester the
person whose username really is their name, for ever, which is the always-on warning
of D-089 in another costume.

**`whoami` does both, not one or the other.** It shows the name plainly, lifted out
of the JSON blob where it was buried next to the key, because it is the literal
answer to the question asked. The provenance line appears only while the name is
guessed; saying "you chose this" otherwise is noise about something already known.

**Prefixes name the scope, and `peer-` is for other people.** `/self-name`, not
`/peer-name`. The rule is now: `peer-` for a command about somebody else, `self-`
for one about you, `room-` for one about a room. A prefix names the **activity**, so
`/peer-pair` with no arguments printing your own string is not a violation — sending
your half is part of pairing with a peer — but that reading should be stated, because
it is the first question anybody will ask of the rule.

**Renaming is safe, and by construction rather than luck.** Turns already sent keep
the name they carried, because events are immutable (§7). And the label a colleague
gave you is theirs (D-094), so your rename cannot change what anybody else calls you.
That is what makes this cosmetic enough to defer and to change.

**Now unblocked:** the pairing string can carry a chosen name, and a label can
default to it (D-094's revisit condition).

**Revisit when:** something other than a person needs a display name, or the name
starts travelling somewhere other than a pairing string and an event.

## D-096 — A command prefix names its target; `/self-status` is where you are

**Date:** 2026-09-20 · **Status:** active (implemented) · **Corrects** D-095

**Context.** D-095 wrote the prefix rule down as naming the **activity**, so that
`/peer-pair` with no arguments printing your own pairing string would not count as a
violation. Pointed out that the existing prefixes plainly name the **target**:
`peer-list`, `peer-forget`, `room-create`, `room-revoke`, `room-join` — every one of
them says what is being acted on.

**That was a rule bent to fit an exception, and the exception was the defect.** Two
commands printed your own string: `/peer-pair` with no arguments, and `/peer-list`
when there are no peers. Both are about other people and both printed you.

**`/self-status` now owns it**, mirroring `/room-status`: "status" already means the
current state of this thing in this set, and you read it for the same reason — what
am I, where am I, can anybody reach me. It replaces nothing, because until now there
was **no slash command for your own identity at all**; `whoami` existed only as a
terminal command, which D-086 says is not a surface to rely on.

`/peer-pair` with no arguments now explains the two halves of pairing and points at
`/self-status`. `/peer-list` with no peers does the same. Neither prints a string
that is not about its target, and `printPairingInvitation` has one caller again.

The cost is a hop for somebody who guessed `/peer-pair` first. Worth paying: the set
will grow, and a rule carrying a documented exception is one people stop trusting.

**No `/help`, and not for want of noticing.** Claude Code owns that name, and the
slash menu already lists all twelve commands with their descriptions, which is the
discovery surface we actually control. What is missing is an entry point that says
what the tool *is* — and that command would be `/<product-name>`, which is what
Phase 13 is blocked on. Do not invent a placeholder: a command name is harder to
change later than a directory is, and the whole point of the blocker is that the
name gets one chance.

**Revisit when:** the name is settled, at which point an overview command has
somewhere to live.

## D-097 — A pairing string carries a chosen name, never a guessed one

**Date:** 2026-09-20 · **Status:** active (implemented) · **Completes** D-094's revisit

**Context.** D-093 made a label compulsory at pairing, which meant somebody had to
type a name for a colleague whose name they obviously know. D-094 wanted the default
to be the name the peer picked, and D-095 made picking one possible. This connects
them.

**The format gains an optional trailing name:**

```
ed25519:KEY@ENDPOINT#Alice
```

`#` separates it, matching the invitation format's use of the same character for a
trailing identifier. Strings without one are ordinary rather than erroneous, so
anything issued before today still parses. The endpoint is untouched, including the
`tc://` form, because the name is stripped before the address is split.

**A guessed name is never carried.** `chosenName` returns nothing while
`NameChosen` is false (D-095). `Ec2-user` arriving as though somebody picked it is
worse than carrying nothing at all: the derived name is at least honest about being
machine-made, whereas a username presented as a name is a claim about a person that
nobody made.

**The receiver's order is: what they typed, then what the sender said, then ask.**
And when the sender's name is used, it says so, because adopting somebody's
description of themselves as your own label is a small act worth stating rather than
performing silently.

**The name is a claim by whoever sent the string**, and a substituted string carries
a substituted name — so a default of "Alice" might come from Mallory. That is safe
only because the label is written after the two words match (D-093), by which point
the string demonstrably came from the person on the call. **Do not move that write
earlier.** A mismatch removes the row entirely, so a substituted name never reaches
the list.

**It is reduced before use.** The name arrives from another machine and becomes a
unique key and an argument to `resolvePeer`, so `sanitizeName` strips what would
break the format (`#`, `@`), span lines, or be invisible, collapses runs of
whitespace, and caps the length at something readable in a list. Asserted both on
what it produces and on the property that matters more: whatever it does, the
endpoint survives.

**Revisit when:** the string carries a third thing, at which point `#` is a
separator with two jobs and wants a real encoding.

## D-098 — A decision that changes what the system is updates the specification in the same pass

**Date:** 2026-09-20 · **Status:** active (spec pass done)

**Context.** Asked whether the answers to two open concerns had ended up in the
specification or only in this log. Only here — and checking found that nine
decisions recorded in one day had produced no specification edits at all, while the
specification had gone on stating the opposite.

**The worst of it was stated as a requirement, not as description.** §29 said *"A
slash command may point at them. It must not perform them"* of pairing and
verification, which is precisely what was built. §31 said pairing and verification
stay at a terminal. Somebody reading the specification first — which CLAUDE.md
instructs, and which section numbers throughout the code invite — would have
concluded the implementation was wrong.

**The division of labour was right and incompletely applied.** This log records why,
including what was rejected; the specification records what the system is. What was
missing is that a decision changing what the system *is* has to update the
specification in the same pass. Recording the rationale felt like finishing, and it
is half.

**This is the same failure as review finding A1**, reopened because the
implementation settled when delivery advances and §19 never caught up. Two instances
of one pattern is a pattern.

**What the pass changed**, with no superseded text left in place:

- **§6** gains three names and what each is for, the distinction between a chosen
  display name and an inferred one, and when a person is offered the choice.
- **§12** gains the third part of a pairing string, the requirement that pairing
  takes a name before the comparison rather than after, and the three endings of an
  attempt. Its worked example and its list of operations were both stale.
- **§20** replaces an interpolated-markup example with the encoded form, and states
  that a fact the receiving side knows is a field of its own rather than text beside
  a claim.
- **§25** gains the local API boundary: loopback is not a boundary against a page in
  this machine's own browser, the requirement is a header such a page cannot send,
  and the referrer is recorded as rejected so it is not added later.
- **§29** replaces two homes with three, retracts the prohibition on a slash command
  performing the ceremony, and states the prefix rule and why there is no command
  named after the system.
- **§31** corrects the phase note that said the ceremony stays at a terminal.

**The specification says what the system is, and nothing about what it was.** The
first version of this pass annotated each correction with the reasoning it replaced
— "this specification previously concluded that they did" — on the theory that the
old rule had been sound and a reader deserved to know why it changed. That is
exactly the burden the no-superseded-text rule exists to remove. These documents are
hard enough to read without carrying every way the system might have worked and
does not, and a reader who wants that has this log, where tracking alternatives is
the whole job. Corrections are clean replacements; the archaeology stays here.

**Revisit when:** never — this is a working rule rather than a decision with a
condition. If it lapses, the symptom is a specification that contradicts the code,
and the check is `git log --name-only` over a day's decisions.

## D-099 — The injected block carries the label, and says it is the name to use

**Date:** 2026-09-20 · **Status:** active (implemented) · **Completes** D-094

**Context.** D-094 put the label in the view and deferred the same field in the
injected block, because `FormatTeamContext` took `verified func(string) bool` across
twenty-two call sites and adding a second function was a signature decision rather
than a field addition.

**The gap was an inconsistency, not an omission.** The view said "alice" and the
model said `Ec2-user` or a word pair, in the same conversation, about the same
person.

**`PeerFacts` groups the two lookups** — `IsVerified` and `Label` — because they are
one idea: what the receiving side knows, as against what the peer asserts. That is
the distinction the block is built on (D-090), and naming it means a third such fact
costs an implementation and nothing at any call site. `Membership` satisfies it
directly. Six callers passed a function and now pass the membership or a `fixedFacts`
stating the same thing about every peer; the rest passed nil and were untouched.

**The framing says what the field is for.** A field the model is given and told
nothing about is a puzzle, so the block states that the label is the name its own
user calls that person by, and that it is preferred over both the self-asserted
`speaker` and the derived `peerName`.

**All three travel.** The label is preferred, not substituted: `speaker` is what the
peer claims and `peerName` is what can be checked, and dropping either to save a
field would remove the thing the preference is safe because of.

**An absent label is an absent field**, not an empty one — `omitempty` — because a
blank value invites a model to wonder what it means.

A test asserts the label reaches the block, that the other two names survive
alongside it, that the framing explains it, and that nothing is emitted when there
is no label. The last check initially scanned the whole block and matched the
framing's own use of the word, which is a reminder that a test over rendered text
should read the payload.

**Revisit when:** a fourth peer fact appears, which should now be an implementation
change only.

## D-100 — Each document answers one question, and a finding is not a commitment

**Date:** 2026-09-20 · **Status:** active · **Generalises** D-098

**Context.** Asked whether findings belong in the specification, in the course of
deciding what a rewritten D-091 should contain. They do not, and working out why
produced a rule that covers more than findings.

**A finding is evidence; a specification statement is a commitment.** Evidence has a
method, a date and a version. A commitment has none of those and is true because it
is required. "Injected context survives compaction" was true of one Claude Code
version under one test; written into the specification it reads as timeless, and
nothing in the sentence says otherwise. The choice then is to let it rot quietly or
to stamp the specification with versions and dates, which turns it into a laboratory
notebook with requirements scattered through it.

The requirement a finding justifies has no such problem. "The delivery watermark is
keyed on session id and does not rewind at compaction" stays true until it is
deliberately changed.

**Review finding C2 is valid and is not asking for this.** It observes that
compaction findings live only in a findings document — and the gap it names is that
the specification makes **no commitment at all** about compaction while the
implementation depends on several. State the requirements; cite the finding only
where a requirement would otherwise look arbitrary. "Findings do not belong in the
specification" and "C2 is a real gap" are both true, because C2 is about the missing
commitment rather than the missing evidence.

**A finding about someone else's software belongs in the behaviour registry.** It is
the only one of these documents that **tests itself**, so a fact recorded there
cannot rot silently — which is the exact failure that makes findings dangerous in a
specification. That is review finding C1 approached from the other side.

**Current-state defects belong in none of them.** This was caught in the draft of
D-091, which listed what the code gets wrong today as evidence for its rule. Those
sentences are false the moment the defects are fixed, and they take the surrounding
entry down with them. A decision states the rule and may say once that the code does
not implement it; the defects are work, and work dies when it is done.

**`CLAUDE.md` is not an exception, though it was first written as one.** The
argument for exempting it — that it loads itself every session, so a duplicate there
is a warning where warnings work — justifies any duplicate anywhere, and its only
real support was that the recitals already existed.

The rule that actually holds is sharper: **what earns a place in an always-loaded
file is what no check covers.** A prohibition, a judgement call, a residual risk.
Where `doctor` already checks a fact, point at the behaviour and state the rule that
depends on it, because the unchecked copy is the one that goes quietly wrong.

Applying it removed fifteen lines and left the file more useful. Two sections
reciting `Stop` and compaction behaviour became statements of what the code does and
why, pointing at B04, B05, B09 and the compaction tier. What survived unchanged is
the one part nothing can assert — that context surviving a compaction is the
summarizer's judgement rather than a format guarantee. The MCP section survived for
the same reason: it records an **absence**, and an absence has no assertion, so the
claim has nowhere else to live.

**Revisit when:** a fifth kind of document appears, at which point the question to
ask of it is which question it answers that none of the others do.

## D-101 — Peer connections are TLS pinned to the key already verified

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asking what a `tc://` address contains led to noticing that `peerURL`
was `http://` and that no `https` appeared anywhere in the codebase.
Confidentiality of room content had never been stated as a requirement, and the
specification discussed it only for pairing strings — where the correct answer is
that none is needed, because the payload is public keys. Room content is a
person's prompts and whatever their Claude said back. Confidentiality is a
requirement.

### What was wrong

When two daemons talked, the request went out as ordinary HTTP over a TCP socket. A
`POST /sync` carrying a JSON body travelled as readable text, and anyone positioned
to carry those bytes — a router, an access point, anyone on the same segment — could
read the conversation.

Signing did not help, and it is worth being exact about why. A signature is a proof
of authorship: it lets a receiver confirm David wrote this event and that nobody
altered it on the way. It does nothing to stop a third party reading it. A signed
postcard is still a postcard.

### What TLS changes

Almost nothing about our HTTP. The routes are the same, the bodies are the same, the
status codes are the same. What changes is that before any of it, the two sides run
a handshake, and afterwards every byte travels inside an encrypted channel.

A dial now goes: open a TCP connection or a tailcat tunnel, which are the same from
here; handshake, in which each side sends a certificate, checks the other's, proves
it holds the matching private key, and agrees an encryption key for this connection
only; then `POST /sync` down that channel; then the response back the same way.
Nothing above the socket changed — `handleSync` still receives an ordinary
`http.Request`.

This happens over **every** transport, including the overlay, which is already
encrypted. The duplication is cheap and it buys the property that matters: nobody
has to reason about whether a particular dial took the safe path.

### What "pinned" means, and why there is a certificate at all

A browser connecting to a bank asks whether the certificate was signed by an
authority it trusts and whether the name matches the site it asked for. That
machinery — authorities, trust stores, expiry, revocation — exists because the
browser has never met that bank and needs a third party to vouch for it.

This is the opposite situation. We already know exactly which key the other machine
holds: a `peerId` **is** an Ed25519 public key (D-042), already confirmed to be that
person's by the two-word comparison (D-055). There is no stranger to vouch for.

So the authority machinery is skipped and one question is asked instead: is the key
inside this certificate the key we already know? That is what `InsecureSkipVerify`
together with a custom `VerifyPeerCertificate` does. The name misleads — it disables
the checks that assume an authority, and our own check runs in their place. It is
stricter than what it replaces, not weaker: an authority will vouch for millions of
keys, and this machine accepts about five.

There is a certificate at all only because the TLS wire format requires one. It is a
container. Each daemon generates a self-signed one at startup from its identity key,
and nobody pins the certificate — they pin the key inside it. That is why
regenerating it on every restart costs nothing, and why its expiry date is
meaningless here.

A client certificate is **required** rather than requested. A peer with nothing to
pin has no business completing a handshake, and refusing there is earlier and
clearer than refusing after a body has been read.

**The test is that the key is recorded, not that it is verified**, and that is not a
weakening. Verification happens *over* this connection: it is how a recorded peer
becomes a verified one. Requiring verification to connect would make the only route
to verification unreachable, so nobody could ever be verified and pairing would
never complete. Tightening this check would read as an obvious hardening and is a
deadlock.

An unverified peer connecting is harmless because the gate that matters is a layer
up: D-054 refuses its events in both directions, so it may complete a handshake and
a ceremony and still move no transcript until a person has compared two words.

Note that being recorded is per machine. Running `/peer-pair` records the colleague
on *this* machine and puts nothing on theirs, which is why the person who types
first cannot connect until the other types too — not because the pin is strict, but
because the far side has no record of them yet. The attempt retries, so typing a
few seconds early is ordinary rather than an error.

### The two situations a dial can be in

**Verifying a specific person.** We know exactly who we are calling: the ceremony is
with David and we hold David's key, so we demand that key. A machine at that address
presenting anything else is refused. This matters because an address can quietly
change hands — a home address gets reassigned, somebody else's machine answers — and
without the demand we would begin a ceremony with a stranger.

**Syncing a room.** We do not know who will answer. A room stores the addresses
where its members listen, not a mapping from member to address, so there is no
particular key to demand. Any key belonging to a peer this machine knows is
accepted, and everybody else refused. Identity is still pinned down a moment later,
because the request inside the connection is signed and the signature says exactly
who sent it (D-044). The handshake narrows it to "a known peer"; the signature makes
it "this peer".

### Why the URL scheme is load-bearing

Go's HTTP client decides whether to encrypt by looking at the URL scheme, not by
whether a TLS config was supplied. An `http://` URL opens the connection and starts
speaking HTTP immediately, ignoring `TLSClientConfig` entirely. An `https://` URL
performs the handshake with that config first and speaks HTTP inside the result.

Peer URLs are `https://peer/sync`. The hostname is a placeholder that is never
resolved, because `clientFor` installs a `DialContext` that ignores the address in
the URL and calls our own dialler. The scheme does one job: it tells Go to run the
handshake.

Changing it back to `http://` would send every connection in plaintext, leave the
certificate config silently unused, and **break nothing** — sync would work
perfectly and unencrypted. That is the worst failure shape available, so a test
asserts the scheme on its own.

### Why this superseded the interim guard

Before TLS existed, a rule was added refusing any bare-TCP connection to an address
not on this machine, on the reasoning that bare TCP carried cleartext and loopback
was the only safe case — an SSH tunnel makes both ends look like loopback and
encrypts between them.

That rule was about a property of bare TCP: that it is unencrypted. The property no
longer holds, because a bare TCP connection now carries a TLS session.

Keeping it would have been actively wrong rather than merely redundant. It would
have gone on refusing exactly the connections TLS had made safe, so building TLS
would have bought nothing — peers on different networks would still be blocked, and
the only way through would have been the escape hatch, which turns the check off for
everybody. The guard went, and the environment variable with it. There is now no
setting that disables encryption, which is the point: a security property with an
off switch is one somebody eventually switches off.

### Considered and rejected: deriving a key exchange from the same keys

The keys are already there, so an exchange could be built directly on them and
certificates avoided. Rejected, because what that avoids is not certificates but a
reviewed implementation: it means writing a handshake, nonce discipline, a replay
window and rekeying, where the primitives are sound and the composition is what
fails silently. Pinning already achieves the property the idea was reaching for —
that the transport's identity and the event's identity are the same key, confirmed
by the same comparison.

### What this does not do

It protects content **in transit only**. Events are stored unencrypted in each
machine's room database, so anyone who can read the disk can read the rooms.

It does nothing to protect a room from its own members. A peer admitted to a room
may read it; that is what admission means. This stops outsiders carrying the bytes,
not insiders holding them.

### A note for whenever a second person runs this

The handshake is a hard version boundary. A peer built before this cannot connect at
all, rather than degrading to something that works less well. Nobody else has run
it, so the cost is zero now and would not be later.

**Revisit when:** content needs protecting from a room's own members, or at rest.

## D-102 — The reveal step tolerates a peer that has already finished

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Making every peer connection TLS (D-101) turned a test that had passed
for weeks into one that failed about one run in six. The handshake did not break it;
it changed the timing enough to expose a race that was always there.

**The two halves of the exchange were not treated alike.** A commit that reached a
peer not yet expecting a verification set `lastErr` and retried, which is right: the
other person may not have run their side yet, and waiting is the whole point of the
ninety-second window. A reveal that got the same answer returned it.

Nothing is not-yet-ready between one call and the next, but something can be
*no longer* ready. The other side can complete between our commit and our reveal,
and its deferred cleanup discards the session, so the reveal lands on a daemon that
is not expecting one. One side then finished with two words while the other reported
that its peer was not expecting a verification.

**It recovers by looping rather than by retrying that call.** By the time the other
side has finished it has already sent its own reveal, so its nonce is in this
session, and the inbound check at the top of the loop finds it. The fix is to give
the reveal the same tolerance the commit has.

**A test that passes is not a test that holds.** This one ran green through the
whole of Phase 5 and the two-machine runs, and the defect was reachable the entire
time — it needed one side to finish inside the window between another's two calls.
Treat a timing change that breaks an old test as evidence about the test's coverage
before assuming it is evidence about the change.

**Revisit when:** the exchange gains a third round trip, which would widen the same
window.

## D-103 — An address belongs to a peer, and is stored in one place

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** Asked why a newer address would not update both stores, and then more
pointedly why an address is stored in more than one place at all. Following that
found a modelling error underneath several problems that had been treated as
separate.

**Addresses live in two tables.** `known_peers.endpoint` holds one per peer, written
at pairing. `room_peers(room_id, endpoint)` holds addresses per room, with **no peer
column**, written when a member synchronizes and when an invitation is accepted.

**Both writers hold the identity and discard it.** The synchronization handler calls
`AddRoomPeer(room.RoomID, req.Endpoint)` one line after `verifyRequest` has
cryptographically established `req.PeerID`. The join path calls it two lines from
`Allow(host, "")`, where the host's identifier came out of the invitation. There is
no path by which an address arrives without an identity attached; the identity is
thrown away at the moment of writing.

**Five problems, one cause.** Each of these had been treated as its own defect:

- verification dials every address this machine knows, because it cannot ask which
  one is a particular peer's;
- an unexpected key therefore cannot be an alarm, because dialling the wrong peer is
  the expected case rather than a surprise;
- room addresses accumulate for ever, because `INSERT OR IGNORE` cannot overwrite
  per peer when there is no peer;
- invalidation has nowhere to live, since expiring "that peer's stale address"
  requires knowing whose it is;
- a tunnel-level allow-list cannot be built from them (D-104), because it needs each
  peer's node key and a bare address does not say whose it is.

**The replacement is a join that already has both halves.** `Guests(roomID)` returns
peer identifiers, and `known_peers.endpoint` holds an address per peer. So *where do
I poll for this room* becomes *the guests of this room, excluding me, and the address
recorded for each* — which yields identity along with every address, so a failure can
be attributed, a peer that moves overwrites one row rather than adding another, and
expiry has something to attach to.

**Checked: no room holds an address for a non-guest.** `verifyRequest` refuses a
request from anybody who is not a guest of that room before `AddRoomPeer` is reached,
and the join path admits the host in the same breath as recording them. The join is
therefore exact rather than approximate.

**One wrinkle, deliberately made explicit.** `Invite` requires only that an
identifier name a key, and `CreateRoom` invites the creator, so **you are a guest of
your own rooms and are never in your own known-peers list**. The join drops you
because no address row exists for you, which is the right outcome reached by
accident. Exclude self by saying so, rather than relying on a missing row that
somebody will later add for an unrelated reason.

**This removes work rather than adding it.** The pruning machinery for accumulated
room addresses becomes unnecessary — one row per peer, replaced when they move, gone
when they are forgotten. So does any reconciliation between the two stores, and so
does the argument for verification's fallback sweep: that argument was that
`room_peers` might hold a fresher address than `known_peers`, and with one store that
divergence cannot occur.

**What the §4 lifecycle assumed.** It was written describing an address that enters
from several sources, ages, and is discarded — all of which is right, and all of
which the schema cannot express while an address has no owner. The prose described a
model the tables could not hold.

**Revisit when:** an address is legitimately held for something that is not a peer.

## D-104 — The overlay address is public and stable; admission moves to a list

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** Somebody prints their pairing string at a coffee shop, is interrupted,
and sends it from home two hours later. Is the address still good? Following that
found the address is not stable at all, and that it contains a secret.

**The address changes on every restart, and the cause was measured.** A tailcat
address encodes three keys and a relay. Two of the keys derive from the node key,
which is persisted. The third is a **pre-shared key**, and the library generates a
fresh one at every `Start()` unless told not to. Restarting a daemon twice and
comparing the published endpoints:

| | two starts identical |
|---|---|
| pre-shared key disabled | yes — byte for byte |
| pre-shared key enabled | no |

So every reboot, plugin update or bad wake from sleep silently invalidates every
pairing string and every invitation that machine has issued. That makes §4's claim
that an address need only be correct once false for the overlay path, which is the
path everybody uses.

**And it puts a secret in a string this design requires to be public.** The
library's own guidance is to treat an address containing a pre-shared key as secret.
§12 says the opposite, and must: the string is pasted into chat and read aloud, and
the two-word comparison exists precisely so that it need not be confidential. A
secret in that string reintroduces the bootstrap problem the ceremony was built to
remove.

**The two cannot both hold at that layer.** The key's purpose is a post-quantum
hedge on recorded traffic, which requires it to stay secret, and it travels inside
the address, which must be published. No arrangement of one node's address escapes
that. A second address would mean a second node key, which is the identity.

**So the access-control half moves to a list, where it is better.** The library
accepts a set of client node keys and refuses anybody else, and a peer's node key is
recoverable from the address already recorded for them. Measured: a peer absent from
the list cannot open a tunnel, and adding one while the server runs takes effect
without a restart.

| | pre-shared key | allow-list |
|---|---|---|
| admits | anyone holding the string | only recorded peers |
| revocable | no; rotating strands everyone at once | per peer |
| secret in the address | yes | none |
| survives a restart | no | yes, rebuilt from the peer list |

**A measured cost: the list refuses by silence.** A peer not on it receives no reply
at all, so the caller waits out its dial timeout rather than being told. Today an
unrecorded peer is refused by TLS in milliseconds. This makes the ordering in D-103
load-bearing rather than tidy: dialling one known address must come before any
sweep, or a verification round spends a minute on addresses that were never going to
answer.

**The hedge is genuinely lost, and is recoverable elsewhere.** Nothing else in the
stack is quantum-resistant — the TLS layer keys on the same elliptic curve — so
removing it removes the only such element. It can be rebuilt at our own layer, from
material the pairing exchange already produces, per peer rather than one secret
shared with everybody ever paired with. That is a separate decision and is not
blocked by this one.

**Also pin the region.** The published address embeds a relay chosen at startup by
latency, with a random fallback when the probe fails. Disabling the pre-shared key
made two starts identical on one network; it does not follow that a start on another
network picks the same relay. Persisting the chosen region alongside the node key
makes the address a property of the identity rather than of the last startup.

**Revisit when:** the post-quantum hedge is wanted, or the library offers a per-peer
pre-shared key.

## D-105 — An invitation travels over the channel pairing already established

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** While tracing how a peer recovers from an address change, the
hand-carried invitation was defended on the grounds that it survives one-way
reachability: the string goes host to guest by human means, and the guest then dials
the host, which is the direction that works when the guest has moved. Answered that
the trade is the wrong way round — recovery from an address change is an outlier and
forming a room is an everyday act.

That is right, and the argument is stronger than it was put. The robustness being
defended protects a case D-104 largely removes: with a stable overlay address, a
host becomes unable to reach a guest only after a restart *and* an unreachable
relay. So the trade was an everyday inconvenience against an outlier of an outlier.

**§12 already describes the intended shape**, and the implementation is what
diverged. The specification has the guest typing a room name, with the host's daemon
recording the admission and the guest's daemon locating the room — and no account of
how it locates it. Lacking a mechanism, the implementation answered by printing a
full invitation string and requiring it. This supplies the missing mechanism rather
than changing the goal.

**An invitation becomes an offer delivered over the paired channel**, with the
string kept as a fallback when the host cannot reach the guest.

**Pushing an offer is not pushing an admission**, which is what makes it safe. The
guest-list row on the host's side is the admission, and it is the host's judgement
to make (D-024). Delivering an offer only tells the guest that row exists. Joining
remains the guest's own act, nothing is entered on their behalf, and reading
anything still requires verification (D-054). §3.7 is untouched: an offer arriving
is data being stored, not a turn being taken.

**The failure is better than the one it replaces.** A host who cannot reach the
guest is told at the moment of inviting and handed the string to send. Today the
string is the only path, and nobody learns whether it was needed.

**Rejected: the guest polling for offers.** It works in the direction that survives
a moved guest, and it matches the anti-entropy pull the rest of the system uses. It
was not taken because it requires every pair of paired peers to maintain contact
indefinitely, whether or not they share a room — standing traffic and standing
metadata, permanently, for something that happens rarely. Push has an immediate and
honest failure with a working fallback, and costs nothing between invitations.

**This mostly dissolves Phase 14.** "First contact without a paste" is the same
problem approached from the other end. Once an invitation reaches a paired guest
over the channel, the only remaining manual exchange is between people who have
never paired, which D-051 declines to support.

**Revisit when:** paired peers acquire a standing reason to poll each other, at
which point pull costs nothing extra and is the better shape.

## D-106 — An offer is delivered when it can work, not when it is made

**Date:** 2026-09-21 · **Status:** active (implemented) · **Refines** D-105

**Context.** D-105 makes an invitation an offer pushed over the paired channel.
Inviting an unverified peer is permitted and inert (§25), so the two together would
make a room that does nothing easier to create than one that works.

**What today's friction was hiding.** A host reads the warning, copies a string,
sends it to the guest. Even a host who ignores the warning passes through a human
moment where noticing is possible. Push removes that: invite, accept, and both
people believe they are in a room which is permanently silent. The join-time warning
becomes the only thing between them and that, and it travels by the weakest channel
there is — a line a command printed, relayed by a model.

**So the admission is recorded and the offer is withheld.** Inviting an unverified
peer still writes the guest-list row, immediately. Whom to admit remains the host's
judgement and §25's separation of the two questions is untouched. What waits is the
*delivery of a notification*, and it waits only as long as it would be useless.

**Why the gate stays open at all, which is the load-bearing half.** Refusing to
invite an unverified peer would be simpler — one fewer state for a pair to be in,
and no queue to flush. It is declined because **wanting to start a room is what
makes somebody willing to verify**. Verification is a chore whose payoff is
invisible until it is needed, and attempting a room is the moment it acquires one.
A refusal blocks a person exactly when they are motivated and sends them off to do
an errand; a queued invitation meets them there. Expect the simplification to be
proposed on the grounds of fewer combinations, and that is the answer: the extra
state is one pending offer with one flush point, bought with the only moment this
system gets somebody's attention for free.

**Which decides the wording, not only the behaviour.** The difference between a
block and a path is whether the invitation succeeded and is waiting. It must read as
admitted, queued, and one step from done — with that step offered where it is
stated, since the two-word check opens in a browser from the same place (D-088). An
invite that reports a refusal throws away the advantage this entry exists to keep.

**Verification gains an effect: it flushes what was waiting.** When two peers
complete the two-word comparison, offers already recorded for that peer are
delivered. So the sequence a host expects — invite, then verify — produces the room
at the end of it, rather than producing a silent one in the middle and repairing it
later.

**The fallback string is withheld on the same terms.** It would otherwise be a way
to reach the outcome the withholding exists to prevent: a guest who pastes a string
joins and gets the same dead room. Inviting an unverified peer therefore produces
neither a delivery nor a string, and says what to do instead.

**Rejected: deliver it, marked unverified.** The guest's daemon could receive the
offer and say that nothing will sync until both verify. It was not taken because a
notification that produces a dead room is worse than no notification — it reads as
progress, and the only thing correcting it is a sentence somebody may not read. An
offer that arrives when it works needs no warning at all.

**Self is a guest of its own rooms and is never in its own peer list** (D-103), so
any check of "is this guest verified" must exclude self rather than conclude that
the creator is unverified and withhold from them.

**Where the two waiting things surface, since they point in opposite directions.**
An offer waiting to be accepted is a room, and belongs with rooms — it is listed
apart from the rooms already joined, because accepting is the act that has not
happened. An invitation withheld for want of a verification is not visible to the
person it is for and is entirely visible to the person who can release it, so it
belongs beside that peer.

**Name the person, do not count the rooms.** A tally was tried first and is close
to useless here: the ordinary room holds two people, so the number is always one
and carries nothing. What somebody can act on is which colleague is unfinished and
what to type. It must also stop being said once the pairing completes, rather than
persisting as a record of work already done.

**The appeal is to finish pairing, not to verify.** Pairing is the act a person
recognises; verifying is our word for a step inside it, and nobody thinks "I must
verify Alice". So the prompt says there is work to do with Alice and offers
`/peer-pair alice`.

That required the command to accept a name. It previously took only a pairing
string, so `/peer-pair alice` hashed the literal text into an identifier and
invented a peer nobody had met — worse than refusing, because it produced a
plausible stranger. It now resolves a recorded peer and runs the ceremony, which is
the second half of an act rather than a new one, so it asks for no label: they were
named when they were recorded.

**Revisit when:** admission and verification stop being separable — if a future
change makes one imply the other, withholding has nothing left to sequence.

## D-107 — Pairing again with somebody already paired does nothing, loudly

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** Asked whether the pairing command is idempotent for a peer who has
completed all or part of it. Partly: resuming an unfinished pairing worked, and
running it on a finished one silently began the whole ceremony again.

**Repeating the ceremony is not a harmless no-op.** It needs the other person at
their machine at the same moment, so an unrequested one leaves somebody watching a
page count down ninety seconds for a colleague who was never asked. A command that
looks idempotent and quietly demands coordination from a third party is worse than
one that refuses.

**So the three states answer differently.**

Not recorded, given a string: record and run the ceremony. Recorded but unfinished,
given a name: resume — this is the second half of an act, and the reason the
command accepts a name at all. Already paired: say so, say since when, and stop.

**A string for somebody already paired changes nothing, including the address.**
This entry first took the address out of it, on the reasoning that a colleague who
moved has no other way to tell you. That was wrong twice over, and is superseded by
D-108.

**Re-verification stays available and must be asked for.** `--again` exists for the
case it is actually needed in: somebody read two words that did not match, or was
told to check. The message names that case rather than offering the flag as a
general option, because a person who re-runs a ceremony without cause learns that
the ceremony is routine, and it is the opposite.

**Checked before anything is named.** The question "what will you call them" must
not be asked of somebody who was named when they were recorded — asking it implies
the answer might change something.

**Revisit when:** a pairing can expire, at which point "already paired" needs to
say until when rather than since when.

## D-108 — Pairing is not how an address is updated

**Date:** 2026-09-21 · **Status:** active (implemented) · **Supersedes** part of D-107

**Context.** D-107 made a pairing string offered for an already-paired peer update
the stored address while skipping the ceremony, on the grounds that a colleague who
moved has no other way to tell you. Rejected on sight, and both halves of the
reasoning turn out to be wrong.

**It is the wrong shape.** Pairing is a security act, and a security act that
silently writes network state is a quiet mutation of the kind this design keeps
removing. It also teaches the wrong reflex: re-pasting somebody's string becomes a
maintenance chore, when handing over a pairing string should be rare, deliberate,
and attached to a ceremony. A string that gets pasted routinely stops being
treated as significant.

**And the gap it filled does not exist.** The case was two peers paired with no
room between them, one of whom moved, so nothing polls and nothing heals. But
nobody needs the address in that state. The next thing that happens is an
invitation, and a host who cannot reach a guest is told so and given a line to send
by hand (D-105). The guest pastes it, joins, and their first synchronization
carries their current address back — repairing the record as a side effect of the
thing they were trying to do anyway.

**An address matters only when it is used, and every use repairs itself.** That is
the general form, and it is why no command for setting one is needed either. A
dedicated address-update command would be a mechanism for a problem that resolves
itself, and one more thing to explain.

**What remains true from D-107:** already paired says so and stops, and `--again`
exists for a comparison somebody has reason to repeat. Only the address write is
withdrawn.

**Revisit when:** a peer can move while sharing no room with anybody and still need
to be reached — which would mean something other than an invitation had come to
depend on a stored address.

---

## D-109 — Two is the target and nothing rules out more

**Date:** 2026-09-21 · **Status:** active (standing constraint)

**Context.** D-032 (the order of work) records in an aside that "pairs remain the
target" and that Phase 6 waits for evidence a third peer is wanted. That is half a
principle, and the recorded half is the one that needs no enforcing. The half that
governs day-to-day work has been applied consistently and never written down: while
bearing down on two, **avoid anything that forecloses more**.

The two are easy to confuse and pull in opposite directions. "Two is the target"
argues for spending nothing on a third peer. "Nothing rules it out" argues for
noticing when a shortcut would make a third peer a rewrite rather than a feature.
Neither is the whole rule, and holding only the first is how a prototype acquires a
ceiling nobody chose.

**Decision.** Build for two people in a room. Spend no effort on a third. But treat
any design that *cannot* extend past two as a defect to be argued for explicitly,
not a saving to be taken quietly.

The distinction is between a **cost** and a **ceiling**. Sync is a pull against a
per-peer watermark and guest lists are per-peer (D-046, the daemon serves many
rooms); both would be O(n) work with more peers and neither breaks — those are
costs, and they are fine. A pairwise ceremony assumed to be the only shape of
admission, or a room identity derived from exactly two identifiers, would be
ceilings. So would describing the tool by a count, which is why the word "several"
does not belong in its first line either.

This is also why several things already built are shaped the way they are.
Transitive relay (§13) and immutable events with per-peer sequences (§7) only pay
for themselves past two peers; both were built anyway, because retrofitting event
identity is not possible once events exist.

**Rejected.**

- *Design for N now.* Every hard problem so far has been specific to the pair in
  front of us, and a generalisation written before the second case is a guess. It
  also costs the thing the prototype is for: evidence about whether two people
  actually use it.
- *Optimise for exactly two and revisit later.* This is the position that sounds
  identical to the decision and is not. It permits the ceiling, and the moment an
  event format or an identity scheme has one, D-058 (signature schemes are added,
  never edited) and §7's immutability make it permanent.
- *Leave it in D-032's aside.* An aside inside a decision about phase ordering is
  not where somebody looks before choosing a data structure, which is the moment
  this rule applies.

**Revisit when** evidence arrives that a third peer is wanted — the trigger D-032
already names. Note that the trigger releases the first half of this rule and not
the second: the second half has no expiry.

---

## D-110 — Host order: Claude Code, CoWork soon after, ChatGPT Desktop much later

**Date:** 2026-09-21 · **Status:** active (ordering; alters neither D-043 nor D-086)

**Context.** The order of target hosts was recorded nowhere. One sentence in D-086
(the terminal is not a user experience) carries it as context for a user-interface
argument — "Claude Code is the first host; Claude CoWork is wanted as a fast follow"
— and ChatGPT Desktop appears nowhere in the repository. An ordering that governs
scope should not be a supporting clause inside a decision about pairing surfaces.

**Decision.** Three hosts, in order: **Claude Code**, **Claude CoWork**, **ChatGPT
Desktop**. The interval from Claude Code to CoWork is short. The interval from
CoWork to ChatGPT Desktop is long, and long enough that the two are not planned
together.

**The intervals are the load-bearing part, not the order.** An order alone says only
that ChatGPT Desktop is third, which nothing acts on. The gap sizes are what decide,
for any given piece of host-independence, whether it is preparation or speculation.

**This licenses nothing.** D-043 (the wire format is not the database row) forbids
`source`/adapter machinery, and D-086 already considered this exact move and refused
it: "naming a second host does not change that." Naming a third does not either. The
reasoning is untouched — capture generalises, injection does not, and every hard
problem so far has been host-specific. This entry records an intention, not a
permission.

**What the short interval does support** is D-086's cheapest-preparation argument,
now with a nearer payoff. The daemon and its view are host-independent by
construction — a local service and a web page, not an extension of anything — so
every piece of experience living there is one a second host does not reimplement.
That was justified on today's host alone. A short gap to a same-vendor host makes it
a good bet as well as a justified one, which is a reason to prefer the view for new
surfaces, and not a reason to abstract anything.

**What the long interval settles** is that nothing is designed against ChatGPT
Desktop. Different vendor, unknown extension model, and the parts most likely to
differ are precisely the ones D-043 named as non-generalising: §3.5's thinking-block
exclusion is a fact about one transcript format, D-014 (delivery is confirmed by
observation) depends on evidence appearing in a specific file, and the whole
injection path assumes a hook that runs before a turn. A host with no hook
equivalent does not need a smaller adapter; it needs a different design, and that
design is not written before the host exists.

**A consequence for the specification, recorded and not resolved.** §3.8 — "Claude
Code is launched and used unchanged" — is the single test applied to every proposal,
and it is written as a statement about one named product. With a host order it must
either become host-general (*the host is launched and used unchanged; we install
only what it already loads*) or stand as a Claude Code rule at the head of a project
with three hosts. Deciding that is a separate pass; naming it here keeps it from
being decided by drift.

**Rejected.**

- *Leave it in D-086.* Where an ordering lives determines whether anybody finds it,
  and the person who needs this one is scoping a host, not choosing a pairing
  surface.
- *Put it in §31's phase order.* A phase needs a placement, and CoWork's placement
  depends on its extension model — which is D-086's own revisit trigger. An ordering
  of intent is not yet a phase.
- *Record the order and omit the intervals.* The order without them is inert: it
  cannot tell anybody whether host-independence work is early or premature, which is
  the only question the ordering is consulted for.

**Revisit when:** CoWork's extension model is known (D-086's trigger, unchanged); or
ChatGPT Desktop moves near enough that the long interval stops being the operative
fact; or a fourth host is wanted, at which point an ordering of intent has probably
become a roadmap and wants a different home.

---

## D-111 — §3.8 is a test about hosts, not about Claude Code

**Date:** 2026-09-21 · **Status:** active (amends §3.8; resolves the question D-110 left open)

**Context.** D-110 (host order) recorded that §3.8 — the single test applied to
every proposal before anything else — is written about one named product, and left
the resolution open rather than letting it be decided by drift. With three hosts
named, §3.8 either becomes host-general or stands as a Claude Code rule at the head
of a project that has committed to more.

**Decision.** §3.8 now reads *The host is launched and used unchanged*. It defines
**host** as the application a session runs in, names Claude Code as the host today,
and states every clause of the test about the host.

**The test itself did not change.** The same three clauses put a proposal out: a
different launch command, installing something the host does not already load, or
depending on how the host renders. It still rules out the PTY wrapper (D-034) and
still permits a view outside the session, on the same grounds as before.

**Generalising exposed two things that had been resting on the single host.**

*First, "things the host already loads" was ambiguous, and had been carrying an
unstated word.* The original said "already loads **on its own**", and dropping it in
summary produced a phrase that could not be read at all. The sense is **something the host would load
anyway** — a sort of extension it loads already, for its own reasons, with nobody
having changed how it starts
— not a file that happens to be loaded, and not something that could be made to
load. Against Claude Code alone the distinction never had to be drawn: hooks, skills
and MCP servers are plainly in, a PTY wrapper is plainly out. Against a host nobody
has examined, "could it be made to load something?" has a yes answer for almost any
program, and the test collapses. §3.8 now says which question it is asking.

*Second, what the test says about a host offering no way in.* It says the host is
out of reach. That is the intended reading rather than a new rule — the constraint
is a product boundary, so a host that cannot be extended is a host this system does
not serve, and is never an argument for wrapping one. This does not conflict with
D-110's note that a host lacking a hook equivalent needs a different design rather
than a smaller adapter: that concerns hosts with some other way in.

**The rule generalises; the evidence for it does not.** The reach argument — Claude
Code is a terminal program, a desktop application and an editor extension, and
extending it through its own mechanisms reaches all three — is a fact about one
product and stays stated as one. A different host has different surfaces, which
changes which mechanisms exist and changes nothing about the reasoning.

**Rejected.**

- *Leave §3.8 naming Claude Code, and add a note that it applies to other hosts.* It
  is consulted as a test, and a test carrying a footnote about which product it
  governs is a test people apply inconsistently.
- *Generalise the whole specification.* §3.8 is the framing test; the rest of the
  document describes Claude Code's actual mechanisms and is correct as written.
  Host-general language throughout would claim a generality nothing else in the
  document has earned, and would make every section quietly harder to check.
- *Wait for CoWork.* The rewrite costs the same now as later, and the ambiguity in
  "already loads" is a defect today, with one host and no prospect of a second
  mattering to it.

**Revisit when** a second host is actually supported — at which point §3.8's Claude
Code examples want a companion rather than a replacement — or if a host appears
whose extension points reach a person directly, which would make §3.8's consequence
subsection an understatement rather than a description.

---

## D-112 — Several sessions from one machine are not told apart in the block

**Date:** 2026-09-21 · **Status:** active (considered and not built)

**Context.** Nothing stops one machine having two live sessions in a room:
`session_rooms` is keyed on the session because sessions are members (§22), while
`room_guests` is keyed on `(room_id, peer_id)` because peers are admitted. They are
different questions and the schema answers them separately. `UndeliveredFor`
excludes `origin_session_id`, not the peer, so the two sessions correctly see each
other's turns.

But identity is per machine, so both carry the same `peerId`, the same derived
`peerName` and the same `label`. The injected block's JSON has `speaker`,
`peerName` and `label` and no session field, so a reader receives two threads
interleaved under one name. `originSessionId` is on the event and unsurfaced, so
the material for a discriminator is already there.

**Considered: a per-block discriminator derived from `originSessionId`**, stable
within a block and not the raw identifier, so threads separate without naming the
agent (D-043).

**Rejected, because it distinguishes without informing.** It reports process
topology, not work. It says turn 7 came from a different session than turn 6 and
nothing about whether that matters — two halves of one feature, or one continuing
after a crash, or two unrelated things all render the same. The reader cannot act
on it.

It is also worse than silence, because it invites an inference it cannot support:
two marked streams read as two topics, or as two people. That is the false
affordance this project already refuses in the view, appearing as a data field
rather than as a control.

**Consistent with how identity is displayed elsewhere.** D-021 suppresses the
derived name on your own turns, where it identifies nothing you did not know, and
D-099 prefers the label because a word pair means nothing to a person weeks later.
A session discriminator is the same kind of thing: an identifier the machine can
derive and the reader cannot use.

**How narrow the problem actually is.** A crash replacement is sequential — the
dead session emits nothing, so nothing interleaves. Unrelated work is
self-correcting, because a session in no room captures nothing and joining is
deliberate (D-064). That leaves deliberately split work, which is one collaboration
by construction, and where **the person knows why and the reader does not**. An
asymmetry of knowledge is not repaired by deriving a marker from a session
identifier.

**What would carry information is a purpose the person supplies** — "this is the
frontend half" — which is a feature with a real cost, and is not justified without
evidence that anybody runs split sessions.

**Revisit when** there is evidence that two live sessions from one machine is an
ordinary thing to do, rather than a thing the schema permits.

---

## D-113 — The install test names a type, and the type must be documented

**Date:** 2026-09-21 · **Status:** active (amends §3.8; tightens D-111)

**Context.** D-111 stated the install clause as "something the host would load
anyway". That says what the test is asking and not how to check it: *would load
anyway* is a claim about a host's behaviour, and behaviour is the one thing nobody
outside the vendor can establish. Against a host nobody here has examined — D-110
names two — it is unfalsifiable, which is fatal for a test whose whole merit is
being applicable without argument.

**Decision.** This project installs only **artifacts of a type included in the
host's demonstrated, documented extension mechanisms**. In Claude Code: hooks,
skills, MCP servers, and the plugin that carries them.

Two halves, both load-bearing.

- **Type, not instance.** What an artifact *does* is unconstrained; what it *is*
  must be on the list. A hook that replicates a conversation to a peer is a hook,
  however little anybody anticipated one. This is not a rule to do only what the
  host imagined.
- **Documented, and demonstrated.** Documented is the vendor saying the mechanism
  exists on purpose and is meant to keep existing. Demonstrated is that it works in
  the version installed, which `doctor` is already the means of establishing. A
  mechanism with one and not the other fails.

**What this catches that the old phrasing did not.** `NODE_OPTIONS="--require
shim.js"` against an npm-distributed host: the runtime loads arbitrary code into the
process, no launch command changes because the variable can be exported from a
profile, and rendering inside the session — the thing D-036 and D-038 keep running
into — becomes possible. Asked as "would the host load it anyway", it invites an
argument about what counts as the host, since Node genuinely does load it. Asked as
"is a require hook among Claude Code's documented extension mechanisms", it is
simply no.

**This does not conflict with the behaviour registry**, which is the first
objection a reader will raise, because `behaviors.go` records undocumented
behaviours this project depends on and there are many. The axes are different. The
registry governs how a *documented mechanism behaves* — hooks are documented,
`Stop.last_assistant_message` carrying only the tail (B04) is not. This rule governs
what *type of artifact* is installed. Depending on undocumented behaviour of a
documented mechanism is the ordinary condition here, and is exactly why the registry
has checks with negative tests. An undocumented artifact type has no such recourse:
there is nothing to check against, because nothing was promised.

**Rejected.**

- *"Something the host would load anyway"* (D-111's phrasing). Right in intent, and
  it asks a reader to reason about a host's internals instead of looking something
  up.
- *Demonstrated alone.* Admits `NODE_OPTIONS`, which demonstrably works.
  Demonstration establishes that a mechanism exists today and never that anybody
  intends to keep it.
- *Documented alone.* Costs nothing to drop the other word and leaves us installing
  against a documentation page the shipped version does not honour — which is the
  failure `doctor` exists to catch, so the second half is already paid for.

**Revisit when** a host's documentation lags a mechanism this project needs and the
mechanism is plainly deliberate. The question then is whether "demonstrated,
documented" should weaken to "demonstrated and publicly intended", which is harder
to apply and is the reason not to reach for it first.

---

## D-114 — §3.1 is a duty not to harm the session, not a claim about data locality

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
limits), D-041 (starting must not delay the session) and D-038 (the view is a
separate program, so a busy room never costs somebody their pane) are all the same
obligation, and nothing said so. Each read as a local judgement; together they are
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
  `spec-review.md` likewise. The term survives once in the spec, at §11's
  "local-first CRDT such as Automerge", where it names a category of software and
  makes no claim about this one.

**Revisit when** a harm is found that none of the six cases anticipates — which is
expected, and is why the duty is general. Add the case; do not narrow the duty.

---

## D-115 — A centralized component must trace to a disclosed tradeoff that benefits the person

**Date:** 2026-09-21 · **Status:** active (standing constraint)

**Context.** §32 lists central services among things not to build, as items on a
non-goals list ending "these can be evaluated after the core experiment". That is a
list, not a rule. It cannot govern a component that turns out to be necessary, and
three centralized or third-party components already exist with nothing governing
them.

The outlook recorded here is not privacy maximalism and not an objection to
centralization as such. It is that the count should be as small as the problem
allows, and that whatever remains should be explicable.

**Decision.** A compromise to privacy or to decentralization is acceptable when all
three hold.

1. **It traces.** A recorded decision says what was given up and why, so the
   reasoning can be found rather than inferred from the code.
2. **It is disclosed.** The person whose data it concerns is told, in something
   they actually read — not in this specification, which they will not.
3. **It benefits them.** The gain accrues to the person using this, not to the
   people building it. A compromise buying us convenience, operational simplicity
   or data is not one of these, however small.

Failing a test is not a prohibition on building the thing. It says the thing is not
yet defensible, and names which part is missing.

**Applied to what exists**, which is what makes this operative rather than
aspirational.

- **The DERP relay.** *Traces* — D-062 evaluated it, and §31's Phase 13 question
  asks openly whether depending on a relay nobody here operates is acceptable.
  *Benefits the person* — two people behind strict NAT cannot reach each other
  otherwise, so the alternative is not a purer system but no collaboration. *Not
  disclosed.* **Two of three.**
- **The model provider.** *Traces* — §28 states that injection carries a
  colleague's conversation to another person's provider under their own account,
  logged as review finding A3. *Benefits the person* — it is the product. *Not
  disclosed*, and it is the least obvious of the three: a colleague's words reach
  **your** provider on **your** subscription, which nobody would assume. **Two of
  three.**
- **The release host — pre-GA, and tagged as such.** *Traces* only partly: D-066
  decided the binary is fetched and verified rather than shipped, and the choice
  of host is not a decision at all but a default in `publish.sh`, a
  personally-operated droplet reached over plain HTTP at a bare IP
  (`plugin/release-url.txt`). Integrity is held by `plugin/checksums.txt` rather
  than by the transport, so tampering is caught; the transport still shows an
  observer which version was fetched. *Benefits us rather than the person*: it was
  what was available. *Not disclosed.* **One of three — and not a standing
  compromise.**

  It exists because there is nowhere else to publish from: a module path must match
  its repository URL, there is no repository, and that is blocked on the name
  (§31, Phase 13). **GA is defined nowhere in this project**, so the tag needs an
  operative trigger rather than a milestone: this is retired when a public release
  host exists, which is the same event that unblocks Phase 13. `publish.sh` already
  anticipates it — "when this moves to a public release host, this script is what
  gets replaced and `release.sh` is what does not."

The asymmetry is worth naming. The relay is **forced** — NAT leaves no alternative.
The release host is **temporary** — it is what pre-GA looks like, not a tradeoff
anybody would defend, and a public host serves the person better on every axis
including this one. Under this rule those are different kinds of thing: one is
defensible as it stands, the other is defensible only as long as the tag is true.

**The disclosure test cannot currently be satisfied by anything.** There is no
user-facing documentation: `plugin/README.md`, the browser view and terminal output
are the only surfaces reaching a person, none mentions what is transmitted, and the
files in `plugin/commands/` are prompts rather than documentation (D-086). So this
rule presently creates an obligation nothing can meet, which is a fact about the
project rather than a defect in the rule. `docs/what-leaves-findings.md` is the
evidence any disclosure would be written from.

**What this does not do.** It does not forbid a version check, which is the
question that produced it. A check plausibly benefits the person, since running a
stale build with a known defect is a harm — so the rule conditions it rather than
refusing it: decide it in the open, disclose it, and be able to say what the person
gets. It is a test, not a veto.

**Rejected.**

- *Treat centralization as a prohibition.* §32 already reads that way and is
  already wrong about the relay, which is both central-ish and necessary. A
  prohibition that reality overrules teaches people to ignore the document.
- *Rely on §32's list.* A list of things not to build cannot govern a thing that
  must be built, and it explicitly expires after the core experiment.
- *Require disclosure only where a person could notice the compromise.* The
  model-provider case is the one nobody would notice and the one most worth being
  told.

**Revisit when** a fourth centralized component is proposed, or when any of the
three above passes all three tests — at which point the question worth asking is
whether the rule was applied or merely satisfied on paper.

---

## D-116 — Verification dials the peer it is verifying, and nobody else

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** `handleVerifyStart` passed `d.syncTargets()` to `RunVerification`, so
verifying one colleague opened a connection to every address this machine knew.

**Most of those addresses could not answer.** `clientConfig(peerID)` pins the dial
to the key being verified, and since D-103 (one address store) put every address in
`known_peers`, each other entry belongs to a different key and is refused at the
handshake. Room peers are a join against the same table, so they add nothing. The
sweep spent a dial and a timeout per peer on candidates excluded by construction.

**The leak is what makes it worth a decision.** Pairing is two people on a call
comparing two words, and this made it visible to everybody else on the list, every
time, as a connection from a recognisable address at a recognisable moment. §4
already ensures no identity goes with it — the connection is abandoned before this
machine presents anything of its own — but timing and an address are still more
than Bob needs to know about Alice.

**It also cost §4 its alarm.** §4 holds that an address answering with an
unexpected key during synchronization is routine, while the same event while
verifying a named peer "is what substitution looks like from the inside, and it is
worth saying so to the person rather than retrying quietly." That could not be said
while a wrong key was the *ordinary* outcome of most dials in a verification.
Against one address recorded for one peer it is the exception §4 describes, and the
machinery already exists: `pinnedVerifier` reports "expected X and reached Y: the
address answers for a different key", `RunVerification` returns it, and
`handleVerifyStart` puts it in front of the person. Nothing needed writing — the
sweep was what made the sentence false.

**Decision.** `verifyTargets(peerID)` returns the address recorded for that peer,
plus `COGMER_PEERS`, and nothing else.

`COGMER_PEERS` stays because those addresses **name no peer**: they are
configured for this machine, so one of them may be the peer wanted and only dialling
can tell. Every other source attributes an address to somebody, and somebody else's
address is never a place to look for this one.

**Why there is no fallback sweep.** A stale address was the reason to keep one, and
it is not a reason. Only one side needs a usable address, so a peer that moved
reaches this one by dialling inward while this side waits on the inbound check.
Where both sides are stale a sweep does not help either, because the addresses it
would try belong to other peers and are refused. A fallback that cannot succeed is
a fallback in name.

**Test.** `TestVerifyDialsOnlyTheNamedPeer` asserts the **absence** of the other
peer's address rather than the presence of the right one — a test checking only for
the right address would pass unchanged if the sweep returned. Confirmed to fail
when the sweep is restored.

**Revisit when** an address source appears that attributes an address to nobody, as
`COGMER_PEERS` does, or when a peer's recorded address can be refreshed by
something other than pairing — which would make the recorded address likelier still
and the case for anything else weaker.

---

## D-117 — The product is named `cogmer`, and nothing is carried forward

**Date:** 2026-09-22 · **Status:** active (implemented)

**Context.** The name is settled: the product, the plugin and the repository are
`cogmer`. D-069 did the expensive half of this a name ago, by breaking the couplings
that would have been permanent once real events existed. What was left was the half
that is merely annoying.

**Decision.** The name reaches the module path, the binary, the command directory,
the plugin manifest, the state directory, the local API header, the environment
variable prefix and the release path. Nowhere else.

**There is no compatibility surface, because there is nothing to be compatible
with.** The only installation that ever existed was on the author's machine, and it
was deleted rather than migrated. That single fact removes work that would
otherwise have been mandatory, and it is worth stating plainly because every
instinct here says otherwise:

- The superseded event signing scheme is **deleted**, not retained. `signingBytes`
  now knows one scheme and refuses every other version, including the zero value
  that used to mean "stored before the column existed". A test pins that refusal,
  because silently guessing a scheme verifies bytes whose provenance nobody
  checked.
- There is **no state directory migration**. A directory under the former name is
  not looked for, moved, or mentioned.

**This does not weaken D-058.** Schemes are still added and never edited, and the
`signingBytes` switch that makes that possible is intact and still tested. What
changed is that the set of schemes worth keeping turned out to be empty. Once
anybody else holds a room, it will not be empty again, and deleting a scheme stops
being available.

**The organisation casing is fixed at the same time.** D-069 recorded that the
inherited module path disagreed with the organisation's actual name, which is why
`go install` failed in D-066's test. A path that had never resolved was free to
leave alone; one about to name a real repository is not.

**Nothing cryptographic moved, which was the point of D-069.**
`protocolNamespace` is still `peer-room` and still arbitrary on purpose.

**The former name is struck from this log rather than preserved.** Where it was
incidental — a path, a command, an environment variable, a module path — the
current name is substituted as though it had always been there. Where the name
itself was the subject, as in D-069's account of namespaces carrying a product
name, the prose now says "the product name" and names nothing. A dead name in a
decision is a thing a reader must hold and resolve, and it buys no understanding:
the reasoning was never about which name it was.

**The command prefixes did not change**, for reasons that are now different from
the ones that put them there. See D-118.

**What this unblocks.** The repository, and through it Phase 13 — which is now
blocked on the repository existing rather than on the name. Also D-096's overview
command, though not in the form D-096 planned for it; D-118 says why.

**Revisit when** somebody other than the author holds a room. At that point the
latitude this entry used — deleting a scheme, deleting state — is gone, and D-058's
rule applies with nothing to soften it.

---

## D-118 — The plugin manifest name is the command namespace, and Claude Code forces it

**Date:** 2026-09-22 · **Status:** active (implemented)

**Context.** D-070 chose `/room-*` over `/team-*` on an empirical finding: a
subdirectory changes how a command is **displayed** and not what a person types,
there was no `plugin:command` invocation form, and so two plugins defining the same
name collided at the only thing a user touches. It named its own revisit trigger —
"if Claude Code gains a `plugin:command` invocation form". That has happened, and
in the stronger form: the prefix is not offered, it is imposed.

**The manifest's `name` is documented as the namespace.** "Unique identifier and
skill namespace. Skills are prefixed with this." And: "Plugin skills are always
namespaced (like `/my-first-plugin:hello`) to prevent conflicts when multiple
plugins have skills with the same name. To change the namespace prefix, update the
`name` field in `plugin.json`."

**Confirmed rather than read.** A throwaway plugin named `nstest`, with one command
in `commands/` and one in `skills/`, surfaced as `nstest:legacyform` and
`nstest:skillform`. Both layouts, both namespaced, each exactly once.

**Decision.** The prefixes stay. `/cogmer:room-create`, not `/cogmer:create`.

D-070's protection becomes belt and braces, which is precisely what D-070 said
would happen. What keeps the prefixes is D-096: a prefix names its target, the set
will grow, and `room-status` and `self-status` are two different questions that
would collapse into one word without it. Losing that distinction to save six
characters in a string the host already made long is a bad trade.

**The cost is stutter, and it is accepted.** `/cogmer:room-create` says the domain
twice. The alternative said it once and made `/cogmer:status` ambiguous between a
room and a person.

**The manifest name is now load-bearing in a way it was not.** It is what a person
types before every colon, so changing `plugin.json`'s `name` renames every command
at once. It is not a cosmetic field.

**D-096's overview command cannot be what D-096 reserved.** It wanted
`/<product-name>` as an entry point saying what the tool is, held back because the
name was unsettled. There is no bare `/cogmer`; it would be `/cogmer:cogmer`. The
compensation is that D-096's stated blocker is gone — Claude Code owns `/help`, but
it does not own `/cogmer:help`, because the namespace is exactly what stops the
collision. Not built here.

**Test.** `TestEveryCommandReferenceCarriesTheManifestNamespace` reads the
namespace out of `plugin.json` rather than duplicating it as a Go constant, and
walks the Go, HTML, Markdown and shell sources for two defects: a command named
with no namespace, and one named with a namespace that is not the manifest's. The
second is the one worth having — renaming the plugin is a single-word edit that
would otherwise break every instruction string the binary prints, with
nothing failing until a person typed one. Both branches confirmed to fail when the
defect is introduced.

**Not in the behaviour registry, and the reason is a limitation.** Registry checks
run against hook payloads inside a live session; this is a plugin load-time
property, which none of them can observe. The half of this finding that does belong
there — that `commands/*.md` is loaded identically to a skill, and so every command
is **model-invocable** — is in `open.md`, because which commands should be reachable
by the model is a decision and not a sweep.

**Revisit when** Claude Code offers an unprefixed alias deliberately rather than as
the behaviour reported in `anthropics/claude-code` issue 15882, or when `commands/`
stops being loaded at all — the documentation already calls it legacy and prefers
`skills/<name>/SKILL.md`.
