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

---

## D-024 — Admission is a guest list; the join code is a fallback for strangers

**Date:** 2026-09-16 · **Status:** active · **Corrects the emphasis of** D-018, D-020

**Context.** The intent behind a guest registry was `claude-team join misty-harbor`
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
archived with it. A developer may know six colleagues and admit two to a room
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

**Surfacing matters as much as detecting.** `claude-team conflicts` exists because a
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

**The database is not where the value is.** A developer's own turns, and the
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

**Decision.** Two listeners. `CLAUDE_TEAM_ADDR` carries hooks and the UI and
**refuses to bind anything but loopback**. `CLAUDE_TEAM_PEER_ADDR` carries
synchronization, defaults to loopback, and warns when bound elsewhere.

Separate listeners rather than one mux with a path filter, because the property that
matters is *which interface can reach an endpoint*, and a filter is a rule someone
can edit later without seeing what it guarded. Tests assert that neither serves the
other's routes.

**Why the hook API is the strict one.** Reaching `/hook/prompt` is equivalent to
being the local developer: it publishes into the room and returns the room's
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
developer saw nothing but Claude's reply.

Claude Code owns its display. No arrangement of hooks produces an ambient view
inside a session, and the specification now says so rather than leaving someone to
rediscover it.

**What the test surfaced that was worth more than the answer.** A developer can
simply *ask*: "what is the team discussing?" gets a full answer — who said what, and
that they are unverified — from context already injected. It needs no code, no view,
and no protocol. For a pair on one problem it answers most of what a view would.

It also demonstrated D-021 reaching the person it was for. The `unverified` marker,
added so a model would not treat a display name as fact, was relayed to the
developer unprompted in the model's own words. An attribution caveat travelling from
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
proposed: `claude-team` would launch the ordinary interactive `claude` inside a PTY,
proxy it, and draw peer turns in a reserved band the child cannot see.

The technique works. A passthrough prototype was byte-for-byte identical to running
`claude` directly — ANSI sequences, terminal dimensions, and exit code — under a
real 24×80 pseudo-terminal. Feasibility was never the problem.

**Decision.** Do not wrap. A view runs *beside* a session as a separate program,
not *around* it.

**Three reasons, compounding.**

*It replaces the entry point.* Everything else this project asks of a developer is
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
the developer's subscription, context window, attention, and repository, none of
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
developer addresses another's Claude directly.

The principle is now scoped to interactive sessions: a session a developer is
working in takes a turn when that developer asks it to, and at no other time. An
interactive session is a working state rather than merely a process, and a turn
arriving unbidden consumes the context window being relied on, may act on the
working tree mid-thought, and destroys the developer's ability to reason about what
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

**Decision.** State it as a principle instead. A developer starts and uses Claude
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
separate program a developer may run, not a change to how they start Claude Code.
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

**The argument.** A session is a developer's conversation with their own Claude,
read closely. A room is a record of what colleagues are doing, glanced at.
Interleaving them buries the glanceable thing inside the closely-read thing, and
interrupts the closely-read thing with arrivals not addressed to it. A developer
loses the thread of their own work in order to be told something they could have
looked at when they chose.

They also scale differently. One colleague interleaved might be tolerable; three is
unreadable — and the cost lands on the developer's own working view, which is the
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

**What using it will settle.** Solo use is sufficient to start: a developer watching
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
`claude plugin install claude-team`. The plugin surface is real and includes
`install`, `uninstall`, `update`, `validate`, `init`, and `marketplace`.

**And the session-start hook starts the daemon.** This is the part that converts the
install from "run these commands and edit this file" to one line — which was the
original objection to the wrapper, now answered without any of the wrapper's costs.
A developer should not have to start the daemon, notice it has stopped, or know it
exists.

**Three requirements, each easy to get wrong.**

*Starting must not delay the session.* Waiting on a daemon nobody asked for is worse
than having no daemon. The same fault was already made once, where the behaviour
preflight ran before the listeners and left a new room unreachable for several
seconds.

*Already running is the ordinary case, not an error.* Several sessions begin at once
on one machine routinely; each attempts the start, at most one succeeds, none
reports anything. A failure to bind is the expected outcome.

*Failure is silent to the developer and recorded by the daemon.* No daemon means no
collaboration, which is degraded rather than broken — the same fail-open rule the
hooks already follow.

**A background process must remain findable.** The daemon outlives the session that
started it, since a room may have members in several sessions and restarting it
repeatedly is worse than leaving it up. That makes it something a developer did not
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

**What this makes possible that was not before.** A developer can be in several
rooms across different sessions, which §12a's constraint describes and a one-room
daemon could not express. A room outlives the session that created it. And nothing
is created by starting a process — the bridge is gone, not because it was removed but
because the situation it papered over no longer arises.

---

## D-047 — The fingerprint is the only manual link, and had the least careful encoding

**Date:** 2026-09-17 · **Status:** open — encoding to be changed

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

**Decision.** Render the fingerprint as words. Digits would also work and are what
Signal uses, but curated wordlists already exist here, and this is the one place
where slow to read beats quick to mishear. The identifier itself stays base64url:
that is for machines and for pasting, where the alphabet is fine.

**The general point worth keeping.** The weakest link in this system is a human
reading a string, and it was given the least design attention of anything in it.
Where a design has exactly one manual step, that step deserves the most care rather
than the least.
