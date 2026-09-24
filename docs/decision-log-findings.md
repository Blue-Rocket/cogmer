# Entries in the decision log that hold more than one decision

**Run:** 2026-09-23, on `docs/decisions.md` at commit `f11e871`, entries D-001 to
D-126. Four readers each read about 31 entries in full and applied W-40 in
`docs/writing.md` (an entry records one decision). Line numbers below are those of
`f11e871`; commit `738a6ff` added a line at five places, so later line numbers have
moved by up to four.

**Result:** 59 of the 126 entries hold more than one decision, 3 should be
tombstones, 53 hold one decision with history or findings to move out, and 11 hold
one decision and nothing else.

## What was run

1. `docs/decisions.md` was divided into four ranges: D-001 to D-031, D-032 to
   D-062, D-063 to D-093, and D-094 to D-126.
2. Each range was read in full. For every entry, the reader recorded each decision
   it holds and where it sits, the alternative each decision after the first has
   of its own, which decision keeps the number, and each part that is not a
   decision: history, a finding, or a reversal.
3. For every entry, citations of its number were found with
   `grep -rn "D-NNN\b" --include=*.md --include=*.go --include=*.sh --include=*.json`
   over the repository, leaving out the entry's own lines, and each citation's few
   words were compared with the decision that keeps the number.

A second decision counts only when it has alternatives of its own and could be
reversed without reversing the first. Reasoning, evidence, limits, consequences and
rejected alternatives do not count.

Quotations of the log, including the entry titles in the headings below, are copied
from it with four changes. Its bold is removed. Its em-dashes appear as " - ". A word
`docs/writing.md` bans is left out and marked "…". A word the guide permits only
with a marker carries one saying it quotes the log. Headings name each entry as
"D-NNN: title".

## What we found

### The verdict for each entry

