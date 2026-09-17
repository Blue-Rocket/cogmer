# Decision log

Decisions made while building this, with the alternatives that were rejected and
why. The rejected options matter more than the chosen ones — without them, a
later reader re-proposes something already ruled out, or "simplifies" code whose
awkwardness was load-bearing.

**Revisit when** ties each decision to the automated check that would invalidate
it, where one exists (`claude-team doctor`, registry in `cmd/claude-team/behaviors.go`).
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
(`claude-team hook prompt`), avoiding Windows backslash-and-space paths inside
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
`~/.claude-team/verified.json`. Forming a tenth room on a verified version is
free. `CLAUDE_TEAM_PREFLIGHT=off` opts out.

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

**Decision.** `cmd/claude-team/behaviors.go` holds behaviors and checks together.
`docs/relied-on-behaviors.md` is generated (`claude-team behaviors --markdown`).

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

## D-012 — The preflight probe uses the `claude-team` binary as its own hook

**Date:** 2026-09-16 · **Status:** active

**Decision.** `claude-team probe-hook <name> <dir>`, registered as the hook
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

**Decision.** Run the probe with `CLAUDE_TEAM_ADDR` pointed at a closed port, so
any ambient `claude-team` hook fails open (D-011) and records nothing.

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
separate them, and neither can the developer reading the result.

**Leaving** is always explicit. **Rejoining** the same room is allowed while it
remains live, and the per-event delivery set from D-014 makes it correct for free
— a returning session receives what it missed and nothing else.

**Moving to a different room is refused once teammate context has been injected.**
This is the sharp edge. Injected context cannot be withdrawn: another developer's
conversation is in that session's context window for the rest of its life, and
anything the session subsequently produces may be shaped by it. Admitting the
session to a second room would publish the first room's conversation into the
second through the model's own output — invisibly, irreversibly, and without
either room's members knowing. The system cannot detect that leak once it has
happened; it can only decline to create the conditions.

A session that has received *no* injected context may move freely, which covers
joining the wrong room and correcting it. A session's own prompts and responses
impose no restriction: that content originated with the developer, so carrying it
forward is their own disclosure, not a leak of someone else's.

**Presence is not membership.** The first draft of this said membership ends when
the Claude Code session ends — which is wrong, because sessions do not end. The
process exits, but the session persists and resumes under the same ID (verified in
Phase 0a, checked by B14). Under that draft, two developers closing their
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
not a value judgement about developers.

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
developer picks will be the name of a project, a client, or a ticket. Rooms named
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
- *Developer-chosen names* — reintroduces project scoping by convention.
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

**Date:** 2026-09-16 · **Status:** active

**Context.** The specification was written for an internal experiment, where
assuming Tailscale was free. For a public release it is not: "install this" and
"install this, create a Tailscale account, put every developer in a configured
tailnet" attract very different numbers of people, and Tailscale's free tier is
framed for personal rather than commercial use, so an evaluating team may read a
free tool as requiring a paid service.

**Decision.** State as a requirement that no network provider is part of room
identity, membership, or replication, and that the system must function with no
VPN at all when peers can already reach one another. Two developers on the same
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

**Where this conflicted with earlier decisions, and how it was resolved.** The
appealing version of zero-configuration joining is `claude-team join misty-canyon`
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
claude-team join misty-canyon#k7qm-2xpr-9vlt
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

**Decision.** The `roomId` and `roomName` are generated at the moment a developer
invites someone. Creating a room and issuing the first invitation are the same
act. The creating session becomes the first member and the room's history starts
there.

Before that, a session is an ordinary Claude Code session: nothing captured,
nothing injected, nothing shared. Collaboration is something a developer starts,
not a state they are in.

**The alternative is a silent disclosure.** The natural implementation — create a
room when a session starts, so capture is always on — means a developer who works
alone for two hours and then invites a colleague hands over all two hours. The
invitation looks like saying hello and behaves like publishing a transcript.
Nothing in the interface would suggest otherwise, and the disclosure is
irreversible.

So the specification now states that *"from its beginning" means the beginning of
the room, never of a session that belongs to it* — the ambiguity that made the
wrong reading available.

**Acknowledged cost.** A developer usually wants to invite someone *because* of
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

An identity belongs to a **machine**, not a person: a developer with a laptop and a
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