| entry | verdict | decisions | citations to repoint |
|---|---|---|---|
| D-001 | one, with non-decision parts to move | 1 | 0 |
| D-002 | split | 2 | 0 |
| D-003 | one, with non-decision parts to move | 1 | 0 |
| D-004 | one, with non-decision parts to move | 1 | 0 |
| D-005 | one | 1 | 0 |
| D-006 | one, with non-decision parts to move | 1 | 0 |
| D-007 | one | 1 | 0 |
| D-008 | split | 2 | 0 |
| D-009 | one | 1 | 0 |
| D-010 | split | 2 | 0 |
| D-011 | one, with non-decision parts to move | 1 | 0 |
| D-012 | one | 1 | 0 |
| D-013 | one | 1 | 0 |
| D-014 | split | 2 | 1 (+1 to cite both) |
| D-015 | split | 2 | 1 |
| D-016 | split | 2 | 0 |
| D-017 | split | 4 | 7 |
| D-018 | split | 2 | 3 |
| D-019 | split | 3 | 9 |
| D-020 | split | 2 | 2 |
| D-021 | split | 4 | 5 (2 match no part) |
| D-022 | one, with non-decision parts to move | 1 | 0 |
| D-023 | split | 4 | 0 |
| D-024 | one, with non-decision parts to move | 1 | 2 |
| D-025 | split | 3 | 3 |
| D-026 | one, with non-decision parts to move | 1 | 0 |
| D-027 | one | 1 | 0 |
| D-028 | tombstone | 1 (withdrawn) | 0 (6 go with D-029's cleanup) |
| D-029 | one, with non-decision parts to move | 1 | 0 |
| D-030 | split | 2 | 0 |
| D-031 | one, with non-decision parts to move | 1 | 0 |
| D-032 | split | 2 | 3 |
| D-033 | split | 2 | 0 |
| D-034 | one, with non-decision parts to move | 1 | 0 |
| D-035 | split | 2 | 0 |
| D-036 | split | 2 | 0 |
| D-037 | split | 2 | 0 |
| D-038 | one, with non-decision parts to move | 1 | 0 |
| D-039 | one, with non-decision parts to move | 1 | 1 |
| D-040 | split | 2 | 0 |
| D-041 | split | 3 | 3 |
| D-042 | split | 4 | 1 |
| D-043 | split | 4 | 2 |
| D-044 | one, with non-decision parts to move | 1 | 0 |
| D-045 | split | 2 | 0 |
| D-046 | split | 4 | 4 |
| D-047 | tombstone | 0 | 11 |
| D-048 | one, with non-decision parts to move | 1 | 0 |
| D-049 | one, with non-decision parts to move | 1 | 0 |
| D-050 | split | 2 | 2 |
| D-051 | one, with non-decision parts to move | 1 | 1 |
| D-052 | split | 4 | 4 |
| D-053 | split | 3 | 3 |
| D-054 | one, with non-decision parts to move | 1 | 0 |
| D-055 | one, with non-decision parts to move | 1 | 0 |
| D-056 | split | 2 | 3 |
| D-057 | split | 3 | 0 |
| D-058 | one, with non-decision parts to move | 1 | 1 |
| D-059 | split | 2 | 0 |
| D-060 | split | 2 | 0 |
| D-061 | split | 4 | 1 |
| D-062 | split | 2 | 3 |
| D-063 | one, with non-decision parts | 1 | 0 |
| D-064 | one, with non-decision parts | 1 | 0 |
| D-065 | one, with non-decision parts | 1 | 0 |
| D-066 | split | 2 | 2 (to the moved finding) |
| D-067 | split | 3 | 0 |
| D-068 | one, with non-decision parts | 1 | 2 (to the moved finding) |
| D-069 | split | 2 | 3 (1 to finding, 2 to cite both) |
| D-070 | one, with non-decision parts | 1 | 1 (mis-citation at 3704) |
| D-071 | one, with non-decision parts | 1 | 1 (stale history comment, main.go:878) |
| D-072 | one, with non-decision parts | 1 | 0 |
| D-073 | one, with non-decision parts | 1 | 0 |
| D-074 | split | 2 | 0 |
| D-075 | split | 2 | 1 |
| D-076 | tombstone (already) | 0 | 0 (its "Replaced by" follows D-077's split) |
| D-077 | split | 2 | 5 |
| D-078 | one, with non-decision parts (partial reversal) | 1 | 0 |
| D-079 | one, with non-decision parts | 1 | 0 |
| D-080 | split | 3 (third folds into D-096) | 7 |
| D-081 | split | 2 | 3 |
| D-082 | split | 2 | 0 |
| D-083 | one, with non-decision parts | 1 | 2 (history references to its open list) |
| D-084 | split | 2 | 0 |
| D-085 | one, with non-decision parts | 1 | 0 |
| D-086 | split | 2 | 2 |
| D-087 | one, with non-decision parts | 1 | 0 |
| D-088 | split | 3 | 2 |
| D-089 | one, with non-decision parts | 1 | 2 (to the moved finding) |
| D-090 | split | 2 | 0 |
| D-091 | split | 2 | 0 |
| D-092 | one, with non-decision parts | 1 | 0 |
| D-093 | split | 2 | 3 |
| D-094 | split | 2 | 0 (6 refer to moved history) |
| D-095 | split | 3 | 5 (one, main.go:1126, is a wrong citation of D-097) |
| D-096 | split | 2 | 4 |
| D-097 | one, with non-decision parts | 1 | 0 |
| D-098 | split | 2 | 0 |
| D-099 | one, with non-decision parts | 1 | 0 |
| D-100 | split | 2 | 0 |
| D-101 | split | 2 | 0 |
| D-102 | one, with non-decision parts | 1 | 0 |
| D-103 | one, with non-decision parts | 1 | 0 |
| D-104 | split | 3 | 7 |
| D-105 | one, with non-decision parts | 1 | 0 |
| D-106 | split | 3 | 3 |
| D-107 | one, with non-decision parts (partial reversal) | 1 | 0 |
| D-108 | one, with non-decision parts | 1 | 0 |
| D-109 | one, with non-decision parts | 1 | 0 |
| D-110 | one, with non-decision parts | 1 | 0 |
| D-111 | one, with non-decision parts (partial reversal) | 1 | 0 (2 refer to the removed part) |
| D-112 | one | 1 | 0 |
| D-113 | one | 1 | 0 |
| D-114 | one, with non-decision parts | 1 | 0 |
| D-115 | one, with non-decision parts | 1 | 0 (3 refer to moved parts) |
| D-116 | one, with non-decision parts | 1 | 0 |
| D-117 | split | 3 | 1 |
| D-118 | one, with non-decision parts | 1 | 0 |
| D-119 | one, with non-decision parts | 1 | 0 |
| D-120 | one, with non-decision parts | 1 | 0 |
| D-121 | split | 2 | 2 |
| D-122 | one, with non-decision parts | 1 | 0 |
| D-123 | one, with non-decision parts | 1 | 0 |
| D-124 | one | 1 | 0 |
| D-125 | one | 1 | 0 |
| D-126 | one | 1 | 0 |

### Nine citations named the wrong entry

Reading every entry turned up nine citations whose number exists but whose words
describe another entry, and commit `738a6ff` corrected all nine. Two of them, at
`docs/decisions.md` lines 4924 and 5979 of `f11e871`, credited D-021 (peer names are
derived from the identity) with hiding the derived name on your own turns, which
D-021 does not say. That choice was made in commit `33ce996` and is recorded in no
decision, so the corrected text says what D-021 does support. The entries below
describe the citations as they stood at `f11e871`.

### D-001: Go, not TypeScript/Node or Python
- verdict: one, with non-decision parts to move
- decisions: 1. Write the system in Go with `modernc.org/sqlite` (pure Go, no cgo). "Decision. Go, with `modernc.org/sqlite`".
- keeps the number: the Go decision.
- move out: finding, "The deciding fact: `claude` resolves to `~/.local/share/claude/versions/…`, a native Mach-O binary - Claude Code no longer ships as an npm package" (also history: "no longer"); finding, "`node:sqlite` in Node 22.12 throws without `--experimental-sqlite` (verified directly)" inside Rejected.
- citations: 11 (5 are test fixtures). None point elsewhere.

### D-002: Stop at Phase 0, and insert Phase 0a before Phase 1
- verdict: split
- decisions:
  1. Put a compaction phase before any daemon work. "Decision. Stop as instructed, then add Phase 0a". Stopping is what §36.10 required, so it is not a choice made here.
  2. Label the new phase "0a" rather than renumbering the phases. The alternative is its own: "*Renumber the phases* - '0a' avoids churn in a specification already referenced by section number".
- keeps the number: 1. The title and the only rejected alternative with substance are about it. No real citation distinguishes the two.
- move out: history. The whole entry reports a step in a process that has finished ("Stop as instructed"; "Phase 0 proved capture and injection, but never exercised compaction - `PreCompact` did not fire"). D-032 (re-sequence the phases) now governs phase order. Decide whether this is still a live decision or should be a tombstone.
- citations: 1 (`citations_test.go:169`, a fixture). None to repoint.

### D-003: Reassemble turns by unioning two incomplete sources
- verdict: one, with non-decision parts to move
- decisions: 1. Union `Stop.last_assistant_message` with the transcript in `ReassembleLastTurn`.
- move out: finding, "Verified by exact string match against a real 2,582-char response"; finding, "*Transcript only* - captured a 110-char preamble of a 2,804-char answer".
- citations: 1.

### D-004: Segment turns positionally, not by identifier
- verdict: one, with non-decision parts to move
- decisions: 1. Assign to a turn every assistant record after the last user record that has a `promptSource`.
- move out: finding, in Rejected.: "observed an assistant record whose `parentUuid` matched no preceding `uuid` in the same file".
- citations: 0.

### D-005: `mergeTail` tolerates a widened `last_assistant_message`
- verdict: one
- decisions: 1. Detect the superset case, and use the field alone if it holds the whole turn.
- citations: 1.

### D-006: No watermark rewind and no re-injection floor after compaction
- verdict: one, with non-decision parts to move
- decisions: 1. Change nothing about delivery after compaction. The title's "and" joins two rejected remedies, not two decisions.
- move out: finding, "Phase 0a tested it twice, including the realistic case … It kept it anyway, with attribution. Both tests ran with `injected=0`". This belongs in `docs/phase0a-findings.md`, cited from Support.
- citations: 3.

### D-007: Record `COMPACTION` events as observability, not remediation
- verdict: one
- decisions: 1. The daemon records a `COMPACTION` event to make compaction visible in room history.
- citations: 3.

### D-008: Key behavior verification on Claude Code version, not on the room
- verdict: split
- decisions:
  1. Cache a verification result on `claude --version` in `~/.cogmer/verified.json`. "cache on `claude --version`". Its alternatives: "*Per room*", "*Time-based expiry*".
  2. Trigger the check automatically at room formation, with `COGMER_PREFLIGHT=off` to opt out. "Check at room formation". Its alternative is its own: "*Manual only* - the failures are silent; nobody runs a check for a problem they cannot see". The trigger could move (for example to daemon start) and the version key would stay.
- keeps the number: 1. It is what the title says.
- citations: 0.

### D-009: A failed behavior check never blocks the room
- verdict: one
- decisions: 1. Report which assumption changed and what it breaks, and carry on.
- citations: 0.

### D-010: The behavior registry is the source of truth; its documentation is generated
- verdict: split
- decisions:
  1. `behaviors.go` holds behaviours and checks together, and `docs/relied-on-behaviors.md` is generated from it. "Decision. `cmd/cogmer/behaviors.go` holds behaviors and checks together." The title's semicolon joins two halves of this one decision.
  2. Every behaviour needs a negative test proving it fails on the regression it claims to catch. "Corollary enforced by test: every behavior needs a negative test". Its alternative is checks with no negative test, the thing that "reads as protection while providing none". Either decision could be dropped and the other kept.
- keeps the number: 1. It is the title.
- move out: finding, "This caught a real error - the first `--deep` run reported B05/B12 failing, which was a bug in the probe's own evidence handling".
- citations: 0.

### D-011: Hooks fail open: exit 0, empty stdout
- verdict: one, with non-decision parts to move
- decisions: 1. Every hook failure path exits 0 and writes nothing to stdout, with diagnostics on stderr.
- move out: finding, "Measured 17 ms when the daemon is down; connection-refused returns immediately".
- citations: 1.

### D-012: The preflight probe uses the `cogmer` binary as its own hook
- verdict: one
- decisions: 1. Register `cogmer probe-hook <name> <dir>` as the probe's hook command.
- citations: 0.

### D-013: The probe points ambient hooks at a closed port
- verdict: one
- decisions: 1. Run the probe with `COGMER_ADDR` pointed at a closed port, so that ambient hooks fail open.
- citations: 0.

### D-014: Derive delivery state from transcript evidence, not from recorded intent
- verdict: split
- decisions:
  1. Delivery is confirmed by finding the injected block's sha256 in a `hook_success` attachment in the transcript, and delivery is recorded as a set. "Decision. Claude Code records a hook's stdout …". Recording delivery as a set follows from this ("a lost injection leaves a hole a contiguous watermark cannot represent"), so it is not a separate decision.
  2. The degraded path: commit provisionally at injection and confirm at `Stop` only while B20 (hook output recorded in the transcript) is recorded as failing for the installed version. Otherwise events stay pending and are offered again. "Consequence: absence of evidence is not evidence of breakage." and "Retained as the degraded path (see below)". Its alternative is its own: fall back to committing on trust whenever no attachment is found. It could be reversed, for example by dropping the fallback entirely, and 1 would stand.
- keeps the number: 1. Nearly every citation means "delivery is confirmed by observation".
- move out: history, "The first implementation fell back to committing on trust … Testing the failure path caught it. The fallback is now gated"; history, "Delivery became a set rather than a watermark"; history, the Status suffix "Supersedes the advance-at-injection behavior reviewed as A1 in `spec-review.md`"; history, Context "an expired or lost reply meant the daemon recorded delivery". The failure that showed the trust fallback wrong belongs in a finding.
- citations: 10. To repoint to 2: `docs/spec-review.md:560` "The delivery fallback re-created the bug it fixed (D-014)". `docs/spec-review.md:48` names both ("with a set rather than a watermark and a fallback gated on B20 failing") and should cite both. `behaviors.go:226`, and `relied-on-behaviors.md:67` generated from it, name the fallback in their Reliance text but cite the observation, so they stay.

### D-015: Rooms are scoped to sessions, not to projects
- verdict: split
- decisions:
  1. A room is a set of linked sessions with a generated id, entered by invitation, and nothing about it is derived from a directory, repository or project. "Decision. A room is a set of linked Claude Code sessions".
  2. A closed room's event log is archived: readable and searchable, never rejoined, synchronized or injected. "Split that makes it work: membership is ephemeral, the record is not." Its alternative is its own: "*Discarding the log when a room closes*". Discarding instead of archiving would leave session scoping untouched.
- keeps the number: 1. Almost every citation means "rooms are session-scoped, never derived from a directory".
- move out: history, Context "A2 found … the proposed fix was a `cwd` → project-config → room lookup. That fix was answering the wrong question"; history, "Two facts made this cheap. Our implementation was already session-centric"; history, "Both sections were amended rather than deleted" inside Cost, stated plainly.; history, the Status suffix "Supersedes the project-scoped room model … dissolves A2"; history, "The strongest argument is one the original specification did not make". Superseded in part: "closed when its last member session ends" contradicts D-016's presence-is-not-membership rule, so W-38 applies. "Settled by D-016: …" is a pointer, not a decision.
- citations: 19. To repoint to 2: `docs/open.md:46` "D-015 (rooms are session-scoped) has a closed room kept as an archive". 1 of 19.

### D-016: One room per session, and presence is not membership
- verdict: split
- decisions:
  1. A session belongs to at most one room at a time, and once it has received injected context from a colleague it cannot move to another room. Before that it may move. "Decision - one room at a time." and "Moving to a different room is refused once teammate context has been injected." One room at a time is argued as following from the event model, and both rejected alternatives about binding ("into a second room with a warning", "Binding … for its entire life") concern the move rule, so the two are one decision.
  2. Presence is not membership. Membership lasts until explicit departure or room closure, presence lapses when a process exits, and a room closes on explicit departure by all members or after long dormancy. "Presence is not membership.", "Leaving is always explicit. Rejoining …", "Consequence for closing." Its alternative is its own: "*Treating process exit as departure*". The dormancy threshold could change without touching 1.
- keeps the number: 1. CLAUDE.md, `commandinvocation_test.go`, `open.md` and decisions.md:373/1170/6429 all mean the room binding.
- move out: history, "The first draft of this said membership ends when the Claude Code session ends - which is wrong … Under that draft, two people closing their terminals for lunch would have archived the room"; history, "§35's existing presence display … already assumed this distinction; the specification simply had not stated it"; history, Status "Closes the open question in D-015" and Context. Finding: "the session persists and resumes under the same ID (verified in Phase 0a, checked by B14)" is support. Keep it with its source.
- citations: 7. None to repoint.

### D-017: A room carries two identifiers: a UUID for synchronization, a generated name for people
- verdict: split
- decisions:
  1. `roomId` (UUID) is the key for all replication, deduplication and storage. `roomName` is non-authoritative display, may be shared by unrelated rooms, and is not carried on events. "Decision. Every room has a `roomId`" and "The name is explicitly non-authoritative." Its alternatives: "*A single identifier*", "*Including the name on every event*".
  2. The name is generated from two curated weather/sky and landscape lists, never chosen by a person. "The name is generated rather than chosen, and that is …." and "Weather-plus-landscape was chosen for four properties". Its alternative is its own: "*Person-chosen names* - reintroduces project scoping by convention".
  3. A name must be unique only among one peer's live rooms, and is regenerated on collision. "Uniqueness is scoped to a peer, not global." Its alternative is its own: "*Globally unique names*".
  4. Names are immutable for the life of the room. "Names are immutable". Its alternative is renaming, rejected because "renaming would invalidate outstanding invitations".
- keeps the number: 1. The most frequent citation is "key on `roomId`, never on `roomName`; names collide by design", and the non-authoritative paragraph holds both.
- citations: 16. To repoint to 2 (generated, speakable, guessable name): `docs/decisions.md:506` "The invitation format from D-017 read `misty-canyon@…`"; `:537` "the name is drawn from a deliberately<!-- writing: quotes the decision log --> small, speakable space"; `:596` "D-017 made room names short, speakable, and therefore guessable"; `:702` "Peer identifiers should be readable, as room names are (D-017)"; `:946` "the name is guessable by construction (D-017)"; `:2421` "names are guessable by design (D-017)"; `Shared Claude Sessions.md:1168` "room names are guessable by design (D-017)". 7 of 16.

### D-018: Identity and reachability are separate; invitations carry an endpoint and a secret
- verdict: split
- decisions:
  1. `machineId` is an identity label that nothing routes by. An endpoint is reachability supplied by the transport, and a daemon discovers it rather than deriving it from a hostname. The endpoint in an invitation is a bootstrap hint, and there may be several. "Decision. Separate the two concerns", "A daemon must discover its endpoint", "The endpoint is a bootstrap hint,". Its alternatives: "*Resolving `machineId` as a hostname*", "*Requiring Tailscale MagicDNS*".
  2. A room name is not a credential: authorization is separate from the name. "A room name is not a credential." Its alternative is its own: "*Relying on the room name for authorization*". The half of it that makes authorization "a separate single-use, expiring secret issued with the invitation", and the title's "and a secret", were reversed by D-026 (no join token), so W-38 applies to that part.
- keeps the number: 1. It is the title's first clause, and spec-review.md:563, decisions.md:605 and :626 cite it.
- move out: finding, "Checked on the development machine: `os.Hostname()` returns `macbookpro.lan`, `scutil --get LocalHostName` returns `pushover` … Tailscale is not installed"; reversed part, "Authorization is a separate single-use, expiring secret … The invitation is consequently a bearer credential, and §25 now says so" (the §25 clause is also history).
- citations: 6. To repoint to 2: `docs/decisions.md:596` "D-018 made authorization a separate secret"; `:643` "is the join secret from D-018 still needed?"; `:892` "Corrects the emphasis of D-018, D-020". 3 of 6.

### D-019: No network provider is required; local discovery is the zero-configuration path
- verdict: split
- decisions:
  1. No network provider belongs in room identity, membership or replication, none is a prerequisite, and which transport connected never surfaces in room identity or event data. "Decision. State as a requirement that no network provider is part of room identity" and "Transports are attempted in order … which one connected is an implementation detail". Its alternative: "*Tailscale as the architectural foundation*". The attempt order (same network, private provider, internet, relay) is stated with no alternative weighed, so it stays inside 1.
  2. Peers on the same network join with no configuration through local discovery, which locates a room and never admits anyone to it. "Two people on the same network is the simplest case and must be the easiest" and "So discovery locates a room; it never admits anyone to one." Its alternative is its own: "*Joining by name alone on a trusted network*". It could be dropped (same-network peers pair like anyone else) and 1 would stand.
  3. Use `tsnet` only inside `TailscaleTransport`. "Note for implementation. `tsnet` … attractive *inside* `TailscaleTransport`, and unacceptable anywhere else." Its alternative is its own: use tsnet elsewhere, or shell out to the Tailscale CLI. This is a small decision. It could instead go to a Limits line of 1.
- keeps the number: 1. CLAUDE.md, `tailcat.go:38`, and decisions.md:3060/3063/3147/3173 all mean "no provider is required; a transport sits under everything".
- move out: reversal, the build order "Implementation order is Local, then Tailscale, then WebRTC" and "Local discovery moved from 'later possibility' to the first transport built", superseded by D-063 (first pair works from home). The reversal text "That priority was wrong, and D-063 corrects it." and the Status clause about build order belong in a finding (W-38). History: "The architectural seam already existed … What changed is the priority"; "Where this conflicted with earlier decisions, and how it was resolved."; "Tailscale is also not installed on the development machine, so local discovery is now the shorter path" (also a finding). Also reversed: the example `cogmer join misty-canyon#k7qm-2xpr-9vlt` and "while the secret remains" / "The secret also disambiguates" describe the join secret D-026 (no join token) removed. Context "The specification was written for an internal experiment" is history.
- citations: 20. To repoint to 2 (local discovery): `docs/open.md:51` "Local network discovery (D-019's zero-configuration path) is not built"; `docs/decisions.md:682` "makes broadcast discovery (D-019) safe to enumerate"; `:4584` "which D-019 names as the zero-configuration path"; `:4703` "discovered on the network we are on now (D-019)"; `:4747` "local discovery lands (D-019's zero-configuration path)"; `:4763` "D-019's zero-configuration path is not built". To repoint to the reversal finding or D-063: `:3141` "D-019 made local discovery the first transport to build"; `Shared Claude Sessions.md:2758` "a reversal of D-019's priority"; `Shared Claude Sessions.md:2853` "Local discovery is last, and that reverses D-019". 9 of 20.

### D-020: A guest list replaces the join secret only once peer identity is cryptographic
- verdict: split
- decisions:
  1. Peer identity must become cryptographic: the identifier derived from a public key, possession proved on connection, and events signed at origin so that relay is verifiable. Until then the guest list, the relay rule and attribution are conventions. "State in the specification that peer identity must become cryptographic" and "The stronger argument for doing the work is unrelated to admission." Its alternative: "*Deferring identity until after Phase 2*".
  2. Once identity is cryptographic, admission prefers a guest list of known keys over a secret. "Once identity is cryptographic, the guest list is the better mechanism" and "They compose rather than compete." Its alternatives are its own: "*Guest list instead of the secret, now*", "*Guest list keyed on `userId` or `machineId`*". This is the decision D-024 (admission is a guest list) records, so it could merge into D-024 rather than take a new number.
- keeps the number: 1. Its citations mean "cryptographic identity, Ed25519, event signing" (`peername.go:57`, `tailcat.go:46`, `spec-review.md:382`, decisions.md:709/760/885/3076, `Shared Claude Sessions.md:2778`).
- move out: reversal, "Decision. Keep the secret for now" and the composing model's "a single-use secret admits a peer that is not yet known", both reversed by D-026 (no join token); W-38 applies. "Finding: not with identity as it stands." is the reasoning for 1 despite its label, but its bold opener is not a template field. History: "This closes review item C7."
- citations: 11. To repoint to 2 (or to D-024): `docs/decisions.md:723` "It comes from D-020's known peers"; `:892` "Corrects the emphasis of D-018, D-020". 2 of 11.

### D-021: Peer names are word pairs derived from the identity, never chosen
- verdict: split
- decisions:
  1. A peer's `peerName` is an adjective and an animal chosen by a hash of `peerId`, derived on load, never stored or chosen, from word lists fit to name a colleague. "Decision. A peer carries two identifiers", "The name is derived, never chosen", "Word lists name colleagues". Its alternatives: "*Self-chosen display names*", "*Storing the name alongside the identity*", "*Indexing the word lists by the first and last bytes*".
  2. A name is shown as itself only for a verified peer, and an unverified speaker is marked inside the injected text, not only in an interface. "The specification now requires that a name be displayed as itself only for a verified peer". Its alternative is its own: marking only in a UI, which "protects the wrong reader". A derived name could stay while the marking rule changed.
  3. A fingerprint comparison renders the whole key, as a word sequence separate from the peer name. "The instinct is sound at a different scale … §25 now requires that such a comparison render the *whole* key." Its alternative is its own: offering the two-word name as a fingerprint check. It sits inside a Rejected. bullet.
  4. Keep the word lists at 8,280 combinations rather than expanding both to 256. "On collisions. … Held off because curating 512 words". Its alternative is its own: expanding to 65,536.
- keeps the number: 1. Most citations mean "attribution anchors on the derived name, never on a chosen one".
- move out: support that belongs in a finding: the slicing measurement ("one distinct value across 2,000 generated ids", "roughly 50% more likely") and the collision arithmetic ("0.5% … 3.6% … 45%"); history, "Implemented as `PeerName()`", "The specification now requires", "§25 now requires".
- citations: 11. To repoint to 2: `cmd/cogmer/ui.go:127` "§6/D-021 require the identifier be shown for any peer whose identity is unverified"; `cmd/cogmer/ui_test.go:275` "Your own turns need no marker and no anchor (D-021)"; `docs/decisions.md:1392` "demonstrated D-021 reaching the person it was for. The `unverified` marker". Citations that match no part of D-021: `docs/decisions.md:4924` "D-021 keeps the derived name off your own turns" and `:5979` "D-021 suppresses the derived name on your own turns". D-021 says nothing about a person's own turns, so find where that is decided. 5 of 11.

### D-022: A room begins when someone is invited into it, and never earlier
- verdict: one, with non-decision parts to move
- decisions: 1. Creating a room and issuing its first invitation are one act, and nothing is captured before it.
- move out: history, "So the specification now states that *'from its beginning' means the beginning of the room*"; Context "The specification said a room is 'created by one peer' … in three places".
- citations: 5.

### D-023: A peer identifier must be safe to know; identities are created once and exchanged on joining
- verdict: split
- decisions:
  1. The peer identifier is the public key or its fingerprint, and it becomes safe to know only once signatures are checked. So no keypairs are generated before verification. "The identifier must be safe to know.", "The order matters, and is easy to get wrong." Its alternatives: "*Treating the current identifier as sensitive*", "*Generating keypairs now as a first step*".
  2. An identity is created once, on a machine's first use, and persists beyond every room, session and invitation. It belongs to a machine, not a person. "When. An identity is created once" and "An identity belongs to a machine, not a person". Its alternative is its own: an identity per room, per session or per invitation, or one per person carried across machines ("Deliberate - a key that never leaves the machine that made it"). Either could change with 1 intact.
  3. Identities are exchanged on joining, with no directory. "How a guest's identifier is obtained: it is not, in advance." Its alternative is its own: "*A directory of peer identifiers*".
  4. A key accepted at first contact is verified once over a channel the invitation did not travel on. "Self-certifying is not self-authenticating. … Verifying once, over a channel the invitation did not travel on, closes that". No alternative is weighed here. It is the decision D-054 (verification gates sync) and D-055 now hold, so it may be support pointing there rather than a new entry.
- keeps the number: 1. Every citation means "safe to know / not verified yet / the ordering warning" (`keys.go:16`, `sync.go:20`, decisions.md:937/1247/1256/1345/1503/1753/1846/1857).
- move out: history, "which is why nothing was implemented here"; Context "Answering the second surfaced that the current identifier is hazardous to share"; "it is a random value that nothing verifies" describes identity as it stood.
- citations: 12. None to repoint.

### D-024: Admission is a guest list; the join code is a fallback for strangers
- verdict: one, with non-decision parts to move
- decisions: 1. Admission is a host's guest list of known public identifiers, proved by possession of the key. The name locates and the list admits. The fallback code in the title was reversed by D-026 (no join token), so it is not a live second decision.
- move out: reversal, "The code survives as a fallback, explicitly weaker." and the title's second clause, reversed by D-026; W-38 rewrites the entry to drop them and records why in a finding. History: "The error was conflating two claims. … I treated it as though it did"; Status "Corrects the emphasis of D-018, D-020"; Context "The specification conceded in one sentence". "Unchanged by this. The ordering from D-023 still governs" is history ("still"), and should become a citation of D-023 in Limits.
- citations: 10. To repoint: `docs/decisions.md:2531` "known peers … and a room's guests … as two lists since D-024" goes to D-025 (two scopes). `:1015` "Supersedes the residual code path in D-024" refers to the reversed part. 2 of 10.

### D-025: The guest list, specified: two scopes, a signed challenge, and approval in the moment
- verdict: split
- decisions:
  1. Two lists at different scopes: known peers per machine, durable; a room's guests per room, archived with it. Each has inspect and correct commands (`peers`/`allow`/`forget`, `guests`/`invite`/`revoke`). "Two lists, at different scopes." Its alternative: "*One combined list*".
  2. Admission is proved by signing a fresh, unpredictable challenge from the host. "Admission is a fresh signed challenge." Its alternative is its own: a replayable proof ("an exchange that can be replayed is a bearer credential with extra steps"). The proof format could change and the two scopes would stand.
  3. A refused peer's request is shown to a present host, who approves it by identifier. A request is never queued for an absent host. "Refusal must be more than silence, and that changes the role of codes." Its alternatives are its own: "*Silent refusal*", "*Queuing requests for an absent host*", "*Approving by displayed name*".
- keeps the number: 1. The title leads with the guest list, and no citation singles out 2 or 3 except spec-review.md:571. This is close; see the summary.
- move out: reversal, "Codes now cover one case only: … *(Superseded by D-026 …)*" carries reversed text inside a live entry; W-38 applies. Status "Completes D-024" and Context "Naming an analogy is not specifying a design" are history.
- citations: 6. To repoint to 3: `docs/spec-review.md:571` "a host present can approve a stranger's request". To repoint to the removed code-case text: `docs/decisions.md:1015` "the residual code path in D-024 and D-025" and `:1017` "D-025 had narrowed join codes to a single case". 3 of 6.

### D-026: There is no join token at all
- verdict: one, with non-decision parts to move
- decisions: 1. There is no join token or invitation secret. Admission is a guest-list entry proved by a key, or a present host's approval.
- move out: history, the Status "Supersedes the residual code path in D-024 and D-025" and Context "D-025 had narrowed join codes"; "Consequence. The system now has no credential … not as a deprecated path, but absent. §25 says so directly".
- citations: 11.

### D-027: A sequence conflict is quarantined, not dropped
- verdict: one
- decisions: 1. Classify each receipt as stored, duplicate or conflict. Keep a conflicting event quarantined with both identifiers, and surface it through `cogmer conflicts`. The bold openers ("Quarantine rather than reject.", "Why not repair it automatically.", "Surfacing matters as much as detecting.") are its reasoning and rejected alternatives.
- citations: 8.

### D-028: Losing a room database ends that peer's membership; recovery is not attempted
- verdict: tombstone
- decisions: 1. (Withdrawn) A peer that loses a room database leaves the room.
- keeps the number: the tombstone keeps the number and title. Status becomes "withdrawn 2026-09-16. Replaced by D-029 (losing a room database)".
- move out: reversal. The whole body goes to a finding: what it decided, the sequence-epoch analysis, and "An inversion worth knowing" (losing identity is the safe failure), which D-029 depends on.
- citations: 7. `docs/spec-review.md:227` "D-028 says its membership ends" is correct against a tombstone. The six in D-029 (lines 1158, 1160, 1168, 1195, 1199, 1210) are history in D-029 and go with its cleanup (W-37: the new decision cites neither).

### D-029: Losing a room database does not end membership; the sequence lives with the identity
- verdict: one, with non-decision parts to move
- decisions: 1. Membership survives loss of a room database. The peer's own sequence position is stored outside the room database, with the identity, and reserved before any event using it is published. "Reserve before publishing." is the condition that makes the stored sequence safe, and reversing it breaks the decision, so it is part of this one and not a second decision. The title's semicolon joins the decision to its mechanism.
- move out: history of D-028 (it goes to the D-028 finding): Status "Supersedes D-028"; Context "D-028 treated a lost room database …"; "The database is not where the value is. … D-028 traded something consequential"; "And it took more than it appeared to. … A disk hiccup cost the afternoon. Neither decision was wrong alone"; "Why this beats the sequence epoch D-028 rejected." (the epoch comparison can remain as a Rejected. bullet, which it already is); "There is no longer any loss that … was the precondition for B4"; "*Ending membership* (D-028) - see above". "Costs, stated so they are expected." is Limits. content.
- citations: 17 (including the D-028 Status line). None to repoint.

### D-030: Two listeners: hooks on loopback, peer sync separately
- verdict: split
- decisions:
  1. Hooks and the UI are served on one listener (`COGMER_ADDR`) that refuses any non-loopback bind, and synchronization on a separate listener, not one mux with a path filter. "Decision. Two listeners." and "Why the hook API is the strict one." Its alternative: "*One listener, path filtering*".
  2. The peer listener (`COGMER_PEER_ADDR`) defaults to loopback and may be bound elsewhere with a warning, rather than being refused until authentication exists. "The peer API is exposed with a warning rather than refused," Its alternatives are its own: "*Exposing the peer API by default*", "*Refusing to expose the peer API until authentication exists*". The exposure policy could change with the two listeners intact.
- keeps the number: 1. It is the title, and the one citation (`ui_test.go:17`, "serving it to peers would hand the room to anyone who can reach the sync port") means the separation.
- move out: history, Context "Making a pair work across two machines was preferred over a three-peer run" (a sequencing choice, now D-032's territory) and "One listener on `127.0.0.1` served hooks, the UI, and synchronization". Possibly stale: "unauthenticated and that nothing verifies who connects - true until identity becomes cryptographic (D-023)".
- citations: 1. None to repoint.

### D-031: Peer synchronization polls; the push worth building is the local UI's
- verdict: one, with non-decision parts to move
- decisions: 1. Peer synchronization polls by default, and push for the peer layer is not built without a reason beyond latency. "The push that does matter is a different one." names §17's live-UI requirement (SSE from the daemon). It weighs no alternative of its own, so it is a pointer and limit, not a second decision (see the summary).
- move out: history, Context "Polling was chosen for the two-peer experiment and recorded only as a code comment … nobody had decided"; the rejected "*Leaving it undecided*" is about process, not an alternative design. Support that belongs in a finding: "Peer propagation is roughly half a second" (C4's measurement).
- citations: 2.

### D-032: Re-sequence the phases, and follow them
- verdict: split
- decisions:
  1. Follow a stated execution order of phases (8 → 9 → 10 → 5 → 7), with actual status recorded against each phase. Opens "Decision. Record actual status against every phase, add the phases…".
  2. Phase numbers are never reused or reassigned, and superseded phases are marked, not rewritten. Opens "Numbers are never reused or reassigned, so references…". Its own alternative: "*Renumber the phases* - breaks every reference in this log…". You could renumber and still follow the same order.
- keeps the number: 1. The title and all three citations mean the ordering entry.
- move out:
  - history: "Context. Work had proceeded opportunistically…" (what was done out of order).
  - history: "*Amended the same day.* The first draft of this bundled room identity…".
  - current state that belongs in `open.md`: "Record actual status against every phase". The per-phase status is state, not a decision.
  - restatement of a standing decision that D-109 (two is the target and nothing rules out more) now holds: "What this does not change. Pairs remain the target. Phase 6 waits…".
- citations: 3. All three mean the "pairs remain the target" aside, which moves out, not the ordering:
  - docs/decisions.md:5763 "D-032 (the order of work) records in an aside that 'pairs remain the target'"
  - docs/decisions.md:5802 "*Leave it in D-032's aside.*"
  - docs/decisions.md:5806 "the trigger D-032 already names" (Phase 6 waits for evidence of a third peer)

  All three sit in D-109 and describe D-032's text. Once the aside leaves D-032 they are history in D-109, not live pointers.

### D-033: The room cannot be displayed inside Claude Code; asking is the free affordance
- verdict: split
- decisions:
  1. The room is not displayed inside a session. The specification names asking the model as the in-session affordance. Opens "Decision. §17 no longer prescribes a browser. It states that the room cannot be shown inside the session, names asking…".
  2. A terminal view and a browser view are different moments rather than competitors, and more than one may exist because both read only from the daemon. Opens "…and treats a terminal view and a browser view as different moments rather than competitors…". Its own alternative: "*Treating the browser as the answer*". D-039 (one renderer until it has been used) later answers this differently while decision 1 stands.
- keeps the number: 1. All 13 citations are "(D-033, D-036)": every extension point delivers to the model, and nothing displays to a person.
- move out:
  - finding: "Finding: it cannot. Tested directly. A hook's standard output becomes context…" (stdout, stderr and `/dev/tty` were each tested).
  - finding: "What the test surfaced… A person can simply *ask*…" (an observed session answering from injected context).
  - finding: "It also demonstrated D-021 reaching the person it was for…" (the model relaying the `unverified` marker unprompted).
  - history: "Context. §17 specified a browser at `localhost`, and a browser was built to it. That turned out not to match…".
- citations: 13. None needs repointing. They cite the display impossibility, which stays with decision 1, though the evidence for it moves to a finding.

### D-034: No terminal wrapper; a view sits beside the session rather than around it
- verdict: one, with non-decision parts to move
- decisions:
  1. No PTY wrapper: a view is a separate program running beside the session. Opens "Decision. Do not wrap." The semicolon joins two halves of one choice.
- move out:
  - finding: "The technique works. A passthrough prototype was byte-for-byte identical…".
  - finding: "Recorded as tested, so it is not re-derived: an invisible pseudo-terminal passthrough…" (the reserved-band technique).
  - history: "the prototype is reverted rather than parked" in Rejected., and the Context's "A pseudo-terminal wrapper was proposed…".
- citations: 4. None needs repointing.

### D-035: A remote peer never initiates local execution
- verdict: split
- decisions:
  1. §3.7: a remote event never causes a turn in an interactive session. It is read only at a turn the person started. Opens "Decision. State it as §3.7…", as narrowed by "The principle is now scoped to interactive sessions: a session a person is working in takes a turn when that person asks…".
  2. The principle is guarded structurally. A test forbids peer-handling files from importing `os/exec` or `syscall` or calling the probe, and is stricter than the principle by design. Opens "Guarded structurally rather than by review.". Its own alternatives are named in the entry: review, and checking call graphs ("Checking imports rather than call graphs is crude, and deliberately<!-- writing: quotes the decision log --> so"). A later need for a separate run could loosen the guard without touching §3.7: "the guard stands until they do".
- keeps the number: 1. There are no citations. §3.7 is cited in place of the number everywhere.
- move out:
  - history: "The implementation already conforms, and not by design so much as by not having written the code…".
  - history: "The specification did not state it, and said something weaker that was also wrong. §16 read…".
  - history: "Scope, corrected the same day. The first draft forbade a remote event starting *any* Claude run…". Keep only the scoped rule it produced.
  - open question for `open.md`: "Whether a peer event may cause a separate run is explicitly left open…". It could also stay under Limits.
- citations: 0.
- note: the title states the pre-correction scope ("never initiates local execution"). The decision as it stands covers interactive sessions only. The entry also has no Rejected. field.

### D-036: MCP logging notifications are not a display channel
- verdict: split
- decisions:
  1. MCP is not used as a display channel: it carries capability to the model and nothing to a person. Opens "Consequence. MCP carries capability *to the model*…". The alternative it answers is the Context's MCP server for ambient display.
  2. No MCP server this project ships exposes sampling, because sampling would let a peer cause inference in an interactive session. Opens "Related, and worth stating before anyone builds an MCP server here for another reason." and ends "Any MCP server this project ships must not expose one." Its own alternative is shipping an MCP server with sampling. You could reverse it without reversing decision 1. CLAUDE.md states it without a number.
- keeps the number: 1. All 12 citations mean "nothing displays to a person".
- move out:
  - finding, most of the entry: "Tested, and it does not work.…", the four places checked, "Tested twice, because the first test was wrong.…", and "The conclusive evidence is the capability record…" with the JSON line. This belongs in a findings document, and decision 1 should cite it.
  - finding: "Whether Claude Code implements sampling was not tested."
- citations: 12. None needs repointing. Sampling is uncited, so no citation moves to the new entry.

### D-037: Claude Code is launched and used unchanged
- verdict: split
- decisions:
  1. §3.8: a person starts and uses Claude Code unchanged, and the system installs only what Claude Code already loads. Opens "Decision. State it as a principle instead."
  2. Ambient awareness, if wanted, comes from an operating-system notification raised by the daemon. Opens "Preferred instead, if ambient awareness is wanted. The daemon can raise an operating-system notification…". Its own alternatives are a terminal pane beside the session (weighed in D-039, one renderer until it has been used) and an undocumented in-session seam. You could choose a pane instead and keep §3.8.
- keeps the number: 1. It has no citations; the title and §3.8 mean it.
- move out:
  - history: "Context. D-034 ruled out a pseudo-terminal wrapper… derived *after* a prototype had already been built" and "Applied earlier, it would have stopped the wrapper before anything was written."
  - finding: "On searching for an undocumented display seam.…the installed artifact is a native binary…'Channels' does not appear in this version…`claude plugin details` reports a *projected token cost*". The decision to decline the seam is a rejected alternative of decision 1 and belongs under Rejected.
  - finding, restated: "Confirmed independently three times: D-033…, D-036…, and skills being markdown instructions". CLAUDE.md already holds this under "Facts that are not obvious".
  - not a decision: "If a display primitive is ever wanted from Anthropic, the request is small…".
- citations: 0. Related: `CLAUDE.md:131` puts "prefer an OS notification from the daemon for ambient awareness" under "(D-038, D-039)", but that preference lives here as decision 2. D-039 leaves announcing open.

### D-038: Separating the room from the session is correct on its merits
- verdict: one, with non-decision parts to move
- decisions:
  1. The room is viewed outside the session because that is right on its merits, and it would still be so if an in-session display became available. Opens "Decision. Record the separation as a design position…".
- move out:
  - history: "Context.…everything since has treated an external view as what remains… That framing was backwards."
  - history: "Consequence for how the constraint is described. §17 no longer opens by saying…" (a record of a specification edit).
- citations: 6. None needs repointing. The entry has no Rejected. field, so W-36 needs one.

### D-039: One renderer until it has been used
- verdict: one, with non-decision parts to move
- decisions:
  1. The browser view is the only renderer, and no second renderer (terminal pane or notification) is built until use shows one is needed. Opens "Decision. Build neither yet.", as settled by "Outcome, 2026-09-17. Used, and judged the right avenue…".
- move out:
  - finding: "Outcome, 2026-09-17. Used, and judged the right avenue… a separate window is consulted rather than forgotten". This is an observation with a date. The decision should state what now holds (the browser is the view), and the observation goes to a finding.
  - history: "The browser view exists; nobody has worked with it." and "What using it will settle.…" were true before the outcome and are not now.
  - open question for `open.md`, where it already is: "Still open is whether arrival wants announcing…".
- citations: 4. One cites the finding that moves: `docs/what-leaves-findings.md:67` "the browser view…, which D-039 records as the one thing consulted rather than forgotten".

### D-040: The injected block is fenced with an unforgeable value, and framed by classification
- verdict: split
- decisions:
  1. The injected block's boundary is unforgeable. A per-injection fence value is stripped from the content, the block ends only at the matching value, and the framing is restated after the content. Opens "Decision - the boundary must be unforgeable.". Rejected: "*Escaping the delimiters*".
  2. The framing classifies what the model is reading instead of instructing it to disregard instructions: nothing inside is addressed to it, and a request inside is a report of a request. Opens "Decision - frame by classification rather than authority.". Its own alternative is in the paragraph: "Instructing a model to disregard instructions invites it to weigh two instructions". You could keep the fence and frame by authority, or frame by classification with a static delimiter.
- keeps the number: 1. `docs/decisions.md:4639` "(fence the block so content cannot close it)" and the fence sentence at `CLAUDE.md:175` mean it.
- move out:
  - finding: "The vulnerability. Content was interpolated raw… Demonstrated rather than theorised."
  - finding: "Verified against a live session, not only in structure. A real session given a forged `SYSTEM OVERRIDE`…".
  - history: "Context.…There was such language - a sentence at the top of the block - and it was defeatable." and "The framing now says…".
- citations: 4. None needs repointing. `CLAUDE.md:175` puts "(D-040)" after the fence clause, and its next sentence, "Frame by classification…", carries no number; it could cite the new entry.

### D-041: Ship as a Claude Code plugin; the session-start hook starts the daemon
- verdict: split
- decisions:
  1. cogmer is packaged as a Claude Code plugin carrying the hooks. Opens "Decision. Package as a plugin carrying the hooks…".
  2. The session-start hook starts the daemon. Starting it never delays the session, finding it already running is the ordinary case, and failure is silent to the person. Opens "And the session-start hook starts the daemon." and "Three requirements, each easy to get wrong.". Its own alternative is the manual daemon start in the Context ("a manual daemon start"). A plugin could ship hooks and still leave the person to start the daemon.
  3. The daemon outlives the session that started it and must be discoverable and stoppable by the person whose machine it runs on. Opens "A background process must remain findable.". Its own alternative: "restarting it repeatedly is worse than leaving it up".
- keeps the number: 1. Three citations mean packaging or the install line (2522, 6566, 6571); the other decisions have two and one.
- move out:
  - history: "Context.…In practice it was still a manual daemon start, a hand-written settings file…".
  - history: "which was the original objection to the wrapper, now answered…".
  - finding: "The plugin surface is real and includes `install`, `uninstall`, `update`, `validate`, `init`, and `marketplace`."
  - history: "The same fault was already made once, where the behaviour preflight ran before the listeners…".
  - partial supersession (W-38): the one-line install is two lines per `docs/decisions.md:6566–6571`, so this entry should be rewritten to state what holds.
  - "Deliberately<!-- writing: quotes the decision log --> not encoded: which surfaces this reaches." is a rejected alternative (a surface matrix) or a limit of decision 1, not a separate decision.
- citations: 6. Three mean decisions other than packaging:
  - docs/decisions.md:4932 "D-041 forbids delaying a session or speaking to the person" → decision 2
  - docs/decisions.md:6102 "D-041 (starting must not delay the session)" → decision 2
  - docs/decisions.md:6651 "D-041 (ship as a plugin; the session-start hook starts the daemon) required the daemon to be discoverable and stoppable" → decision 3

### D-042: Peer identity is an Ed25519 key pair; events are signed at origin
- verdict: split
- decisions:
  1. A peer's identifier is its Ed25519 public key, rendered `ed25519:<base64url>`, not a fingerprint of it. Opens "Decision. A peer's identifier *is* its Ed25519 public key…" and "Why the identifier is the key rather than a fingerprint of it.".
  2. Events are signed at origin over length-prefixed fields with a leading purpose tag, and every receiver verifies each one against the key its identifier names. Opens "Events are signed at origin over a length-prefixed encoding…" and "Signing covers length-prefixed fields with a purpose tag.". Its own alternative is in the paragraph: "Concatenating fields directly would let a boundary move". You could make the identifier a key and sign only requests, or sign events under a fingerprint-style identifier.
  3. The private key lives in its own file, never in `identity.json`. Opens "The private key lives in its own file.". Its own alternative: one identity file, "an identity that cannot be shown without checking what else is in it".
  4. An event with a failed signature is refused and logged, not quarantined. Opens "Rejection, not quarantine.". Its own alternative is quarantine as D-027 (a sequence conflict is quarantined) does for sequence conflicts.
- keeps the number: 1. Every specific citation means "a peerId is a key" or "knowing an identifier grants nothing".
- move out:
  - finding: "Verified live: a peer impersonating another and offering an event signed by nobody was rejected".
  - history: "What this does not do, stated because the startup warning used to overclaim.…Proof of possession on connection is Phase 10…the warning now says exactly<!-- writing: quotes the decision log --> this". The limit on integrity and attribution can stay in Limits.; the Phase 10 and warning narration goes.
  - history: "Migration. An `identity.json` whose identifier is not the local key is rewritten…".
  - history: "Context. Phase 9…§25 asked for signable identity and got a random string."
- citations: 12. One means something else:
  - docs/decisions.md:4884 "D-042 already calls it 'a mnemonic for an identity already verified'". D-042 contains no such words. They are in D-021 (peer names are word pairs derived from the identity) at `docs/decisions.md:724` and in the spec at `Shared Claude Sessions.md:2011`. Repoint to D-021.
  - Related: `cmd/cogmer/offer.go:47` cites D-058 for "a captured offer could be replayed as something else". That is decision 2 here (the purpose tag), so it should point at decision 2's new number.

### D-043: The wire format is defined separately from the stored row
- verdict: split
- decisions:
  1. The wire format is its own type in its own file, with explicit conversion to and from the stored row. Opens "Decision. Define the wire format in its own file, as its own type…" and "Why a separate type when the fields are currently identical.".
  2. A second host is prepared for only by naming, never by machinery. The session field is renamed `originSessionId`, and there is no `source` field, adapter architecture or second-host design. Opens "Rename the session field to `originSessionId`…" and "What was deliberately<!-- writing: quotes the decision log --> not done. No `source: claude-code | codex` field, no adapter architecture…". Its own alternative, rejected in the entry, is "cross-agent collaboration becomes nearly free once events are normalised". You could keep one struct for wire and row and still refuse adapters, or separate them and build adapters.
  3. Every sync exchange carries a protocol version, and a peer speaking a different one is refused. Opens "Carry a protocol version in every sync exchange and refuse a peer that speaks a different one." and "Why the version moved to v2.". D-065 (the daemon reads a range of wire versions) since changed the refusal without touching decision 1, which shows it is separable.
  4. Rooms are migrated on open, so a column added later reaches rooms created earlier. Opens "Rooms are now migrated on open, and adding a column to that list…" at the end of "A bug this surfaced.". The entry weighs no alternative (a schema version table or rebuilding are the obvious ones), but CLAUDE.md treats it as a standing choice: "Migrate, do not orphan".
- keeps the number: 2. Eight of ten citations mean "no adapter machinery for a second host". The title names decision 1, so D-043 would be retitled and the wire-and-row split would take a new number. See the note in the report.
- move out:
  - history: "Context. A suggestion that the daemon's database schema should not become the protocol. We were half-violating it…".
  - history: "Why now. No room existed that anyone would mind losing…".
  - finding: "A bug this surfaced. `CREATE TABLE IF NOT EXISTS` creates a table and then ignores it forever… An existing room failed with *no such column: signature*…".
- citations: 10. Two mean the wire-and-row decision rather than the one that keeps the number:
  - docs/decisions.md:3277 "D-043 (the wire format is not the database row) keeps the two structs separate precisely<!-- writing: quotes the decision log --> so a column added for local bookkeeping cannot become protocol" → decision 1
  - docs/decisions.md:5831 "D-043 (the wire format is not the database row) forbids `source`/adapter machinery". The sentence means decision 2 and its few words name decision 1, so the words need fixing even though the number stays.

  The other eight mean decision 2: CLAUDE.md:127, CLAUDE.md:238, docs/decisions.md:2857, 4367, 4369, 5814, 5848 and 5966.

### D-044: Sync requests are signed; authentication is not admission
- verdict: one, with non-decision parts to move
- decisions:
  1. Every sync request carries the caller's identifier, a timestamp and a nonce, signed with a purpose tag, and is refused outside a two-minute window or when the nonce has already been seen. Opens "Decision. Every sync request carries…". "Authentication is not admission" is this decision's limit, not a second decision. Signing requests rather than holding a session, the two-minute window and the unsigned `have` map are parameters of the same choice.
- move out:
  - finding: "two NTP-synced machines measured 408ms apart".
  - finding: "A stranger generated a key pair, authenticated correctly, and read a private room - while the host logged nothing…". The limit it shows stays under Limits.
  - history: "So what Phase 9 delivers is the ability to make an admission decision, not the decision. The guest list is Phase 10, and until it exists the startup warning…". D-045 exists now.
  - history: "Context. Phase 9's third part…".
- citations: 14. None needs repointing. The ones meaning "authenticates correctly and must still be refused" (`membership_test.go:40`, `membership.go:753`, `auth.go:132`) cite the limit, which stays.

### D-045: Rooms are records with a guest list; admission is enforced
- verdict: split
- decisions:
  1. A room is a record in `membership.db` (UUID, generated name, guest list), and a sync request is refused unless its authenticated peer is a guest. Opens "Decision. Rooms become records rather than arbitrary strings… A sync request is refused unless its authenticated peer is a guest."
  2. Only a key can be admitted: `allow` and `invite` refuse an identifier that names no key. Opens "Only keys can be admitted.". Its own alternative is recording `alice` or an old `peer-8f3a…` identifier, "recording a hope". You could enforce admission and still accept names.
- keeps the number: 1. All six citations mean "refused unless its peer is a guest".
- move out:
  - finding: "Verified by repeating the test that failed. The same uninvited stranger now reads nothing…".
  - restatement of D-025 (the guest list specified: two scopes, a signed challenge, approval in the moment) and D-024 (admission is a guest list): "Two scopes, as §12 requires. `known_peers` is machine-wide… `forget`… `revoke`…". D-053 notes the two lists date from D-024. Cite, do not restate.
  - history: "The out-of-band step is a public key, and the tooling says so. `allow` prints the full fingerprint…". D-053 made `allow` print UNVERIFIED and D-055 (one way to verify a peer) removed fingerprint comparison, so this is reversed text.
  - history: "A bridge, noted as such. A daemon pointed at a room nobody created makes one…". D-046 removed it.
  - history: "Context. Phase 10. D-044 left the confidentiality gap open…".
- citations: 6. None needs repointing. The entry has no Rejected. field, so W-36 needs one.

### D-046: The daemon serves many rooms; a session says which one it is in
- verdict: split
- decisions:
  1. The daemon is the machine's local service, not a room: it opens a store per room on demand. Opens "Decision. The daemon is the machine's local service, not a room."
  2. A session is in no room until it binds to one, and binds at first sight. A session in no room is an ordinary Claude Code session. Opens "a session binds to a room on first sight. A session in no room is an ordinary Claude Code session…" and "Binding at first sight rather than asking". Its own alternative is asking, which is rejected ("there is nobody to ask at that moment"). You could have a multi-room daemon that assigns rooms by some other rule.
  3. An invitation carries the room's name, id, reachable address and the inviter's identifier, and joining admits the inviter. Opens "An invitation now carries the room's name, its identity, and where to reach it…" and "An invitation now carries the inviting peer's identifier, and joining admits them." Its alternative is a token, excluded by D-026 (there is no join token at all).
  4. A peer advertises the address it listens on, signed. Opens "so a peer now advertises where it listens, signed, because a peer acts on that address by polling it and an unsigned one would redirect polling." Its own alternative is an unsigned address.
- keeps the number: 1. Five citations mean the many-rooms daemon (main.go:228, phase5-findings.md:80 and 85, decisions.md:2959, and half of spec-review.md:619). The rest mean the per-peer guest-list finding.
- move out:
  - history: "A machine-level *current room* answers instead: `join` sets it, and sessions started afterwards enter it." D-080 (there is no current room) removed it.
  - history: "Once a session has been offered teammate context it is marked…". D-056 (a session's room is fixed at first sight) removed the mark.
  - finding: "Two gaps only the end-to-end test exposed.…*A guest knew no room existed.*…*Synchronisation was one-way.*…". The decisions (3, 4) stay and the observed failures go.
  - finding: "a guest list is per-peer, so two peers can disagree about who belongs.".
  - history: "Context. A daemon served exactly<!-- writing: quotes the decision log --> one room…" and "the bridge is gone, not because it was removed but…".
  - "What this makes possible that was not before." is consequence and history.
- citations: 9. Four mean something other than decision 1:
  - docs/decisions.md:3082 "the defect behind the asymmetric guest list (D-046)" → per-peer guest-list finding
  - docs/decisions.md:3758 "Guest lists are per-peer (D-046), so two peers can already disagree" → finding
  - docs/decisions.md:5780 "guest lists are per-peer (D-046, the daemon serves many rooms)" → finding
  - docs/spec-review.md:619 "and a session binds to a room on first sight (D-046)" → partly decision 2. D-016 (one room per session) and D-056 also hold the rule.

### D-047: The fingerprint is the only manual link, and had the least careful encoding
- verdict: tombstone
- decisions:
  1. None stands. The entry is "CLOSED, not implemented - the construction it argued about was removed". Its decision, "Render the fingerprint as words", was closed by D-055 (one way to verify a peer). "`Fingerprint` survives as a display" repeats D-055's "What is kept".
- keeps the number: the tombstone. It keeps the title and a Status line pointing at D-055 and at the finding.
- move out:
  - finding (the reason for the reversal, W-37): the substitution demonstration ("an attacker substituted her own identifier in transit… Zero refusals.").
  - finding: the grinding measurements ("three characters fell in 339,297 tries and under a second; four take about thirty seconds; eight are 2^48…sixteen are 2^96").
  - finding: "a person… will read the first group, the last group, and skim the middle".
  - finding: the base64url dictation hazard (`l`, `I`, `_`).
  - finding: "The weakest link in this system is a human reading a string…".
  - history: "Closed without implementing it (2026-09-18).…".
- citations: 11. Once it is a tombstone, every citation that uses its content must point at the finding:
  - cmd/cogmer/auth.go:141 "every check above passes just as well for whoever substituted it in transit (D-047)" → finding (substitution demo)
  - docs/decisions.md:2179 "which D-047 demonstrated end to end with zero refusals" → finding
  - docs/decisions.md:2214 "(D-047 measured it)" → finding
  - docs/decisions.md:2587 "D-047 demonstrated exactly<!-- writing: quotes the decision log --> that end to end with zero refusals" → finding
  - docs/decisions.md:2653 "D-047 measured the cost: shown forty-three characters…" → finding
  - docs/decisions.md:2172, 2242 "D-047 remains held" / "D-047 is held, not cancelled" → history in D-048, deleted with it
  - docs/decisions.md:2265, 2510 "the full-length comparison of D-047 is the only option" → history in D-048 and D-052, contradicted by D-055
  - docs/decisions.md:2677 "D-047 is closed without being implemented" → history in D-055
  - CLAUDE.md:236 reading map "pairing, verification, the two words | D-055, D-088, D-093, D-047" → the finding, not the tombstone

### D-048: Verification should bind a live exchange, not a standing identifier (ZRTP's SAS)
- verdict: one, with non-decision parts to move
- decisions:
  1. Verification is a live commit-then-reveal exchange that derives two words from both identity keys and both fresh nonces, compared by the two people on a call. A mismatch is conspicuous and never presented as retryable. Opens "Decision. Adopt the live-exchange form…" and "The hazard to implement against.".
- move out:
  - history (reversed placement, W-38): "This entry placed that at join… D-052 rejected the placement… D-053 then moved it to pairing…" and the Status line's "its placement was superseded… D-047 remains held".
  - history: consequence 3, "D-047 is held, not cancelled.… A `verified` flag - which today does not exist anywhere in `membership.go`…". D-047 is closed and the flag exists (D-052).
  - history: "What it corrects here. §25 said… §25 now says so". The offline-precomputation versus online-guess distinction is support and stays as support.
  - support that belongs in a finding: "What ZRTP does.…" (external construction; needs a URL or a findings doc under W-34).
  - history: the Revisit when clause, "so both renderings may need to exist", which D-055 contradicts.
- citations: 4. None needs repointing.

### D-049: The sync request addresses a room by id, never by name
- verdict: one, with non-decision parts to move
- decisions:
  1. A room is named by `roomId` on the wire in both directions, and peer-supplied identifiers resolve only through `RoomByID`. Opens "Decision. `roomId` on the wire…". The `wireVersion` and signing-tag bump is a consequence of it.
- move out:
  - finding: "What the collision actually costs - the first reading was wrong.…".
  - history: "That bug is not fixed by this entry and is recorded here so it is not mistaken for fixed." D-050 fixed it.
  - history: "Context. Asked whether the room creator's daemon signs its polling requests…".
- citations: 3. None needs repointing.

### D-050: Room names may collide locally; the schema stops forbidding it
- verdict: split
- decisions:
  1. Room names may collide locally, and the uniqueness constraint on `rooms.room_name` is dropped. Opens "Decision. Drop the constraint." The semicolon joins two halves of one choice. Rejected: "Renaming a joined room locally to keep names unique".
  2. An ambiguous name is reported with both identities rather than guessed at or answered as unknown. Opens "`FindRoom` reports three outcomes, not two." Its own alternatives are in the paragraph: answering "no such room", or "returning whichever row came back first". You could allow collisions and silently pick one. Later decisions (`docs/decisions.md:3806`) cite this as a pattern of its own.
- keeps the number: 1. It is the title, and citations split two and two, so the title decides.
- move out:
  - history: "Context. Found while making the wire address rooms by id… `runJoin` called `log.Fatalf`…".
  - finding: "Names are drawn from 7,656 combinations… Around a hundred rooms over a machine's lifetime makes it a coin flip."
  - "`CreateRoom` asks whether a name is free" and "`migrateMembership` rebuilds the table" are implementation consequences of decision 1, not decisions.
- citations: 4. Two mean decision 2:
  - docs/spec-review.md:673 "an ambiguous name is reported with both identities rather than guessed at (D-050)"
  - docs/decisions.md:3806 "Same shape as D-050: report the ambiguity, name both"

### D-051: Stranger pairing is not a supported case
- verdict: one, with non-decision parts to move
- decisions:
  1. Pairing with someone unknown is not a design target: no affordance presents it as intended and no claim is made that verification protects it. Whom to admit remains the host's judgement. Opens "Decision. Exclude stranger pairing as a design target…". "Whom to admit remains the host's judgement" is stated as this decision's limit ("What is not decided. The mechanism does not forbid it"), not as a second decision.
- move out:
  - open item for `open.md`: "Also not decided: whether the host-approval path is built at all…".
  - open item for `open.md`: "Recorded as open in §12a: whether a request may arrive unsolicited…" together with the expecting-someone window.
  - history: "Noted because the two were previously argued as one…" and the Context's account of §12a's earlier wording.
- citations: 6. One cites the open item that moves:
  - docs/open.md:31 "a host approving an unsolicited join request - the second undecided rather than pending (§12a, D-051)" → the open question. The citation sits in `open.md`, where the item would now live, so it drops to §12a.

### D-052: The SAS is its own act, not part of joining or approving
- verdict: split
- decisions:
  1. Verification is its own command, `cogmer verify <peer>`, run by both people at once on a call, and separate from joining and from host approval. Opens "Decision. `cogmer verify <peer>`, run by both people at the same time…" and "The two are orthogonal.".
  2. A daemon takes part in a verification only while its own person has asked for one. Otherwise it answers 409 and displays nothing, so no inbound request creates anything. Opens "Both sides run it, and that is what keeps §12a's open question closed.". Its own alternative is an inbound verification request that prompts the host, "a prompt that can be trained away". You could keep verification its own act and accept inbound requests.
  3. Only a person's confirmation records a verification (`MarkVerified`), a mismatch records nothing, and a verified peer loses the `unverified` marker. Opens "Only a person may record a verification." and "The marker now means something.". Its own alternative is a failed-verification state: "storing one invites an interface that offers to retry".
  4. The two words come from the PGP biometric word list, alternating even and odd, disjoint from the peer-name and room-name vocabularies. Opens "Wordlists.". Its own alternative is reusing the peer-name or room-name vocabularies, "one mnemonic mistakable for the other".
- keeps the number: 1. It is the title, and five of nine citations mean the separate act.
- move out:
  - history: "The exchange is symmetric - true of a single round, and not of the session around it, which D-059 had to correct…". The address-discovery rule in that paragraph ("keeping the one that answers signed by the identity asked for") is part of decision 1 and stays.
  - history: "Before this the tag was true of every peer forever…".
  - history: the Revisit when clause, "D-047's full-length rendering is the only option, so both may need to exist", which D-055 contradicts.
  - history: "Context. D-048 accepted ZRTP's short authentication string in principle and put it 'at join'…".
  - "Not done. The words are 16 bits… Nothing rate-limits attempts." is a limit and stays under Limits.
- citations: 9. Four mean decisions other than 1:
  - cmd/cogmer/daemon.go:82 "held only while a person has asked for one. Nothing here is created by an incoming request (D-052)" → decision 2
  - cmd/cogmer/daemon.go:499 "a peer verified over a recognising channel (D-052) is not marked" → decision 3
  - cmd/cogmer/membership.go:31 "When two people compared a SAS and said it matched (D-048/D-052)" → decision 3
  - docs/decisions.md:2891 "D-052 described the exchange as symmetric - 'each side sends its commitment…'" → the symmetry text, which moves out as history. This line is itself history inside D-059.

### D-053: `pair` is machine scope and `invite` is room scope; the commands now say so
- verdict: split
- decisions:
  1. `cogmer pair` is the machine-scope act: it records the peer and a bootstrap address and runs the two-word comparison in one command, so pairing precedes any room. `invite` stays room-scoped. Opens "Decision. `cogmer pair <identifier>[@address] [name]` is the durable act…" and "The ordering gap this closes.".
  2. `allow` survives as a low-level record-without-verifying for scripts and tests, and prints UNVERIFIED. Opens "`allow` survives as the low-level 'record without verifying' for scripts and tests…". Its own alternative is removing `allow`, or keeping its old fingerprint output. You could add `pair` and delete `allow`.
  3. An interactive act that ends in something a person must read unaltered (pair, verify) runs at a terminal. A non-interactive room act (invite) may be a slash command. Opens "Why the naming matters more than it looks. The split decides where each act can live…". Its own alternative is everything at a terminal, "what 'commands are typed at a terminal' had quietly become". D-057 cites this as a decision about §29.
- keeps the number: 1. Five of nine citations mean pairing as the machine-scope act.
- move out:
  - reversed part (W-38): the Status line's "except its gating position, reversed by D-054 the same day" and "Gating - see D-054. The warnings this entry added…". The entry should be rewritten to state what holds, with the reversal in a finding.
  - history: "§12 is retitled from 'Session Pairing' to 'Forming a Room'…".
  - history: "`verify` previously found a peer only through addresses learned from room membership… the sequence is now pair → create → invite → join rather than…".
  - history: "Context. Asked why the commands are typed at a terminal… The … answer was that nothing is packaged yet…".
- citations: 9. Three mean decisions other than 1:
  - cmd/cogmer/ui_test.go:206 "Recorded with no label of its own, as a script would (D-053)" → decision 2
  - cmd/cogmer/main.go:645 "It used to name `allow`, which D-053 reserves for scripts and tests" → decision 2
  - docs/decisions.md:2748 "§29 had already split commands between a session and a terminal (D-053)" → decision 3

### D-054: Verification gates synchronization and injection, not just a marker
- verdict: one, with non-decision parts to move
- decisions:
  1. An unverified peer's events are not served, not stored (judged at the origin), and not injected, and they are held rather than dropped. Opens "Decision. Three gates, because there are three ways in:". "Held, not discarded", explaining the silence, and "inviting an unverified peer remains permitted and inert" are properties and limits of the same gate.
- move out:
  - history: "Context. Asked whether we plan to admit unverified guests. We did - by omission… `IsVerified` was consulted in exactly<!-- writing: quotes the decision log --> three places…".
  - history: "Why the previous position did not hold. It rested on D-051…". The fact it rests on (every check passes for a substituted key) is support and stays, citing the D-047 finding.
  - finding: "The existing `authDaemon` fixture now verifies its guest, which is itself evidence the gate bites…".
- citations: 32. None needs repointing. `CLAUDE.md:30-31` and `docs/writing.md:108-109` use the number as an example with its correct few words.

### D-055: There is exactly<!-- writing: quotes the decision log --> one way to verify a peer
- verdict: one, with non-decision parts to move
- decisions:
  1. The live two-word comparison is the only way to verify a peer. There is no whole-key fallback, and `Fingerprint` is display-only. Opens "Decision. One ceremony." "What is kept." (display-only `Fingerprint`) is this decision's limit.
- move out:
  - finding and history: "It did not exist. `Fingerprint` had no callers outside its own test…".
  - history: "Consequence. D-047 is closed without being implemented."
  - history: "Context. §25 retained the whole-key comparison as a fallback… Asked why."
- citations: 13. None needs repointing.

### D-056: A session's room is fixed at first sight; the `injected` flag is removed
- verdict: split
- decisions:
  1. A session binds to a room at first sight and never moves, with no exception for a session that has received nothing. Opens "Decision. Keep the strict rule and delete the machinery for the loose one. A session binds on first sight and stays." This overlaps D-016 (one room per session), which CLAUDE.md cites for the same rule.
  2. A column encoding a removed rule is dropped by migration, not left in place. Opens "The column is dropped, not left.". Its own alternative is in the paragraph: "It would have been harmless: it has a default and nothing writes it". You could keep the strict rule and leave the column.
- keeps the number: 1. It is the title, and `docs/spec-review.md:680` "keeping the strict behaviour and deleting the flag" means it.
- move out:
  - history: "`injected`, `MarkInjected` and `HasReceivedContext` are gone" and the title's second clause, "the `injected` flag is removed".
  - history: "Context. Initialized CodeGraph and ran a dead-symbol sweep…" and "The specification was therefore looser than the code…".
  - finding: "What the sweep says about the method.… A symbol with no callers is worth treating as a question…".
- citations: 4. Three mean something other than decision 1:
  - cmd/cogmer/membership.go:165 "a column encoding a rule that was removed is a rule somebody will later find and reinstate (D-056)" → decision 2
  - cmd/cogmer/membership.go:78 "this one was written and never read (D-056)" → the history of the flag
  - docs/decisions.md:2925 "the third such column found this week, after `injected` (D-056)" → the dead-symbol finding

### D-057: A slash command is a thin wrapper over the CLI; the session names itself
- verdict: split
- decisions:
  1. A slash command shells out to the corresponding CLI command, with one implementation and two entry points. Opens "Decision. Slash commands shell out."
  2. A session-scoped command learns its session from `CLAUDE_CODE_SESSION_ID`, and without it the command refuses rather than falling back to a machine-level setting. An explicit override exists for tests. Opens "They pass no session id, because the CLI reads `CLAUDE_CODE_SESSION_ID`…" and "The variable is absent at a terminal, so a session-scoped command run there… refuses." Its own alternatives are passing the session id explicitly, or a machine-level current room (rejected in Revisit when). You could keep thin wrappers that pass the id as an argument.
  3. A command that cannot be wrapped (pair, verify) has a slash counterpart that tells the person to run it in a terminal. Opens "Not every command can be wrapped… Their slash counterparts print an instruction… A signpost is … in a way a proxy would not be." Its own alternative is a proxy.
- keeps the number: 1. Its only citation is `Shared Claude Sessions.md:2744` "A slash command is a thin wrapper… (D-057)".
- move out:
  - finding, already B21: "`CLAUDE_CODE_SESSION_ID` is exported into the environment of every Bash tool call… Verified on 2.1.275…". Cite B21 and do not restate it.
  - held elsewhere: "What it allows us to delete. The machine-level *current room*…". D-080 (there is no current room) holds that decision.
  - history: "Context.…the assumption underneath was that they would need a second implementation" and "The objection that had blocked this was wrong.".
  - open item: "The model may retry.… Either make creation idempotent per session… or have it refuse." It is undecided and belongs in `open.md`, or is settled by a later entry.
- citations: 1. None needs repointing.

### D-058: Signature schemes are kept, never replaced
- verdict: one, with non-decision parts to move
- decisions:
  1. An event records the signature scheme it was signed under and is verified under that scheme. Old schemes are added beside, never edited, and an unknown scheme is refused as a version problem. Opens "Decision. An event records the scheme it was signed under…". "Zero means v2", "the version is not covered by the signature" and "unknown scheme refused as a version problem" are the mechanism's details, not separate decisions.
- move out:
  - history: "`signingBytes()` hard-coded a single event tag… and `Verify()` always recomputed with today's code."
  - history: "The related hazard, recorded and not fixed. `wireVersion` is a hard refusal…". D-065 (the daemon reads a range of wire versions) has since resolved it.
  - history and support: "What was already right, and worth not disturbing: `originSessionId`…". It restates D-043.
  - history: "Context. Asked whether anything in the current approach would make backwards compatibility hard…".
- citations: 10. One means another decision:
  - cmd/cogmer/offer.go:47 "a signing scheme of its own rather than a reuse of the sync one… or a captured offer could be replayed as something else (D-058)" → D-042's purpose-tag rule (decision 2 there, which takes a new number)

### D-059: Only one side drives a verification; the other completes from inbound
- verdict: split
- decisions:
  1. Either side completes a verification from inbound once it holds the other's revealed nonce, so only one side has to drive the exchange. Opens "Decision. Either side may complete from inbound."
  2. The verification endpoint answers 409 for "I do not know you yet" and reserves 401 for a signature that did not verify. Opens "Fixed by answering 409 for 'I do not know you yet'…401 now means only that a signature did not verify". Its own alternative is 401 for an unknown peer, treated as final. You could fix the stranding and keep 401 for unknown peers, or the reverse.
- keeps the number: 1. It is the title and the only citation.
- move out:
  - finding: "Context. Phase 5's two-machine run. Pairing failed twice…", "First: whoever typed first lost.…", and "Second, and deeper: the side that finished stranded the other.…".
  - history: "D-052 described the exchange as symmetric - …That was true of a single round and false of the session around it."
  - finding: "Why a test did not catch it. `TestTwoDaemonsReachTheSameWords` starts both sides in goroutines with no delay…".
- citations: 1. None needs repointing. The entry has no Rejected. field, so W-36 needs one.

### D-060: A sequence is reserved outside the room before the event that uses it
- verdict: split
- decisions:
  1. A sequence is reserved and recorded in `membership.db` before the event that uses it is published, and `Store.Append` takes the sequence rather than deriving one. Opens "Decision. `Membership.ReserveSequence` issues the number…" and "Reserve, then publish, and not the reverse." Keeping the sequence outside the room is D-029's design (losing a room database does not end membership), and this entry adds the order and the API shape.
  2. Losing a room's state is reported when the room is opened. Opens "The loss is reported, because it is otherwise invisible. `reportLostState` runs when a room is opened…". Its own alternative is silent recovery, which the paragraph rejects ("otherwise invisible"). §8 requires the report; the choice of the open-time moment is this entry's. You could reserve sequences and still recover silently.
- keeps the number: 1. Both citations mean reservation.
- move out:
  - history: "Context. Review C-1… D-029 settled the design a fortnight ago… None of it was built." and "`rooms.issued_sequence` existed in the schema and was written by nobody…".
  - finding: "What that cost.… Reproduced during the review: the daemon kept serving from its open file handle after the file was deleted…".
  - history, unrelated to this decision: "Also fixed, from the Phase 5 findings. `log`, `conflicts` and `seed` resolved a room through `config.json`… now use the current room from `membership.db`. `whoami` deliberately<!-- writing: quotes the decision log --> does not…". It is a separate fix, and it cites a current room that D-080 removed.
  - "Recovery needed no new mechanism." is support for decision 1.
- citations: 2. None needs repointing. The entry has no Rejected. field, and "Reserve, then publish, and not the reverse" is the alternative to move into one.

### D-061: Phase 7's last three: one dissolved, two built
- verdict: split
- decisions:
  1. There is no outbound queue, because synchronization is a pull and the event store is the only record of what a peer has not fetched. Opens "### The outbound queue is dissolved, not deferred".
  2. A peer outage is reported on transition: when a peer stops answering, every ten minutes while it stays gone, and once when it returns with the duration. Opens "### Peer health: report transitions, not polls". Its own alternative is logging once per poll.
  3. Injected context has a whole-block character budget applied after the event cap and measured on rendered turns. It drops from the front, all limits are configurable, and a zero or unparsable limit is ignored. Opens "### Context size: the limit that was missing bounded nothing" and "A whole-block budget now applies after the event cap…". Its own alternative is the per-turn and per-count caps alone.
  4. The token count is estimated from characters at four to one, not measured with a tokenizer. Opens "On estimated token count… derived from characters at four to one rather than measured." Its own alternative: "A tokenizer would have to track a model this system does not choose".
- keeps the number: 2. The only citation (`cmd/cogmer/ui.go:296`) is in the peer-health section. The title is a topic ("Phase 7's last three"), not a decision, and needs replacing under W-31 whichever part keeps it.
- move out:
  - history: "Recorded in §7 as struck through with the reason, rather than removed."
  - finding: "which was observed filling a log through the Phase 5 partition".
  - history: "Also fixed: `peerStatus` enumerated only `COGMER_PEERS`, so every peer learned by pairing… was invisible in the browser view…".
  - finding: "Forty turns of eleven thousand characters pass both. Measured: 404,635 characters…".
  - history: "§21 is amended to say what each limit is *for*…".
  - history: "Context. Phase 7 listed a local outbound queue… produced one deletion and two findings."
- citations: 1. It cites history rather than a decision:
  - cmd/cogmer/ui.go:296 "peers learned by pairing or by joining a room were invisible here until D-061" → the `peerStatus` fix, which moves out as history. Reword the comment or point it at the finding.

### D-062: Tailcat evaluated for Phase 15: a good fit, adopted behind an interface if at all
- verdict: split
- decisions:
  1. Tailcat, if used, sits behind a narrow interface of our own (a Dial and a Listen) and decides nothing about who may speak. Opens the title's "adopted behind an interface if at all" and "… code behind an unstable API is why it should sit behind a narrow interface of our own - a Dial and a Listen…". Adoption itself is D-068 (tailcat is the cross-network transport, behind our own dialer), per the Status line.
  2. Tailcat's `AllowedClients` is not used, so admission has one authority. Opens "Do not use `AllowedClients`.". Its own alternative is a third allowlist keyed on the WireGuard key. You could adopt tailcat behind an interface and still use `AllowedClients`.
- keeps the number: 1. Most citations mean the evaluation findings, which move out, or D-019's "decides nothing". Decision 1 is the part the title names.
- move out:
  - history: the Status line's "Was: investigated, not yet adopted, but the likely answer rather than a contingency…".
  - finding, most of the entry: "What it is.…", "Why it fits this design unusually well. The API is `net.Conn`-shaped…", "Measured, not assumed." with the size table, "526 dependencies against our current one", and "Three risks to weigh at Phase 15, not now." (no stability promise, DERP rendezvous, rate-limited public relays).
  - history, restating D-063: "What would make it the answer was whether Phase 13's first outside user is remote. They are (D-063)…" and "The DERP objection also weakens for this pair specifically…".
  - restatement of D-019 (a transport must sit under everything) and D-026 (no join tokens): "And it lands exactly<!-- writing: quotes the decision log --> where D-019 said a transport must…" and "Two distinctions to hold… A `tc…` address… Tailcat has its own WireGuard keypair…". The transport key versus `peerId` distinction is a limit of decision 1 or of D-068.
  - history: "Every cross-network run so far… used an SSH tunnel".
- citations: 5. Three mean the evaluation findings, not decision 1:
  - docs/decisions.md:3157 "What this does to the tailcat evaluation (D-062)… 526 dependencies and a roughly doubled binary" → finding
  - docs/decisions.md:3313 "Tailcat would roughly double it (D-062)" → size finding
  - docs/decisions.md:6160 "*Traces* - D-062 evaluated it" (the DERP relay) → finding

  `cmd/cogmer/tailcat.go:38` and `docs/decisions.md:3173` "(D-019, D-062)" mean "decides nothing" and stay with decision 1.

### D-063: The first pair is remote, so cross-network reach comes before local discovery
- verdict: one, with non-decision parts to move
- decisions:
  1. Phase 15 (cross-network reach) is built before Phase 12 (local discovery) and is a prerequisite for Phase 13. Where: "Decision. Phase 15 moves ahead of Phase 12".
- keeps the number: decision 1, the only one.
- move out:
  - history: "Context. The phase order placed local discovery…" ("That fact is now known") and "It is simply no longer first".
  - history: "What that invalidates." is an account of D-019's earlier build order. The D-019 requirement it restates ("D-019's *requirement* is untouched") is a citation, not support.
  - support that belongs in a finding or in D-068: "What this does to the tailcat evaluation (D-062)." (526 dependencies, doubled binary, VPN / public bind / SSH tunnel all worse) and "The DERP objection weakens for this specific pair" (self-hosted relay on the droplet).
  - restated decision: "What has not changed." restates D-019 and D-062 (a transport decides nothing). Keep only as a citation.
- citations: 11. All mean the build order. None to repoint.

### D-064: A session's room is the one somebody chose inside it, never a machine default
- verdict: one, with non-decision parts to move
- decisions:
  1. `RoomForSession` reports the room a session was put in by a command run inside it, and never puts it in one. Where: "Decision. `RoomForSession` reports the room…".
- keeps the number: decision 1.
- move out:
  - history: "This entry was never written." paragraph (reconstruction from `e4fd2d8`).
  - history: "What became of the other pointer." (D-076, D-077, D-080, the `current_room` schema comment and `settings` table, "no longer exists").
  - history: "Context. One pointer was answering two different questions" (the machine-level `current_room` no longer exists).
  - Rejected header note "what the code rules out, not what was considered" is provenance history.
- citations: 12. All mean the session-chosen room rule. None to repoint.

### D-065: The daemon reads a range of wire versions, so upgrading is not a flag day
- verdict: one, with non-decision parts to move
- decisions:
  1. A build declares `wireVersion` and `minWireVersion`, accepts anything between, reads an absent version as 1, and reports a peer outside the range. Where: "Decision. A build declares the newest version…" with "Zero means one." as part of the same range rule, and "Why the floor moves rarely." as its limit.
- keeps the number: decision 1.
- move out:
  - history: "This entry was never written" paragraph.
  - history: "Why this stopped being optional at Phase 11." (the number of people changed).
  - "Today that is 2 and 1" is current state, which goes stale at the next bump.
- citations: 2. None to repoint.

### D-066: The binary is fetched and verified, never shipped in the plugin
- verdict: split
- decisions:
  1. The binary is fetched at first run into `~/.cogmer/bin`, and nothing runs unless its hash is in the committed `plugin/checksums.txt`, which `release.sh` generates from the bytes it built. Where: "Decision. Fetch at first run into `~/.cogmer/bin`, and run nothing that cannot be verified."
  2. The installer runs detached, outside the plugin directory, one at a time behind a `mkdir` lock, and waits an hour after a failure before trying again. Where: "Three lessons taken from the only comparable bootstrap…" (install outside plugin dir, `mkdir` lock, one-hour cooldown) and "And one of our own: it runs detached." Its own alternatives: install inside the plugin directory ("a plugin update must not discard a working binary"), concurrent installs writing one path, retrying every session, and a session waiting on the download (§29). Any of these could change and fetch-and-verify would still hold.
- keeps the number: decision 1. The title, the Phase 11 row, D-067 and D-115's release-host item all mean fetch-and-verify.
- move out:
  - finding: "What the ecosystem does - stated more carefully than it first was." (53 plugins, the `terraform` case), including its self-correction ("stated more carefully than it first was", "asserted here before it was checked").
  - support that belongs in a finding: "The arithmetic." (56 MB, git history growth, the `.gcs-sha` archive observation).
  - support that belongs in a finding: "What a registry would have given us, and what we gave up." and the goreleaser framing. These are really Rejected items: ship in the repository, a registry or `npx`, build from source.
  - finding: "Verified by running it, including the case that matters."
  - history / open.md state: "It is inert today, and deliberately<!-- writing: quotes the decision log --> so." (releases exist now).
  - finding, now history: "A finding that changes what must happen next." (a private repository makes `go install` fail; Phase 13 depends on a release).
- citations: 6. Two mean the private-repository finding and go to its findings document once it is moved:
  - docs/decisions.md:3564 "which is why `go install` failed in D-066's test"
  - docs/decisions.md:6319 "which is why `go install` failed in D-066's test"
  - Nothing cites decision 2.

### D-067: Release assets are served from the droplet; the host is data, not code
- verdict: split
- decisions:
  1. Release assets are served from the droplet until the code is public, when GitHub Releases takes over. Where: "Decision. Serve the assets from the droplet…"
  2. The release host is a committed data file, `plugin/release-url.txt`, versioned with the checksums and `VERSION`. Building (`release.sh`, host-agnostic) is separate from publishing (`publish.sh`, host-specific), and `publish.sh` fetches back what the host serves and checks it against the pinned hashes. Where: "The host is data.", "That also decides where the seam goes.", "publish.sh verifies what the host actually serves". Its own alternative: a host written into code or scripts. It would carry over unchanged to a GitHub Releases host.
  3. Assets are served over plain HTTP, because the pinned sha256 is what authorises execution, and the pin must not be removed as redundant. Where: "Plain HTTP, and the checksum is why." Its own alternative: HTTPS on the droplet ("HTTPS would still be better"). It could be reversed by putting TLS on the droplet without moving the host.
- keeps the number: decision 1. It is the first half of the title, and the only citation (the Phase 11 row) means the release arrangement in general.
- move out:
  - history: "Context." ("there is no git remote at all", `github.com/Blue-Rocket/cogmer` inherited, "Both paths dead"). This describes a state that D-117 changed.
  - finding: "Verified end to end, from a clean state…" (11 MB, republish replaced in place, nginx autoindex).
  - "What this costs." is Limits for decision 1 (availability dependency), not a separate decision.
- citations: 1. None to repoint.

### D-068: Tailcat is the cross-network transport, behind our own dialer
- verdict: one, with non-decision parts to move
- decisions:
  1. Tailcat carries cross-network sync, confined to `tailcat.go` behind the `Dialer` interface and a `net.Listener`, so one set of routes serves both transports. Where: "Confined to one file." There is no Decision. field; the title and this paragraph are the decision.
- keeps the number: decision 1.
- move out (most of the entry):
  - finding: "What was verified, stated precisely<!-- writing: quotes the decision log --> because the first version of this was overstated." (the 291 ms spike was NAT-to-public, then a full no-tunnel run upgraded to direct UDP).
  - finding / open.md state: "Still not tested: neither side able to accept inbound." and "What is nearly certain regardless."
  - finding: "Two defects found by running it…", "A client is not a connection." (per-peer client cache, 30s timeout) and "A timeout is the signature of the packet filter." (`ServedTCPPorts`). The resulting code choices belong in comments at `clientFor` / `ServedTCPPorts`, with the finding as source.
  - finding: "A third defect, found and unrelated to tailcat." (`session_id` vs `sessionId`; the Phase 5 run's degenerate attribution). This belongs in the Phase 5 findings, not here.
  - finding: "Costs, measured." (566 dependencies, 21 MB).
  - history: "Context. … Phases 2 and 5 substituted an SSH tunnel".
- citations: 2. Both mean the status or history, not the decision:
  - Shared Claude Sessions.md:2392 "built and working between two machines; NAT-to-NAT awaits the real peer (D-068)". This is state and belongs to the finding.
  - docs/decisions.md:3037 "adopted - see D-068 for what was built". This is history. It stays pointed at D-068.

### D-069: Signing namespaces and the state directory are decoupled from the name
- verdict: split
- decisions:
  1. Every signing domain-separation tag uses `protocolNamespace` (`peer-room`), which is arbitrary on purpose and never changes, so nothing cryptographic depends on the product name. Where: "The signing namespaces." through "They should never have carried a product name". Moving the three live-exchange tags outright rather than versioning them is part of this ("The other three tags moved outright rather than gaining a version").
  2. The state directory path is one constant, so a rename changes one line. Where: "The state directory. `~/.cogmer` is where the name reaches the filesystem." Its own alternative: the literal repeated wherever it is used (implicit, "rather than a search"). It is independent of 1.
- keeps the number: decision 1. CLAUDE.md, the spec, `offer.go`, D-070 and D-117 all cite "nothing cryptographic".
- move out:
  - history: "D-058's mechanism got its first real use…" (`signingBytesV3`; the note that D-117 later deleted the scheme).
  - finding: "And a latent bug found by looking." (`install.sh` honoured `COGMER_HOME`, the binary did not).
  - history: "What is deliberately<!-- writing: quotes the decision log --> not done." (module path left alone). D-117 fixed it.
  - history / stale: the Revisit list ("`/team-*` commands"). The name is settled (D-117).
  - "Context. Asked to push the repository…" is history.
- citations: 9. The ones that need checking:
  - docs/decisions.md:6317 "D-069 recorded that the inherited module path disagreed with the organisation's actual name". This means the module-path paragraph (history). Point it at the finding once that is moved.
  - docs/decisions.md:3596 "Unlike the couplings in D-069". Plural: it covers both decisions. Cite both.
  - docs/decisions.md:6289 "D-069 did the expensive half of this… by breaking the couplings". Plural: covers both decisions. Cite both.
  - Total to check or repoint: 3.

### D-070: Slash commands carry a distinctive prefix, because invocation is not namespaced
- verdict: one, with non-decision parts to move (a supersession candidate; see the note)
- decisions:
  1. Room commands use the `room-` prefix instead of `team-`, to avoid collisions. Where: "Decision. `/team-*` becomes `/room-*`."
- keeps the number: decision 1.
- move out:
  - finding: "Context." (a subdirectory changes display, not invocation; `hookify` and `ralph-loop` both define `/help`). D-118 found this no longer true.
  - "The prefix is tied to the protocol namespace…" is reasoning that D-096 (a prefix names its target) replaced. D-118 says "What keeps the prefixes is D-096". Per W-38 this reasoning goes, or the whole entry becomes a tombstone pointing at D-096 and D-118.
  - history: "becomes" and "It is being done now" phrasing.
- citations: 3. One is a mis-citation:
  - docs/decisions.md:3704 (in D-072) "for the same reason the view is not summarised (D-070's sibling concern)". D-070 says nothing about summarising, so this citation points at nothing in the entry.

### D-071: Leaving is a pause, and the row that records it is a tombstone
- verdict: one, with non-decision parts to move
- decisions:
  1. Leaving is a per-session pause. The session may rejoin the room it was in and no other, so its `session_rooms` row is kept and marked `left_at` rather than deleted. Where: "The first one sharpened the invariant.", "The trap, which required the tombstone." and "Leaving is per-session". The tombstone row is the mechanism that makes a pause compatible with §12a, not a separate choice: deleting the row, the only alternative the entry names, would break §12a as well as the pause.
- keeps the number: decision 1.
- move out:
  - history: "Context. … `leave` did not leave" (the current-room pointer, the never-read `state` column).
  - history: "The third one caught a live defect." (leaving blanked the view; "Leaving no longer touches the pointer"). The pointer no longer exists (D-080).
  - open.md state: "Still not implemented, and now the only part of §22's lifecycle that is not." (rooms never close).
  - "What leaving deliberately<!-- writing: quotes the decision log --> does not touch." is Limits (events, guest list, and synchronization, which is left open).
- citations: 7. One refers to the history part, and its code comment is stale:
  - cmd/cogmer/main.go:878 "It deliberately<!-- writing: quotes the decision log --> leaves the current-room pointer alone. Clearing it used to be the whole of leaving, and it blanked the browser view… (D-071)". The pointer is gone (D-080). This goes to the finding or is removed. The same stale claim, without its own citation, is in membership.go:688 ("does not change which room command-line commands or the browser view act on").

### D-072: A refused join is explained, … refused
- verdict: one, with non-decision parts to move
- decisions:
  1. A refused join states the mechanism (the first room's context would leak into the second through what the session says next) and the way out (a new session, or rejoin the same room). It is carried by a typed `BoundElsewhereError` naming both rooms and relayed unabridged by `/room-join`. Where: "Decision. Explain the mechanism and give the way out." The typed error and "the slash command is told not to condense it" are how the explanation reaches the person, not separate choices.
- keeps the number: decision 1.
- move out: history, the quoted old refusal and `log.Fatalf` in "Context.". A mis-citation to fix: "(D-070's sibling concern)" (see D-070).
- citations: 3. None to repoint.

### D-073: Forgetting a peer discards their admissions with them
- verdict: one, with non-decision parts to move
- decisions:
  1. `Forget` deletes the identity and every admission it carried in one transaction, so meeting again is a first meeting. Where: "Decision. `Forget` deletes admissions and identity in one transaction."
- keeps the number: decision 1.
- move out: history, "Context." (`Forget` deleted only `known_peers`) and "What that cost." (masked by D-054; readmission on re-verify). The reason is support once restated as a present fact. "Revoke is unchanged" is history phrasing for a scope limit.
- citations: 5. None to repoint.

### D-074: A name means one key, and a collision is where a key change surfaces
- verdict: split
- decisions:
  1. A local label belongs to one key. `Allow` refuses a name held by a different key with a `NameTakenError` whose explanation treats the collision as a possible substitution and gives the `forget` route. Where: "Decision. A name may belong to one key." The title's second half ("a collision is where a key change surfaces") is the reasoning for this, not a second decision.
  2. `resolvePeer` refuses a name that matches more than one key and names both, rather than choosing. Where: "`resolvePeer` refuses an ambiguous name rather than choosing." Its own alternative: pick one ("picking between them would admit a peer nobody named"), or migrate the duplicates away. It governs databases written before 1 and could change (say, to a migration) without touching 1.
- keeps the number: decision 1. D-090, D-093, D-094 and the code comments on `NameTakenError` and `TestANameMeansOneKey` all mean uniqueness.
- move out:
  - finding: "And it passed silently." (two alices, `resolvePeer` returned the first).
  - history: "Nothing implemented that alarm", read against `pair`'s old message.
  - "What this does not do." is Limits.
- citations: 8. None need repointing. cmd/cogmer/main.go:573 sits in `resolvePeer`'s comment, but its words ("Recording two keys under one name is now prevented (D-074)") mean decision 1. It could add a citation of the new number for its own rule.

### D-075: The install has a state, and absence is not an inference
- verdict: split
- decisions:
  1. The installer writes one state file (`installing`, `failed` with reason, removed on success, `stalled` derived after five minutes). The command wrapper and the session-start hook, which passes it to the model as `additionalContext` only while the binary is absent, both read it. Where: "Decision. One state file, written by the installer, read by everything." "What each reader does with it." and "What it does not do." are part of this.
  2. A plugin and the binary it pins are released as one artefact, with the release script writing `VERSION` so they cannot drift. Where: "Released together, because they are not separable." ("A plugin and the binary it pins are one artefact"). Its own alternative: version the plugin and binary independently, which is what let v0.2.0 reference subcommands its binary lacked. It is independent of the install mark.
- keeps the number: decision 1. `common.sh`, `cli.sh`, `main.go:1415`, open.md and D-084 all mean the install state.
- move out:
  - history: the v0.3.0 / v0.2.0 release account inside "Released together…".
  - finding: "Verified from nothing".
  - history: "Context. Asked what options exist…".
- citations: 7. One to repoint to decision 2:
  - cmd/cogmer/membership_test.go:781 "The plugin and the binary version together (D-075)"

### D-076: A command inside a session acts on that session's room
- verdict: tombstone
- decisions: none; it is a tombstone already in W-37 form.
- keeps the number: n/a.
- citations: 12, all to the withdrawn decision (findings, the `structure_test.go` fixtures, a test comment and history in D-064 / D-077). None to repoint. The tombstone itself names "Replaced by D-077 (every room has its own URL, and no ambient value picks one)". Once D-077 is split, it should name the new ambient-value entry (see D-077).

### D-077: Every room has its own URL, and no ambient value picks one
- verdict: split
- decisions:
  1. Every room has its own view URL. `/room/<name>` serves it, `/` lists rooms and redirects only when there is one, the events and stream endpoints take the room as a parameter, and `watchLine` prints the room's own address. Where: "Per-room URLs."
  2. No ambient value chooses a room: a room is named in the invocation or it is the session's. `COGMER_ROOM` is removed and a test fails if anything reads it. Where: "A room is named per invocation, or it is the session's." and the "Context." / "The pattern worth naming…" paragraphs. Its own alternative: an environment variable ranked above the session (D-076). It could be reversed without removing per-room URLs, and the reverse is also true.
- keeps the number: decision 1. `ui.go:98`, `daemon.go:359`, `main.go:259`, `ui_test.go:30`, and D-064, D-080 and D-087 all mean per-room URLs (7 against 4–5).
- move out:
  - history: "What that settles about the pointer." (the terminal fallback that D-080 removed).
  - history: "Context. Two objections…" ("introduced an hour earlier", "It is gone, along with the dead `LoadConfig`").
  - stale: the Revisit when paragraph ("Today the fallback answers") describes a fallback D-080 deleted.
- citations: 13 (one is a `structure_test.go` fixture string, not a reference). Five to repoint to decision 2:
  - docs/decisions.md:3883 D-076 tombstone "Replaced by D-077 (every room has its own URL, and no ambient value picks one)". What replaced D-076 is decision 2.
  - docs/room-choice-findings.md:33 "D-077 … states the rule that replaced it: a room is named in the invocation, or it is the session's"
  - cmd/cogmer/membership_test.go:729 "COGMER_ROOM was consulted ahead of the session's own room, which quietly reinstated the failure D-064 removed, at higher precedence (D-077)"
  - docs/decisions.md:3235 "D-077 answers it by naming a room per invocation rather than by storing one"
  - CLAUDE.md:239 "rooms, membership, session binding | … D-077". This row is about session binding, so it means decision 2. It is ambiguous and could cite both.

### D-078: Granting access names its room; showing something may guess
- verdict: one, with non-decision parts to move (a partial reversal; see the note)
- decisions:
  1. A command that changes who can read a room must not guess the room, while one that only shows a room may, because being wrong is self-announcing. Where: "Decision. `roomToChange` does not guess… `roomToShow` may still fall back". Its status line says this distinction stands.
- keeps the number: decision 1, the showing/changing distinction. `main.go:995,1020` and D-080 cite it as "(D-078, D-080)" / "(D-078, D-079)".
- move out:
  - reversal: the `--room <name>` flag, the refusal text that prints it, and "The slash commands need no flag…" were superseded by D-079. Under W-38 they go to a finding.
  - reversal: "What this leaves of the fallback." (read-only terminal fallback). D-080 deleted the fallback, leaving only `conflicts` by name.
  - history: "Context. … split what I had been treating as one decision" and the "stored pointer, set possibly weeks earlier".
  - The status line itself is a partial-supersession note that W-38 says to replace by rewriting.
- citations: 5. None to repoint. 3937, 3986 and 3988 refer to the superseded flag and are themselves history text in D-078 and D-079.

### D-079: Changing a room requires standing in it, which only a session has
- verdict: one, with non-decision parts to move
- decisions:
  1. `invite` and `revoke` act only from a session that is in the room, with no flag, fallback or pointer. At a terminal they are refused with a pointer to the slash command. Where: "Decision. `invite` and `revoke` require the invoking session to be in the room."
- keeps the number: decision 1.
- move out:
  - reversal / stale: "What stays." (`log`, `guests`, `conflicts` fall back at a terminal). D-080 removed the fallback.
  - history: "Why this took three attempts, which is the part worth keeping." and "Context. … That answered the wrong question", plus the status clause "supersedes the `--room` flag added in D-078 hours earlier".
- citations: 5. None to repoint.

### D-080: There is no current room; a terminal command exists to be tested or to work when the plugin cannot
- verdict: split
- decisions:
  1. There is no machine-level current room: `SetCurrentRoom` and `CurrentRoom` are deleted, and a room-scoped command gets its room only from the invoking session. The one exception is `conflicts`, which takes a room by name because reading a room is not standing. Where: "Decision. `SetCurrentRoom` and `CurrentRoom` are deleted." and "The exception, and why it is one."
  2. A terminal command exists only for one of five reasons: it cannot pass through a model, it must work when the plugin path is broken, the daemon's lifecycle, machine scope, or testing. Every other command gets a slash command, with a stated exclusion list. Where: "The reasons that survive." and "Slash commands now cover every command that can have one." Its own alternatives: other purposes for the terminal, a slash command for `doctor`/`behaviors`, and so on. It could change (D-086 did amend reason 1) without the current room returning.
  3. Slash-command prefixes follow scope: `room-` for one room, `peer-` for relationships that outlast rooms (hence `/room-pair` → `/peer-pair`). Where: "Two prefixes, because there are two scopes (§12)". Its own alternative: keep `/room-pair`. It is independent of 1 and 2, and D-096 (a command prefix names its target) now holds this rule in general form. Fold it into D-096 rather than giving it a new number.
- keeps the number: decision 1. It is the title's first half, and 8 citations (`daemon.go:167`, `main.go:194,995,1020`, `membership_test.go:434,700`, room-choice-findings:42, decisions:3222) mean it, against 7 for decision 2.
- move out:
  - history: the list of commands added ("Added `/room-revoke`… `/room-pair` became `/peer-pair`") and "the machine-level pointer had no remaining user".
  - history / finding: "A test now asserts every command file names a real subcommand" (v0.2.0 shipped a mismatch). The test is a consequence of D-075's decision 2, so cite that.
  - "Context. Asked for the reasons…" is history.
- citations: 15. Seven mean decision 2:
  - cmd/cogmer/pairview.go:25 "it was the only thing available that was not the model (D-080)"
  - docs/decisions.md:4336 "amends D-080, does not alter D-055"
  - docs/decisions.md:4352 "D-080's first reason is amended, not withdrawn."
  - docs/decisions.md:4357 "D-080 could not see that option because discovery was unsettled"
  - docs/decisions.md:4381 "What the terminal keeps (D-080's other four reasons, all intact)"
  - docs/decisions.md:4479 "it was the only thing available that was not the model (D-080)"
  - docs/decisions.md:6652 "D-080 (there is no current room) put its lifecycle at the terminal". The words name decision 1, but the claim is decision 2's reason 3. Both the few words and the number need changing.

### D-081: Untrusted turns are JSON, the policy is stated separately, and the relay is measured
- verdict: split
- decisions:
  1. Room turns in the injected block are JSON-encoded, and the per-injection fence is kept alongside. Where: "JSON, because escaping by hand is escaping by hand." and "The fence stays."
  2. The standing policy for reading room content is stated once at session start through `additionalContext`, in addition to the in-block framing. Where: "So the standing policy is now stated once at session start". Its own alternative: framing only inside the block, where tool-result-like content is discounted. The finding shows the in-block framing alone refused the attack. It could be removed without touching JSON encoding.
  - The title's third clause, "the relay is measured", is a finding, not a decision.
- keeps the number: decision 1. `transcript_test.go:210,265`, `behaviors.go:276`, `daemon.go:462,477` and D-090 all mean JSON encoding.
- move out:
  - finding: "Then measured, because a defence nobody tested is a hope." (the four-run table) and "And the … result about the new policy: it could not be shown to help."
  - finding / support: "What this settles about reaching a person." (the model relay is accurate and silent when irrelevant).
  - open.md / behaviour registry: "What it does not settle." (whether a `hook_success` attachment is treated as a tool result). It says itself that this belongs in the registry.
  - "Not adopted: screening tool output through a classifier." is a Rejected item for decision 2, not a separate decision.
  - history: "Context.", and "Turns were interpolated into markup…".
- citations: 9. Three to repoint to decision 2:
  - plugin/hooks-handlers/session-start.sh:46 "the rule is delivered separately from the data it governs… (D-081)"
  - docs/open.md:184 "the standing policy for room content (D-081) is never given"
  - docs/open.md:198 "D-081 could not show that policy helping". This means the finding, so it could point at the findings document instead.

### D-082: The daemon can reach a person directly; there is no setsid hazard
- verdict: split
- decisions:
  1. The view is reached by the daemon calling `open`, which from a detached process brings the browser forward (B23). Where: "Auto-open works, and takes focus. … This is the mechanism discovery now rests on". Rejected items: terminal writes, and an `.app` bundle.
  2. OS notifications are used only to announce, never for anything that must be known to have landed, because delivery is unobservable (D-014). Where: "Notifications: kept, with a stated limit." Its own alternatives: no notifications, notifications as a confirmed channel, and the `.app` bundle for a clickable identity. It could be dropped or changed without affecting auto-open. CLAUDE.md's "prefer an OS notification from the daemon for ambient awareness" rests on it.
  - The title's second half ("there is no setsid hazard") is a finding.
- keeps the number: decision 1. `browser.go:14`, D-083 and D-086 all mean auto-open.
- move out:
  - finding / history: "An earlier finding said the opposite, and was wrong." (macOS has no `setsid`).
  - finding: "So the setsid branch in `start_daemon_if_needed` is not a hazard." (the `launchctl managername` table; "No code change was made").
  - finding: the measurements in "Terminal writes: possible, rejected." (`ENXIO`, tty via `ps`, no `PIPE_BUF` atomicity). Its conclusion is a Rejected item for 1.
  - "Context." is history.
- citations: 3. None to repoint.

### D-083: The view is opened at a first pairing, not at room creation
- verdict: one, with non-decision parts to move
- decisions:
  1. The view first opens at the first pairing, not at the first `create`/`join`. Where: "So the first open belongs at the first pairing." (there is no Decision. field).
- keeps the number: decision 1.
- move out:
  - history / open state: "Not settled here:" (drive or display, and whether later pairings re-open). D-088 answered both.
  - The status "not yet implemented" is stale state if D-088's pairing view implements it. Check.
- citations: 4. Two refer to the open list, not the decision. Both are themselves history text:
  - docs/decisions.md:4394 "Open questions from D-083 remain open"
  - docs/decisions.md:4521 "Also settled from D-083's open list:"

### D-084: Silent to the person is not silent to the log
- verdict: split
- decisions:
  1. Hook and install output goes to the log by default, `/dev/null` only when you can say what the discarded output would have said, and never to stdout. Where: "The rule adopted: redirect to the log by default…", with "§3.1 requires silence toward the person, not toward the log," and "Never to stdout, in any of this." (`json_safe`) as part of it.
  2. A busy port at daemon start is told apart: if `/healthz` shows the holder is one of ours the daemon exits quietly, and otherwise it writes `daemon-state`, which the session-start hook relays to the model. Where: "A busy port is two different events and now says which." Its own alternative: `log.Fatalf` for both. It is independent of the logging rule.
- keeps the number: decision 1. `install.sh:42` and `session-start.sh:23` both mean the logging rule.
- move out:
  - finding: "Context." and "Four things stacked, and each alone was survivable." / "What broke it was a positive observable" (the setsid detection story). This is the same event as D-082's finding.
  - history: "What changed." (`ct_say`, curl split, `install.log`, `mkdir` made audible).
- citations: 2. None to repoint.

### D-085: A line telling somebody to run `cogmer` is a line that fails
- verdict: one, with non-decision parts to move
- decisions:
  1. Wherever a person is told to type a command, the binary prints its own invocation (`invocation()`, from `os.Executable()` with `~`), and command docs relay it without shortening. Where: "The binary prints its own invocation." and "The command docs now relay rather than restate."
- keeps the number: decision 1.
- move out:
  - finding: "The instructions did not work." (`command not found` for every plugin install).
  - history: "Not done: making pairing work from a slash command." D-088 did it.
- citations: 3. None to repoint.

### D-086: The terminal is not a user experience; the view is the surface
- verdict: split
- decisions:
  1. What a user does, including pairing, belongs in the browser view, and the terminal is an operator surface (diagnostics, daemon lifecycle, machine-scope identity, testing). Where: the title, "D-080's first reason is amended, not withdrawn." and "What the terminal keeps". There is no Decision. field.
  2. Files in `plugin/commands/` are prompts, not documentation, so they carry no roadmap or meta-commentary, and interim states are recorded in the decision log instead. Where: "Interim state, deliberately<!-- writing: quotes the decision log --> not marked in the files." Its own alternative: annotate `peer-pair.md` as provisional. It holds whatever surface pairing uses.
- keeps the number: decision 1. D-087, D-088, D-096's `whoami` note and D-110's references (4983, 5814, 5816, 5832, 5838, 5865, 5869, 5875) mean the view-as-surface direction or its host argument.
- move out:
  - finding / state: "What actually depends on a terminal today: one bit." (`fmt.Scanln`; D-088 removed it).
  - open.md state, now history: "Next, and not yet built:" (D-088 built it).
  - "D-055 is untouched." and "D-043 still holds…" restate other decisions. They become citations in Limits.
  - "But this direction and that constraint point the same way." is support for 1 (the cheapest-preparation argument).
  - history: "Context." (the host-order sentence, which D-110 now holds). Under W-38, D-080's reason 1 should be rewritten, not amended from here.
- citations: 12. Two to repoint to decision 2:
  - docs/what-leaves-findings.md:74 "prompts the model reads, not documentation a person reads - D-086 settled that about `peer-pair.md`"
  - docs/decisions.md:6198 "the files in `plugin/commands/` are prompts rather than documentation (D-086)"
  - Also check docs/decisions.md:5816. Its words ("One sentence in D-086 … carries it as context") refer to the host-order context sentence that moves out. It can stay as history inside D-110 or be dropped.

### D-087: The local API requires a header a web page cannot send
- verdict: one, with non-decision parts to move
- decisions:
  1. State-changing local routes require `X-Cogmer: 1` and POST, with `Origin` as a second layer and reads left unguarded. Where: "The rule is REQUIRE, not refuse.", "Also fixed: state-changing routes require POST", "Not guarded, deliberately<!-- writing: quotes the decision log -->:". POST and `Origin` are layers of the same guard, not separate choices.
- keeps the number: decision 1.
- move out:
  - finding: "It was exploitable, and was demonstrated rather than argued." (the cross-port POST marked a peer verified; `text/plain` simple request).
  - finding: the measurements inside "The rule is REQUIRE…" (the `OPTIONS` with `Access-Control-Request-Headers`, no POST followed), "Rejected: `Referer`…" (the meta-referrer measurement) and "Rejected: a redirect…" (307 measurement). The rejections stay; the measurements go to a finding.
  - history: "Context." ("no handler … checked `r.Method` or `Origin` - not one").
- citations: 8. None to repoint.

### D-088: The pairing ceremony lives in the view, and every pairing gets its own URL
- verdict: split
- decisions:
  1. The two-word ceremony is shown and confirmed in the browser view, with the terminal path kept for `--terminal` and for machines that cannot open a browser. Where: the title, "D-055 is unchanged, and that is …." and "The terminal path is kept, and is not legacy." There is no Decision. field.
  2. Every pairing has its own URL carrying a 128-bit id that is the capability. The page names a pairing (`pairId`), never a peer, and the daemon resolves it, so an expired link fails as a link. Where: "Every pairing gets its own URL" and "The page names a pairing, never a peer." Its own alternative: one fixed pairing address whose contents are rewritten. It could change without moving the ceremony out of the view.
  3. The 90-second window starts when the person presses Start, not when the page loads, and a timeout returns to the ready state. Where: "The 90-second window starts when the person is ready, not when the tab loads." Its own alternative: start on load, which spends the deadline on getting onto a call. It is independent of 1 and 2.
- keeps the number: decision 1. CLAUDE.md:236, `pairview.go:20`, `main.go:1501`, D-092 and D-102 (5630) mean the ceremony in the view.
- move out:
  - finding: "Measured, and it refined the premise." (`open` opens a new tab every time; 2→3→4→5).
  - finding: "Verified end to end".
  - history: "The first build ran the exchange on load", and "Also settled from D-083's open list:", which answers D-083's open questions. Rewrite D-083 instead (W-38).
  - "Context." is history.
- citations: 7. Two to repoint to decision 2:
  - cmd/cogmer/daemon.go:86 "Pairings awaiting their ceremony, one per opened page (D-088)"
  - cmd/cogmer/verify.go:347 "PairID lets the view name a pairing rather than a peer… the daemon decides what that link means (D-088)"

### D-089: The reachability warning asked the wrong question, and fired always
- verdict: one, with non-decision parts to move
- decisions:
  1. The warning that a pairing string cannot be reached lives in `printPairingInvitation`, so it goes wherever a pairing string is printed. It asks about `AdvertisedEndpoint()`, not the bind address, and names which of two causes applies (`endpointRecorded()`). Where: "The warning tested the bind address, not the advertised one.", "It was attached to a command rather than to the string." and "The two causes need different answers, and got one." There is no Decision. field. The entry is written as a defect report, and the title is not a decision (W-31).
- keeps the number: decision 1.
- move out:
  - finding / history: the always-on warning ("Observed directly: `whoami` printing a `tc://` string…").
  - history / finding: "A fallback that needed the thing that had just failed." (the `beginCeremony` daemon check) and "Found by a test, immediately:" (`ParseEndpoint` strictness). These are defect fixes.
  - reversal: "Not taken: advertising the machine's LAN address" together with the blockquote "The conclusion holds; the reason is restated in D-091.". D-091 holds this now. Under W-38, remove it and record the dropped reason as a finding.
  - "Have we done everything to avoid a loopback endpoint?" is support or state.
- citations: 6. Two mean the finding (always-on warning), not the decision:
  - docs/decisions.md:4891 "Same failure as D-089: a warning that is always on trains somebody to ignore the one that matters"
  - docs/decisions.md:4940 "which is the always-on warning of D-089 in another costume"
  - The others: `main.go:1507` and 4930 ("D-089 consolidated them") mean decision 1. 4652 and 4737 are D-091's supersession of the LAN part.

### D-090: Attribution is structured; the derived name is a field, not part of a string
- verdict: split
- decisions:
  1. In the injected block, the derived name, the verified state and the turn kind are separate fields, and nothing of ours is concatenated with a peer's free text. Where: "So the derived name became a field".
  2. The self-asserted display name has no uniqueness constraint, because two colleagues may share a name. Where: "The self-asserted display name has no constraint, correctly". Its own alternative: refuse or disambiguate a colliding display name, which is the question the Context asks. It could be reversed without undoing the field structure. The derived-name and local-label paragraphs only restate D-050 and D-074 and decide nothing here.
- keeps the number: decision 1. `transcript_test.go:98`, `sas_test.go:339`, `daemon.go:396`, CLAUDE.md:237 and D-099 all mean structured attribution.
- move out:
  - finding: "The hole was structural rather than a collision." (`Alice (quiet-otter) (prudent-wagtail, unverified)`; the escaping test passed).
  - finding / history: "That already happened: the first two-peer run had both daemons assert the same display name".
  - open.md: "Still open, and not fixed here:" (the view's visual weight). This is already at open.md:219.
  - "The derived name … can collide and that is accepted" and "The local label…" restate D-050 and D-074.
- citations: 6. One refers to the open-question part:
  - docs/open.md:219 "D-090 left this open: the display name and the derived name are separate elements but carry similar weight". It can stay, since it is history of where the item came from, or be dropped.
  - Nothing cites decision 2.

### D-091: Identity is advertised; location is discovered
- verdict: split
- decisions:
  1. Only identity-shaped addresses (`tc://…`) are advertised and stored durably. A location (IP and port) enters only as a discovered candidate with an expiry, and private ranges never appear in a pairing string, invitation or durable record. Where: "Addresses are of two kinds, and only one of them can be advertised." and "A private address is … stale…". The overlay rendezvous paragraph is a limit of this.
  2. The receiver orders candidates: a remembered winner first (timestamped, and dropped on failure), then by derived class (same-host, same-network, public, relayed), tried concurrently. There is no configurable sort, and real requirements are filters. Where: "A remembered winner is a snapshot too." and "On ordering, asked directly: no, a daemon should not have a configurable sort." Its own alternatives: a configurable sort, a sender-expressed order, or a remembered winner kept as a record. It could change without changing what may be advertised.
- keeps the number: decision 1. CLAUDE.md:241, D-089's blockquote, D-092 and D-098 (5154, 5183) all mean advertise-identity / discover-location.
- move out:
  - finding: "An overlay address is not purely an identity…" (the CBOR decode: three keys plus a home relay; the relay mesh forwarding).
  - history: "This entry first argued that the cost was disclosure… TLS removed it (D-101)", and "an earlier form of this entry overturned it" in "D-089 rejected advertising…".
  - open.md state: "None of this is implemented."
  - The status "Supersedes part of D-089" is a partial-supersession note. Under W-38, rewrite D-089 instead.
  - "The sender cannot know which address works…" is support (ICE, happy-eyeballs).
- citations: 5. None to repoint.

### D-092: An address is learned once, out of band, and only one side needs one
- verdict: one, with non-decision parts to move
- decisions:
  1. Pairing proceeds when the pairing string has no address, because only one side needs to dial: `pair` says which side must dial and starts anyway. Where: "Only one side needs a usable address." and "And `pair` refused to allow it." ("It now says which of them has to dial and starts anyway"). The title's first half ("learned once, out of band") describes the code (`parsePairing`, `SetPeerEndpoint`, `RemoteAddr` never read). It is a finding about current state, not a choice.
- keeps the number: decision 1.
- move out:
  - finding: "When: once, at `pair` time. How: carried by a person." (current-state description).
  - history: "Seven more instruction sites were still naming a bare `cogmer` (D-085)". This applies D-085 and D-088, and the operator-surface exemption belongs in D-085.
  - history: the old "there is nowhere to reach them yet" behaviour.
- citations: 1. None to repoint.

### D-093: A two-interaction flow must choose what an unfinished second one means
- verdict: split
- decisions:
  1. A label is required at pairing. It is taken with the pairing string, held in the pending pairing, and written only when the words match. `NameFree` checks for a clash before the ceremony, and both surfaces run through one pairing record. Where: "Collecting is not asserting. … The name is now taken with the pairing string, held in the pending pairing, and written only on a match." "The name clash is checked before the ceremony", "Both surfaces now run through one pairing record" and "How somebody learns the requirement" are part of this. The title's general principle is the reasoning, and its Revisit when gives the method.
  2. A pairing's three endings leave different state: abandoned leaves a resumable unverified row, matched writes the chosen label, and differed removes only what this pairing created, never an existing relationship. Where: "The three endings are now distinct, and were not before:" (the table) and "A mismatch removes only what the pairing created." Its own alternatives: leave the mismatched key recorded under the colleague's name (as abandonment did), or discard an existing relationship on a mismatch (D-073). It could change without changing when the label is collected.
- keeps the number: decision 1. D-097 (5007, 5036), `pairview.go:54`, `membership.go:291`, `main.go:1122,1130,1263,1319` all mean the label taken up front and written on a match. The title would change to state that decision.
- move out:
  - history: "Optional did not mean unlabelled." (`Allow` filled an empty name) and "Asking after the words matched was worse, and was the first plan." The latter's argument is a Rejected item for 1.
  - finding: "A hazard found while moving the write:" (`SetPeerEndpoint` `UPDATE` affecting no row).
  - "Context." is history.
- citations: 12. Three to repoint to decision 2:
  - cmd/cogmer/pairview_test.go:212 "these three endings are genuinely different. Abandoning is not mismatching… (D-093)"
  - cmd/cogmer/verify.go:421 "the three endings here are genuinely different (D-093)"
  - cmd/cogmer/offer_test.go:78 "which is the state an abandoned pairing leaves behind (D-093)"
  - CLAUDE.md:236 (pairing reading list) could cite both.

### D-094: The name you chose leads the view; the unverified marker becomes a fact
- verdict: split
- decisions:
  1. The view leads with the label you chose at pairing, with the derived name beside it, and a label equal to the derived name counts as absent. Lives at "The label now leads, with the derived name kept beside it".
  2. The view carries whether a turn's peer is verified as a field, and shows the unverified marker only when that field says so, where it had been on every remote turn. Lives at "And the unverified marker was on every remote turn.". Its own alternative: the constant marker on every remote turn, rejected as the always-on warning of D-089 ("a warning that is always on trains somebody to ignore the one that matters"). Reversible alone: the view could drop the marker entirely, or keep it constant, and the label would still lead.
- keeps the number: 1. Every code citation (ui.go, ui_test.go, identity.go, membership.go, main.go) is about the label, and the title leads with it.
- move out:
  - history: the Context. table and "`known_peers.name` was read only by the CLI `peers` listing" (what each name reached before the change); belongs in a finding.
  - history: "a comment written before D-054 explained that every remote peer was unverified" and "the marker was permanently on".
  - history (done since): "Deferred: carrying the label into the injected block." D-099 (the injected block carries the label) decided it.
  - history (done since): "Not done, and blocked: defaulting the label to the name the peer chose.", including the `$USER` / `Ec2-user` account and the first two-peer run. D-095 (chosen or guessed) and D-097 (the string carries a chosen name) decided it. The two-peer-run event belongs in a finding.
  - the Revisit when names a condition D-095 and D-097 already met, so it is history too.
- citations: 12. Repoint to decision 2: none. Refer to a moved part (the deferred or blocked halves): decisions.md:4918 ("D-094 stopped short of defaulting a peer's label"), decisions.md:4960 ("D-094's revisit condition"), decisions.md:5005 ("Completes D-094's revisit"), decisions.md:5008 ("D-094 wanted the default to be the name the peer picked"), decisions.md:5110 ("Completes D-094"), decisions.md:5112 ("D-094 put the label in the view and deferred the same field"). All six sit in Context paragraphs or status lines of D-095, D-097 and D-099 that are history themselves.

### D-095: A name is chosen or guessed, and the difference is recorded
- verdict: split
- decisions:
  1. Whether the display name was chosen is recorded as a flag (`NameChosen`), false at creation and set by any deliberate set, never inferred by comparing the name with the guess. Lives at "Chosen is recorded, not inferred.".
  2. A person is offered the choice of name where their identity first travels, in `printPairingInvitation`, and `whoami` shows the name plainly with a provenance line only while it is guessed. Lives at "So the moment is the first time it travels" and "`whoami` does both, not one or the other.". Its own alternatives: offering it "with a command", or at identity creation ("There is no earlier candidate"), and for `whoami` showing only one of the two. Reversible alone: the offer could move to a command while the flag stays.
  3. The command that sets your name is `/self-name`, under `self-` for commands about you. Lives at "Prefixes name the scope, and `peer-` is for other people.". Its own alternative: `/peer-name`. The prefix rule stated here, including the "names the activity" reading, is the rule D-096 holds, and D-096 corrected the activity reading; so the rule moves to D-096 and only the choice of `/self-name` is a decision here, if it is kept at all (it is one application of D-096 and could live there).
- keeps the number: 1. keys_test.go:234, identity.go:39, decisions.md:5009 and decisions.md:5024 all mean the recorded flag, and the title states it.
- move out:
  - reversal (partial, W-38): "A prefix names the activity, so `/peer-pair` with no arguments printing your own string is not a violation". D-096 corrects it; the reason goes in a finding.
  - support that belongs elsewhere: "Renaming is safe, and by construction rather than luck." is support for decision 1 or 2, not a decision; keep it under Support..
  - history: "Now unblocked: the pairing string can carry a chosen name" (an event, and D-097 did it); the Context. paragraph ("D-094 stopped short ... Asked what the natural point is").
  - finding: "which is exactly<!-- writing: quotes the decision log --> why `Ec2-user` could travel for weeks" (an observed event).
- citations: 11. Repoint:
  - CLAUDE.md:149 "`self-` you (D-095, D-096)": the prefix rule, which is D-096's. Drop D-095 there.
  - main.go:1590 "a command that sets your own name has no business in that namespace (D-095)": the prefix, decision 3 (or D-096).
  - main.go:1578 "The offer appears exactly<!-- writing: quotes the decision log --> while it is true and stops at the first deliberate act (D-095)": decision 2.
  - main.go:1126 "A GUESSED name is never carried ... (D-095)": this is D-097 (a pairing string carries a chosen name, never a guessed one), not D-095 at all. A wrong citation regardless of the split.
  - decisions.md:4969 "D-095 wrote the prefix rule down as naming the activity": the moved prefix reading; it is D-096's Context, which is history itself.
  - CLAUDE.md:243 (table row "the name, a slash command, the plugin manifest"): general, holds whichever entries result.

### D-096: A command prefix names its target; `/self-status` is where you are
- verdict: split
- decisions:
  1. A command prefix names the command's target: `peer-` somebody else, `room-` a room, `self-` you. Your own identity and pairing string are shown by `/self-status`, and no `peer-` command prints them. Lives at the title and "`/self-status` now owns it". (`/self-status` is kept with the rule: the only alternative the entry weighs for it, `/peer-pair` printing your string, is the violation of the rule.)
  2. There is no `/help` and no overview command, and no placeholder name is invented for one. Lives at "No `/help`, and not for want of noticing.". Its own alternative: a placeholder command name ("Do not invent a placeholder"). Reversible alone: an overview command could be added without touching the prefix rule, and D-118 (the manifest name is the command namespace) has already changed what this part rests on.
- keeps the number: 1. CLAUDE.md:149, main.go:650, main.go:1202, decisions.md:6370 and decisions.md:6450 all mean the prefix rule.
- move out:
  - history: "That was a rule bent to fit an exception, and the exception was the defect." and the Context. paragraph (what D-095 wrote and who pointed it out); "It replaces nothing, because until now there was no slash command for your own identity at all"; "`printPairingInvitation` has one caller again". The reason D-095's activity reading was wrong goes in a finding.
  - history (overtaken): "which is what Phase 13 is blocked on" and "the name gets one chance" (D-117 settled the name); "the slash menu already lists all twelve commands" is a count that went stale.
- citations: 10. Repoint to decision 2 (the overview command):
  - decisions.md:6337 "Also D-096's overview command"
  - decisions.md:6338 "not in the form D-096 planned for it"
  - decisions.md:6383 "D-096's overview command cannot be what D-096 reserved"
  - decisions.md:6386 "D-096's stated blocker is gone - Claude Code owns `/help`"
  (4 citations; decisions.md:6383 counts once though it names D-096 twice.) CLAUDE.md:243 is the general table row.

### D-097: A pairing string carries a chosen name, never a guessed one
- verdict: one, with non-decision parts to move
- decisions:
  1. A pairing string may end in `#<name>`, carrying the sender's chosen name and never a guessed one, which the receiver uses as the label's default after what they typed and before asking, saying when it does. Lives at "The format gains an optional trailing name:" through "The receiver's order is". The `sanitizeName` reduction and "Do not move that write earlier." are conditions of the same decision (the name is untrusted input written after the two words match), not separate choices.
- move out:
  - history: the Context. paragraph ("D-093 made a label compulsory ... This connects them"); "anything issued before today still parses".
- citations: 1 (sas_test.go:449, the guessed name). Also note main.go:1126 cites D-095 for this entry's rule; it should cite D-097.

### D-098: A decision that changes what the system is updates the specification in the same pass
- verdict: split
- decisions:
  1. A decision that changes what the system is updates the specification in the same pass. Lives at "The division of labour was right and incompletely applied.".
  2. The specification states only what the system is: a correction is a clean replacement, with no superseded text and no note of what the old text said. Lives at "The specification says what the system is, and nothing about what it was.". Its own alternative: annotating each correction with the reasoning it replaced ("this specification previously concluded that they did"), rejected in the same paragraph. Reversible alone: the spec could carry annotated corrections while still being updated in the same pass.
- keeps the number: 1. CLAUDE.md:58 cites D-098 for "updates the spec in the same pass".
- move out:
  - finding: the Context. ("nine decisions recorded in one day had produced no specification edits") and "The worst of it was stated as a requirement" (§29, §31 contradicting the code).
  - finding: "This is the same failure as review finding A1".
  - history: "What the pass changed" and its six-item list; "The first version of this pass annotated each correction" (the event behind decision 2).
  - the status "(spec pass done)" and "Revisit when: never" are not template fields; the second is a limit.
- citations: 2. Repoint: none. decisions.md:5151 ("Generalises D-098") means the document division, which is decision 1's ground. CLAUDE.md's "Corrections are clean replacements: no superseded text" carries no citation, but is decision 2 and could cite the new entry.

### D-099: The injected block carries the label, and says it is the name to use
- verdict: one, with non-decision parts to move
- decisions:
  1. The injected block carries the label beside `speaker` and `peerName`, omitted when absent, and its framing says the label is the name to prefer. The `PeerFacts` interface is how the lookup reaches `FormatTeamContext`; it is the mechanism of this decision, not a second one.
- move out:
  - history: the Context. paragraph (what D-094 deferred, twenty-two call sites); "The gap was an inconsistency, not an omission." (what the view and model said before); "Six callers passed a function and now pass the membership".
  - finding: "The last check initially scanned the whole block and matched the framing's own use of the word, which is a reminder that a test over rendered text should read the payload."
- citations: 5.

### D-100: Each document answers one question, and a finding is not a commitment
- verdict: split
- decisions:
  1. Each document answers one question: a finding is evidence and stays out of the specification, which states the requirement it justifies; a finding about someone else's software goes in the behaviour registry; current-state defects go in none of them. Lives at "A finding is evidence; a specification statement is a commitment.", "A finding about someone else's software belongs in the behaviour registry." and "Current-state defects belong in none of them.".
  2. An always-loaded file (`CLAUDE.md`) holds only what no check covers, and points at a behaviour rather than restating it where `doctor` checks it. Lives at "`CLAUDE.md` is not an exception, though it was first written as one." and "The rule that actually holds is sharper: what earns a place in an always-loaded file is what no check covers.". Its own alternative: exempting `CLAUDE.md` as the place where a duplicate "is a warning where warnings work", rejected. Reversible alone: CLAUDE.md could hold recitals again while every other document keeps its one question.
- keeps the number: 1, the title's decision. No citation exists to decide it.
- move out:
  - history: "This was caught in the draft of D-091, which listed what the code gets wrong today"; "though it was first written as one"; "Applying it removed fifteen lines ... The MCP section survived for the same reason" (what the pass did to CLAUDE.md).
  - finding: the compaction example ("was true of one Claude Code version under one test").
  - support that belongs in a finding or support list: "Review finding C2 is valid and is not asking for this." and the C1 remark.
- citations: 0.

### D-101: Peer connections are TLS pinned to the key already verified
- verdict: split
- decisions:
  1. Every peer connection, over every transport including the overlay, is TLS with a required client certificate, each side accepting only a key this machine has recorded: the named peer's key when verifying, any recorded peer's key when syncing. Lives at the title, "### What TLS changes", "### What \"pinned\" means", "### The two situations a dial can be in" and "### Why the URL scheme is …". "The test is that the key is recorded, not that it is verified" is part of this decision (its rejected alternative, pinning to verified keys, deadlocks).
  2. No setting disables encryption between peers. Lives at the end of "### Why this superseded the interim guard": "There is now no setting that disables encryption, which is …: a security property with an off switch is one somebody eventually switches off." Its own alternative: an escape hatch or environment variable that turns the check off. Reversible alone: an off switch could be added for testing and TLS pinning would stand. This is the weakest split in the range; see the hard cases.
- keeps the number: 1. Every citation means TLS pinning.
- move out:
  - history: "### What was wrong" (plain HTTP before); "### Why this superseded the interim guard" except its last sentence; "A peer built before this cannot connect at all" in "### A note for whenever a second person runs this"; the Context. ("led to noticing that `peerURL` was `http://`").
  - the "### ..." subheadings are not template fields; "### Considered and rejected" is Rejected., "### What this does not do" is Limits..
  - "Note that being recorded is per machine ... The attempt retries" is support, not a decision.
- citations: 14. Repoint: none. (what-leaves-findings.md:27 and :31, decisions.md:4708 and the code comments all mean pinned TLS.)

### D-102: The reveal step tolerates a peer that has already finished
- verdict: one, with non-decision parts to move
- decisions:
  1. A reveal that finds the other side no longer expecting a verification goes back to the loop, as a commit does, so the inbound check finds the nonce the finished side already sent.
- move out:
  - history and finding: the Context. (TLS changed the timing; "failed about one run in six") and "The two halves of the exchange were not treated alike." (what the code did); the observed failure goes in a finding.
  - finding (a lesson, not a decision): "A test that passes is not a test that holds."
- citations: 1.

### D-103: An address belongs to a peer, and is stored in one place
- verdict: one, with non-decision parts to move
- decisions:
  1. A peer's address is stored once, in `known_peers`, and a room's sync targets are its guests other than self joined to that table, with self excluded explicitly. Lives at "The replacement is a join that already has both halves." and "One wrinkle, deliberately<!-- writing: quotes the decision log --> made explicit." (the explicit self-exclusion is how the join is defined, not a separate choice).
- move out:
  - finding: "Addresses live in two tables." and "Both writers hold the identity and discard it." (the schema as it was); "Five problems, one cause." (support whose facts are about the old schema); "Checked: no room holds an address for a non-guest." (a check performed).
  - history: "This removes work rather than adding it." (what became unnecessary); "What the §4 lifecycle assumed."; the Context.
- citations: 13.

### D-104: The overlay address is public and stable; admission moves to a list
- verdict: split
- decisions:
  1. The overlay runs with no pre-shared key, so its published address holds no secret and is the same at every start. Lives at "The address changes on every restart, and the cause was measured." through "The two cannot both hold at that layer." and "The hedge is genuinely lost, and is recoverable elsewhere.".
  2. The overlay admits only recorded peers, through the library's allow-list of client node keys, rebuilt from the peer list and updated without a restart. Lives at "So the access-control half moves to a list, where it is better.". Its own alternative: the pre-shared key as the admission mechanism (the comparison table), or no tunnel-level admission at all with the TLS pin as the only gate. Reversible alone: the allow-list could be dropped and the address would stay public and stable.
  3. The overlay's relay region is chosen once and persisted with the node key. Lives at "Also pin the region.". Its own alternative: choosing a region by latency at every start (the library default), which `open.md:38` reopens as "re-pick when it cannot be reached". Reversible alone: yes, and `open.md` already proposes a partial reversal.
- keeps the number: 1. The citations divide three, four and three (see below), so no decision has a clear majority; 1 is the title's first statement and the one the other two follow from. If the rule is applied by count alone, 2 has four.
- move out:
  - finding: the restart measurement and its table ("Restarting a daemon twice and comparing the published endpoints"); "Measured: a peer absent from the list cannot open a tunnel"; "A measured cost: the list refuses by silence." (the measurement; the consequence for D-103's ordering is support).
  - history: the Context. (the coffee-shop question; "found the address is not stable at all"); "Today an unrecorded peer is refused by TLS in milliseconds".
- citations: 11. Repoint:
  - to 2 (allow-list): pairview.go:167 "rather than being refused by silence (D-104)"; daemon.go:94 "let through the tunnel without a restart (D-104)"; main.go:392 "A peer not on it cannot open a tunnel at all ... needs no secret in the published address (D-104)" (names both 1 and 2); decisions.md:5436 "a tunnel-level allow-list cannot be built from them (D-104)".
  - to 3 (region): tailcat.go:237 "a changed address strands everybody holding the old one (D-104)" (the region comment); open.md:38 "D-104 pins it so the address is stable"; what-leaves-findings.md:31 "D-104 pins a region".
  - stay with 1: tailcat.go:187, open.md:210, decisions.md:5556. CLAUDE.md:241 is the general table row.
  (7 to repoint.)

### D-105: An invitation travels over the channel pairing already established
- verdict: one, with non-decision parts to move
- decisions:
  1. The host's daemon delivers an invitation to a paired guest as an offer over the paired channel, and hands the host the string to send only when it cannot reach the guest. Lives at "An invitation becomes an offer delivered over the paired channel". Guest polling is its rejected alternative.
- move out:
  - history: the Context. (the exchange about one-way reachability); "§12 already describes the intended shape, and the implementation is what diverged" (support in part, history in part); "Today the string is the only path".
  - history / open: "This mostly dissolves Phase 14." is a consequence for the plan, which belongs in `open.md`.
- citations: 10.

### D-106: An offer is delivered when it can work, not when it is made
- verdict: split
- decisions:
  1. Inviting an unverified peer records the admission at once and withholds the offer and the fallback string until the two have verified, and completing verification delivers what was waiting; the invite reads as admitted and queued, never as a refusal. Lives at "So the admission is recorded and the offer is withheld.", "Why the gate stays open at all", "Verification gains an effect", "The fallback string is withheld on the same terms." and "Which decides the wording".
  2. An offer waiting to be accepted is listed with rooms, apart from joined ones, and a withheld invitation is shown beside the peer who can release it, naming the colleague and what to type rather than a count. Lives at "Where the two waiting things surface" and "Name the person, do not count the rooms.". Its own alternative: a tally of withheld rooms ("A tally was tried first and is close to useless here"). Reversible alone: the pending state could surface anywhere, or as a count, without changing when offers are delivered.
  3. `/peer-pair <name>` resolves a recorded peer and resumes its pairing ceremony without asking for a label. Lives at "The appeal is to finish pairing, not to verify." and "That required the command to accept a name." Its own alternative: `/peer-pair` taking only a pairing string. Reversible alone: the prompt could say "verify" and offer another command. D-107 (pairing again does nothing, loudly) builds on it: "the reason the command accepts a name at all".
- keeps the number: 1. Seven of ten citations mean withholding and delivery.
- move out:
  - history: "What today's friction was hiding." (what hosts did before push); "It previously took only a pairing string, so `/peer-pair alice` hashed the literal text into an identifier and invented a peer" (a defect that goes in a finding); "A tally was tried first".
  - support: "Self is a guest of its own rooms and is never in its own peer list (D-103)" is a fact the decision needs, not a decision.
- citations: 10. Repoint to 2: offer_test.go:140 "The count is what lets /cogmer:peer-list say what verifying would release (D-106)"; membership.go:905 "WithheldFor counts the rooms a peer has been admitted to but cannot be told about (D-106)"; main.go:671 "Named, not counted ... which colleague it is and what to type (D-106)". None cite decision 3 by D-106. (3 to repoint.)

### D-107: Pairing again with somebody already paired does nothing, loudly
- verdict: one, with non-decision parts to move
- decisions:
  1. `/peer-pair` for a peer already paired says so, says since when, and stops, before asking for any name; re-verification needs `--again`, which the message offers only for the case that needs it.
- move out:
  - reversal (partial, W-38): "A string for somebody already paired changes nothing, including the address. This entry first took the address out of it ... That was wrong twice over, and is superseded by D-108." Keep "changes nothing" if wanted, drop the narration; the reason is already D-108's, and goes in a finding.
  - history: the Context. ("running it on a finished one silently began the whole ceremony again").
- citations: 5. (open.md:295 and decisions.md:6081 mean the entry as a whole; decisions.md:5722, 5724 and 5749 are D-108's Context and history.)

### D-108: Pairing is not how an address is updated
- verdict: one, with non-decision parts to move
- decisions:
  1. Neither pairing nor any dedicated command updates a peer's stored address; it is repaired by the peer's next synchronization. Lives at "It is the wrong shape." and "An address matters only when it is used, and every use repairs itself.". The "no address-update command" half is kept here: it rests on the same support and the entry weighs it as the general form of the same point (see the hard cases).
- move out:
  - history: the Context. (what D-107 did, "Rejected on sight"); "What remains true from D-107: ... Only the address write is withdrawn."; the status "Supersedes part of D-107".
- citations: 3.

### D-109: Two is the target and nothing rules out more
- verdict: one, with non-decision parts to move
- decisions:
  1. Build for two people and spend nothing on a third, but treat a design that cannot extend past two as a defect to be argued for.
- move out:
  - history: the Context. ("has been applied consistently and never written down", D-032's aside).
- citations: 1.

### D-110: Host order: Claude Code, CoWork soon after, ChatGPT Desktop much later
- verdict: one, with non-decision parts to move
- decisions:
  1. The hosts are Claude Code, then Claude CoWork after a short interval, then ChatGPT Desktop after a long one, and the ordering licenses no adapter machinery.
- move out:
  - history / open item since resolved: "A consequence for the specification, recorded and not resolved." D-111 (§3.8 is a test about hosts) resolved it.
  - history: the Context. ("recorded nowhere ... appears nowhere in the repository").
- citations: 6. (decisions.md:5884 and 5886 cite the resolved paragraph, from D-111's own status and Context.)

### D-111: §3.8 is a test about hosts, not about Claude Code
- verdict: one, with non-decision parts to move
- decisions:
  1. §3.8 reads "The host is launched and used unchanged", defines the host, and states every clause about the host; a host with no way in is out of reach.
- move out:
  - reversal (partial, W-38): "*First, "things the host already loads" was ambiguous*" is a second decision (the install clause means "something the host would load anyway"), and D-113 (the install test names a documented type) replaced it and rejects this phrasing by name. Remove it from D-111 and record the reason in a finding.
  - history: "The test itself did not change."; "The original said "already loads on its own", and dropping it in summary produced a phrase that could not be read"; the Context.
  - support: "The rule generalises; the evidence for it does not."
- citations: 4. Refer to the removed part: decisions.md:6006 "D-111 stated the install clause as "something the host would load anyway"" and decisions.md:6049 ""Something the host would load anyway" (D-111's phrasing)". Both are in D-113, and would point at the finding.

### D-112: Several sessions from one machine are not told apart in the block
- verdict: one
- decisions:
  1. The injected block does not mark which session of one machine a turn came from.
- citations: 1. (Its status "considered and not built" is not a W-30 status, and its Context. holds history; minor.)

### D-113: The install test names a type, and the type must be documented
- verdict: one
- decisions:
  1. The project installs only artifacts of a type among the host's documented and demonstrated extension mechanisms.
- citations: 2. (Its Context. is history about D-111's phrasing.)

### D-114: §3.1 is a duty not to harm the session, not a claim about data locality
- verdict: one, with non-decision parts to move
- decisions:
  1. §3.1 is "First, do no harm", stated as a general duty with known cases rather than a list.
- move out:
  - finding: "Twelve citations of §3.1 across the repository are about hooks (6), silence toward the person (4)" and "the section did not contain what they cite" (a count measured at a date).
  - history: "§3.1 was titled "Local first" and said two things"; "(now including a dead daemon and a failed hook)".
- citations: 0.

### D-115: A centralized component must trace to a disclosed tradeoff that benefits the person
- verdict: one, with non-decision parts to move
- decisions:
  1. A compromise to privacy or decentralization is acceptable when it traces to a recorded decision, is disclosed to the person it concerns, and benefits them; failing a test names what is missing rather than forbidding it.
- move out:
  - finding / current state: "Applied to what exists" (the relay, the model provider, the release host, each scored); it is an assessment at a date and belongs in `what-leaves-findings.md` or `open.md`.
  - history: the release-host item and "It exists because there is nowhere else to publish from ... retired when a public release host exists"; D-121 (the repository is the release host) retired it, so this now describes a component that no longer exists.
  - current state: "The disclosure test cannot currently be satisfied by anything." belongs in `open.md`.
  - history: the Context. ("three ... already exist with nothing governing them").
- citations: 6. Refer to a moved part: what-leaves-findings.md:32 ("A pre-GA tag, and nothing else") and what-leaves-findings.md:59 ("It is tagged pre-GA rather than defended (D-115)"), both about the release-host application, which D-121 made stale; decisions.md:6552 "D-115 counts three centralized components" (the tally). No citation needs a different decision.

### D-116: Verification dials the peer it is verifying, and nobody else
- verdict: one, with non-decision parts to move
- decisions:
  1. `verifyTargets(peerID)` returns the address recorded for that peer plus `COGMER_PEERS`, with no fallback sweep.
- move out:
  - history: the Context. ("`handleVerifyStart` passed `d.syncTargets()`"); "The sweep spent a dial and a timeout per peer"; "the sweep was what made the sentence false".
- citations: 0.

### D-117: The product is named `cogmer`, and nothing is carried forward
- verdict: split
- decisions:
  1. The name `cogmer` reaches the module path (with the organisation's casing), the binary, the command directory, the manifest, the state directory, the API header, the environment prefix and the release path, and nothing cryptographic. Lives at "Decision.", "The organisation casing is fixed at the same time." and "Nothing cryptographic moved".
  2. The rename carries no compatibility: the superseded signing scheme is deleted and `signingBytes` refuses every other version, and no state directory under the former name is migrated. Lives at "There is no compatibility surface, because there is nothing to be compatible with.". Its own alternative: retaining the old scheme and migrating the state directory, which "every instinct here says". Reversible alone: the old scheme could have been kept under the new name.
  3. The former name is struck from the decision log rather than preserved. Lives at "The former name is struck from this log rather than preserved.". Its own alternative: preserving the name where it appeared. Reversible alone: it is a rule about this document, independent of the rename's scope.
- keeps the number: 1. CLAUDE.md:243 and decisions.md:6488 ("Noticed during the rename (D-117)") mean the rename.
- move out:
  - history: "What this unblocks." (Phase 13, D-096's overview command); "the zero value that used to mean "stored before the column existed""; "The only installation that ever existed ... was deleted rather than migrated" (an event, whose fact is support for 2); "The command prefixes did not change" (history; D-118 holds it).
- citations: 3. Repoint to 2: decisions.md:3545 "(D-117 later deleted that scheme, on the ground that no such event existed anywhere". (1 to repoint.)

### D-118: The plugin manifest name is the command namespace, and Claude Code forces it
- verdict: one, with non-decision parts to move
- decisions:
  1. Commands keep their `room-`/`peer-`/`self-` prefixes under the manifest namespace (`/cogmer:room-create`), and a test requires every command reference to carry the manifest's namespace. Lives at "Decision. The prefixes stay." The title states the finding that makes the decision necessary rather than the decision itself (W-31).
- move out:
  - finding (behaviour): "The manifest's `name` is documented as the namespace." and "Confirmed rather than read." (the `nstest` plugin). These are support; the observation goes in a finding.
  - history: the Context. (what D-070 chose and why); "D-070's protection becomes belt and braces"; "The manifest name is now … in a way it was not." (a consequence, fine as support without "now").
  - consequence for another entry: "D-096's overview command cannot be what D-096 reserved." belongs with D-096's overview decision (the new entry from the D-096 split) or in `open.md`.
  - history: "The half of this finding that does belong there ... is in `open.md`" (D-119 decided it since).
- citations: 10. Repoint: none.

### D-119: No slash command is reachable by the model
- verdict: one, with non-decision parts to move
- decisions:
  1. Every file in `plugin/commands/` carries `disable-model-invocation: true`, and a test requires it.
- move out:
  - history / finding: "this was a live violation of the duty ... on every session, from the moment the plugin shipped"; "every command was a model-invocable skill: confirmed with a throwaway plugin" (the confirmation goes in a finding).
  - open: "Not done here. Migrating to the `skills/<name>/SKILL.md` layout" already sits in `open.md`.
- citations: 2.

### D-120: The manifest version is the release version, and `release.sh` writes it
- verdict: one, with non-decision parts to move
- decisions:
  1. A release has one version, which `release.sh` writes into `plugin.json` along with `VERSION` and `checksums.txt`, and a test compares the two. The title's "and" joins what the number is with who writes it, which is one decision.
- move out:
  - history: the Context. (`0.1.0` against `0.7.0`, noticed during the rename); "The manifest is now `0.7.0`"; "So the stale number was a release nobody would receive." (support stated as an event).
- citations: 2.

### D-121: The repository is both the release host and the marketplace
- verdict: split
- decisions:
  1. Releases are served from the repository's GitHub releases, with `release-url.txt` and `publish.sh` pointing there and `publish.sh` checking every asset against `checksums.txt` over the public URL. Lives at "The release host." and "The check.".
  2. The repository carries `.claude-plugin/marketplace.json` beside `plugin/`, so the marketplace and the plugin share one repository name. Lives at "The marketplace.". Its own alternative: "*A separate marketplace repository.*" Reversible alone: the marketplace could move to its own repository while releases stay on GitHub.
- keeps the number: 1. scripts/publish.sh:8 means the release host.
- move out:
  - history: "The address is also removed from the history" (an act performed on the repository); "D-041's count is superseded; nothing else in it is" (a change to another entry, which W-38 says is made in D-041).
  - history: the Context. ("Phase 13 was blocked on one thing").
  - support: "What is not gained is trust - the sha256 in `checksums.txt` is still what authorises a binary, exactly<!-- writing: quotes the decision log --> as it was when the transport was plain HTTP" (drop the history clause).
- citations: 3. Name both decisions and would need both numbers: Shared Claude Sessions.md:2793 "carries the marketplace manifest beside the plugin, and serves the release assets ... (D-121)"; docs/open.md:25 "carries the marketplace manifest beside the plugin, and serves releases ... (D-121)". (2 to repoint, by adding the new number.)

### D-122: The two READMEs are split by whether you have installed it
- verdict: one, with non-decision parts to move
- decisions:
  1. `README.md` serves somebody who has not installed cogmer, and `plugin/README.md` somebody who has; neither repeats the other and current state lives in `open.md`. Leading `README.md` with the §30 result is part of what `README.md` contains, and its alternative is already under Rejected.
- move out:
  - history: the Context. ("Until the repository was public ... a lab notebook"); "The second reason is rot, and it had already happened." (the event goes in a finding; the fact, two documents owning one fact, is support); "*One README serving both.* It is what we had."; "That promise was already made and already broken".
- citations: 1.

### D-123: `stop` finds a daemon by the addresses it holds, and signals only cogmer
- verdict: one, with non-decision parts to move
- decisions:
  1. `cogmer stop` finds the process listening on each of this installation's addresses and stops it only if its executable is named `cogmer`, SIGTERM then SIGKILL, reporting what it stopped and declined. The title's two statements are how one process is found and which one is signalled; the rejected alternatives cover both together.
- move out:
  - finding: the 09-22 incident in the Context. ("a daemon left from a test ... held that port"), and its reuse in "on 09-22 it was not" and the first rejected alternative.
- citations: 4. Not citations: structure_test.go:525 is a test fixture heading ("## D-123 - Old"); writing.md:402 uses the number as the template cutoff ("every decision after D-123"). Real citations: open.md:53, open.md:137.

### D-124: The writing test reads documents as Markdown, through goldmark
- verdict: one
- decisions:
  1. `writing_test.go` parses documents with goldmark and checks only prose.
- citations: 19, of which 17 are test fixtures using the number as a sample heading (writing_test.go:303-305, structure_test.go:227, 526-538). Real citations: structure_test.go:269, writing.md:367.

### D-125: A permitted use of a judgement word carries a marker with its reason
- verdict: one
- decisions:
  1. A permitted use of one of the three judgement words carries `<!-- writing: <reason> -->`, and the test fails every other use.
- citations: 1.

### D-126: Rules that need a reader are reviewed by a skill a maintainer runs
- verdict: one
- decisions:
  1. The rules no test can decide are reviewed by the `writing-review` skill, run by a maintainer, which reports quoted findings, changes nothing and never blocks a commit. (One support item carries a choice of the skill's, reporting a long item by the kind of detail rather than a shorter wording; see the hard cases.)
- citations: 3, of which 2 are test fixtures (structure_test.go:519 and 532 use it as a sample replacement number). Real citation: writing.md:413.

## What this does not show

Each range had one reader, and whether a choice counts as a second decision is
judgement. The readers flagged several counts as possibly one too high, where an
extra decision is an implementation detail, so 87 new entries is an upper bound.
Citation counts include test fixtures that use a number as a string and cite
nothing, and a grep cannot see a citation that names an entry by its title alone.
