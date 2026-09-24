# Entries in the decision log that hold more than one decision

**Run:** 2026-09-23, on `docs/decisions.md` at commit `f11e871`, entries D-001 to
D-126. Every count, line number and quotation below describes `f11e871`, for every
file it names, and later commits move them. Four readers each read about 31 entries
in full and applied W-40 in `docs/writing.md` (an entry records one decision).

**Result:** 59 of the 126 entries hold more than one decision. D-076 (a command inside
a session acts on that session's room) is a tombstone,
an entry reduced to its number, its title and a line naming what replaced it (W-37,
a reversed decision becomes a tombstone). Of the 66 entries that hold one decision,
57 also hold history, findings or other content that is not a decision, and 9 hold
nothing else. Two of the 57, D-028 (losing a room database ends that peer's
membership) and D-047 (the fingerprint is the only manual link), hold a decision that
a later entry reversed, and keep their full text.

## What was run

1. `docs/decisions.md` at `f11e871` was divided into four ranges: D-001 to D-031,
   D-032 to D-062, D-063 to D-093, and D-094 to D-126.
2. Each range was read in full. For every entry, the reader recorded each decision
   it holds and where it sits, the alternative the entry weighs for each decision
   after the first, and each part that is not a decision: history, a finding, a
   reversal by a later entry, or current state.
3. For every entry, citations of its number were found with
   `git grep -n "D-NNN\b" f11e871 -- "*.md" "*.go" "*.sh" "*.json"`, leaving out the
   entry's own lines, and each citation's few words were compared with what the
   entry holds.

A second decision counts only when it has alternatives of its own and could be
reversed without reversing the first. Reasoning, evidence, limits, consequences and
rejected alternatives do not count.

Quotations of the log, including the entry titles in the headings below, are copied
from it with four changes. Its bold is removed. Its em-dashes appear as " - ". A cut
in a quotation is marked "…", including where a word `docs/writing.md` bans is left
out. A word the guide permits only
with a marker carries one saying it quotes the log. Headings name each entry as
"D-NNN: title".

## What we found

### The decisions and citations in each entry

The last column counts the citations of an entry that mean a decision after its
first, a part that is not a decision, or something the entry does not state. Test
fixtures, which use a number as a string and cite nothing, are not counted. A row of
the reading list in `CLAUDE.md` names an entry as a whole; where it could mean either
of two decisions, the row gives both counts.

| entry | decisions | other content | citations meaning something other than the first decision |
|---|---|---|---|
| D-001 | 1 | finding, history | 0 |
| D-002 | 2 | history | 0 |
| D-003 | 1 | finding | 0 |
| D-004 | 1 | finding | 0 |
| D-005 | 1 | none | 0 |
| D-006 | 1 | finding | 0 |
| D-007 | 1 | none | 0 |
| D-008 | 2 | none | 0 |
| D-009 | 1 | none | 0 |
| D-010 | 2 | finding | 0 |
| D-011 | 1 | finding | 0 |
| D-012 | 1 | none | 0 |
| D-013 | 1 | none | 0 |
| D-014 | 2 | history, finding | 4, or 5 with `CLAUDE.md:238` |
| D-015 | 2 | history, reversed in part by D-016 (presence is not membership), pointer | 4 |
| D-016 | 2 | history, reversed in part by D-056 (a session's room is fixed at first sight) | 2, or 3 with `CLAUDE.md:239` |
| D-017 | 4 | history | 7 |
| D-018 | 2 | finding, reversed in part by D-026 (no join token) | 3 |
| D-019 | 3 | history, finding, reversed in part by D-063 (the first pair is remote) and D-026 (no join token) | 9, or 10 with `CLAUDE.md:241` |
| D-020 | 2 | history, reversed in part by D-026 (no join token) | 4 |
| D-021 | 4 | finding, history | 5 |
| D-022 | 1 | history | 0 |
| D-023 | 3 | history, a requirement with no alternative weighed, decided by D-054 (verification gates sync) | 0 |
| D-024 | 2 | history, second decision reversed by D-026 (no join token) | 4 |
| D-025 | 3 | history, reversed in part by D-026 (no join token) | 3 |
| D-026 | 1 | history | 0 |
| D-027 | 1 | history | 0 |
| D-028 | 1 | reversed by D-029 (losing a room database does not end membership), history | 0 |
| D-029 | 1 | history | 0 |
| D-030 | 2 | history | 0 |
| D-031 | 1 | history, finding, pointer | 0 |
| D-032 | 2 | history, current state, restatement of D-109 (two is the target) | 3 |
| D-033 | 2 | history, finding | 0 |
| D-034 | 1 | history, finding | 0 |
| D-035 | 2 | history, open question | 0 |
| D-036 | 2 | finding | 0 |
| D-037 | 2 | history, finding, a request to Anthropic | 0 |
| D-038 | 1 | history | 0 |
| D-039 | 1 | history, finding, open question | 1 |
| D-040 | 2 | history, finding | 1 |
| D-041 | 3 | history, finding, current state, changed in part by D-121 (the repository is the marketplace) | 3 |
| D-042 | 4 | history, finding | 1 |
| D-043 | 4 | history, finding | 9 |
| D-044 | 1 | history, finding | 2 |
| D-045 | 2 | history, finding, restatement of D-024 and D-025 (the guest list), reversed in part by D-053 (pair and invite scopes) and D-055 (one way to verify a peer) | 0 |
| D-046 | 4 | history, finding, reversed in part by D-080 (no current room) and D-056 (room fixed at first sight) | 3, and 2 in part |
| D-047 | 1, no longer standing | reversed by D-055 (one way to verify a peer), finding, history | 8 |
| D-048 | 1 | history, support with no source, reversed in part by D-052 (the SAS is its own act) and D-053 (pair and invite scopes) | 1 |
| D-049 | 1 | history, finding | 0 |
| D-050 | 2 | history, finding | 2, and 1 in part |
| D-051 | 1 | history, open questions | 1 |
| D-052 | 4 | history, limit | 5 |
| D-053 | 3 | history, reversed in part by D-054 (verification gates sync) | 3 |
| D-054 | 1 | history, finding | 0 |
| D-055 | 1 | history, finding | 1 |
| D-056 | 2 | history, finding | 3 |
| D-057 | 3 | history, current state, finding held by B21, restatement of D-080 (no current room), open question | 0 |
| D-058 | 1 | history, restatement of D-043 (the wire format is defined separately from the stored row) | 1 |
| D-059 | 2 | history, finding | 0 |
| D-060 | 2 | history, finding, a separate fix | 0, and 2 in part |
| D-061 | 4 | history, finding | 1 |
| D-062 | 2 | history, finding, restatement of D-019 (no provider required), D-026 (no join token) and D-063 (the first pair is remote), restatement of D-042 (peer identity is an Ed25519 key pair) | 3 |
| D-063 | 1 | history, restated decisions | 6 |
| D-064 | 1 | history | 1 |
| D-065 | 1 | history, current state | 0 |
| D-066 | 2 | history, finding, current state | 3 |
| D-067 | 3 | history, finding, current state, reversal by D-121 (the repository is both the release host and the marketplace) | 0 |
| D-068 | 1 | history, finding, current state | 2 |
| D-069 | 2 | history, finding | 3 |
| D-070 | 1 | finding, history, reasoning replaced by D-118 (the plugin manifest name is the command namespace) | 2 |
| D-071 | 1 | history, current state | 2 |
| D-072 | 1 | history | 0 |
| D-073 | 1 | history | 0 |
| D-074 | 2 | history, finding | 0 |
| D-075 | 2 | history, finding | 1 |
| D-076 | 0 | withdrawn, a tombstone | 0 |
| D-077 | 2 | history, finding, reversal by D-080 (there is no current room) | 4, or 5 with `CLAUDE.md:239` |
| D-078 | 1 | history, reversal by D-079 (changing a room requires standing in it) and D-080 (there is no current room) | 2 |
| D-079 | 1 | history, reversal by D-080 (there is no current room) | 0 |
| D-080 | 3 | history, finding, partial change by D-086 (the terminal is not a user experience) | 7 |
| D-081 | 2 | history, finding, open question | 3 |
| D-082 | 2 | history, finding | 0 |
| D-083 | 1 | open questions answered by D-088 (the pairing ceremony lives in the view), current state | 3 |
| D-084 | 2 | history, finding | 0 |
| D-085 | 1 | history, finding, current state | 0 |
| D-086 | 2 | history, current state, restated decisions | 5 |
| D-087 | 1 | history, finding | 0 |
| D-088 | 3 | history, finding | 2 |
| D-089 | 1 | history, finding, current state, reason replaced by D-091 (identity is advertised, location is discovered) | 4 |
| D-090 | 2 | finding, open question, restated decisions | 1 |
| D-091 | 2 | history, finding, current state | 2 |
| D-092 | 1 | history, finding | 0 |
| D-093 | 2 | history, finding | 3, or 4 with `CLAUDE.md:236` |
| D-094 | 2 | history, finding | 6 |
| D-095 | 3 | partly reversed by D-096 (a command prefix names its target), history, finding | 7, or 8 with `CLAUDE.md:243` |
| D-096 | 2 | history | 4, or 5 with `CLAUDE.md:243` |
| D-097 | 1 | history | 0 |
| D-098 | 2 | finding, history, a limit under Revisit when | 0 |
| D-099 | 1 | history, finding | 1 |
| D-100 | 2 | history, finding | 0 |
| D-101 | 1 | history | 0 |
| D-102 | 1 | history, finding | 0 |
| D-103 | 1 | finding, history | 1 |
| D-104 | 3 | finding, history | 7, or 8 with `CLAUDE.md:241` |
| D-105 | 1 | history, current state | 0 |
| D-106 | 3 | history, finding | 3 |
| D-107 | 1 | partly reversed by D-108 (pairing is not how an address is updated), history | 3 |
| D-108 | 1 | history, partly reverses D-107 (pairing again does nothing, loudly) | 0 |
| D-109 | 1 | history | 0 |
| D-110 | 1 | history | 2 |
| D-111 | 1 | partly reversed by D-113 (the install test names a documented type), history | 3 |
| D-112 | 1 | none | 0 |
| D-113 | 1 | history | 1 |
| D-114 | 1 | finding, history | 0 |
| D-115 | 1 | finding, history, current state | 3 |
| D-116 | 1 | history | 0 |
| D-117 | 3 | history | 1, or 2 with `CLAUDE.md:243` |
| D-118 | 1 | finding, history, a consequence for D-096 (a command prefix names its target) | 2 |
| D-119 | 1 | history, finding, current state | 1 |
| D-120 | 1 | history | 0 |
| D-121 | 2 | history | 2 |
| D-122 | 1 | history, finding | 0 |
| D-123 | 1 | finding | 1 |
| D-124 | 1 | none | 0 |
| D-125 | 1 | none | 0 |
| D-126 | 1 | none | 0 |

### Nine citations that commit 738a6ff corrected

Commit `738a6ff` corrected nine citations that, at `f11e871`, named an entry that
exists but credited it with words or a fact it does not hold. Other citations of that
kind remain at `f11e871`, and the entries below count them in the table's last column,
such as `docs/decisions.md:451` under D-015 (rooms are scoped to sessions) and
`docs/spec-review.md:571` under D-024 (admission is a guest list). For five of them,
another entry holds what the citation described. One gave D-080 (there is no current
room) the few words of its other decision. One credited D-070 (slash commands carry a
distinctive prefix) with a precedent that `plugin/commands/room-status.md` holds.
Two, at `docs/decisions.md` lines 4924 and 5979, credited D-021 (peer names are
derived from the identity) with leaving the derived name off a user's own turns.
D-021 does not say this. The view leaves it off a user's own turns because of commit
`33ce996`, and no decision records that choice. The entries below describe the
citations as they stood at `f11e871`, before that correction.

### D-001: Go, not TypeScript/Node or Python

D-001 holds one decision. It writes the system in Go with `modernc.org/sqlite`,
which is pure Go with no cgo, and it sits in the paragraph "Decision. Go, with
`modernc.org/sqlite`".

The rest of the entry holds two findings. The paragraph "The deciding fact: `claude`
resolves to `~/.local/share/claude/versions/…`, a native Mach-O binary - Claude Code
no longer ships as an npm package" records an observation, and its "no longer" is
history under W-33 (a decision uses none of the history words). The Rejected. bullet
for TypeScript/Node holds the second: "`node:sqlite` in Node 22.12 throws without
`--experimental-sqlite` (verified directly)".

At `f11e871`, 11 lines outside the entry cite D-001. Six are test fixtures, which use
the number as a string: `cmd/cogmer/citations_test.go:153`, `:168` and `:173`,
`cmd/cogmer/structure_test.go:508` and `:527`, and `cmd/cogmer/writing_test.go:295`.
The other five are `cmd/cogmer/ui.go:16`, `docs/decisions.md:256`, `:1450` and
`:6720`, and `docs/phase2-experiment.md:131`, and all five mean the decision.

### D-002: Stop at Phase 0, and insert Phase 0a before Phase 1

D-002 holds two decisions. The first puts a compaction phase before any daemon work,
in the paragraph "Decision. Stop as instructed, then add Phase 0a". Stopping is what
§36.10 (stop after the integration spike) required, so it is not a choice this entry
makes. The second labels the new phase "0a" rather than renumbering the phases. It
sits in the Rejected. bullet that weighs its own alternative: "*Renumber the
phases* - '0a' avoids churn in a specification already referenced by section number".
The
title and the Rejected. bullet "*Proceed to Phase 1 and handle compaction when it
appears*" concern the first decision.

The whole entry records a step in a process that has finished, which is history under
W-32 (a decision describes only the present). The Context paragraph reports it:
"Phase 0 proved capture and injection, but never exercised compaction - `PreCompact`
did not fire". The Decision paragraph opens "Stop as instructed". At `f11e871`, the
phase order is stated by D-032 (re-sequence the phases).

At `f11e871`, one line outside the entry cites D-002. It is a test fixture, which uses
the number as a string: `cmd/cogmer/citations_test.go:169`, "See D-002.". No other
line cites it.

### D-003: Reassemble turns by unioning two incomplete sources

D-003 holds one decision. It reassembles a turn from the union of
`Stop.last_assistant_message` and the transcript, in the paragraph "Decision. Union
both in `ReassembleLastTurn`."

The entry holds two findings. The Decision paragraph goes on "Verified by exact string
match against a real 2,582-char response", and the Rejected. bullet "*Transcript
only* - captured a 110-char preamble of a 2,804-char answer" records a second
measurement.

At `f11e871`, one line outside the entry cites D-003: `docs/decisions.md:121`, in the
Context of D-005 (`mergeTail` tolerates a widened `last_assistant_message`), "D-003
appends `last_assistant_message` to the transcript-derived text". It means the
decision.

### D-004: Segment turns positionally, not by identifier

D-004 holds one decision. It assigns to a turn every assistant record after the last
user record that has a `promptSource`, in the paragraph "Decision. Take every
assistant record following the last user record bearing a `promptSource`."

The Rejected. bullet about the `parentUuid` chain holds a finding: "observed an
assistant record whose `parentUuid` matched no preceding `uuid` in the same file".

At `f11e871`, no line outside the entry cites D-004.

### D-005: `mergeTail` tolerates a widened `last_assistant_message`

D-005 holds one decision. It detects the superset case, and uses the field alone if
it holds the whole turn, in the paragraph "Decision. Detect the superset case rather
than assume it cannot happen."

At `f11e871`, one line outside the entry cites D-005: `docs/spec-review.md:206`,
"widening (D-005)". It means the decision.

### D-006: No watermark rewind and no re-injection floor after compaction

D-006 holds one decision. It changes nothing about delivery after compaction, in the
paragraph "Decision. Change nothing." The title's "and" joins two rejected remedies,
not two decisions.

The Decision paragraph goes on to record a test result, which is a finding: "Phase 0a
tested it twice, including the realistic case … It kept it anyway, with attribution.
Both tests ran with `injected=0`". The same two tests are recorded in
`docs/phase0a-findings.md`, section 4, "The critical question: injected context
survives".

At `f11e871`, three lines outside the entry cite D-006: `docs/spec-review.md:482`,
and `docs/decisions.md:165` and `:171` in D-007 (record `COMPACTION` events as
observability). All three mean the decision.

### D-007: Record `COMPACTION` events as observability, not remediation

D-007 holds one decision. The daemon records a `COMPACTION` event so that compaction
is visible in room history, in the paragraph "Decision. When the daemon is built,
record a `COMPACTION` event".

At `f11e871`, three lines outside the entry cite D-007: `docs/spec-review.md:479`,
`:481` and `:547`. All three mean the decision.

### D-008: Key behavior verification on Claude Code version, not on the room

D-008 holds two decisions, and both sit in the Decision paragraph's first sentence,
"Check at room formation; cache on `claude --version` in `~/.cogmer/verified.json`."
The first caches a verification result on `claude --version`, and its alternatives
are "*Per room*" and "*Time-based expiry*". The second runs the check automatically
at room formation, with `COGMER_PREFLIGHT=off` to opt out. Its own alternative is
"*Manual only* - the failures are silent; nobody runs a check for a problem they
cannot see". The trigger could move, for example to daemon start, and the version key
would stand. The title states the first decision.

At `f11e871`, no line outside the entry cites D-008.

### D-009: A failed behavior check never blocks the room

D-009 holds one decision. It reports which assumption changed and what it breaks, and
carries on, in the paragraph "Decision. Report which assumption changed and what it
breaks; carry on."

At `f11e871`, no line outside the entry cites D-009.

### D-010: The behavior registry is the source of truth; its documentation is generated

D-010 holds two decisions. The first keeps behaviours and their checks together in
`behaviors.go` and generates `docs/relied-on-behaviors.md` from it, in the paragraph
"Decision. `cmd/cogmer/behaviors.go` holds behaviors and checks together." The
title's semicolon joins two halves of this one decision, and the title states it. The
second requires every behaviour to have a negative test proving it fails on the
regression it claims to catch, in the paragraph "Corollary enforced by test: every
behavior needs a negative test". The alternative it weighs is a check with no
negative test, which the entry says "reads as protection while providing none".
Either decision could be reversed without reversing the other.

The same paragraph holds a finding: "This caught a real error - the first `--deep`
run reported B05/B12 failing, which was a bug in the probe's own evidence handling".

At `f11e871`, no line outside the entry cites D-010.

### D-011: Hooks fail open: exit 0, empty stdout

D-011 holds one decision. Every hook failure path exits 0 and writes nothing to
stdout, in the paragraph "Decision. Every failure path exits 0 writing nothing." The
Rejected. bullet about stdout sends diagnostics to stderr.

The Decision paragraph holds a finding: "Measured 17 ms when the daemon is down;
connection-refused returns immediately".

At `f11e871`, one line outside the entry cites D-011: `docs/decisions.md:269`, in
D-013 (the probe points ambient hooks at a closed port), "any ambient `cogmer` hook
fails open (D-011)". It means the decision.

### D-012: The preflight probe uses the `cogmer` binary as its own hook

D-012 holds one decision. It registers `cogmer probe-hook <name> <dir>` as the
probe's hook command, in the paragraph "Decision. `cogmer probe-hook <name> <dir>`,
registered as the hook command".

At `f11e871`, no line outside the entry cites D-012.

### D-013: The probe points ambient hooks at a closed port

D-013 holds one decision. It runs the probe with `COGMER_ADDR` pointed at a closed
port, so that ambient hooks fail open, in the paragraph "Decision. Run the probe with
`COGMER_ADDR` pointed at a closed port".

At `f11e871`, no line outside the entry cites D-013.

### D-014: Derive delivery state from transcript evidence, not from recorded intent

D-014 holds two decisions. The first confirms delivery by finding the injected block,
matched on a sha256 of the emitted text, in a `hook_success` attachment in the
transcript, and records delivery as a set. It sits in the paragraph "Decision. Claude
Code records a hook's stdout in the transcript as a `hook_success` attachment."
Recording delivery as a set follows from it ("a lost injection leaves a hole a
contiguous watermark cannot represent"), so it is not a separate decision. The second
is the degraded path: while B20 (injected hook output is recorded as a
`hook_success` attachment) is recorded as failing for the installed version, the
daemon commits provisionally at injection and confirms at `Stop`, and otherwise
events stay pending and are offered again. It sits in the Rejected. bullet "Retained
as the degraded path (see below)" and in the paragraph "Consequence: absence of
evidence is not evidence of breakage." Its own alternative is falling back to
committing on trust whenever no attachment is found. It could be reversed, for
example by removing the fallback entirely, and the first decision would stand.

The Consequence paragraph also holds history: "The first implementation fell back to
committing on trust … Testing the failure path caught it. The fallback is now gated".
What testing the failure path showed is a finding. The Decision paragraph's
"Delivery became a set rather than a watermark" is history. So are the Status suffix
"Supersedes the advance-at-injection behavior reviewed as A1 in `spec-review.md`"
and the Context clause "an expired or lost reply meant the daemon recorded delivery".

At `f11e871`, 10 lines outside the entry cite D-014. Six mean the first decision
alone, and one of those, `CLAUDE.md:238`, is the reading-list row "capture,
reassembly, injection", which could mean either decision.
`cmd/cogmer/behaviors.go:226`, and `docs/relied-on-behaviors.md:67` generated
from it, cite D-014 for "Delivery confirmation" and describe the degraded path in
their Reliance text, so they mean both decisions. So does `docs/spec-review.md:48`:
"with a
set rather than a watermark and a fallback gated on B20 failing".
`docs/spec-review.md:560`, "The delivery fallback re-created the bug it fixed
(D-014)", means the Consequence paragraph's account of the first implementation,
which is the history and finding that the second decision answers.

### D-015: Rooms are scoped to sessions, not to projects

D-015 holds two decisions. The first makes a room a set of linked sessions with a
generated id, entered by invitation, with nothing about it derived from a directory,
repository or project. It sits in the paragraph "Decision. A room is a set of linked
Claude Code sessions". The second archives a closed room's event log, readable and
searchable, never rejoined, synchronized or injected, in the paragraph "Split that
makes it work: membership is ephemeral, the record is not." Its own alternative is
"*Discarding the log when a room closes*". Discarding instead of archiving would
leave session scoping untouched.

The rest of the entry holds history, a partial reversal and a pointer. The history is
the Context paragraph, "A2 found … the proposed fix was a `cwd` → project-config →
room lookup. That fix was answering the wrong question"; the paragraph "The strongest
argument is one the original specification did not make"; the paragraph "Two facts
made this cheap. Our implementation was already session-centric"; "Both sections were
amended rather than deleted" inside "Cost, stated plainly."; and the Status suffix
"Supersedes the project-scoped room model … dissolves A2". The Decision paragraph's
clause "closed when its last member session ends" contradicts D-016 (presence is not
membership), whose "Consequence for closing." closes a room on explicit departure by
all members or on prolonged dormancy. That is a partial change of the kind W-38 (a
partial change rewrites the earlier entry) covers. "Settled by D-016: a session holds
membership in at most one room at a time." is a pointer to D-016 (one room per
session), not a decision.

At `f11e871`, 19 lines outside the entry cite D-015. Fifteen mean the first
decision: rooms are session-scoped and never derived from a directory.
`docs/decisions.md:451` in the Context of D-017 (a room carries two identifiers)
says "D-015 gave rooms a generated id plus a human-chosen label", and D-015 holds the
generated id and no label, so it credits D-015 with something the entry does not
state. `docs/open.md:46`, "D-015 (rooms are session-scoped) has
a closed room kept as an archive", means the second decision.
`docs/decisions.md:382`, D-016's Status "Closes the open question in D-015", and
`:384`, "D-015 left open whether a session could hold membership in two rooms at
once", mean the question that the pointer "Settled by D-016" answers.

### D-016: One room per session, and presence is not membership

D-016 holds two decisions. The first binds a session to at most one room at a time,
and forbids a session that has received a colleague's injected context from moving
to another room, while a session that has received none may move. It sits in the
paragraphs "Decision - one room at a time." and "Moving to a different room is
refused once teammate context has been injected." The entry argues that one room at
a time follows from the event model, so it weighs no alternative to it. Both rejected
alternatives about binding, "*Allowing a session into a second room with a warning*"
and "*Binding a session to one room for its entire life, with no exception*",
concern the move rule. The binding and the move rule are therefore one decision. The
second makes presence separate from membership: membership
lasts until explicit departure or room closure, presence lapses when a process exits,
and a room closes on explicit departure by all members or after long dormancy. It
sits in the paragraphs "Presence is not membership.", "Leaving is always explicit.
Rejoining …" and "Consequence for closing." Its own alternative is "*Treating process
exit as departure*". The dormancy threshold could change without touching the first
decision.

The rest of the entry holds history. The paragraph on presence opens "The first draft
of this said membership ends when the Claude Code session ends - which is wrong …
Under that draft, two people closing their terminals for lunch would have archived
the room". A later paragraph says "§35's existing presence display … already assumed
this distinction; the specification simply had not stated it". The Status "Closes the
open question in D-015" and the Context paragraph are history as well. The clause
"the session persists and resumes under the same ID (verified in Phase 0a, checked by
B14)" is support, and it names its source. D-056 (a session's room is fixed at first
sight) reverses the first decision in part: it removes the exception that lets a
session that has received no teammate context move to another room.

At `f11e871`, seven lines outside the entry cite D-016. Four mean the first decision,
the room binding: `cmd/cogmer/commandinvocation_test.go:14`, `docs/open.md:260`, and
`docs/decisions.md:373` and `:1170`. Two credit D-016 with fixing a session's room at
its first prompt, which is what D-056 decided: `CLAUDE.md:201`, "is fixed at its
first prompt and never changes (D-016)", and `docs/decisions.md:6429`, "D-016 fixes
that at the first prompt". The seventh,
`CLAUDE.md:239`, is the reading-list row "rooms, membership, session binding", which
could mean either decision.

### D-017: A room carries two identifiers: a UUID for synchronization, a generated name for people

D-017 holds four decisions. The first makes `roomId`, a UUID, the key for all
replication, deduplication and storage, and makes `roomName` a non-authoritative
display name that unrelated rooms may share and that events do not carry. It sits in
the paragraphs "Decision. Every room has a `roomId`" and "The name is explicitly
non-authoritative." Its alternatives are "*A single identifier*" and "*Including the
name on every event*". The second generates the name from two curated lists, weather
or sky and landscape, so that nobody chooses it. It sits in the paragraphs "The name
is generated rather than chosen, and that is …." and "Weather-plus-landscape was
chosen for four properties". Its own alternative is "*Person-chosen names* -
reintroduces project scoping by convention". The third requires a name to be unique
only among one peer's live rooms, regenerated on collision, in the paragraph
"Uniqueness is scoped to a peer, not global." Its own alternative is "*Globally unique
names*". The fourth fixes a name for the life of the room, in the paragraph "Names are
immutable". Its own alternative is renaming, which the entry turns down because
"renaming would invalidate outstanding invitations".

The Context paragraph, "D-015 gave rooms a generated id plus a human-chosen label. The
label was the weak part.", describes what came before the decision, which is history
under W-32 (a decision describes only the present).

At `f11e871`, 16 lines outside the entry cite D-017. Nine mean the first decision. The
most frequent of them says that nothing keys on `roomName` because names collide by
design, as `CLAUDE.md:199` does, and the non-authoritative paragraph holds both halves
of it. Seven mean the
second decision, the generated, speakable and guessable name:

- `docs/decisions.md:506`, "The invitation format from D-017 read `misty-canyon@…`".
  D-017 holds the name `misty-canyon` and no invitation format.
- `docs/decisions.md:537`, "the name is drawn from a deliberately<!-- writing: quotes the decision log --> small, speakable space".
- `docs/decisions.md:596`, "D-017 made room names short, speakable, and therefore
  guessable".
- `docs/decisions.md:702`, "Peer identifiers should be readable, as room names are
  (D-017)".
- `docs/decisions.md:946`, "the name is guessable by construction (D-017)".
- `docs/decisions.md:2421`, "names are guessable by design (D-017)".
- `Shared Claude Sessions.md:1168`, "room names are guessable by design (D-017)".

No line means the third or the fourth.

### D-018: Identity and reachability are separate; invitations carry an endpoint and a secret

D-018 holds two decisions. The first makes `machineId` an identity label that nothing
routes by. It treats an endpoint as reachability that the transport supplies, which a
daemon discovers rather than derives from a hostname. The endpoint in an invitation
is a bootstrap hint, and there may be several. It sits in the paragraphs "Decision.
Separate the two concerns", "A daemon must discover its endpoint" and "The endpoint is
a bootstrap hint,". Its alternatives are "*Resolving `machineId` as a hostname*" and
"*Requiring Tailscale MagicDNS*". The title's first clause states it. The second holds
that a room name is not a credential, so authorization is separate from the name. It
sits in the paragraph "A room name is not a credential." Its own alternative is
"*Relying on the room name for authorization*".

The rest of the entry holds a finding and a reversed part. The finding is the Context
paragraph "Checked on the development machine: `os.Hostname()` returns
`macbookpro.lan`, `scutil --get LocalHostName` returns `pushover` … Tailscale is not
installed". The reversed part is the second half of the paragraph that holds the
second decision, "Authorization is a separate single-use, expiring secret … The
invitation is consequently a bearer credential, and §25 now says so", together with
the title's "and a secret". D-026 (no join token) reverses it, a partial change of the
kind W-38 (a partial change rewrites the earlier entry) covers. The clause naming §25
(security) is also history.

At `f11e871`, six lines outside the entry cite D-018. Two mean the first decision:
`docs/decisions.md:605` and `:626`. `docs/spec-review.md:563`, "`machineId` was never
an address (D-018)", means the first decision and restates the finding that supports
it. Three mean the join secret, the reversed part:

- `docs/decisions.md:596`, "D-018 made authorization a separate secret".
- `docs/decisions.md:643`, "is the join secret from D-018 still needed?".
- `docs/decisions.md:892`, the Status of D-024 (admission is a guest list), "Corrects
  the emphasis of D-018, D-020".

### D-019: No network provider is required; local discovery is the zero-configuration path

D-019 holds three decisions. The first keeps every network provider out of room
identity, membership and replication, makes none a prerequisite, and keeps which
transport connected out of room identity and event data. It sits in the paragraph
"Decision. State as a requirement that no network provider is part of room identity"
and the paragraph "Transports are attempted in order … which one connected is an
implementation detail". Its alternative is "*Tailscale as the architectural
foundation*". The attempt order, same network, then a private provider, then
internet, then relay, is stated with no alternative weighed, so it is part of the
first decision. The second lets peers on the same network join with no configuration
through local discovery, which locates a room and never admits anyone to it. It sits
in the sentences "Two people on the same network is the simplest case and must be the
easiest" and "So discovery locates a room; it never admits anyone to one." Its own
alternative is "*Joining by name alone on a trusted network*". It could be reversed,
with same-network peers pairing like anyone else, and the first decision would stand.
The third uses `tsnet` only inside `TailscaleTransport`, in the paragraph "Note for
implementation. `tsnet` … attractive *inside* `TailscaleTransport`, and unacceptable
anywhere else." Its own alternative is using `tsnet` elsewhere, or shelling out to
the Tailscale CLI, which the paragraph names. It is a small decision.

The rest of the entry holds a reversal, history and a finding. The build order,
"Implementation order is Local, then Tailscale, then WebRTC" and "Local discovery
moved from 'later possibility' to the first transport built", is reversed by D-063
(the first pair is remote). The paragraph "That priority was wrong, and D-063
corrects it." and the Status clause "Its build order is superseded by D-063" record
that reversal inside the entry, a partial change of the kind W-38 (a partial change
rewrites the earlier entry) covers. The example `cogmer join
misty-canyon#k7qm-2xpr-9vlt`, "while the secret remains" and "The secret also
disambiguates" describe the join secret that D-026 (no join token) removes. The
history is "The architectural seam already existed … What changed is the priority";
the paragraph "Where this conflicted with earlier decisions, and how it was
resolved."; and the Context paragraph "The specification was written for an internal
experiment". "Tailscale is also not installed on the development machine, so local
discovery is now the shorter path" is history and a finding.

At `f11e871`, 20 lines outside the entry cite D-019. Eleven mean the first decision,
that no provider is required and a transport sits under everything. They include
`CLAUDE.md:140` and `:241`, `cmd/cogmer/tailcat.go:38`, and `docs/decisions.md:3060`,
`:3063`, `:3147` and `:3173`. `CLAUDE.md:241` is the reading-list row "addresses,
transport, reachability", which could mean either decision. Six mean the second
decision, local discovery, and
three mean the build order that D-063 reverses. Those nine are:

- `docs/open.md:51`, "Local network discovery (D-019's zero-configuration path) is not
  built".
- `docs/decisions.md:682`, "makes broadcast discovery (D-019) safe to enumerate".
- `docs/decisions.md:4584`, "which D-019 names as the zero-configuration path".
- `docs/decisions.md:4703`, "discovered on the network we are on now (D-019)".
- `docs/decisions.md:4747`, "local discovery lands (D-019's zero-configuration
  path)".
- `docs/decisions.md:4763`, "D-019's zero-configuration path is not built".
- `docs/decisions.md:3141`, "D-019 made local discovery the first transport to
  build", which means the build order.
- `Shared Claude Sessions.md:2758`, "a reversal of D-019's priority", which means the
  build order.
- `Shared Claude Sessions.md:2853`, "Local discovery is last, and that reverses
  D-019", which means the build order.

No line means the third decision.

### D-020: A guest list replaces the join secret only once peer identity is cryptographic

D-020 holds two decisions. The first requires peer identity to become cryptographic:
the identifier derived from a public key, possession proved on connection, and events
signed at origin so that relay can be verified. Until then the guest list, the relay
rule and attribution are conventions. It sits in the Decision paragraph's "State in
the specification that peer identity must become cryptographic" and in the paragraph
"The stronger argument for doing the work is unrelated to admission." Its alternative
is "*Deferring identity until after Phase 2*". The second prefers, once identity is
cryptographic, a guest list of known keys over a secret. It sits in the paragraphs
"Once identity is cryptographic, the guest list is the better mechanism" and "They
compose rather than compete." Its own alternatives are "*Guest list instead of the
secret, now*" and "*Guest list keyed on `userId` or `machineId`*". D-024 (admission is
a guest list) holds the same preference as its ordinary path.

The rest of the entry holds a reversed part and history. The Decision paragraph's
"Decision. Keep the secret for now" and the composing model's "a single-use secret
admits a peer that is not yet known" are both reversed by D-026 (no join token), a
partial change of the kind W-38 (a partial change rewrites the earlier entry) covers.
The paragraph "Finding: not with identity as it stands." is the reasoning for the
first decision despite its label, and its bold opener is not a template field. "This
closes review item C7." is history.

At `f11e871`, 11 lines outside the entry cite D-020. Seven mean the first decision,
cryptographic identity and event signing: `cmd/cogmer/peername.go:57`,
`docs/spec-review.md:382` and `:497`, `docs/decisions.md:709`, `:760` and `:885`, and
`Shared Claude Sessions.md:2778`. Four mean something else:

- `docs/decisions.md:3076`, "a `peerId`, which is an Ed25519 key (D-020)", and
  `cmd/cogmer/tailcat.go:45` to `:46`, "an Ed25519 key that signs events and is what
  two people compare two words against (D-020)", credit D-020 with Ed25519, which
  D-020 never names; D-042 (peer identity is an Ed25519 key pair) decides it.

- `docs/decisions.md:723`, "It comes from D-020's known peers", which means the second
  decision.
- `docs/decisions.md:892`, the Status of D-024 (admission is a guest list), "Corrects
  the emphasis of D-018, D-020". What D-024 corrects is the secret D-020 keeps for
  now, the reversed part.

### D-021: Peer names are word pairs derived from the identity, never chosen

D-021 holds four decisions. The first derives a peer's `peerName`, an adjective and an
animal, from a hash of `peerId` on load, never stores or chooses it, and draws it from
word lists fit to name a colleague. It sits in the paragraphs "Decision. A peer
carries two identifiers", "The name is derived, never chosen" and "Word lists name
colleagues". Its alternatives are "*Self-chosen display names*", "*Storing the name
alongside the identity*" and "*Indexing the word lists by the first and last bytes of
the identifier*". The second shows a name as itself only for a verified peer, and
marks an unverified speaker inside the injected text, not only in an interface. It
sits in the sentence "The specification now requires that a name be displayed as
itself only for a verified peer". Its own alternative is marking only in a UI, which
"protects the wrong reader". A derived name could stay while the marking rule
changed. The third renders the whole key for a fingerprint comparison, as a word
sequence separate from the peer name. It sits inside a Rejected. bullet: "The
instinct is sound at a different scale. … §25 now requires that such a comparison
render the *whole* key." Its own alternative is offering the two-word name as a
fingerprint check. The fourth keeps the word lists at 8,280 combinations rather than
expanding both to 256 entries, in the paragraph "On collisions. … Held off because
curating 512 words". Its own alternative is expanding to 65,536 combinations.

The rest of the entry holds findings and history. The Rejected. bullet on slicing
records a measurement, "one distinct value across 2,000 generated ids" and "roughly
50% more likely", and the paragraph on collisions records arithmetic, "0.5% … 3.6% …
45%". Both are findings. "Implemented as `PeerName()`", "The specification now
requires" and "§25 now requires" are history.

At `f11e871`, 11 lines outside the entry cite D-021. Six mean the first decision:
attribution anchors on the derived name, never on a chosen one. Three mean the second
decision, and two match no part of D-021:

- `cmd/cogmer/ui.go:127`, "§6/D-021 require the identifier be shown for any peer
  whose identity is unverified", which means the second decision.
- `cmd/cogmer/ui_test.go:275`, "Your own turns need no marker and no anchor (D-021)",
  which names the second decision's marker and also speaks of your own turns.
- `docs/decisions.md:1392`, "demonstrated D-021 reaching the person it was for. The
  `unverified` marker", which means the second decision.
- `docs/decisions.md:4924`, "D-021 keeps the derived name off your own turns", which
  matches no part.
- `docs/decisions.md:5979`, "D-021 suppresses the derived name on your own turns",
  which matches no part.

D-021 says nothing about a user's own turns. That choice was made in commit
`33ce996` and is recorded in no decision.

### D-022: A room begins when someone is invited into it, and never earlier

D-022 holds one decision. Creating a room and issuing its first invitation are one
act, and nothing is captured before it. It sits in the paragraph "Decision. The
`roomId` and `roomName` are generated at the moment a person invites someone."

The entry holds history in two places: "So the specification now states that *'from
its beginning' means the beginning of the room, never of a session that belongs to
it*", and the Context paragraph "The specification said a room is 'created by one
peer' … in three places".

At `f11e871`, five lines outside the entry cite D-022: `docs/spec-review.md:142` and
`:147`, and `docs/decisions.md:834`, `:2023` and `:2036`. All five mean the decision.

### D-023: A peer identifier must be safe to know; identities are created once and exchanged on joining

D-023 holds three decisions. The first makes the peer identifier the public key or its
fingerprint, safe to know only once signatures are checked, so that no keypairs are
generated before verification. It sits in the paragraphs "The identifier must be safe
to know." and "The order matters, and is easy to get wrong." Its alternatives are
"*Treating the current identifier as sensitive*" and "*Generating keypairs now as a
first step*". The second creates an identity once, on a machine's first use, to
persist beyond every room, session and invitation, and gives it to a machine rather
than to a human. It sits in the paragraphs "When. An identity is created once" and
"An identity belongs to a machine, not a person". Its own alternative is an identity
per room, per session or per invitation, or one per human carried across machines,
which the entry answers with "Deliberate - a key that never leaves the machine that
made it". Either could change with the first decision intact. The third exchanges
identities on joining, with no directory, in the paragraph "How a guest's identifier
is obtained: it is not, in advance." Its own alternative is "*A directory of peer
identifiers*". The paragraph "Self-certifying is not self-authenticating. … Verifying
once, over a
channel the invitation did not travel on, closes that" states that a key accepted at
first contact is verified once, over another channel. The entry weighs no alternative
for it, so under the test this document applies it is a requirement, not a fourth
decision. D-023 does not cite D-054, which is a later entry. D-054 (verification gates
synchronization and injection) makes verification
a condition of synchronization, and D-055 (one way to verify a peer) holds the
comparison that performs it.

The rest of the entry holds history: "which is why nothing was implemented here", and
the Context sentence "Answering the second surfaced that the current identifier is
hazardous to share". "it is a random value that nothing verifies" describes identity
as it stood before D-042 (peer identity is an Ed25519 key pair).

At `f11e871`, 12 lines outside the entry cite D-023, and all twelve mean the first
decision, that an identifier is safe to know, is not verified yet, or must follow the
ordering warning: `cmd/cogmer/keys.go:16`, `cmd/cogmer/sync.go:20`,
`docs/spec-review.md:288` and `:567`, and `docs/decisions.md:937`, `:1247`, `:1256`,
`:1345`, `:1503`, `:1753`, `:1846` and `:1857`.

### D-024: Admission is a guest list; the join code is a fallback for strangers

D-024 holds two decisions. The first is that admission is a host's guest list of
known public identifiers, proved by possession of the key, so the name locates and
the list admits. It sits in the paragraph "Decision. A guest list is the ordinary
path." and its sentence "The name locates, the list admits." The second keeps a join
code as a fallback for strangers, in the title's second clause and the paragraph "The
code survives as a fallback, explicitly weaker." Its own alternative is the Rejected.
bullet "*Removing codes entirely*". D-026 (no join token) reversed the second decision
and left the first standing, a partial change of the kind W-38 (a partial change
rewrites the earlier entry) covers, and adopted that alternative.

The rest of the entry holds history. The history is "The error was conflating two
claims. … I
treated it as though it did"; the Status "Corrects the emphasis of D-018, D-020"; and
the Context sentence "The specification conceded in one sentence". The paragraph
"Unchanged by this. The ordering from D-023 still governs" restates the ordering that
D-023 (a peer identifier must be safe to know) holds, and describes it as unchanged,
which is history under W-32 (a decision describes only the present).

At `f11e871`, 10 lines outside the entry cite D-024. Six mean the first decision.
Four mean something else:

- `docs/decisions.md:2531`, "known peers … and a room's guests … as two lists since
  D-024", which means the first decision of D-025 (the guest list, specified), its
  two scopes.
- `docs/decisions.md:1015`, the Status of D-026, "Supersedes the residual code path in
  D-024", which means the second decision.
- `docs/spec-review.md:571`, "The bearer token was never necessary (D-024, D-025)",
  which credits D-024 with what D-026 decided, since D-024 keeps the code.
- `CLAUDE.md:142`, "No join tokens, and no join-by-name on a trusted network (D-024,
  D-026)", which credits D-024 with D-026's decision in the same way.

### D-025: The guest list, specified: two scopes, a signed challenge, and approval in the moment

D-025 holds three decisions. The first keeps two lists at different scopes: known
peers per machine, durable, and a room's guests per room, archived with it, each with
commands to inspect and correct it (`peers`, `allow`, `forget`; `guests`, `invite`,
`revoke`). It sits in the paragraph "Two lists, at different scopes." Its alternative
is "*One combined list*". The title leads with it. The second proves admission by
signing a fresh, unpredictable challenge from the host, in the paragraph "Admission is
a fresh signed challenge." Its own alternative is a replayable proof, which the entry
turns down because "an exchange that can be replayed is a bearer credential with
extra steps". The proof format could change and the two scopes would stand. The third
shows a refused peer's request to a present host, who approves it by identifier, and
never queues a request for an absent host. It sits in the paragraph "Refusal must be
more than silence, and that changes the role of codes." Its own alternatives are
"*Silent refusal*", "*Queuing requests for an absent host*" and "*Approving by
displayed name*".

The rest of the entry holds a reversed part and history. The paragraph "Codes now
cover one case only: … *(Superseded by D-026 …)*" carries text reversed by D-026 (no
join token) inside a live entry, a partial change of the kind W-38 (a partial change
rewrites the earlier entry) covers. The Status "Completes D-024" and the Context
sentence "Naming an analogy is not specifying a design" are history.

At `f11e871`, six lines outside the entry cite D-025. Three mean the first decision or
admission as a whole: `docs/phase2-experiment.md:215`, and `docs/spec-review.md:142`
and `:149`. None means the second decision. Three mean something else:

- `docs/spec-review.md:571`, "a host present can approve a stranger's request", which
  means the third decision.
- `docs/decisions.md:1015`, "the residual code path in D-024 and D-025", which means
  the reversed code paragraph.
- `docs/decisions.md:1017`, "D-025 had narrowed join codes to a single case", which
  means the reversed code paragraph.

### D-026: There is no join token at all

D-026 holds one decision. There is no join token or invitation secret, and admission
is a guest-list entry proved by a key, or a present host's approval. It sits in the
paragraph "Decision. Remove join tokens from the design."

The entry holds history: the Status "Supersedes the residual code path in D-024 and
D-025", the Context sentence "D-025 had narrowed join codes", and the paragraph
"Consequence. The system now has no credential … not as a deprecated path, but
absent. §25 says so directly".

At `f11e871`, 11 lines outside the entry cite D-026, and all eleven mean the
decision.

### D-027: A sequence conflict is quarantined, not dropped

D-027 holds one decision. It classifies each receipt as stored, duplicate or conflict,
keeps a conflicting event quarantined with both identifiers, and surfaces it through
`cogmer conflicts`. It sits in the paragraph "Decision. Distinguish the three outcomes
on receipt". The paragraphs that open "Quarantine rather than reject.", "Why not
repair it automatically." and "Surfacing matters as much as detecting." are its
reasoning and rejected alternatives.

The Context sentence "`INSERT OR IGNORE` against `UNIQUE(peer_id, peer_sequence)`
absorbed that as an ordinary duplicate" describes the system before the decision,
which is history under W-32 (a decision describes only the present).

At `f11e871`, eight lines outside the entry cite D-027, and all eight mean the
decision.

### D-028: Losing a room database ends that peer's membership; recovery is not attempted

D-028 holds one decision, and D-029 (losing a room database does not end membership)
reverses all of it. A peer that loses a room database leaves the room, does not
rejoin and does not resume publishing, in the paragraph "Decision. Treat it as the end
of that peer's membership in that room."

The whole entry is a reversed decision. Its Status reads "SUPERSEDED BY D-029 - the
cost of ending membership was assessed wrongly, and the mechanism that avoids it is
cheaper than the sequence epoch this entry rejected." Under W-37 (a reversed decision
becomes a tombstone), a tombstone holds only the number, the title and a status line.
The body holds what D-028 decided, the sequence-epoch analysis, and the paragraph "An
inversion worth knowing", that losing identity is the safe failure, which D-029
depends on. "Under the project-scoped model this decision replaced, the same event
would have cost months" is history under W-32 (a decision describes only the
present).

At `f11e871`, seven lines outside the entry cite D-028, and all seven mean the
withdrawn decision. `docs/spec-review.md:227`, "D-028 says its membership ends",
states it correctly. The other six are in D-029, at `docs/decisions.md:1158`,
`:1160`, `:1168`, `:1195`, `:1199` and `:1210`, all in the parts of D-029 that
describe D-028. W-37 says the decision that replaces a reversed one cites neither the
tombstone nor the finding.

### D-029: Losing a room database does not end membership; the sequence lives with the identity

D-029 holds one decision. Membership survives the loss of a room database, and the
peer's own sequence position is stored outside the room database, with the identity,
and reserved before any event using it is published. It sits in the paragraph
"Decision. Membership survives." and the paragraph "Reserve before publishing." The
reservation is the condition that makes the stored sequence safe, and reversing it
breaks the decision, so it is part of this one and not a second decision. The title's
semicolon joins the decision to its mechanism.

The rest of the entry holds history of D-028 (losing a room database ends that peer's
membership): the Status "Supersedes D-028"; the Context "D-028 treated a lost room
database …"; "The database is not where the value is. … D-028 traded something
consequential"; "And it took more than it appeared to. … A disk hiccup cost the
afternoon. Neither decision was wrong alone"; the paragraph "Why this beats the
sequence epoch D-028 rejected."; "There is no longer any loss that … was the
precondition for B4"; and the Rejected. bullet "*Ending membership* (D-028) - see
above". The epoch comparison also appears as the Rejected. bullet "*A sequence
epoch* - correct, and unnecessary once the counter cannot be lost." The paragraph
"Costs,
stated so they are expected." states what the decision gives up, which the template
puts under Limits.

At `f11e871`, 17 lines outside the entry cite D-029, including the Status line of
D-028 at `docs/decisions.md:1099`, and all seventeen mean the decision.

### D-030: Two listeners: hooks on loopback, peer sync separately

D-030 holds two decisions. The first serves hooks and the UI on one listener,
`COGMER_ADDR`, that refuses any non-loopback bind, and synchronization on a separate
listener rather than one mux with a path filter. It sits in the paragraphs "Decision.
Two listeners." and "Why the hook API is the strict one." Its alternative is "*One
listener, path filtering*". The title states it. The second lets the peer listener,
`COGMER_PEER_ADDR`, default to loopback and be bound elsewhere with a warning, rather
than be refused until authentication exists. It sits in the paragraph "The peer API is
exposed with a warning rather than refused,". Its own alternatives are "*Exposing the
peer API by default*" and "*Refusing to expose the peer API until authentication
exists*". The exposure policy could change with the two listeners intact.

The Context paragraph is history: "Making a pair work across two machines was
preferred over a three-peer run" is a sequencing choice, and D-032 (re-sequence the
phases) states the phase order, with its Context recording that Phase 6 was
deferred. "One listener on `127.0.0.1` served hooks, the UI, and synchronization" is
history as well. The warning paragraph says the peer API is "unauthenticated and that
nothing verifies who connects - true until identity becomes cryptographic (D-023)".
At `f11e871`, `cmd/cogmer/sync.go` refuses an unauthenticated sync request and D-042
(peer identity is an Ed25519 key pair) is implemented, so that clause describes the
peer API as it stood when D-030 was written.

At `f11e871`, one line outside the entry cites D-030: `cmd/cogmer/ui_test.go:17`,
"serving it to peers would hand the room to anyone who can reach the sync port". It
means the first decision, the separation.

### D-031: Peer synchronization polls; the push worth building is the local UI's

D-031 holds one decision. Peer synchronization polls by default, and push for the peer
layer is not built without a reason beyond latency. It sits in the paragraph
"Decision. Polling is the default for peer synchronization, not a placeholder." The
paragraph "The push that does matter is a different one." names the live-UI
requirement of §17 (shared conversation UI), served by server-sent events from the
daemon. It weighs no alternative of its own, so it is a pointer and a limit, not a
second decision.

The rest of the entry holds history and a finding. The Context paragraph is history:
"Polling was chosen for the two-peer experiment and recorded only as a code comment …
nobody had decided". The Rejected. bullet "*Leaving it undecided*" is about process,
not an alternative design. "Peer propagation is roughly half a second" is a finding:
the measurement from review item C4 in `docs/spec-review.md` ("Real time" means two
different things, and one of them is slow).

At `f11e871`, two lines outside the entry cite D-031: `cmd/cogmer/sync.go:22` and
`docs/decisions.md:2982`. Both mean the decision.

### D-032: Re-sequence the phases, and follow them

D-032 holds two decisions. The first sets an order of work to follow (Phase 8, then 9,
10, 5 and 7), and it sits in the Decision paragraph, which opens "Decision. Record
actual status against every phase, add the phases the original sequence lacked, and
state an execution order to follow from here". The second is that phase numbers are
never reused or reassigned and superseded phases are marked rather than rewritten, in
the paragraph opening "Numbers are never reused or reassigned, so references in this
log and in the code still resolve." The entry weighs its own alternative for the
second under Rejected.: "*Renumber the phases* - breaks every reference in this log
and in the findings documents". The phases could be renumbered while the same order
was followed, so the second can be reversed without the first.

The Context paragraph, opening "Context. Work had proceeded opportunistically", is
history under W-32 (a decision describes only the present): it records what was done
out of order. The paragraph opening "*Amended the same day.* The first draft of this
bundled room identity" is history as well. The status the Decision paragraph asks to
be recorded against every phase is current state, and the status itself is not a
decision. The paragraph opening "What this does not change. Pairs remain the target.
Phase 6 waits" restates a standing position that D-109 (two is the target and nothing
rules out more) holds.

At f11e871, 3 lines outside the entry cite D-032. None is a test fixture that uses the
number as a string. All three sit in D-109 and describe D-032's text, and all three
mean the "pairs remain the target" aside rather than the order of work, although the
first names D-032 by its order:

- `docs/decisions.md:5763` "D-032 (the order of work) records in an aside that "pairs remain the target""
- `docs/decisions.md:5802` "*Leave it in D-032's aside.*"
- `docs/decisions.md:5806` "the trigger D-032 already names", which is Phase 6 waiting for evidence that a third peer is wanted

### D-033: The room cannot be displayed inside Claude Code; asking is the free affordance

D-033 holds two decisions. The first is that the room is not displayed inside a
session, and that asking the model is the in-session affordance; it sits in the
Decision paragraph, which opens "Decision. §17 no longer prescribes a browser. It
states that the room cannot be shown inside the session, names asking…". The second is
that a terminal view and a browser view are different moments rather than competitors,
and that more than one may exist because each reads only from the local daemon. It
sits in the same paragraph, from "…and treats a terminal view and a browser view as
different moments rather than competitors…". Its own alternative is under Rejected.:
"*Treating the browser as the answer*". D-039 (one renderer until it has been used)
answers the renderer question differently while the first decision holds.

The paragraph opening "Finding: it cannot. Tested directly. A hook's standard output
becomes context" is a finding: standard output, standard error and `/dev/tty` were
each tested. The paragraph opening "What the test surfaced that was worth more than
the answer. A person can simply" is a finding about an observed session answering from
injected context. The paragraph opening "It also demonstrated D-021 reaching the
person it was for" is a finding: the model relayed the `unverified` marker, the tag
injected text carries for an unverified peer, without being asked. D-021 is the
decision that peer names are word pairs derived from the identity. The Context
paragraph, opening "Context. §17 specified a browser at `localhost`, and a browser was
built to it", is history under W-32 (a decision describes only the present).

At f11e871, 13 lines outside the entry cite D-033. None is a test fixture that uses
the number as a string. All 13 mean that nothing Claude Code offers displays to a
person, which the first decision states in §17 (shared conversation UI) and the
finding "Finding: it cannot." shows. Nine are written "(D-033, D-036)". The other four
are `docs/decisions.md:1423` "D-033 established that Claude Code surfaces nothing a
hook writes", `docs/decisions.md:1625` "D-033 (hooks display nothing)",
`docs/decisions.md:1662` "D-033 concluded that the room cannot be displayed inside a
Claude Code session", and `docs/decisions.md:6445` "D-033 and D-036 settled that every
extension point delivers to the model and nothing displays to a person". No citation
means the second decision.

### D-034: No terminal wrapper; a view sits beside the session rather than around it

D-034 holds one decision. It is that no pseudo-terminal wrapper is built, and a view
is a separate program running beside the session; it sits in the Decision paragraph,
which opens "Decision. Do not wrap." The title's semicolon joins two halves of this
one choice.

The paragraph opening "The technique works. A passthrough prototype was byte-for-byte
identical" is a finding. The paragraph opening "Recorded as tested, so it is not
re-derived: an invisible pseudo-terminal passthrough" is a finding about the
reserved-band technique. The Rejected. item's clause "the prototype is reverted rather
than parked" is history under W-32 (a decision describes only the present), and so is
the Context sentence "A pseudo-terminal wrapper was proposed".

At f11e871, 4 lines outside the entry cite D-034. None is a test fixture that uses the
number as a string. All four mean the decision that there is no wrapper.

### D-035: A remote peer never initiates local execution

D-035 holds two decisions. The first is §3.7 (a remote event never drives an
interactive session): a remote event never causes a turn in an interactive session. It
sits in the Decision paragraph, which opens "Decision. State it as §3.7", as narrowed
by the paragraph opening "The principle is now scoped to interactive sessions: a
session a person is working in takes a turn when that person asks it to, and at no
other time." The second is that the principle is guarded by a structural test, which
forbids the files handling peer traffic from importing `os/exec` or `syscall` or
calling `RunProbe`, the only function that starts a Claude run, and which is stricter
than the principle. It sits in the paragraph
opening "Guarded structurally rather than by review." The entry names its alternatives
in that paragraph: review, and checking call graphs, as in "Checking imports rather
than call graphs is crude, and deliberately<!-- writing: quotes the decision log -->
so". A later need for a separate run could loosen the guard without changing §3.7,
since "the guard stands until they do".

The paragraph opening "The implementation already conforms, and not by design so much
as by not having written the code" is history under W-32 (a decision describes only
the present). So is the paragraph opening "The specification did not state it, and
said something weaker that was also wrong. §16 read", which describes the old wording
of §16 (propagation). The paragraph opening "Scope, corrected the same day. The first
draft forbade a remote event starting *any* Claude run" is history, and the rule it
produced is the first decision. The paragraph opening "Whether a peer event may cause
a separate run is explicitly left open" is an open question. The title, "A remote peer
never initiates local execution", states the scope of the first draft, while the first
decision as corrected covers interactive sessions only. The entry has no Rejected.
field, which W-36 (a decision always has Rejected.) requires.

At f11e871, no line outside the entry cites D-035. Other text refers to the rule as
§3.7, which 15 lines outside the entry cite.

### D-036: MCP logging notifications are not a display channel

D-036 holds two decisions. The first is that MCP is not used as a display channel,
because it carries capability to the model and nothing to a person; it sits in the
paragraph opening "Consequence. MCP carries capability…". The alternative it answers
is the one in the Context paragraph, an MCP server as the carrier for ambient display.
The second is that no MCP server this project ships exposes sampling, because sampling
would let a peer cause inference in an interactive session. It sits in the paragraph
opening "Related, and worth stating before anyone builds an MCP server here for
another reason." and ending "Any MCP server this project ships must not expose one."
Its own alternative is shipping an MCP server with sampling, which the entry describes
as "a direct route to violating §3.7". The second could be reversed without the first.
`CLAUDE.md:171` states it without a number: "An MCP server must not expose sampling,
for the same reason."

Most of the entry is a finding: the paragraph opening "Tested, and it does not work.",
the four places checked, the paragraph opening "Tested twice, because the first test
was wrong.", and the paragraph opening "The conclusive evidence is the capability
record" with its JSON line. The sentence "Whether Claude Code implements sampling was
not tested." is a finding too. The entry has no Rejected. field, which W-36 (a
decision always has Rejected.) requires.

At f11e871, 12 lines outside the entry cite D-036. None is a test fixture that uses
the number as a string. All 12 mean the first decision, that nothing displays to a
person, including `docs/decisions.md:1626` "D-036 (MCP logging is never rendered)". No
citation means the sampling rule.

### D-037: Claude Code is launched and used unchanged

D-037 holds two decisions. The first is §3.8 (the host is launched and used
unchanged): a user starts and uses Claude Code unchanged, and the system installs only
what Claude Code already loads. It sits in the Decision paragraph, which opens
"Decision. State it as a principle instead." The second is that ambient awareness, if
wanted, comes from an operating-system notification raised by the daemon. It sits in
the paragraph opening "Preferred instead, if ambient awareness is wanted. The daemon
can raise an operating-system notification directly". Its own alternatives are a
terminal pane beside the session, which D-039 (one renderer until it has been used)
weighs, and an undocumented display seam inside the session. A pane could be chosen
instead while §3.8 held.

The Context paragraph, opening "Context. D-034 ruled out a pseudo-terminal wrapper by
enumerating its costs", is history under W-32 (a decision describes only the present),
including its clause "it was derived *after* a prototype had already been built". The
sentence "Applied earlier, it would have stopped the wrapper before anything was
written." is history too. The paragraph opening "On searching for an undocumented
display seam." is a finding: it reports that the installed artifact is a native
binary, that "Channels" does not appear in this version, and that `claude plugin
details` reports a projected token cost. Declining the seam is a rejected alternative
of the first decision as well as one of the second decision's own alternatives. The
sentence opening "Confirmed independently three times:
D-033 (hooks display nothing), D-036 (MCP logging is never rendered)" restates a
finding that `CLAUDE.md` holds under "Facts that are not obvious from the code". The
paragraph opening "If a display primitive is ever wanted from Anthropic, the request
is small" describes a request to another party and is not a decision. The entry has no
Rejected. field, which W-36 (a decision always has Rejected.) requires.

At f11e871, no line outside the entry cites D-037. The `CLAUDE.md` bullet at lines 131
to 133 says "prefer an OS notification from the daemon for ambient awareness" under
"(D-038, D-039)", although that preference is the second decision here. D-039 leaves
announcing open.

### D-038: Separating the room from the session is correct on its merits

D-038 holds one decision. It is that the room is viewed outside the session because
that is right on its merits, and would be even if an in-session display became
available; it sits in the Decision paragraph, which opens "Decision. Record the
separation as a design position rather than a consequence."

The Context paragraph, opening "Context. D-033 concluded that the room cannot be
displayed", is history under W-32 (a decision describes only the present), down to
"That framing was backwards." The paragraph opening "Consequence for how the
constraint is described. §17 no longer opens by saying" is history: it records an edit
to §17 (shared conversation UI). The entry has no Rejected. field, which W-36 (a
decision always has Rejected.) requires.

At f11e871, 6 lines outside the entry cite D-038. None is a test fixture that uses the
number as a string. All six mean the decision.

### D-039: One renderer until it has been used

D-039 holds one decision. It is that the browser view, the page that shows the room in
a browser, is the only renderer, and that no second renderer (a terminal pane or a
notification) is built until use shows one is needed. It sits in the Decision
paragraph, which opens "Decision. Build neither yet.", as settled by the paragraph
opening "Outcome, 2026-09-17. Used, and judged the right avenue."

The paragraph opening "Outcome, 2026-09-17. Used, and judged the right avenue." is a
dated observation and so a finding: "a separate window is consulted rather than
forgotten". The sentence "The browser view exists; nobody has worked with it." and the
paragraph opening "What using it will settle." describe the time before that outcome
and are history under W-32 (a decision describes only the present). The paragraph
opening "Still open is whether arrival wants announcing" is an open question, which
`docs/open.md:223` holds. The entry has no Rejected. field, which W-36 (a decision
always has Rejected.) requires.

At f11e871, 4 lines outside the entry cite D-039. None is a test fixture that uses the
number as a string. Three mean the decision, including `docs/open.md:223` "D-039 (the
browser view is the settled avenue)". One means the outcome finding:

- `docs/what-leaves-findings.md:67` "the browser view at 145 lines, which D-039 records as the one thing consulted rather than forgotten"

### D-040: The injected block is fenced with an unforgeable value, and framed by classification

D-040 holds two decisions. The first is that the injected block's boundary is
unforgeable: a fence, a value generated for each injection that marks where the block
ends, is stripped from the content, the block ends only at the matching value, and the
framing is restated after the content. It sits in the paragraph opening "Decision -
the boundary must be unforgeable.", and its alternative is under Rejected.: "*Escaping
the delimiters*". The second is that the framing classifies what the model is reading
instead of instructing it to disregard instructions: nothing inside is addressed to
it, and a request inside is a report of a request. It sits in the paragraph opening
"Decision - frame by classification rather than authority." The entry weighs its own
alternative in that paragraph: "Instructing a model to disregard instructions invites
it to weigh two instructions". The fence could be kept with framing by authority, or
classification used with a static delimiter.

The paragraph opening "The vulnerability. Content was interpolated raw" is a finding,
ending "Demonstrated rather than theorised." The paragraph opening "Verified against a
live session, not only in structure. A real session given a forged `SYSTEM OVERRIDE`"
is a finding. The Context paragraph, from "There was such language - a sentence at the
top of the block - and it was defeatable.", is history under W-32 (a decision
describes only the present), and so is the phrase "The framing now says".

At f11e871, 4 lines outside the entry cite D-040. None is a test fixture that uses the
number as a string. `docs/decisions.md:4639` "D-040 (fence the block so content cannot
close it)" and `CLAUDE.md:175`, which puts "(D-040)" after the fence clause, mean the
first decision. `CLAUDE.md:238`, the reading-map row for capture, reassembly and
injection, covers the first decision. The sentence after the fence clause in
`CLAUDE.md`, "Frame by classification, never by asserting authority.", states the
second decision and carries no number. One citation means the vulnerability finding,
that content from peers can carry instructions aimed at the model, rather than the
fence:

- `docs/decisions.md:2526` "without passing through a model that reads room content from unverified peers (D-040)"

### D-041: Ship as a Claude Code plugin; the session-start hook starts the daemon

D-041 holds three decisions. The first is that cogmer is packaged as a Claude Code
plugin carrying the hooks, in the Decision paragraph, which opens "Decision. Package
as a plugin carrying the hooks". The second is that the session-start hook starts the
daemon, that starting it never delays the session, that finding it already running is
the ordinary case, and that failure is silent to the user. It sits in the paragraphs
opening "And the session-start hook starts the daemon." and "Three requirements, each
easy to get wrong." Its own alternative is the manual start in the Context paragraph,
"a manual daemon start". A plugin could ship hooks and still leave the user to start
the daemon. The third is that the daemon outlives the session that started it and is
discoverable and stoppable by the user whose machine it runs on. It sits in the
paragraph opening "A background process must remain findable." Its own alternative is
restarting the daemon with each session, which the entry weighs as "restarting it
repeatedly is worse than leaving it up".

The Context sentence "In practice it was still a manual daemon start, a hand-written
settings file" is history under W-32 (a decision describes only the present). So is
the clause "which was the original objection to the wrapper, now answered", and the
sentence opening "The same fault was already made once, where the behaviour preflight
ran before the listeners". The sentence "The plugin surface is real and includes
`install`, `uninstall`, `update`, `validate`, `init`, and `marketplace`." is a
finding. D-121 (the repository is both the release host and the marketplace) changes
this entry in part, at `docs/decisions.md:6566` to `6571`: the install is two lines
rather than one, and "D-041's count is superseded; nothing else in it is". W-38
(a partial change rewrites the earlier entry) describes a rewrite that D-041 has not
had, and at `f11e871` it still states the one-line install. The paragraph opening
"Deliberately<!-- writing: quotes the decision
log --> not encoded: which surfaces this reaches." is a rejected alternative, a
surface matrix, or a limit of the first decision, and not a separate decision. The
Status line, "active (specified; not implemented)", is current state, and at
`f11e871` it no longer holds: the plugin and its session-start hook exist under
`plugin/`. The
entry has no Rejected. field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 6 lines outside the entry cite D-041. None is a test fixture that uses the
number as a string. Three mean the first decision, packaging or the install line:
`docs/decisions.md:2522`, `6566` and `6571`. The line at 2522, "nothing is packaged
yet (D-041)", sits in the Context of D-053 (`pair` is machine scope and `invite` is
room scope). Three mean the other decisions:

- `docs/decisions.md:4932` "D-041 forbids delaying a session or speaking to the person", which means the second decision
- `docs/decisions.md:6102` "D-041 (starting must not delay the session)", which means the second decision
- `docs/decisions.md:6651` "D-041 (ship as a plugin; the session-start hook starts the daemon) required the daemon to be discoverable and stoppable", which means the third decision

### D-042: Peer identity is an Ed25519 key pair; events are signed at origin

D-042 holds four decisions. The first is that a peer's identifier is its Ed25519
public key, rendered `ed25519:<base64url>`, not a fingerprint of it. It sits in the
Decision paragraph, which opens "Decision. A peer's identifier *is* its Ed25519 public
key", and in the paragraph opening "Why the identifier is the key rather than a
fingerprint of it." The second is that events are signed at origin over
length-prefixed fields with a leading purpose tag, and every receiver verifies each
one against the key its identifier names. It sits in the Decision paragraph from
"Events are signed at origin over a length-prefixed encoding", and in the paragraph
opening "Signing covers length-prefixed fields with a purpose tag." The entry weighs
its own alternative there: "Concatenating fields directly would let a boundary move".
The identifier could be a key while only requests were signed, or events could be
signed under a fingerprint-style identifier. The third is that the private key lives
in its own file, never in `identity.json`, in the paragraph opening "The private key
lives in its own file." Its own alternative is one identity file, which the entry
weighs as "an identity that cannot be shown without checking what else is in it". The
fourth is that an event with a failed signature is refused and logged, not
quarantined, in the paragraph opening "Rejection, not quarantine." Its own alternative
is quarantine, which D-027 (a sequence conflict is quarantined, not dropped) applies
to sequence conflicts.

The sentence "Verified live: a peer impersonating another and offering an event signed
by nobody was rejected" is a finding. The paragraph opening "What this does not do,
stated because the startup warning used to overclaim." is history under W-32 (a
decision describes only the present) in its account of Phase 10 and of the warning,
including "the warning now says exactly<!-- writing: quotes the decision log -->
this". Its statement that the entry gives integrity and attribution, and does not
decide who may connect, is a limit of the first two decisions. The paragraph opening
"Migration. An `identity.json` whose identifier is not the local key is rewritten" is
history. The Context paragraph, down to "§25 asked for signable identity and got a
random string.", is history too. The entry has no Rejected. field, which W-36 (a
decision always has Rejected.) requires.

At f11e871, 12 lines outside the entry cite D-042. None is a test fixture that uses
the number as a string; `cmd/cogmer/auth_test.go:62` and
`cmd/cogmer/membership_test.go:635` are comments. A thirteenth line,
`cmd/cogmer/pair.html:2`, lies outside the file types the search covers. Eleven mean
the first decision, "a peerId is a key" or that knowing an identifier grants nothing.
One attributes to D-042 words it does not contain:

- `docs/decisions.md:4884` "D-042 already calls it "a mnemonic for an identity already verified"". The words are in D-021 (peer names are word pairs derived from the identity) at `docs/decisions.md:724`, and in the specification at `Shared Claude Sessions.md:2011`.

`cmd/cogmer/offer.go:47` cites D-058 (signature schemes are kept, never replaced) for
"a captured offer could be replayed as something else", which is the purpose tag of
the second decision here.

### D-043: The wire format is defined separately from the stored row

D-043 holds four decisions. The first is that the wire format is its own type in its
own file, with explicit conversion to and from the stored row. It sits in the Decision
paragraph, which opens "Decision. Define the wire format in its own file, as its own
type", and in the paragraph opening "Why a separate type when the fields are currently
identical." The second is that a second host is prepared for only by naming, never by
machinery: the session field is renamed `originSessionId`, and there is no `source`
field, adapter architecture or second-host design. It sits in the Decision paragraph
from "Rename the session field to `originSessionId`", and in the paragraph opening
"What was deliberately<!-- writing: quotes the decision log --> not done. No `source:
claude-code | codex` field, no adapter architecture". Its own alternative, which the
entry rejects, is the argument that "cross-agent collaboration becomes nearly free
once events are normalised". One struct could serve wire and row while adapters were
refused, or the two could be separated and adapters built. The third is that every
sync exchange carries a protocol version, and a peer speaking a different one is
refused. It sits in the Decision paragraph from "Carry a protocol version in every
sync exchange and refuse a peer that speaks a different one.", and in the paragraph
opening "Why the version moved to v2." D-065 (the daemon reads a range of wire
versions) changed the refusal without changing the first decision, which shows the
third is separable. The fourth is that rooms are migrated on open, so a column added
later reaches rooms created earlier. It sits at the end of the paragraph opening "A
bug this surfaced.", from "Rooms are now migrated on open, and adding a column to that
list". The entry weighs no alternative for it; a schema version table and rebuilding a
room are the obvious ones. `CLAUDE.md` lists it among the choices this project has
made: "Migrate, do not orphan".

The Context paragraph, opening "Context. A suggestion that the daemon's database
schema should not become the protocol. We were half-violating it", is history under
W-32 (a decision describes only the present). The paragraph opening "Why now. No room
existed that anyone would mind losing" is history. The paragraph opening "A bug this
surfaced. `CREATE TABLE IF NOT EXISTS` creates a table and then ignores it forever" is
a finding, up to "An existing room failed with *no such column: signature*". The entry
has no Rejected. field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 10 lines outside the entry cite D-043. None is a test fixture that uses
the number as a string. One means the first decision: `docs/decisions.md:3277` "D-043
(the wire format is not the database row) keeps the two structs separate precisely<!--
writing: quotes the decision log --> so a column added for local bookkeeping cannot
become protocol". Nine mean the second decision, no adapter machinery for a second
host. One of them, `docs/decisions.md:5831`, names the first decision in its few words
while meaning the second. The nine are:

- `CLAUDE.md:127` "No adapter machinery for a second host (D-043, D-113)"
- `CLAUDE.md:238`, the reading-map row "capture, reassembly, injection"
- `docs/decisions.md:2857` "`originSessionId` rather than `claudeSessionId` (D-043)"
- `docs/decisions.md:4367` "D-043 still holds: no adapter machinery"
- `docs/decisions.md:4369` "The argument in D-043 was that capture generalises and injection does not"
- `docs/decisions.md:5814` "alters neither D-043 nor D-086"
- `docs/decisions.md:5831` "D-043 (the wire format is not the database row) forbids `source`/adapter machinery"
- `docs/decisions.md:5848` "the ones D-043 named as non-generalising"
- `docs/decisions.md:5966` "without naming the agent (D-043)"

### D-044: Sync requests are signed; authentication is not admission

D-044 holds one decision. It is that every sync request carries the caller's
identifier, a timestamp and a nonce, signed with a purpose tag, and is refused outside
a two-minute window or when the nonce has already been seen. It sits in the Decision
paragraph, which opens "Decision. Every sync request carries the caller's identifier".
"Authentication is not admission" is this decision's limit, not a second decision.
Signing requests rather than holding a session, the two-minute window and the unsigned
`have` map are parameters of the same choice.

The measurement "two NTP-synced machines measured 408ms apart", in the paragraph
opening "Two minutes of tolerance", is a finding. The sentence "A stranger generated a
key pair, authenticated correctly, and read a private room - while the host logged
nothing" is a finding, and the limit it shows is part of the decision. The paragraph
opening "So what Phase 9 delivers is the ability to make an admission decision, not
the decision. The guest list is Phase 10, and until it exists the startup warning" is
history under W-32 (a decision describes only the present): the guest list it
describes as not yet built is D-045 (rooms are records with a guest list). The Context
paragraph, opening "Context. Phase 9's third part.", is history too. The entry has no
Rejected. field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 14 lines outside the entry cite D-044. None is a test fixture that uses
the number as a string; `cmd/cogmer/membership_test.go:40` is a comment. Twelve mean
the decision, and two of those, `cmd/cogmer/membership_test.go:40` and
`cmd/cogmer/membership.go:753`, mean its limit that a caller who authenticates
correctly may still be refused. Two mean the stranger finding:

- `cmd/cogmer/auth.go:132` "what let a stranger with a freshly generated key read a private room (D-044)"
- `docs/decisions.md:1991` "D-044 left the confidentiality gap open and said so: a stranger generated a key, authenticated correctly, and read a private room", in the Context of D-045

### D-045: Rooms are records with a guest list; admission is enforced

D-045 holds two decisions. The first is that a room is a record in `membership.db` (a
UUID, a generated name and a guest list), and that a sync request is refused unless
its authenticated peer is a guest. It sits in the Decision paragraph, which opens
"Decision. Rooms become records rather than arbitrary strings" and ends "A sync
request is refused unless its authenticated peer is a guest." The second is that only
a key can be admitted: `allow` and `invite` refuse an identifier that names no key. It
sits in the paragraph opening "Only keys can be admitted." Its own alternative is
recording `alice` or an old `peer-8f3a…` identifier, which the entry calls "recording
a hope". Admission could be enforced while names were still accepted.

The paragraph opening "Verified by repeating the test that failed. The same uninvited
stranger now reads nothing" is a finding. The paragraph opening "Two scopes, as §12
requires. `known_peers` is machine-wide" restates D-025 (the guest list specified: two
scopes, a signed challenge, and approval in the moment) and D-024 (admission is a
guest list), including `forget` and `revoke`. D-053 (`pair` is machine scope and
`invite` is room scope) notes that the two lists date from D-024. The paragraph
opening "The out-of-band step is a public key, and the tooling says so. `allow` prints
the full fingerprint" is text later decisions reversed: D-053 has `allow` print
UNVERIFIED, and D-055 (one way to verify a peer) removed fingerprint comparison. The
paragraph opening "A bridge, noted as such. A daemon pointed at a room nobody created
makes one" is history under W-32 (a decision describes only the present); D-046 (the
daemon serves many rooms) records the bridge as gone. The Context paragraph, opening
"Context. Phase 10. D-044 left the confidentiality gap open", is history too. The
entry has no Rejected. field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 6 lines outside the entry cite D-045. None is a test fixture that uses the
number as a string. All six mean the first decision, that a request is refused unless
its peer is a guest.

### D-046: The daemon serves many rooms; a session says which one it is in

D-046 holds four decisions. The first is that the daemon is the machine's local
service, not a room, and opens a store per room on demand. It sits in the Decision
paragraph, which opens "Decision. The daemon is the machine's local service, not a
room." The second is that a session is in no room until it binds to one, and binds at
first sight, so that a session in no room is an ordinary Claude Code session. It sits
in the Decision paragraph from "a session binds to a room on first sight. A session in
no room is an ordinary Claude Code session", and in the paragraph opening "Binding at
first sight rather than asking". Its own alternative is asking, which the entry
rejects because "there is nobody to ask at that moment". A daemon serving many rooms
could assign them by another rule. The third is that an invitation carries the room's
name, its id, where to reach it, and the inviter's identifier, and that joining admits
the inviter. It sits in the sentences "An invitation now carries the room's name, its
identity, and where to reach it" and "An invitation now carries the inviting peer's
identifier, and joining admits them." Its alternative is a token, which D-026 (there
is no join token at all) excludes. The fourth is that a peer advertises the address it
listens on, signed. It sits in "so a peer now advertises where it listens, signed,
because a peer acts on that address by polling it and an unsigned one would redirect
polling." Its own alternative is an unsigned address.

The sentence "A machine-level *current room* answers instead: `join` sets it, and
sessions started afterwards enter it." is text a later decision reversed: D-080 (there
is no current room) removed the current room. The sentence opening "Once a session has
been offered teammate context it is marked" is reversed as well: D-056 (a session's
room is fixed at first sight) removed the mark. The paragraph opening "Two gaps only
the end-to-end test exposed." is a finding in its observed failures, "*A guest knew no
room existed.*" and "*Synchronisation was one-way.*", while the third and fourth
decisions sit inside it. The sentence "a guest list is per-peer, so two peers can
disagree about who belongs." is a finding. The Context paragraph, opening "Context. A
daemon served exactly<!-- writing: quotes the decision log --> one room", is history
under W-32 (a decision describes only the present), and so is "the bridge is gone, not
because it was removed but". The paragraph opening "What this makes possible that was
not before." is consequence and history. The entry has no Rejected. field, which W-36
(a decision always has Rejected.) requires.

At f11e871, 9 lines outside the entry cite D-046. None is a test fixture that uses the
number as a string. Four mean the first decision: `cmd/cogmer/main.go:228`,
`docs/phase5-findings.md:80` and `85`, and `docs/decisions.md:2959`. Two mean the
first and second decisions together: `docs/spec-review.md:619` "opening a store per
room on demand, and a session binds to a room on first sight (D-046)", and
`CLAUDE.md:239`, the reading-map row "rooms, membership, session binding". D-016 (one
room per session) and D-056 also hold the binding rule. Three mean the per-peer
guest-list finding:

- `docs/decisions.md:3082` "the defect behind the asymmetric guest list (D-046)"
- `docs/decisions.md:3758` "Guest lists are per-peer (D-046), so two peers can already disagree"
- `docs/decisions.md:5780` "guest lists are per-peer (D-046, the daemon serves many rooms)", whose few words name the first decision

### D-047: The fingerprint is the only manual link, and had the least careful encoding

D-047 holds one decision, and it no longer stands. It rendered the fingerprint as
words, in the paragraph opening "Decision at the time. Render the fingerprint as
words." The Status line reads "CLOSED, not implemented - the construction it argued
about was removed".

D-055 (one way to verify a peer) reversed the decision, so under W-37 (a reversed
decision becomes a tombstone) this entry is a reversed decision. The sentence
"`Fingerprint` survives as a display" repeats D-055's "What is kept." The rest of the
entry is findings and history. The substitution demonstration, "an attacker
substituted her own identifier in transit" ending "Zero refusals.", is a finding, and
it is the fact D-054 (verification gates sync) rests on. D-055 (one way to verify a
peer) cites the reading finding below in the second of its three reasons, the paragraph
opening "It is a downgrade path.". The grinding measurements are a finding: "three
characters fell in 339,297 tries and under a second; four take about thirty seconds;
eight are 2^48", and sixteen are 2^96. The sentence "Show someone forty-three
characters of base64 to check over a telephone and they will read the first group, the
last group, and skim the middle" is a finding. So is the base64url dictation hazard,
in which `l`, `I` and `_` are confused when spoken, and the paragraph opening "The
general point worth keeping. The weakest link in this system is a human reading a
string". The paragraph opening
"Closed without implementing it (2026-09-18)." is history under W-32 (a decision
describes only the present). The entry has no Rejected. field, which W-36 (a decision
always has Rejected.) requires.

At f11e871, 11 lines outside the entry cite D-047. None is a test fixture that uses
the number as a string. Three mean the decision: `docs/decisions.md:2172` "D-047
remains held" and `docs/decisions.md:2242` "D-047 is held, not cancelled", both
history inside D-048 (verification binds a live exchange), and
`docs/decisions.md:2677` "D-047 is closed without being implemented", history inside
D-055. Eight mean other parts:

- `cmd/cogmer/auth.go:141` "every check above passes just as well for whoever substituted it in transit (D-047)", which means the substitution finding
- `docs/decisions.md:2179` "which D-047 demonstrated end to end with zero refusals", which means the substitution finding
- `docs/decisions.md:2214` "(D-047 measured it)", which means the grinding finding
- `docs/decisions.md:2587` "D-047 demonstrated exactly<!-- writing: quotes the decision log --> that end to end with zero refusals", which means the substitution finding
- `docs/decisions.md:2653` "D-047 measured the cost: shown forty-three characters", which means the reading finding
- `docs/decisions.md:2265` "the full-length comparison of D-047 is the only option", the Revisit when clause of D-048 (verification should bind a live exchange), which D-055 contradicts
- `docs/decisions.md:2510` "D-047's full-length rendering is the only option", the Revisit when clause of D-052 (the SAS is its own act), which D-055 contradicts
- `CLAUDE.md:236`, the reading-map row "pairing, verification, the two words", which means the findings

### D-048: Verification should bind a live exchange, not a standing identifier (ZRTP's SAS)

D-048 holds one decision. It is that verification is a live commit-then-reveal
exchange that derives two words from both identity keys and both fresh nonces, which
both users compare aloud on a call, and that a mismatch is conspicuous and never
presented as worth retrying. It sits in the Decision paragraph, which opens "Decision.
Adopt the live-exchange form", and in the paragraph opening "The hazard to implement
against."

The paragraph opening "This entry placed that at join" is a placement later decisions
reversed, and under W-38 (a partial change rewrites the earlier entry) it is history
in this entry: "D-052 rejected the placement" and "D-053 then moved it to pairing",
where D-053 is the decision that `pair` is machine scope and `invite` is room scope.
The Status line's "its placement was superseded" and "D-047 remains held" are the same
history. Consequence 3, opening "D-047 is held, not cancelled.", is history: it says a
`verified` flag "today does not exist anywhere in `membership.go`", while D-052 (the
SAS is its own act) added `known_peers.verified_at` and D-055 (one way to verify a
peer) closed D-047. The paragraph opening "What it corrects here. §25 said" is history
in its account of the old §25 (security) and the sentence "§25 now says so"; its
distinction between offline precomputation and one online guess is support. The
paragraph opening "What ZRTP does." states an external construction with no URL or
findings document, which W-34 (every supporting fact names a source) requires of a
supporting fact. The Revisit when clause, "so both renderings may need to exist", is
contradicted by D-055. The entry has no Rejected. field, which W-36 (a decision always
has Rejected.) requires.

At f11e871, 4 lines outside the entry cite D-048. None is a test fixture that uses the
number as a string. Three mean the decision: `cmd/cogmer/sas.go:12`,
`cmd/cogmer/main.go:1070`, and `cmd/cogmer/membership.go:31`, which cites
"(D-048/D-052)". One means the placement at join:

- `docs/decisions.md:2439` "D-048 accepted ZRTP's short authentication string in principle and put it "at join, where both daemons are connected."", in the Context of D-052

### D-049: The sync request addresses a room by id, never by name

D-049 holds one decision. It is that a room is named by `roomId` on the wire in both
directions, and that peer-supplied identifiers resolve only through `RoomByID`; it
sits in the Decision paragraph, which opens "Decision. `roomId` on the wire". The
`wireVersion` and signing-tag bump is a consequence of it.

The paragraph opening "What the collision actually costs - the first reading was
wrong." is a finding. The sentence "That bug is not fixed by this entry and is
recorded here so it is not mistaken for fixed." is history under W-32 (a decision
describes only the present); D-050 (room names may collide locally) fixed the bug. The
Context paragraph, opening "Context. Asked whether the room creator's daemon signs its
polling requests", is history. The entry has no Rejected. field, which W-36 (a
decision always has Rejected.) requires.

At f11e871, 3 lines outside the entry cite D-049. None is a test fixture that uses the
number as a string. All three mean the decision.

### D-050: Room names may collide locally; the schema stops forbidding it

D-050 holds two decisions. The first is that room names may collide locally, and the
uniqueness constraint on `rooms.room_name` is dropped. It sits in the Decision
paragraph, which opens "Decision. Drop the constraint.", and the title's semicolon
joins two halves of this one choice. Its alternative sits under "What was not done.",
which is not a Rejected. field: "Renaming a joined room locally to keep names unique
was the obvious alternative and is worse". The second is that an ambiguous name is
reported with both identities rather than guessed at or answered as unknown. It sits
in the item "`FindRoom` reports three outcomes, not two." The entry names its own
alternatives in that item: answering "no such room", and "returning whichever row came
back first". Collisions could be allowed while one row was picked silently.
`docs/decisions.md:3806` treats the second as a pattern of its own.

The Context paragraph, opening "Context. Found while making the wire address rooms by
id (D-049). `rooms.room_name` was `NOT NULL UNIQUE`", is history under W-32 (a
decision describes only the present) through "`runJoin` called `log.Fatalf`". The
paragraph opening "Not hypothetical at any real scale. Names are drawn from 7,656
combinations" is a finding, ending "Around a hundred rooms over a machine's lifetime
makes it a coin flip." The items "`CreateRoom` asks whether a name is free" and
"`migrateMembership` rebuilds the table" are consequences of the first decision in its
implementation, not decisions. The entry has no Rejected. field, which W-36 (a
decision always has Rejected.) requires.

At f11e871, 4 lines outside the entry cite D-050. None is a test fixture that uses the
number as a string. One means the first decision, `docs/decisions.md:4606` "D-050 says
names may collide and identities do not". One means both: `docs/spec-review.md:118`
"D-050 lets names collide locally and reports the ambiguity rather than guessing at
it". Two mean the second decision:

- `docs/spec-review.md:673` "an ambiguous name is reported with both identities rather than guessed at (D-050)"
- `docs/decisions.md:3806` "Same shape as D-050: report the ambiguity, name both"

### D-051: Stranger pairing is not a supported case

D-051 holds one decision. It is that pairing with someone unknown is not a design
target: no affordance presents it as intended, and no claim is made that verification
protects it. It sits in the Decision paragraph, which opens "Decision. Exclude
stranger pairing as a design target". That whom to admit remains the host's judgement
is this decision's limit, stated in the paragraph opening "What is not decided. The
mechanism does not forbid it", and not a second decision.

The paragraph opening "Also not decided: whether the host-approval path is built at
all" is an open question. The paragraph opening "Recorded as open in §12a: whether a
request may arrive unsolicited" is an open question too, together with the window in
which the host is expecting someone; §12a is room membership. The sentence opening
"Noted because the two were previously argued as one" is history under W-32 (a
decision describes only the present), and so is the Context paragraph's account of the
earlier wording of §12a. The entry has no Rejected. field, which W-36 (a decision
always has Rejected.) requires.

At f11e871, 6 lines outside the entry cite D-051. None is a test fixture that uses the
number as a string. Five mean the decision or its limit. One means the open question
about unsolicited requests:

- `docs/open.md:31` "a host approving an unsolicited join request - the second undecided rather than pending (§12a, D-051)". It sits in `docs/open.md` and cites §12a beside the number.

### D-052: The SAS is its own act, not part of joining or approving

D-052 holds four decisions. The SAS is the short authentication string, the two words
both users compare. The first decision is that verification is its own command,
`cogmer verify <peer>`, run by both users at once on a call and separate from joining
and from host approval. It sits in the Decision paragraph, which opens "Decision.
`cogmer verify <peer>`, run by both people at the same time", and in the paragraph
opening "The two are orthogonal." The second is that a daemon takes part in a
verification only when "its own user has asked for one", and otherwise answers 409 and
displays nothing, so no inbound request creates anything. It sits in the paragraph
opening "Both sides run it, and that is what keeps §12a's open question closed." Its
own alternative is an inbound verification request that prompts the host, which the
entry rules out in the words "no prompt that can be trained away". Verification could
be its own act
while inbound requests were accepted. The third is that only a person's confirmation
records a verification (`MarkVerified`), a mismatch records nothing, and a verified
peer loses the `unverified` marker, the tag injected text carries for a peer not yet
verified. It sits in the paragraphs opening "Only a person may record a verification."
and "The marker now means something." Its own alternative is a failed-verification
state: "storing one invites an interface that offers to retry". The fourth is that the
two words come from the PGP biometric word list, alternating even and odd, and
disjoint from the peer-name and room-name vocabularies. It sits in the paragraph
opening "Wordlists." Its own alternative is reusing those vocabularies, where "one
mnemonic mistakable for the other is how somebody compares the wrong thing".

The paragraph opening "The exchange is symmetric - true of a single round, and not of
the session around it, which D-059 had to correct" is history under W-32 (a decision
describes only the present); D-059 is the decision that only one side drives a
verification. The address rule in that paragraph, "keeping the one that answers signed
by the identity asked for", is part of the first decision. The sentence opening
"Before this the tag was true of every peer forever" is history. The Revisit when
clause, "D-047's full-length rendering is the only option, so both may need to exist",
is contradicted by D-055 (one way to verify a peer). The Context paragraph, opening
"Context. D-048 accepted ZRTP's short authentication string in principle", is history.
The paragraph opening "Not done. The words are 16 bits", which includes "Nothing
rate-limits attempts.", is a limit. The entry has no Rejected. field, which W-36 (a
decision always has Rejected.) requires.

At f11e871, 9 lines outside the entry cite D-052. None is a test fixture that uses the
number as a string. Four mean the first decision, the separate act:
`docs/decisions.md:2144`, `2171`, `2228` and `2231`. Five mean other parts:

- `docs/decisions.md:4034` "(D-052, §29)" for "interactive, blocking on another person, and the compared words must reach a person's eyes unaltered", which D-052 does not state; those words are in D-053 (`pair` is machine scope and `invite` is room scope)

- `cmd/cogmer/daemon.go:82` "held only while a person has asked for one. Nothing here is created by an incoming request (D-052)", which means the second decision
- `cmd/cogmer/daemon.go:499` "a peer verified over a recognising channel (D-052) is not marked", which means the third decision
- `cmd/cogmer/membership.go:31` "When two people compared a SAS and said it matched (D-048/D-052)", which means the third decision
- `docs/decisions.md:2891` "D-052 described the exchange as symmetric", which means the symmetry text, and is itself history inside D-059

### D-053: `pair` is machine scope and `invite` is room scope; the commands now say so

D-053 holds three decisions. The first is that `cogmer pair` is the machine-scope act:
it records the peer and a bootstrap address and runs the two-word comparison in one
command, so pairing precedes any room, while `invite` stays room-scoped. It sits in
the Decision paragraph, which opens "Decision. `cogmer pair <identifier>[@address]
[name]` is the durable act", and in the paragraph opening "The ordering gap this
closes." The second is that `allow` is kept as a low-level way to record a peer
without verifying, for scripts and tests, and prints UNVERIFIED. It sits in the
Decision paragraph from "`allow` survives as the low-level "record without verifying"
for scripts and tests". Its own alternatives are removing `allow`, or keeping its old
fingerprint output. `pair` could be added and `allow` deleted. The third is that an
interactive act ending in something a person must read unaltered (pairing, verifying)
runs at a terminal, while a non-interactive room act (inviting) may be a slash
command. It sits in the paragraph opening "Why the naming matters more than it looks.
The split decides where each act can live". Its own alternative is every act at a
terminal, "what "commands are typed at a terminal" had quietly become". D-057 (a slash
command is a thin wrapper over the CLI) cites this as what §29 (the experience we
want) records, a split of commands between a session and a terminal.

The Status line's "except its gating position, reversed by D-054 the same day" and the
paragraph opening "Gating - see D-054. The warnings this entry added" record a part
that D-054 (verification gates sync) reversed. Under W-38 (a partial change rewrites
the earlier entry), D-053 is an entry a later decision changed in part, and its Status
line records the reversal. The paragraph opening "§12 is retitled from "Session
Pairing" to "Forming a Room"" is history under W-32 (a decision describes only the
present). The sentence opening "`verify` previously found a peer only through
addresses learned from room membership" is history, and so is "the sequence is now
pair → create → invite → join rather than create → invite → join → verify". The
Context paragraph, opening "Context. Asked why the commands are typed at a terminal",
is history, including "The … answer was that nothing is packaged yet". The entry has
no Rejected. field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 9 lines outside the entry cite D-053. None is a test fixture that uses the
number as a string; `cmd/cogmer/ui_test.go:206` is a comment. Six mean the first
decision, pairing as the machine-scope act: `docs/decisions.md:2171`, `2230`, `2231`,
`3005`, `4222`, and `CLAUDE.md:240`, the reading-map row "identity, keys, admission".
Three mean other decisions:

- `cmd/cogmer/ui_test.go:206` "Recorded with no label of its own, as a script would (D-053)", which means the second decision
- `cmd/cogmer/main.go:645` "It used to name `allow`, which D-053 reserves for scripts and tests", which means the second decision
- `docs/decisions.md:2748` "§29 had already split commands between a session and a terminal (D-053)", which means the third decision

### D-054: Verification gates synchronization and injection, not just a marker

D-054 holds one decision. It is that an unverified peer's events are not served, not
stored (judged at their origin), and not injected, and that they are held rather than
dropped. It sits in the Decision paragraph, which opens "Decision. Three gates,
because there are three ways in:". The paragraphs opening "Held, not discarded.",
"Silence had to be explained." and "Inviting an unverified peer remains permitted and
remains inert" describe properties and limits of the same gates.

The Context paragraph, opening "Context. Asked whether we plan to admit unverified
guests. We did - by omission", is history under W-32 (a decision describes only the
present), including "`IsVerified` was consulted in exactly<!-- writing: quotes the
decision log --> three places". The paragraph opening "Why the previous position did
not hold. It rested on D-051" is history. The fact it rests on, that every check
passes for a key substituted in transit, is support, and it comes from the
substitution demonstration in D-047 (the fingerprint is the only manual link), which
records that a swapped key passed every check with zero refusals. The sentence opening
"The existing `authDaemon` fixture now verifies its guest, which is itself evidence
the gate bites" is a finding. The entry has no Rejected. field, which W-36 (a decision
always has Rejected.) requires.

At f11e871, 31 lines outside the entry cite D-054. None is a test fixture that uses
the number as a string; `cmd/cogmer/peertls_test.go:61`, `cmd/cogmer/ui_test.go:240`
and `cmd/cogmer/offline_test.go:74` are comments. `CLAUDE.md:30` and `31`, and
`docs/writing.md:47`, use the number as an example of a reference with its few words,
"D-054 (verification gates sync)". All 31 mean the decision.

### D-055: There is exactly<!-- writing: quotes the decision log --> one way to verify a peer

D-055 holds one decision. It is that the live two-word comparison is the only way to
verify a peer, with no whole-key fallback, and that `Fingerprint` is for display only.
It sits in the Decision paragraph, which opens "Decision. One ceremony." The paragraph
opening "What is kept.", which keeps `Fingerprint` as a display, is this decision's
limit.

The paragraph opening "It did not exist. `Fingerprint` had no callers outside its own
test" is a finding and history. The sentence "Consequence. D-047 is closed without
being implemented." is history under W-32 (a decision describes only the present). The
Context paragraph, opening "Context. §25 retained the whole-key comparison as a
fallback" and containing "Asked why.", is history. The entry has no Rejected. field,
which W-36 (a decision always has Rejected.) requires.

At f11e871, 13 lines outside the entry cite D-055. None is a test fixture that uses
the number as a string. Twelve mean the decision. One means the finding that
`Fingerprint` had no callers:

- `docs/decisions.md:2726` "represented a decision recorded but never wired up (D-055)", inside D-056 (a session's room is fixed at first sight)

### D-056: A session's room is fixed at first sight; the `injected` flag is removed

D-056 holds two decisions. The first is that a session binds to a room at first sight
and never moves, with no exception for a session that has received nothing. It sits in
the Decision paragraph, which opens "Decision. Keep the strict rule and delete the
machinery for the loose one. A session binds on first sight and stays." It overlaps
D-016 (one room per session), which `CLAUDE.md` cites for the same rule. The second is
that a column encoding a removed rule is dropped by migration, not left in place. It
sits in the paragraph opening "The column is dropped, not left." The entry weighs its
own alternative there: "It would have been harmless: it has a default and nothing
writes it". The strict rule could be kept while the column was left.

The sentence "`injected`, `MarkInjected` and `HasReceivedContext` are gone" is history
under W-32 (a decision describes only the present), and so is the title's second
clause, "the `injected` flag is removed". The Context paragraph, opening "Context.
Initialized CodeGraph and ran a dead-symbol sweep", is history, and so is the
paragraph opening "The specification was therefore looser than the code". The
paragraph opening "What the sweep says about the method." is a finding, including "A
symbol with no callers is worth treating as a question". The entry has no Rejected.
field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 4 lines outside the entry cite D-056. None is a test fixture that uses the
number as a string. One means the first decision, `docs/spec-review.md:680` "keeping
the strict behaviour and deleting the flag (D-056)". Three mean other parts:

- `cmd/cogmer/membership.go:165` "a column encoding a rule that was removed is a rule somebody will later find and reinstate (D-056)", which means the second decision
- `cmd/cogmer/membership.go:78` "this one was written and never read (D-056)", which means the history of the flag
- `docs/decisions.md:2925` "the third such column found this week, after `injected` (D-056)", which means the dead-symbol finding

### D-057: A slash command is a thin wrapper over the CLI; the session names itself

D-057 holds three decisions. The first is that a slash command shells out to the
corresponding CLI command, with one implementation and two entry points. It sits in
the Decision paragraph, which opens "Decision. Slash commands shell out." The second
is that a session-scoped command learns its session from `CLAUDE_CODE_SESSION_ID`, and
without it refuses rather than falling back to a machine-level setting, with an
explicit override for tests. It sits in "They pass no session id, because the CLI
reads `CLAUDE_CODE_SESSION_ID`" and in "The variable is absent at a terminal, so a
session-scoped command run there has no session to bind and refuses." Its own
alternatives are passing the session id explicitly, and a machine-level current room,
which Revisit when rejects. Thin wrappers could pass the id as an argument. The third
is that a command that cannot be wrapped (pairing, verifying) has a slash counterpart
that tells the user to run it in a terminal. It sits in the paragraph opening "Not
every command can be wrapped", with "Their slash counterparts print an instruction to
run them in a terminal." and "A signpost is … in a way a proxy would not be." Its own
alternative is a proxy.

The paragraph opening "The objection that had blocked this was wrong." holds a finding
that B21 (`CLAUDE_CODE_SESSION_ID` is exported into a tool call's environment)
records: "`CLAUDE_CODE_SESSION_ID` is exported into the environment of every Bash tool
call", "Verified on 2.1.275". The paragraph opening "What it allows us to delete. The
machine-level *current room*" restates what D-080 (there is no current room) decides.
The Context paragraph's "the assumption underneath was that they would need a second
implementation" is history under W-32 (a decision describes only the present), and so
is "The objection that had blocked this was wrong." The item opening "The model may
retry." is an open question: it offers "Either make creation idempotent per session"
or "have it refuse", and the entry leaves the choice open. The Status line, "decided;
commands not yet built", is current state, and at `f11e871` it no longer holds:
`plugin/commands/` holds the commands. The entry has no Rejected.
field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 1 line outside the entry cites D-057. It is not a test fixture. It means
the first decision: `Shared Claude Sessions.md:2744` "A slash command is a thin
wrapper, never a reimplementation (D-057)".

### D-058: Signature schemes are kept, never replaced

D-058 holds one decision. It is that an event records the signature scheme it was
signed under and is verified under that scheme, that a new scheme is added beside the
old ones, that an
old one is never edited, and that an unknown scheme is refused as a version problem.
It sits in
the Decision paragraph, which opens "Decision. An event records the scheme it was
signed under". "Zero means v2", "The version is not covered by the signature" and "An
unknown scheme is refused as a version problem, not a forgery" are details of its
mechanism, not separate decisions.

The sentence "`signingBytes()` hard-coded a single event tag and a fixed field list,
and `Verify()` always recomputed with today's code." is history under W-32 (a decision
describes only the present). The paragraph opening "The related hazard, recorded and
not fixed. `wireVersion` is a hard refusal" is history; D-065 (the daemon reads a
range of wire versions) resolved the hazard. The paragraph opening "What was already
right, and worth not disturbing: `originSessionId`" is history and support, and it
restates D-043 (the wire format is defined separately from the stored row). The
Context paragraph, opening "Context. Asked whether anything in the current approach
would make backwards compatibility hard", is history. The entry has no Rejected.
field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 10 lines outside the entry cite D-058. None is a test fixture that uses
the number as a string. Nine mean the decision. One means a rule of another entry:

- `cmd/cogmer/offer.go:47` "Two messages that mean different things must not be interchangeable under one signature, or a captured offer could be replayed as something else (D-058)", which means the purpose-tag rule of D-042 (peer identity is an Ed25519 key pair), its second decision

### D-059: Only one side drives a verification; the other completes from inbound

D-059 holds two decisions. The first is that either side completes a verification from
inbound once it holds the other's revealed nonce, so only one side has to drive the
exchange. It sits in the Decision paragraph, which opens "Decision. Either side may
complete from inbound." The second is that the verification endpoint answers 409 for a
peer it does not know yet and keeps 401 for a signature that did not verify. It sits
in the paragraph opening "Fixed by answering 409 for "I do not know you yet"", which
continues "401 now means only that a signature did not verify". Its own alternative is
401 for an unknown peer, treated as final. The stranding could be fixed while 401 was
kept for unknown peers, or the reverse.

The Context paragraph, opening "Context. Phase 5's two-machine run. Pairing failed
twice", is a finding, and so are the paragraphs opening "First: whoever typed first
lost." and "Second, and deeper: the side that finished stranded the other." The
paragraph opening "D-052 described the exchange as symmetric" is history under W-32 (a
decision describes only the present); D-052 is the decision that the SAS is its own
act. The paragraph opening "Why a test did not catch it.
`TestTwoDaemonsReachTheSameWords` starts both sides in goroutines with no delay" is a
finding. The entry has no Rejected. field, which W-36 (a decision always has
Rejected.) requires.

At f11e871, 1 line outside the entry cites D-059. It is not a test fixture. It means
the first decision: `docs/decisions.md:2463` "which D-059 had to correct after a
two-machine run".

### D-060: A sequence is reserved outside the room before the event that uses it

D-060 holds two decisions. The first is that a sequence is reserved and recorded in
`membership.db` before the event that uses it is published, and that `Store.Append`
takes the sequence rather than deriving one. It sits in the Decision paragraph, which
opens "Decision. `Membership.ReserveSequence` issues the number", and in the paragraph
opening "Reserve, then publish, and not the reverse." Keeping the sequence outside the
room and reserving it before publishing are D-029's design (losing a room database
does not end membership), stated in D-029's paragraph "Reserve before publishing."
This entry adds the API: `Membership.ReserveSequence` issues the number and
`Store.Append` takes it. The second is that losing a room's state is reported when the
room is opened. It sits in the paragraph opening "The loss is reported, because it is
otherwise invisible. `reportLostState` runs when a room is opened". Its own
alternative is recovering silently, which the paragraph rejects as "otherwise
invisible". §8 (event identity and ordering) requires the report, and choosing the
moment the room is opened is this entry's. Sequences could be reserved while recovery
stayed silent.

The Context paragraph, opening "Context. Review C-1", with "D-029 settled the design a
fortnight ago" and "None of it was built.", is history under W-32 (a decision
describes only the present). So is the paragraph opening "`rooms.issued_sequence`
existed in the schema and was written by nobody". The paragraph opening "What that
cost." is a finding, including "Reproduced during the review: the daemon kept serving
from its open file handle after the file was deleted". The paragraph opening "Also
fixed, from the Phase 5 findings. `log`, `conflicts` and `seed` resolved a room
through `config.json`" is history of a separate fix, unrelated to either decision. It
says the commands "now use the current room from `membership.db`", a current room that
D-080 (there is no current room) removed, and it ends with "`whoami` deliberately<!--
writing: quotes the decision log --> does not". The paragraph opening "Recovery needed
no new mechanism." is support for the first decision. The entry has no Rejected.
field, which W-36 (a decision always has Rejected.) requires. The paragraph "Reserve,
then publish, and not the reverse" names the reverse order as the alternative to the
first decision.

At f11e871, 2 lines outside the entry cite D-060. Neither is a test fixture. Both mean
the first decision, and both also cover the second: `docs/spec-review.md:588` "See
D-060. Sequences are reserved in `membership.db` before the event that uses them"
continues "The loss is reported when the room is opened", and `CLAUDE.md:244` is the
reading-map row "sequences, recovery from local loss".

### D-061: Phase 7's last three: one dissolved, two built

D-061 holds four decisions. The first is that there is no outbound queue, because
synchronization is a pull and the event store is the only record of what a peer has
not fetched. It sits in the section headed "The outbound queue is dissolved, not
deferred". The second is that a peer outage is reported on each transition: when a
peer stops answering, every ten minutes while it stays gone, and once when it returns
with how long it was gone. It sits in the section headed "Peer health: report
transitions, not polls", and its own alternative is logging once per poll. The third
is that injected context has a character budget for the whole block, applied after the
event cap and measured on rendered turns, that it drops from the front, that all
limits are configurable, and that a zero or unparsable limit is ignored. It sits in
the section headed "Context size: the limit that was missing bounded nothing", from "A
whole-block budget now applies after the event cap". Its own alternative is the
per-turn and per-count caps alone. The fourth is that the token count is estimated
from characters at four to one, not measured with a tokenizer. It sits in the
paragraph opening "On estimated token count", in "derived from characters at four to
one rather than measured." Its own alternative is a tokenizer: "A tokenizer would have
to track a model this system does not choose". The title, "Phase 7's last three",
names a topic rather than a decision, and W-31 (a decision's title states the
decision) asks that a title state the decision.

The sentence "Recorded in §7 as struck through with the reason, rather than removed."
is history under W-32 (a decision describes only the present). The struck-through queue
is at `Shared Claude Sessions.md:2686`, in Phase 7 of §31 (implementation order), and
not in §7 (event model).
The clause "which was observed filling a log through the Phase 5 partition" is a
finding. The paragraph opening "Also fixed: `peerStatus` enumerated only
`COGMER_PEERS`, so every peer learned by pairing or by joining a room was invisible in
the browser view" is history. The sentence "Forty turns of eleven thousand characters
pass both. Measured: 404,635 characters" is a finding. The paragraph opening "§21 is
amended to say what each limit is *for*" is history; §21 is context window management.
The Context paragraph, from "Phase 7 listed a local outbound queue" to "produced one
deletion and two findings.", is history. The entry has no Rejected. field, which W-36
(a decision always has Rejected.) requires.

At f11e871, 1 line outside the entry cites D-061. It is not a test fixture. It sits in
the peer-health section's code and means the `peerStatus` fix, which is history rather
than a decision:

- `cmd/cogmer/ui.go:296` "peers learned by pairing or by joining a room were invisible here until D-061, which is most of them"

### D-062: Tailcat evaluated for Phase 15: a good fit, adopted behind an interface if at all

D-062 holds two decisions. Phase 15 is cross-network reach. The first decision is that
tailcat, if used, sits behind a narrow interface of our own (a Dial and a Listen) and
decides nothing about who may speak. It sits in the title's "adopted behind an
interface if at all", in "A tailcat connection carries bytes and decides nothing.",
and in "… code behind an unstable API is why it should sit behind a narrow interface
of our own - a Dial and a Listen". Adopting it is D-068 (tailcat is the cross-network
transport, behind our own dialer), as the Status line says. The second is that
tailcat's `AllowedClients` is not used, so admission has one authority. It sits in the
paragraph opening "Do not use `AllowedClients`." Its own alternative is a third
allowlist keyed on the WireGuard key. Tailcat could be adopted behind an interface
while `AllowedClients` was used.

The Status line's "Was: investigated, not yet adopted, but the likely answer rather
than a contingency" is history under W-32 (a decision describes only the present).
Most of the entry is a finding: the paragraphs opening "What it is.", "Why it fits
this design unusually well. The API is `net.Conn`-shaped", and "Measured, not
assumed." with the size table, "526 dependencies against our current one", and the
list opening "Three risks to weigh at Phase 15, not now." (no stability promise, DERP
rendezvous, rate-limited public relays). The paragraph opening "What would make it the
answer was whether Phase 13's first outside user is remote. They are (D-063)" is
history, and so is the paragraph opening "The DERP objection also weakens for this
pair specifically"; both restate D-063 (the first pair is remote, so cross-network
reach comes first). The sentence "And it lands exactly<!-- writing: quotes the
decision log --> where D-019 said a transport must" and the paragraphs opening "Two
distinctions to hold" restate D-019 (no network provider is required), D-026 (there is
no join token at
all) for the `tc…` address, and D-042 (peer identity is an Ed25519 key pair) for
tailcat's own
WireGuard keypair. D-062 cites D-020 (a guest list replaces the join secret only once
peer identity is cryptographic) for that fact, and D-020 never names Ed25519. The
distinction between the transport key and `peerId` is a limit of the
first decision or of D-068. The Context sentence "Every cross-network run so far -
Phase 2's and Phase 5's - used an SSH tunnel" is history. The entry has no Rejected.
field, which W-36 (a decision always has Rejected.) requires.

At f11e871, 5 lines outside the entry cite D-062. None is a test fixture that uses the
number as a string. Two mean the first decision: `cmd/cogmer/tailcat.go:38` "It sits
UNDER everything and decides nothing (D-019, D-062)", and `docs/decisions.md:3173` "A
transport still decides nothing (D-019, D-062)". Both mean the rule that a transport
decides nothing, which D-062 states ("carries bytes and decides nothing") and later
text credits to D-019 and D-062 together. Three mean the evaluation findings:

- `docs/decisions.md:3157` "What this does to the tailcat evaluation (D-062).", which goes on to "526 dependencies and a roughly doubled binary"
- `docs/decisions.md:3313` "Tailcat would roughly double it (D-062)", which means the size finding
- `docs/decisions.md:6160` "*Traces* - D-062 evaluated it", about the DERP relay

### D-063: The first pair is remote, so cross-network reach comes before local discovery

D-063 holds one decision. It decides that Phase 15 (reaching a peer on another
network, in §31, the implementation order) is built before Phase 12 (discovery on a
local network) and is a prerequisite for Phase 13 (somebody else uses it). It sits
in the paragraph that opens "Decision. Phase 15 moves ahead of Phase 12".

The rest of the entry is not a decision. The Context paragraph, which opens
"Context. The phase order placed local discovery", describes the build order before
the decision, which is history under W-32 (a decision describes only the present).
It also states the fact the decision rests on: the first colleague to use cogmer and
its author both work from home, and "They will never share a network." The paragraph
that opens "What that invalidates." is an account of the build order that D-019 (no
network provider is required) set, which is history, and the paragraph after it,
"D-019's requirement is untouched", restates D-019. The sentence "It is simply no
longer first" is history. The paragraph that opens "What this does to the tailcat
evaluation (D-062)." records costs from D-062 (the evaluation of tailcat, the
library that carries a connection across two home routers): 526 dependencies and a
roughly doubled binary, with a company VPN, a public bind and an SSH tunnel each
worse. The paragraph that opens "The DERP objection weakens for this specific pair"
argues that DERP, the relay tailcat uses to introduce two machines, can run on the
droplet, a server the maintainer operates. Both paragraphs are about tailcat, which
is the subject of D-068 (tailcat is the cross-network transport), not about the
build order. The paragraph that opens "What has not changed." restates D-019 and
D-062: a transport decides nothing.

At f11e871, 11 lines outside the entry cite D-063. None is a test fixture. Five mean
the build order: `Shared Claude Sessions.md` lines 2389, 2758 and 2860, and
`docs/decisions.md` lines 561 and 587. Six mean something else. Five mean the fact
in the Context paragraph, that the first pair work from home:

- `docs/decisions.md:3039`, in D-062, "D-063 established that the first pair are
  remote"
- `docs/decisions.md:3114`, in D-062, "They are (D-063): colleagues at one company,
  both working from home"
- `docs/decisions.md:3454`, in D-068, "neither can bind an address the other can
  reach (D-063)"
- `cmd/cogmer/transport.go:21`, "which for two people working from home is never
  (D-063)"
- `cmd/cogmer/tailcat.go:25`, "for a pair who pair daily that is the product failing
  at its first step (D-063)"

One means the tailcat-cost paragraph: `docs/decisions.md:3508`, in D-068, "Both are
prices D-063 already accepted".

### D-064: A session's room is the one somebody chose inside it, never a machine default

D-064 holds one decision. It decides that `RoomForSession` reports the room a
session was put in by a command run inside it, and never puts a session in one. It
sits in the paragraph that opens "Decision. `RoomForSession` reports the room a
session was put in and never puts it in one."

Four parts are history. The paragraph that opens "This entry was never written."
says the entry was reconstructed from commit `e4fd2d8` and the citations that commit
made, and the Status line's "reconstructed 2026-09-21" says the same. The Context
paragraph, which opens "Context. One pointer was answering two different
questions.", describes the machine-level `current_room` pointer, a stored record of
which room a terminal command acts on, which no longer exists, so it is history
under W-32 (a decision describes only the present). The paragraph that opens "What
became of the other pointer." is history of D-076 (a command inside a session acts
on that session's room, withdrawn), D-077 (every room has its own URL) and D-080
(there is no current room), and of a schema comment and a `settings` table that were
removed. The heading of the Rejected field, "what the code rules out, not what was
considered", is history of how the entry was written. The paragraphs "The hazard is
silent, which is why it needed a decision." and "Enforced by a test rather than by
care." are support.

At f11e871, 12 lines outside the entry cite D-064. None is a test fixture. Eleven
mean the decision. One means the paragraph "This entry was never written.":
`docs/decisions.md:3245`, in D-065 (the daemon reads a range of wire versions), "for
the same reason and in the same commit as D-064".

### D-065: The daemon reads a range of wire versions, so upgrading is not a flag day

D-065 holds one decision. It decides that a build declares `wireVersion`, the newest
protocol version it speaks, and `minWireVersion`, the oldest it can read; accepts
anything between; reads an absent version as 1; and reports a peer outside the
range. It sits in the paragraph that opens "Decision. A build declares the newest
version it speaks". The paragraph "Zero means one." is part of the same rule, and
the paragraph that opens "Why the floor moves rarely." is its limit. A flag day, in
the title, is a change that every user has to take at the same moment.

Four parts are not decisions. The paragraph that opens "This entry was never
written" is history of the entry: it says the entry was reconstructed from
`protocol.go` and `sync.go`, and the Status line's "reconstructed 2026-09-21" says
the same. The paragraph that opens "Why this stopped being optional at Phase 11." is
history: it says the range became necessary when Phase 11 (installation) shipped the
plugin to a second user. The sentence "Today that is 2 and 1", inside the Decision
paragraph, is current state; at f11e871 `protocol.go` still sets `wireVersion` to 2
and `minWireVersion` to 1, and the sentence changes with the next version. The
heading of the Rejected field, "what the code rules out, not what was considered",
is history of how the entry was written, as in D-064 (a session's room is the one
chosen inside it). The Context paragraph is support: it describes the equality
comparison that the Rejected field names as its first alternative.

At f11e871, 2 lines outside the entry cite D-065: `cmd/cogmer/protocol.go:23` and
`cmd/cogmer/sync.go:253`. Neither is a test fixture, and both mean the decision.

### D-066: The binary is fetched and verified, never shipped in the plugin

D-066 holds two decisions. The first decides that the binary is fetched at first run
into `~/.cogmer/bin`, and that nothing runs unless its hash is in the committed
`plugin/checksums.txt`, which `scripts/release.sh` generates from the bytes it just
built. It sits in the paragraph that opens "Decision. Fetch at first run into
`~/.cogmer/bin`, and run nothing that cannot be verified." The second decides how
the installer runs: outside the plugin directory, one at a time behind a `mkdir`
lock, waiting an hour after a failure, and detached from the session. It sits in the
list that opens "Three lessons taken from the only comparable bootstrap in the
marketplace" and the paragraph that opens "And one of our own: it runs detached."
The entry weighs an alternative for each part: installing inside the plugin
directory ("A plugin update must not discard a working binary and re-fetch it"),
concurrent installs ("Two concurrent installs writing one path is a corrupt
binary"), retrying after every failure ("Never retry a failure every session"), and
a session that waits for the download, which §29 (the experience we want) forbids
("nothing waits on a download"). Each could change while fetch-and-verify holds.

Most of the rest is not a decision. The Context paragraph, which opens "Context. §29
requires that a participant installs one thing.", says that nothing put the binary
on the machine, which is history under W-32 (a decision describes only the present),
and names the three candidates. The paragraph that opens "What the ecosystem does -
stated more carefully than it first was." is a finding: 53 plugins in the official
marketplace, of which only `terraform` has a compiled server. It also corrects its
own earlier form, as does the phrase "which was asserted here before it was checked"
in the paragraph after "The arithmetic.". The paragraphs that open "The arithmetic."
record a measurement, which is a finding: 56 MB for five targets, the growth of git
history, and the marketplace arriving as an archive with a `.gcs-sha`. The paragraph
that opens "The runner pattern is also unavailable to us" is support that rests on
D-001 (Go, not TypeScript/Node or Python). The paragraphs that open "What a registry
would have given us, and what we gave up." and "And the framing that resolves it"
argue against a registry, `npx` and the other candidates the Context names, and
relate the choice to goreleaser; they are alternatives written as prose, since the
entry has no Rejected field. The paragraph that opens "Verified by running it,
including the case that matters." is a finding. The paragraph that opens "It is
inert today, and deliberately<!-- writing: quotes the decision log --> so." and the
Status line's "inert until a release exists" are current state, and at f11e871 they
no longer hold: `plugin/checksums.txt` lists five 0.7.1 assets. The paragraph that
opens "A finding that changes what must happen next." is a finding: a private
repository makes `go install` fail for everyone but its owner, so Phase 13 (somebody
else uses it) depends on a published release.

At f11e871, 6 lines outside the entry cite D-066. None is a test fixture. Three mean
the first decision: `Shared Claude Sessions.md:2388`, the Phase 11 row;
`docs/decisions.md:3397`, in D-067 (release assets are served from the droplet),
"D-066 built verified acquisition and left it inert", which also names the inert
state; and `docs/decisions.md:6171`, in D-115 (a centralized component must trace to
a disclosed tradeoff), "D-066 decided the binary is fetched and verified rather than
shipped". Three mean a finding:

- `docs/decisions.md:3422`, in D-067, "A tampered response is refused
  exactly<!-- writing: quotes the decision log --> as a tampered file is -
  demonstrated in D-066", which means the paragraph "Verified by running it"
- `docs/decisions.md:3564`, in D-069 (signing namespaces are decoupled from the
  name), "which is why `go install` failed in D-066's test", which means the
  private-repository finding
- `docs/decisions.md:6319`, in D-117 (the product is named `cogmer`), "`go install`
  failed in D-066's test", which means the same finding

No line cites the second decision.

### D-067: Release assets are served from the droplet; the host is data, not code

D-067 holds three decisions. The first decides that release assets are served from
the droplet, a server the maintainer operates, until the code is public. It sits in
the paragraph that opens "Decision. Serve the assets from the droplet that already
exists". The second decides that the release host is a committed data file,
`plugin/release-url.txt`, versioned with `plugin/checksums.txt` and
`plugin/VERSION`; that building (`release.sh`, which knows no host) is separate from
publishing (`publish.sh`, which does); and that `publish.sh` fetches back what the
host serves and compares it with the pinned hashes. It sits in the paragraphs that
open "The host is data.", "That also decides where the seam goes." and "publish.sh
verifies what the host actually serves". The only alternative it names is the title's
"not code", a host written into code. Its reason for data is that the URL and the
hashes change in one commit ("hashes that did not move with it would cost a refusal
nobody could explain"). The third decides that assets are served over plain HTTP,
because
the pinned sha256 is what authorises running a binary, and that the pin is not to be
removed as redundant. It sits in the paragraph that opens "Plain HTTP, and the
checksum is why." The alternative it weighs is HTTPS on the droplet ("HTTPS would
still be better").

The rest is not a decision, and a later decision reverses the first. The Context
paragraph, which opens "Context. D-066 built verified acquisition and left it
inert", is history: it says there is no git remote and a private repository returns
404, which D-121 (the repository is both the release host and the marketplace)
changed by creating the repository. The paragraph that opens "Verified end to end,
from a clean state with no environment overrides." is a finding: 11 MB installed, a
republish replaced in place, and nginx serving one directory. The paragraph that
opens "What this costs." is the first decision's limit, an availability dependency.
The Status line's "v0.1.0 published" is current state; at f11e871 `plugin/VERSION`
is 0.7.1. D-121 reverses the first decision: at f11e871 `plugin/release-url.txt`
names `https://github.com/Blue-Rocket/cogmer/releases/download`, and D-121 rejects
"Publishing to both the droplet and the release". With the droplet gone, the third
decision's plain HTTP no longer describes the host, and D-121 keeps its reason: "the
sha256 in `checksums.txt` is still what authorises a binary". The Revisit when
paragraph describes the move D-121 made.

At f11e871, 1 line outside the entry cites D-067: `Shared Claude Sessions.md:2388`,
the Phase 11 row, "a verified binary fetch (D-066, D-067)". It is not a test
fixture, and it means the release arrangement as a whole, which the first decision
names.

### D-068: Tailcat is the cross-network transport, behind our own dialer

D-068 holds one decision. It decides that tailcat, the library that carries a
connection across two home routers, carries cross-network sync, confined to
`tailcat.go` behind the `Dialer` interface and a `net.Listener`, so that one set of
routes serves both transports. It sits in the title and the paragraph that opens
"Confined to one file."; the entry has no Decision field.

Nearly all of the rest is findings and state. The Context paragraph, which opens
"Context. Phase 15.", is history ("Phases 2 and 5 substituted an SSH tunnel") and
restates the fact D-063 (the first pair is remote) rests on. The paragraphs that
open "What was verified, stated precisely<!-- writing: quotes the decision log -->
because the first version of this was overstated." and "The full integration was
then run" are a finding: the 291 ms spike was a connection to a public address, and
a later run with no tunnel went through DERP, the relay that introduces the two
machines, and then upgraded to direct UDP. They also correct the entry's first form.
The paragraphs that open "Still not tested: neither side able to accept inbound."
and "What is nearly certain regardless." are current state, as is the Status line's
"NAT-to-NAT still unproven". The paragraphs that open "Two defects found by running
it, neither visible from reading it.", "A client is not a connection." and "A
timeout is the signature of the packet filter." are findings, each with the code
change it led to: a client cached per peer, a 30s sync timeout, and `ServedTCPPorts`
set explicitly. The paragraph that opens "A third defect, found and unrelated to
tailcat." is a finding about the Phase 5 run, whose events were captured under an
empty session id because a harness sent `sessionId` for `session_id`. The paragraph
that opens "Costs, measured." is a finding: 566 dependencies and 21 MB.

At f11e871, 2 lines outside the entry cite D-068. Neither is a test fixture, and
neither means the decision alone:

- `Shared Claude Sessions.md:2392`, "built and working between two machines;
  NAT-to-NAT awaits the real peer (D-068)", which means the verification finding and
  the untested state
- `docs/decisions.md:3037`, the Status line of D-062 (the tailcat evaluation),
  "adopted - see D-068 for what was built and what remains unproven", which means
  the decision together with the paragraph "Still not tested"

### D-069: Signing namespaces and the state directory are decoupled from the name

D-069 holds two decisions. The first decides that every signing domain-separation
tag uses `protocolNamespace` (`peer-room`), which is arbitrary on purpose and never
changes, so nothing cryptographic depends on the product name. It sits in the
paragraphs that open "The signing namespaces." and "They should never have carried a
product name.", and moving the three live-exchange tags without a new version is
part of it ("The other three tags moved outright rather than gaining a version").
The second decides that the state directory path is one constant, so a rename
changes one line. It sits in the paragraph that opens "The state directory.
`~/.cogmer` is where the name reaches the filesystem." The alternative it weighs is
the literal repeated wherever it is used ("rather than a search"), and it is
independent of the first.

Five parts are not decisions. The Context paragraph, which opens "Context. Asked to
push the repository, which meant choosing a name", is history. The paragraph that
opens "D-058's mechanism got its first real use, …" is history of `signingBytesV3`,
and says that D-117 (the product is named `cogmer`) later deleted the superseded
scheme. The paragraph that opens "And a latent bug found by looking." is a finding:
`install.sh` honoured `COGMER_HOME` and the binary did not. The paragraph that opens
"What is deliberately<!-- writing: quotes the decision log --> not done." is
history: it leaves the module path alone until the name is settled, and D-117
records the organisation casing as fixed. Because D-117 substituted names throughout
the log, the paragraph now gives `github.com/Blue-Rocket/cogmer` as the module path
and `Blue-Rocket` as the organisation, so it no longer states the disagreement it
describes. The Revisit when paragraph waits for "the name is settled" and names the
`/team-*` commands; D-117 settled the name.

At f11e871, 9 lines outside the entry cite D-069. None is a test fixture. Six mean
the first decision: `CLAUDE.md:227`, `Shared Claude Sessions.md:2404`,
`cmd/cogmer/offer.go:31`, and `docs/decisions.md` lines 3592, 6322 and 6328. Three
mean something else. Two name both decisions as the couplings the entry broke, and
one means the module-path paragraph:

- `docs/decisions.md:3596`, in D-070 (slash commands carry a distinctive prefix),
  "Unlike the couplings in D-069"
- `docs/decisions.md:6289`, in D-117, "D-069 did the expensive half of this a name
  ago, by breaking the couplings"
- `docs/decisions.md:6317`, in D-117, "D-069 recorded that the inherited module path
  disagreed with the organisation's actual name"

### D-070: Slash commands carry a distinctive prefix, because invocation is not namespaced

D-070 holds one decision. It decides that room commands use the prefix `room-`
instead of `team-`, to avoid colliding with other plugins' commands. It sits in the
paragraph that opens "Decision. `/team-*` becomes `/room-*`."

The rest is not a decision, and a later decision replaces its reasons. The Context
paragraph, which opens "Context. Observed that command names need namespacing", is a
finding: a subdirectory changes how a command is displayed and not what a user
types, and `hookify` and `ralph-loop` both define `/help`. D-118 (the plugin
manifest name is the command namespace, and Claude Code forces it) records the
opposite at f11e871: Claude Code prefixes every command with the manifest name, so
the title's reason, "because invocation is not namespaced", no longer holds. The
paragraph that opens "The prefix is tied to the protocol namespace, not the product
name." is reasoning that D-118 replaces with D-096 (a command prefix names its
target): "What keeps the prefixes is D-096". The prefix itself stands, so this is a
partial change under W-38 (a partial change rewrites the earlier entry). The
paragraph that opens "Unlike the couplings in D-069, this one is cheap to revisit."
describes the change being made ("It is being done now"), which is history. Both
conditions in the Revisit when paragraph have occurred: D-117 (the product is named
`cogmer`) settled the name, and D-118 records the `plugin:command` form.

At f11e871, 3 lines outside the entry cite D-070. None is a test fixture. One means
the decision and its Context finding: `docs/decisions.md:6350`, in D-118, "D-070
chose `/room-*` over `/team-*` on an empirical finding". Two mean something else:

- `docs/decisions.md:6369`, in D-118, "D-070's protection becomes belt and braces,
  which is precisely<!-- writing: quotes the decision log --> what D-070 said would
  happen", which means the Revisit when paragraph
- `docs/decisions.md:3704`, in D-072 (a refused join is explained), "for the same
  reason the view is not summarised (D-070's sibling concern)", which names no part
  of D-070: the entry says nothing about summarising or about the view

### D-071: Leaving is a pause, and the row that records it is a tombstone

D-071 holds one decision. It decides that leaving is a per-session pause: the
session may rejoin the room it was in and no other, so its `session_rooms` row is
kept and marked `left_at` rather than deleted. The tombstone in the title is that
kept row. It sits in the paragraphs that open "The first one sharpened the
invariant.", "The trap, which required the tombstone." and "Leaving is per-session".
The kept row is how a pause stays compatible with §12a (room membership), not a
separate choice: deleting the row, the only alternative the entry names, would break
§12a as well as the pause.

Four parts are not decisions. The Context paragraph, which opens "Context. Tracing
the room lifecycle against the code found that `leave` did not leave.", is history:
it describes a machine-level current-room pointer and a `state` column that nothing
read. The paragraph that opens "The third one caught a live defect." is history:
leaving blanked the view, the browser page the daemon serves for a room, because
leaving cleared that pointer. The pointer no longer exists. D-080 (there is no
current room) deleted it, and at f11e871 no Go function sets or reads one. The
paragraph that opens "What leaving
deliberately<!-- writing: quotes the decision log --> does not touch." is the
decision's limit: events, the guest list, and synchronization, which is left open.
The paragraph that opens "Still not implemented, and now the only part of §22's
lifecycle that is not." is current state: rooms never close. The paragraph "Leaving
is per-session" cites §22 (persistence) for "Membership is held by a Claude Code
session, identified by its session ID"; at f11e871 that sentence is in §3.6
(session-scoped rooms), at `Shared Claude Sessions.md:213`.

At f11e871, 7 lines outside the entry cite D-071. None is a test fixture. Five mean
the decision: `CLAUDE.md:239`, `cmd/cogmer/membership.go:87`,
`cmd/cogmer/membership_test.go:443`, and `docs/decisions.md` lines 3670 and 3695.
Two refer to the paragraph "The third one caught a live defect.", and both describe
a pointer that does not exist at f11e871:

- `cmd/cogmer/main.go:878`, the end of a comment that says "Clearing it used to be
  the whole of leaving, and it blanked the browser view"
- `cmd/cogmer/membership.go:686`, a comment that cites D-071 for three things
  leaving does not do, one of which is "it does not change which room command-line
  commands or the browser view act on"; the other two mean the decision and its
  limit

### D-072: A refused join is explained, … refused

D-072 holds one decision. It decides that a refused join states the mechanism, that
the first room's context would reach the second through what the session says next,
and gives the way out: a new session, or rejoining the same room. It sits in the
paragraph that opens "Decision. Explain the mechanism and give the way out." The
typed `BoundElsewhereError`, in the paragraph that opens "A typed error, not a
formatted string.", and the instruction to `/room-join`, in the paragraph that opens
"And the slash command is told not to condense it.", are how the explanation reaches
the user. The typed error exists so the explanation can name both rooms, so it
serves the decision rather than standing apart from it.

One part is not a decision. The Context paragraph, which opens "Context. A session
that has been in one room is refused a second", is history: it quotes the old
refusal and its delivery through `log.Fatalf`. Within the paragraph "And the slash
command is told not to condense it.", the citation "(D-070's sibling concern)" for
not summarising the view names no part of D-070 (slash commands carry a distinctive
prefix), which says nothing about summarising. The paragraph that opens "Why that
is worse here than it would be elsewhere." is support.

At f11e871, 3 lines outside the entry cite D-072: `cmd/cogmer/main.go:303`,
`cmd/cogmer/membership.go:707` and `cmd/cogmer/membership_test.go:533`. None is a
test fixture, and all three mean the decision.

### D-073: Forgetting a peer discards their admissions with them

D-073 holds one decision. It decides that `Forget` deletes a peer's identity and
every admission it carried in one transaction, so meeting again is a first meeting.
It sits in the paragraph that opens "Decision. `Forget` deletes admissions and
identity in one transaction."

Three parts are history. The Context paragraph, which opens "Context. Tracing a
guest's lifecycle through the code rather than the specification.", says that
`Forget` deleted the row in `known_peers` and left every row in `room_guests`. The
paragraphs that open "What that cost." describe that defect: a forgotten peer still
appeared in `guests`, and meeting them again readmitted them to every room. It was
masked because D-054 (verification gates synchronization and injection) refused an
unverified peer's sync, and it opened when the peer was verified again. The reason
these paragraphs give, that without the cascade meeting again readmits, holds now
and is the decision's support. The paragraph that opens "Revoke is unchanged and
remains the narrow act." states a scope limit by comparison with the earlier
behaviour ("now they do"), which W-32 (a decision describes only the present) counts
as history. The paragraph that opens "Tests." is support.

At f11e871, 5 lines outside the entry cite D-073: `CLAUDE.md:240`,
`cmd/cogmer/membership.go:366`, `cmd/cogmer/membership_test.go:565`, and
`docs/decisions.md` lines 3801 and 4833. None is a test fixture, and all five mean
the decision. The two in `membership.go` and `membership_test.go` cite it for the
readmission described in "What that cost.", which is the decision's reason.

### D-074: A name means one key, and a collision is where a key change surfaces

D-074 holds two decisions. The first decides that a local label, the name a user
gives a peer on this machine, belongs to one key: `Allow` refuses a name held by a
different key with a `NameTakenError`, whose explanation treats the collision as a
possible substitution and gives the `forget` route. It sits in the paragraph that
opens "Decision. A name may belong to one key." The title's second half, "a
collision is where a key change surfaces", is the reasoning for it. The second
decides that `resolvePeer` refuses a name that matches more than one key and names
both, rather than choosing. It sits in the paragraph that opens "`resolvePeer`
refuses an ambiguous name rather than choosing." The alternative it weighs is
choosing one: "picking between them would admit a peer nobody named". It governs
databases written before the first decision, and could change without changing the
first.

Three parts are not decisions. The paragraph that opens "Nothing implemented that
alarm" is history, since it says no code acted on the alarm; it also holds support: a
`peerId` is a key, so a new key is a peer this machine has never seen. The paragraph
that opens "And it passed silently." is a finding: a second row was created, `peers`
listed two alices, and `resolvePeer` returned the first. The paragraph that opens
"What this does not do." is the limit. The Context paragraph is support: it describes
how an identity is created, and it quotes the message `pair` prints at `f11e871`, at
`main.go:1391` ("if this key changes, that is an alarm").

At f11e871, 8 lines outside the entry cite D-074. None is a test fixture, and all
eight mean the first decision. `cmd/cogmer/main.go:573` sits in `resolvePeer`'s
comment, but its words, "Recording two keys under one name is now prevented
(D-074)", mean the first decision. No line cites the second.

### D-075: The install has a state, and absence is not an inference

D-075 holds two decisions. The first decides that the installer writes one state
file (`installing`; `failed`, with the reason; removed on success; `stalled`,
derived after five minutes), and that the command wrapper and the session-start hook
both read it, the hook passing it to the model as `additionalContext` only while the
binary is absent. It sits in the paragraph that opens "Decision. One state file,
written by the installer, read by everything.", and the paragraphs that open "What
each reader does with it." and "What it does not do." are part of it. The second
decides that a plugin and the binary it pins are released as one artefact, with the
release script writing `VERSION` so that they cannot drift. It sits in the paragraph
that opens "Released together, because they are not separable." The entry names no
alternative for it in words; its account of v0.2.0 describes a plugin edited after
the release it pins, which referenced subcommands its binary did not have. It is
independent of the install state.

Three parts are not decisions. The account of v0.3.0 and v0.2.0 inside "Released
together, because they are not separable." is history. The paragraph that opens
"Verified from nothing" is a finding. The Context paragraph, which opens "Context.
Asked what options exist for telling a session about a download in progress", is
history, and the paragraph after it, "The second framing is the right one.", is
support: it lists the three things the absence of the binary could mean.

At f11e871, 7 lines outside the entry cite D-075. None is a test fixture. Six mean
the first decision: `cmd/cogmer/main.go:1415`, `docs/decisions.md:4296`,
`docs/open.md:84`, `plugin/cli.sh:12`, and `plugin/hooks-handlers/common.sh` lines
61 and 125. One means the second decision: `cmd/cogmer/membership_test.go:781`, "The
plugin and the binary version together (D-075)".

### D-076: A command inside a session acts on that session's room

D-076 holds no decision. It is a tombstone in the form W-37 (a reversed decision
becomes a tombstone) sets, with only its status line: "withdrawn 2026-09-20.
Replaced by D-077 (every room has its own URL, and no ambient value picks one)". Its
"Why:" names `docs/room-choice-findings.md`, "An ambient variable ranked above the
session answers for every session", a heading that exists at f11e871, at line 21 of
that file. D-077 holds two decisions, and what replaced D-076 is the second of them,
that no ambient value chooses a room.

At f11e871, 12 lines outside the entry cite D-076. Five are test fixtures that use
the number as a string: `cmd/cogmer/structure_test.go` lines 530 to 534. The other
seven all mean the withdrawn decision, which is the only thing the number names.
Four mean its ranking of an environment variable above the session's own room, the
part that was withdrawn: `docs/room-choice-findings.md` lines 9 and 23, and
`docs/decisions.md` lines 3219, in D-064 (a session's room is the one chosen inside
it), and 3899, in D-077. Two mean its rule that a command inside a session acts on
that session's room, which `docs/room-choice-findings.md` records under "The rest of
D-076 survived": `cmd/cogmer/membership_test.go:697` and
`docs/room-choice-findings.md:37`. One, `docs/room-choice-findings.md:41`, means the
machine-level pointer that D-076 kept as a fallback.

### D-077: Every room has its own URL, and no ambient value picks one

D-077 holds two decisions. The first decides that every room has its own address in
the view, the browser page the daemon serves: `/room/<name>` serves a room, `/`
lists rooms and redirects only when there is one, the events and stream endpoints
take the room as a parameter, and `watchLine` prints the room's own address. It sits
in the paragraph that opens "Per-room URLs." The second decides that no ambient
value, one set once for a whole machine such as an environment variable, chooses a
room: a room is named in the invocation or it is the session's, `COGMER_ROOM` is
gone, and a test fails if anything reads it. It sits in the paragraph that opens "A
room is named per invocation, or it is the session's.", with its support in the
Context paragraphs and the paragraph that opens "The pattern worth naming". The
alternative it weighs is the variable ranked above the session, "D-076 had just
placed it above the session's own room". Either decision could be reversed without
the other.

The rest is not a decision, and a later decision reverses part of it. The Context
paragraphs, which open "Context. Two objections, and the second landed on something
introduced an hour earlier", are history, including "It is gone, along with the dead
`LoadConfig`". The paragraph that opens "What that settles about the pointer."
describes a terminal fallback to the machine-level pointer, and D-080 (there is no
current room) deleted that fallback. The Revisit when paragraph ("Today the fallback
answers") describes the same fallback. The paragraph that opens "The pattern worth
naming, because it recurred three times in a day." is a finding, which
`docs/room-choice-findings.md` also records under "The same pattern appeared three
times in one day". That paragraph cites §22 (persistence) for "A session is the unit
of membership"; at f11e871 §3.6 (session-scoped rooms) states it.

At f11e871, 13 lines outside the entry cite D-077. One is a test fixture that uses
the number as a string: `cmd/cogmer/structure_test.go:532`. Of the other twelve,
seven mean the first decision: `cmd/cogmer/daemon.go:359`, `cmd/cogmer/main.go:259`,
`cmd/cogmer/ui.go:98`, `cmd/cogmer/ui_test.go:30`, and `docs/decisions.md` lines
3221, in D-064 (a session's room is the one chosen inside it), 4049, in D-080, and
4464, in D-087 (the local API requires a header). Four mean the second decision, and
one could mean either:

- `docs/decisions.md:3883`, the D-076 tombstone, "Replaced by D-077 (every room has
  its own URL, and no ambient value picks one)"
- `docs/room-choice-findings.md:33`, "D-077 (every room has its own URL, and no
  ambient value picks one) states the rule that replaced it: a room is named in the
  invocation, or it is the session's"
- `cmd/cogmer/membership_test.go:729`, "reinstated the failure D-064 removed, at
  higher precedence (D-077)"
- `docs/decisions.md:3235`, in D-064, "D-077 answers it by naming a room per
  invocation rather than by storing one"
- `CLAUDE.md:239`, in the reading-list row "rooms, membership, session binding",
  whose subject matches the second decision and could cover both

### D-078: Granting access names its room; showing something may guess

D-078 holds one decision. It decides that a command that changes who can read a room
does not guess the room, while one that only shows a room may, because a wrong guess
there shows itself. It sits in the paragraph that opens "Decision. `roomToChange`
does not guess", and the Status line says "the showing/changing distinction stands".

Most of the rest has been reversed by later decisions. The `--room <name>` flag,
which the Decision paragraph requires at a terminal, the refusal text that prints
it, and the paragraph that opens "The slash commands need no flag and that is not an
oversight." are reversed by D-079 (changing a room requires standing in it), whose
Status line says it "supersedes the `--room` flag". The Decision paragraph's
"`roomToShow` may still fall back" and the paragraph that opens "What this leaves of
the fallback." describe a read-only terminal fallback that D-080 (there is no
current room) deleted. At f11e871, `main.go` has no `roomToShow`; `guests` and `log`
resolve their room through `sessionRoom`, which refuses at a terminal, and only
`conflicts` takes a room by name. The Status line records a partial change in a note
rather than in a rewrite, which W-38 (a partial change rewrites the earlier entry)
classifies as a partial change not yet applied. The Context paragraph, which opens
"Context. Asked why a terminal command should be able to act", is history, and the
paragraph that opens "`invite` grants access." describes "a stored pointer, set
possibly weeks earlier", which no longer exists.

At f11e871, 5 lines outside the entry cite D-078. None is a test fixture. Three mean
the showing/changing distinction as it applies to `conflicts`: `cmd/cogmer/main.go`
lines 995 and 1020, "(D-078, D-080)", and `docs/decisions.md:4059`, in D-080,
"(D-078, D-079)". Two mean the flag that D-079 reverses, and both are history text
in D-079:

- `docs/decisions.md:3986`, "supersedes the `--room` flag added in D-078 hours
  earlier"
- `docs/decisions.md:3988`, "D-078 made `invite` and `revoke` refuse to guess a
  room, and gave them `--room <name>`", which means the decision and the flag
  together

`docs/decisions.md:3937` is inside D-078 and cites D-079, so it is not among the
five.

### D-079: Changing a room requires standing in it, which only a session has

D-079 holds one decision. It decides that `invite` and `revoke` act only from a
session that is in the room, with no flag, fallback or pointer, and that at a
terminal they are refused with a pointer to the slash command. Standing is the
entry's word for the right to change a room, which only a session in that room has.
The decision sits in the paragraph that opens "Decision. `invite` and `revoke`
require the invoking session to be in the room."

Three parts are not decisions. The paragraph that opens "What stays." says that
`log`, `guests` and `conflicts` fall back at a terminal; D-080 (there is no current
room) reversed that, and at f11e871 only `conflicts` takes a room without a session.
The paragraph that opens "Why this took three attempts, which is the part worth
keeping." is history of three answers to the same question: an environment variable,
a stored pointer, and a flag. The Context paragraph's "That answered the wrong
question" and the Status line's "supersedes the `--room` flag added in D-078 hours
earlier" are history. The Context paragraph cites §22 (persistence) for putting
membership in a session; at f11e871 that sentence is in §3.6 (session-scoped rooms).

At f11e871, 5 lines outside the entry cite D-079: `cmd/cogmer/main.go:199`,
`cmd/cogmer/membership_test.go:753`, and `docs/decisions.md` lines 3937, in D-078
(granting access names its room), 4059 and 4063, in D-080. None is a test fixture,
and all five mean the decision.

### D-080: There is no current room; a terminal command exists to be tested or to work when the plugin cannot

D-080 holds three decisions. The first decides that there is no machine-level
current room: `SetCurrentRoom` and `CurrentRoom` are deleted, and a room-scoped
command gets its room only from the invoking session. The one exception is
`conflicts`, which takes a room by name because "naming a room in order to read it
is not an exercise of standing", standing being D-079's word for the right to change
a room, which only a session in it has. It sits in the paragraphs that open
"Decision. `SetCurrentRoom` and `CurrentRoom` are deleted." and "The exception, and
why it is one." The second decides that a terminal command exists only for one of
five reasons (it cannot pass through a model, it must work when the plugin path is
broken, the daemon's lifecycle, machine scope, or testing), and that every other
command gets a slash command, with a stated exclusion list. It sits in the list that
opens "The reasons that survive." and the paragraph that opens "Slash commands now
cover every command that can have one." The alternative it weighs is a slash command
for `doctor` and `behaviors`, which "would imply the plugin path is a reasonable way
to diagnose the plugin path"; the exclusion list that says so sits in the paragraph
on prefixes. It could change without the current room returning, and D-086 (the
terminal is not a user experience) amends its first reason. The third decides that a
command's prefix follows its scope: `room-` for one room, and `peer-` for
relationships that outlast every room, so `/room-pair` became `/peer-pair`. It sits
in the paragraph that opens "Two prefixes, because there are two scopes (§12)", §12
being forming a room. The alternative it weighs is `room-` on pairing, which "was
saying otherwise". It is independent of the other two, and D-096 (a command prefix
names its target) states the same rule in general form.

Four parts are not decisions. The Context paragraph, which opens "Context. Asked for
the reasons a terminal command should exist at all", is history. The sentence "So
the machine-level pointer had no remaining user" is history. The list of commands
added in "Slash commands now cover every command that can have one." ("Added
`/room-revoke`", "`/room-pair` became `/peer-pair`") is history. The paragraph that
opens "A test now asserts every command file names a real subcommand" is history and
a finding, since v0.2.0 shipped a mismatch; the test follows from D-075's second
decision, that the plugin and the binary are released together. D-086 amends the
first reason from its own entry, and D-080's text of that reason is unchanged, which
W-38 (a partial change rewrites the earlier entry) classifies as a partial change
not yet applied.

At f11e871, 15 lines outside the entry cite D-080. None is a test fixture. Eight
mean the first decision: `cmd/cogmer/daemon.go:167`, `cmd/cogmer/main.go` lines 194,
995 and 1020, `cmd/cogmer/membership_test.go` lines 434 and 700,
`docs/room-choice-findings.md:42`, and `docs/decisions.md:3222`. Seven mean the
second decision:

- `cmd/cogmer/pairview.go:25`, "it was the only thing available that was not the
  model (D-080)"
- `docs/decisions.md:4336`, in D-086, "amends D-080, does not alter D-055"
- `docs/decisions.md:4352`, in D-086, "D-080's first reason is amended, not
  withdrawn."
- `docs/decisions.md:4357`, in D-086, "D-080 could not see that option because
  discovery was unsettled"
- `docs/decisions.md:4381`, in D-086, "What the terminal keeps (D-080's other four
  reasons, all intact)"
- `docs/decisions.md:4479`, in D-088 (the pairing ceremony lives in the view), "it
  was the only thing available that was not the model (D-080)"
- `docs/decisions.md:6652`, in D-123 (`stop` finds a daemon by its addresses),
  "D-080 (there is no current room) put its lifecycle at the terminal", whose few
  words name the first decision while its claim is the second decision's third
  reason

### D-081: Untrusted turns are JSON, the policy is stated separately, and the relay is measured

D-081 holds two decisions. The first decides that room turns in the injected block,
the room content a hook adds to a user's prompt, are JSON-encoded, and that the
per-injection fence, a random value that marks where the block ends, is kept beside
the encoding. It sits in the paragraphs that open "JSON, because escaping by hand is
escaping by hand." and "The fence stays." The second decides that the policy for
reading room content is also stated once at session start, through
`additionalContext`, in addition to the framing inside the block. It sits in the
paragraph that opens "So the standing policy is now stated once at session start",
where a standing policy is one stated once for the whole session. The alternative it
weighs is the framing inside the block alone, which travels "in the same blob as the
room content", where a model may discount it. It could be withdrawn without touching
the JSON encoding. The title's third clause, "the relay is measured", describes a
finding, not a decision.

Five parts are not decisions. The paragraph that opens "Then measured, because a
defence nobody tested is a hope." (the table of four runs) and the paragraph that
opens "And the … result about the new policy: it could not be shown to help." are
findings. The paragraph that opens "What this settles about reaching a person." is a
finding: the model relays accurately when a question is relevant and stays silent
when it is not. The paragraph that opens "What it does not settle." is an open
question about someone else's software, whether a `hook_success` attachment is
treated as tool-result content, and says itself that it "belongs in the behaviour
registry". The paragraph that opens "Not adopted: screening tool output through a
classifier." is an alternative rejected for the second decision, written as prose,
since the entry has no Rejected field. The Context paragraph and the sentence "Turns
were interpolated into markup" are history.

At f11e871, 9 lines outside the entry cite D-081. None is a test fixture. Six mean
the first decision: `cmd/cogmer/behaviors.go:276`, `cmd/cogmer/daemon.go` lines 462
and 477, `cmd/cogmer/transcript_test.go` lines 210 and 265, and
`docs/decisions.md:4640`, in D-090 (attribution is structured). Three mean something
else:

- `plugin/hooks-handlers/session-start.sh:46`, the end of a comment that says "the
  rule is delivered separately from the data it governs", which means the second
  decision
- `docs/open.md:184`, "the standing policy for room content (D-081) is never given",
  which means the second decision
- `docs/open.md:198`, "D-081 could not show that policy helping", which means the
  finding that the policy could not be shown to help

### D-082: The daemon can reach a person directly; there is no setsid hazard

D-082 holds two decisions. The first decides that the view, the browser page the
daemon serves, is reached by the daemon calling `open`, which from a detached
process brings the browser forward; B23 (a process the daemon's shape keeps the GUI
session, so it can open the view and notify) records the behaviour. It sits in the
paragraph that opens "Auto-open works, and takes focus." ("This is the mechanism
discovery now rests on"). The alternatives it weighs are writing to the terminal
("Terminal writes: possible, rejected.") and an `.app` bundle ("Not adopted: an
`.app` bundle."). The second decides that OS notifications are used only to
announce, never for anything that must be known to have arrived, because their
delivery cannot be observed, which D-014 (delivery state comes from transcript
evidence) makes decisive. It sits in the paragraph that opens "Notifications: kept,
with a stated limit." The alternatives it weighs are a notification relied on as
delivered ("unfit for anything that must be known to have landed") and the `.app`
bundle's own notification identity and click that opens the view. It could change
without affecting auto-open. `CLAUDE.md:133`, "prefer an OS notification from the
daemon for ambient awareness", rests on it. The title's second half, "there is no
setsid hazard", is a finding.

Four parts are not decisions. The paragraph that opens "An earlier finding said the
opposite, and was wrong." is a finding that corrects an earlier finding: macOS ships
no `setsid`, so the earlier test never launched anything. The paragraph that opens
"So the setsid branch in `start_daemon_if_needed` is not a hazard." is a finding,
with the `launchctl managername` table and "No code change was made". The
measurements in "Terminal writes: possible, rejected." (`/dev/tty` is `ENXIO`, the
tty is readable from `ps`, and a tty has no `PIPE_BUF` atomicity) are a finding, and
its conclusion is a rejected alternative for the first decision. The Context
paragraph is history, as is the Status line's "supersedes an earlier false finding".

At f11e871, 3 lines outside the entry cite D-082: `cmd/cogmer/browser.go:14`, and
`docs/decisions.md` lines 4217, in D-083 (the view opens at a first pairing), and
4358, in D-086 (the terminal is not a user experience). None is a test fixture, and
all three mean the first decision.

### D-083: The view is opened at a first pairing, not at room creation

D-083 holds one decision. It decides that the view, the browser page the daemon
serves, first opens at the first pairing, the two-word comparison with a colleague,
and not at the first `create` or `join`. It sits in the paragraph that opens "So the
first open belongs at the first pairing."; the entry has no Decision field. The
alternative it weighs is the first `create` or `join`, "the moment a room first
exists".

Two parts are not decisions. The paragraph that opens "Not settled here:" is open
state: whether pairing is driven from the view or only shown in it, and whether
later pairings open it again. D-088 (the pairing ceremony lives in the view) answers
both in its paragraph "Also settled from D-083's open list:". The Status line's "not
yet implemented" is current state, and at f11e871 it no longer holds.
`beginCeremony` opens the pairing page with `openInBrowser` (`main.go:1341`) unless
`--terminal` is given, and that is the only call to `openInBrowser`. `runPair` and
`runVerify` both call `beginCeremony`, so the view opens at every pairing and every
re-verification, and nothing opens it at room creation. This was read from the code,
not observed.

At f11e871, 4 lines outside the entry cite D-083. None is a test fixture. One means
the decision: `docs/decisions.md:4358`, in D-086 (the terminal is not a user
experience), "D-083 already placed the first opening of the view at a first
pairing". Three mean the list "Not settled here:". One is
`docs/decisions.md:4329`, in D-085 (a line telling somebody to run `cogmer` fails),
"That belongs with D-083, not here", whose subject is pairing from a surface that
shows the words, an item of that list. The other two are history text in the entries
that hold them:

- `docs/decisions.md:4394`, in D-086, "Open questions from D-083 remain open"
- `docs/decisions.md:4521`, in D-088, "Also settled from D-083's open list:"

### D-084: Silent to the person is not silent to the log

D-084 holds two decisions. The first decides that hook and install output goes to
the log by default, to `/dev/null` only when the discarded output can be said in
advance, and never to stdout. It sits in the paragraph that opens "The rule adopted:
redirect to the log by default", and the paragraphs that open "§3.1 requires silence
toward the person, not toward the log," and "Never to stdout, in any of this."
(`json_safe`) are part of it; §3.1 is first, do no harm. The second decides that a
busy port at daemon start is told apart: if `/healthz` shows the holder is a cogmer
daemon, the new one exits quietly, and otherwise it writes `daemon-state`, which the
session-start hook relays to the model. It sits in the paragraph that opens "A busy
port is two different events and now says which." The alternative it weighs is
`log.Fatalf` for both ("`net.Listen` failing was `log.Fatalf` for both"). It is
independent of the logging rule.

Two parts are not decisions. The Context paragraph and the paragraphs that open
"Four things stacked, and each alone was survivable." and "What broke it was a
positive observable" are a finding: how the missing `setsid` went undetected. It is
the same event that the paragraph of D-082 (the daemon can reach a person directly)
"An earlier finding said the opposite, and was wrong." records. The paragraph that
opens "What changed." is history: a shared `ct_say`, a split between curl refusing
and curl missing, `install.log`, and a `mkdir` failure that now says something.

At f11e871, 2 lines outside the entry cite D-084. Neither is a test fixture, and
both mean the first decision. `plugin/hooks-handlers/session-start.sh:23` states the
rule ("Never to stdout"). `plugin/hooks-handlers/install.sh:42`, "it used to be the
quietest line in the file (D-084)", takes its words from "What changed.", about a
`mkdir` failure the logging rule made audible.

### D-085: A line telling somebody to run `cogmer` is a line that fails

D-085 holds one decision. It decides that wherever a user is told to type a command,
the binary prints its own invocation (`invocation()`, from `os.Executable()`, with
`$HOME` written as `~`), and the command files relay it without shortening. It sits
in the paragraphs that open "The binary prints its own invocation." and "The command
docs now relay rather than restate."

Four parts are not decisions. The Context paragraph, which opens "Context. Reviewing
documentation for the assumption that every user is a person.", is history. The
paragraph that opens "The instructions did not work." is a finding: every plugin
install got `command not found`. The paragraph "The command docs now relay rather
than restate." says that `/peer-pair` "walks through opening a terminal, pasting,
and pressing return"; at f11e871 that is not so, since
`plugin/commands/peer-pair.md` says "Do not tell them to open a terminal." The
paragraph that opens "Not done: making pairing work from a slash command." is open
state that D-088 (the pairing ceremony lives in the view) answered: at f11e871
`/cogmer:peer-pair` runs `pair`, which opens the pairing page.

At f11e871, 3 lines outside the entry cite D-085: `CLAUDE.md:243`,
`cmd/cogmer/commandname_test.go:34`, and `docs/decisions.md:4781`, in D-092 (only
one side needs an address). None is a test fixture, and all three mean the decision.

### D-086: The terminal is not a user experience; the view is the surface

D-086 holds two decisions. The first decides that what a user does, including
pairing, belongs in the view, the browser page the daemon serves, and that the
terminal is an operator surface for diagnostics, the daemon's lifecycle,
machine-scope identity and testing. It sits in the title and the paragraphs that
open "D-080's first reason is amended, not withdrawn." and "What the terminal
keeps"; the entry has no Decision field. The second decides that the files in
`plugin/commands/` are prompts, not documentation, so they carry no roadmap or
commentary, and interim state is recorded in the decision log instead. It sits in
the paragraph that opens "Interim state,
deliberately<!-- writing: quotes the decision log --> not marked in the files." The
alternative it weighs is annotating `peer-pair.md` as provisional ("It is not
annotated as provisional"). It holds whatever surface pairing uses.

Six parts are not decisions. The paragraph that opens "What actually depends on a
terminal today: one bit." is current state as of the entry: a single `fmt.Scanln`.
At f11e871 that `fmt.Scanln` is still in `verifyWith` (`main.go:1382`), which is the
terminal path D-088 (the pairing ceremony lives in the view) keeps for `--terminal`
and for machines that cannot open a browser. The paragraph that opens "Next, and not
yet built:" is open state that D-088 built. The paragraph "Interim state" says that
`/peer-pair` "currently walks a person through opening a terminal", which at f11e871
`plugin/commands/peer-pair.md` does not do. The paragraphs that open "D-055 is
untouched." and "D-043 still holds" restate D-055 (there is one way to verify a
peer) and D-043 (a second host is prepared for only by naming, never by machinery). The
Context paragraph is history, and its sentence on host order is now held by D-110
(host order: Claude Code, CoWork soon after, ChatGPT Desktop much later). D-086
amends D-080's first reason (a command that cannot pass through a model) from its
own entry, which W-38 (a partial change rewrites the earlier entry) classifies as a
partial change not yet applied to D-080. The paragraph that opens "But this
direction and that constraint point the same way." is support for the first
decision.

At f11e871, 12 lines outside the entry cite D-086. None is a test fixture. Seven
mean the first decision: `docs/decisions.md` lines 4404, in D-087 (the local API
requires a header), 4471, in D-088, 4983, in D-096 (a command prefix names its
target), and 5814, 5838, 5869 and 5875, in D-110; the last two mean its Revisit when
condition. Five mean something else:

- `docs/what-leaves-findings.md:74`, "prompts the model reads, not documentation a
  person reads - D-086 settled that about `peer-pair.md`", which means the second
  decision
- `docs/decisions.md:6198`, "the files in `plugin/commands/` are prompts rather than
  documentation (D-086)", which means the second decision
- `docs/decisions.md:5816`, in D-110 (host order), "One sentence in D-086 (the
  terminal is not a user experience) carries it as context", which means the
  Context's sentence on host order
- `docs/decisions.md:5865`, in D-110, "Leave it in D-086.", a rejected alternative
  that means the same sentence
- `docs/decisions.md:5832`, in D-110, "D-086 already considered this exact move and
  refused it", which means the paragraph that restates D-043

### D-087: The local API requires a header a web page cannot send

D-087 holds one decision. It decides that state-changing local routes require the
header `X-Cogmer: 1` and POST, with `Origin` as a second layer, and that reads are
left unguarded. It sits in the paragraphs that open "The rule is REQUIRE, not
refuse.", "Also fixed: state-changing routes require POST" and "Not guarded,
deliberately<!-- writing: quotes the decision log -->:". POST and `Origin` are
layers of the same guard, not separate choices.

Three parts are not decisions. The paragraphs that open "It was exploitable, and was
demonstrated rather than argued." are a finding: a page on another port marked a
peer verified with one `text/plain` POST. The measurements inside "The rule is
REQUIRE, not refuse." (an `OPTIONS` carrying `Access-Control-Request-Headers`, with
no POST after it), "Rejected: `Referer`, and rejected on measurement." (a `<meta
name="referrer">` tag suppressing it) and "Rejected: a redirect to control the
referrer." (a 307) are findings; the rejections themselves are alternatives the
decision weighs, written as paragraphs rather than under a Rejected field. The
Context paragraph ("no handler on the local HTTP server checked `r.Method` or
`Origin` - not one") is history, as is the word "fixed" in "Also fixed".

At f11e871, 8 lines outside the entry cite D-087: `CLAUDE.md` lines 178 and 237,
`cmd/cogmer/daemon.go:131`, `cmd/cogmer/localguard.go:9`,
`cmd/cogmer/pairview.go:39`, `cmd/cogmer/pairview_test.go:84`, and
`docs/decisions.md` lines 4473 and 6688. None is a test fixture, and all eight mean
the decision.

### D-088: The pairing ceremony lives in the view, and every pairing gets its own URL

D-088 holds three decisions. The first decides that the pairing ceremony, the
two-word comparison of D-055 (there is one way to verify a peer), is shown and
confirmed in the view, the browser page the daemon serves, with the terminal path
kept for `--terminal` and for machines that cannot open a browser. It sits in the
title and the paragraphs that open "D-055 is unchanged, and that is …." and "The
terminal path is kept, and is not legacy."; the entry has no Decision field. The
second decides that every pairing has its own URL carrying a 128-bit id that acts as
the capability, and that the page names a pairing (`pairId`), never a peer, so an
expired link fails as a link. It sits in the paragraphs that open "Every pairing
gets its own URL" and "The page names a pairing, never a peer." The alternative it
weighs is "A fixed address whose contents we rewrote". It could change without
moving the ceremony out of the view. The third decides that the 90-second window
starts when the user presses Start, not when the page loads, and that a timeout
returns to the ready state. It sits in the paragraph that opens "The 90-second
window starts when the person is ready, not when the tab loads." The alternative it
weighs is starting on load, which "spends the deadline on however long it takes two
people to get on a call". It is independent of the other two.

Five parts are not decisions. The paragraph that opens "Measured, and it refined the
premise." is a finding: `open` makes a new tab every time, and the tab count climbed
from 2 to 5 over repeated opens. The paragraph that opens "Verified end to end" is a
finding. The sentence "The first build ran the exchange on load" is history. The
paragraph that opens "Also settled from D-083's open list:" answers the open
questions of D-083 (the view opens at a first pairing) from this entry, which W-38
(a partial change rewrites the earlier entry) classifies as a partial change not yet
applied to D-083. The Context paragraph is history.

At f11e871, 7 lines outside the entry cite D-088. None is a test fixture. Five mean
the first decision: `CLAUDE.md:236`, `cmd/cogmer/main.go:1501`,
`cmd/cogmer/pairview.go:20`, and `docs/decisions.md` lines 4783, in D-092 (only one
side needs an address), and 5630, in D-106 (an offer is delivered when it can work).
Two mean the second decision:

- `cmd/cogmer/daemon.go:86`, "Pairings awaiting their ceremony, one per opened page
  (D-088)"
- `cmd/cogmer/verify.go:347`, the end of a comment that opens "PairID lets the view
  name a pairing rather than a peer" and ends "the daemon decides what that link
  means (D-088)"

### D-089: The reachability warning asked the wrong question, and fired always

D-089 holds one decision. It decides that the warning that a pairing string cannot
be reached lives in `printPairingInvitation`, so it goes wherever a pairing string
is printed, asks about `AdvertisedEndpoint()` rather than the bind address, and
names which of two causes applies (`endpointRecorded()`). A pairing string is what
one user sends another to pair, carrying a key and an address. The decision sits in
the paragraphs that open "The warning tested the bind address, not the advertised
one.", "It was attached to a command rather than to the string." and "The two causes
need different answers, and got one."; the entry has no Decision field. It is
written as a defect report, and its title names a defect rather than a decision, so
it does not meet W-31 (a decision's title states the decision).

Five parts are not decisions. The sentence that opens "Observed directly: `whoami`
printing a `tc://` string" is a finding: the warning appeared under every pairing
string. The paragraphs that open "A fallback that needed the thing that had just
failed." (the daemon check in `beginCeremony`) and "Found by a test, immediately:"
(`ParseEndpoint` accepting a malformed address) are findings, each with the defect
fix it led to. The paragraph that opens "Not taken: advertising the machine's LAN
address as a fallback." gives a reason that D-091 (identity is advertised; location
is discovered) replaces, and the blockquote that opens "The conclusion holds; the
reason is restated in D-091." records that. D-091's Status line says it "Supersedes
part of D-089", which W-38 (a partial change rewrites the earlier entry) classifies
as a partial change not yet applied. The paragraph that opens "Have we done
everything to avoid a loopback endpoint?" is current state. The Context paragraph is
history.

At f11e871, 6 lines outside the entry cite D-089. None is a test fixture. Two mean
the decision: `cmd/cogmer/main.go:1507` and `docs/decisions.md:4930`, both "D-089
consolidated them". Four mean something else:

- `docs/decisions.md:4891`, in D-094 (the chosen name leads the view), "Same failure
  as D-089: a warning that is always on trains somebody to ignore the one that
  matters", which means the always-on warning finding
- `docs/decisions.md:4940`, in D-095 (a name is chosen or guessed), "which is the
  always-on warning of D-089 in another costume", which means the same finding
- `docs/decisions.md:4652`, the Status line of D-091, "Supersedes part of D-089",
  which means the LAN-address paragraph
- `docs/decisions.md:4737`, in D-091, "D-089 rejected advertising this machine's LAN
  address, and was right.", which means the LAN-address paragraph

### D-090: Attribution is structured; the derived name is a field, not part of a string

D-090 holds two decisions. The first decides that in the injected block, the room
content a hook adds to a user's prompt, the derived name (the word pair derived from
a peer's key), the verified state and the kind of turn are separate fields, and that
nothing cogmer states is joined to a peer's free text. It sits in the paragraph that
opens "So the derived name became a field". The second decides that the display name
a peer asserts for itself has no uniqueness constraint, because two colleagues may
share a name. It sits in the paragraph that opens "The self-asserted display name
has no constraint, correctly". The alternative it weighs is a constraint on a
colliding display name, which is the question the Context asks ("what should happen
when a peer's self-identified name collides with one already in the list"). It could
be reversed without undoing the fields.

Four parts are not decisions. The paragraph that opens "The hole was structural
rather than a collision." is a finding: a display name of `Alice (quiet-otter)`
produced `Alice (quiet-otter) (prudent-wagtail, unverified)`, and the escaping test
passed. The sentence that opens "That already happened: the first two-peer run" is a
finding. The paragraph that opens "Still open, and not fixed here:" is an open
question, the view's similar visual weight for the two names, and at f11e871 it is
an item in `docs/open.md` at line 219. The paragraphs that open "The derived name
(`quiet-otter`) can collide and that is accepted" and "The local label you assign is
settled by D-074" restate D-021 (peer names are derived from the identity), which
records how often derived names collide, and D-074 (a name means one key), and decide
nothing here. The first also cites D-050 (room names may collide locally) and D-017
for the principle that names may collide and identities do not.

At f11e871, 6 lines outside the entry cite D-090. None is a test fixture. Five mean
the first decision: `CLAUDE.md:237`, `cmd/cogmer/daemon.go:396`,
`cmd/cogmer/sas_test.go:339`, `cmd/cogmer/transcript_test.go:98`, and
`docs/decisions.md:5123`, in D-099 (the injected block carries the label). One means
the open question: `docs/open.md:219`, "D-090 left this open: the display name and
the derived name are separate elements but carry similar weight". No line cites the
second decision.

### D-091: Identity is advertised; location is discovered

D-091 holds two decisions. The first decides that only addresses that name a node
(`tc://…`, an overlay address) are advertised and stored durably, that a location
(an IP and port) enters only as a discovered candidate with an expiry, and that
private ranges never appear in a pairing string, an invitation or a durable record.
It sits in the paragraphs that open "Addresses are of two kinds, and only one of
them can be advertised." and "A private address is … stale; it can be confidently
wrong about a different machine." The paragraphs on the overlay's rendezvous relay
are a limit of it. The second decides that the receiver orders candidates: a
remembered winner first, carrying its time and discarded on failure, then by class
(same-host, same-network, public, relayed), tried concurrently, with no configurable
sort, and with real requirements expressed as filters. It sits in the paragraphs
that open "A remembered winner is a snapshot too." and "On ordering, asked directly:
no, a daemon should not have a configurable sort." The alternatives it weighs are a
configurable sort, an order the sender expresses ("any order the sender expresses is
a preference rather than knowledge"), and a remembered winner kept as a record ("It
is a hint, not a record"). It could change without changing what may be advertised.

Six parts are not decisions. The paragraphs that open "An overlay address is not
purely an identity, and the difference matters." are a finding: the decoded value
holds three keys and a home relay, and the relays forward to each other. The
paragraph that opens "This entry first argued that the cost was disclosure" is
history, with a measurement recorded in D-101 (peer connections are TLS pinned to
the key already verified). The phrase "an earlier form of this entry overturned it",
in the paragraph that opens "D-089 rejected advertising this machine's LAN address,
and was right.", is history. The paragraph "None of this is implemented." and the
Status line's "decided, not implemented" are current state. The Status line's
"Supersedes part of D-089" records a partial change that W-38 (a partial change
rewrites the earlier entry) says is made by rewriting D-089. The paragraph that
opens "The sender cannot know which address works, and should not try." is support,
and the Context paragraph is history.

At f11e871, 5 lines outside the entry cite D-091. None is a test fixture. Three mean
the first decision: `CLAUDE.md:241`, and `docs/decisions.md` lines 4589, the
blockquote in D-089 (the reachability warning), and 4787, in D-092 (only one side
needs an address). Two, in D-100 (each document answers one question), name D-091 as
an entry being written, and mean neither decision:

- `docs/decisions.md:5154`, "deciding what a rewritten D-091 should contain"
- `docs/decisions.md:5183`, "D-091, which listed what the code gets wrong today as
  evidence for its rule"

### D-092: An address is learned once, out of band, and only one side needs one

D-092 holds one decision. It decides that pairing proceeds when the pairing string,
what one user sends another to pair, has no address, because only one side needs to
dial: `pair` says which side has to dial and starts anyway. It sits in the
paragraphs that open "Only one side needs a usable address." and "And `pair` refused
to allow it." ("It now says which of them has to dial and starts anyway"). The
title's first half, "learned once, out of band", describes the code, not a choice.

Four parts are not decisions. The paragraph that opens "When: once, at `pair` time.
How: carried by a person." is a finding about the code: `parsePairing` splits the
string, `SetPeerEndpoint` stores the address, and `RemoteAddr` is never read. The
paragraph "And `pair` refused to allow it." also holds history, the earlier message
"there is nowhere to reach them yet". The paragraph that opens "Seven more
instruction sites were still naming a bare `cogmer` (D-085)" is history that applies
D-085 (a line telling somebody to run `cogmer` is a line that fails) and D-088 (the
pairing ceremony lives in the view). It also holds an exemption to D-085's rule,
"Operator surfaces - the daemon log, `doctor`'s stderr, the generated behaviours
header - keep the short form", which D-085 does not state. The Context paragraph is
history.

At f11e871, 1 line outside the entry cites D-092: `cmd/cogmer/sas_test.go:294`,
"carried no address, which made a working arrangement look broken (D-092)". It is
not a test fixture, and it means the decision.

### D-093: A two-interaction flow must choose what an unfinished second one means

D-093 holds two decisions. The first decides that a label, the name a user gives a
peer, is required at pairing: it is taken with the pairing string, held in the
pending pairing, and written only when the words match; `NameFree` looks for a clash
before the ceremony; and both surfaces run through one pairing record. It sits in
the paragraph that opens "Collecting is not asserting.", and the paragraphs that
open "The name clash is checked before the ceremony, not after.", "Both surfaces now
run through one pairing record." and "How somebody learns the requirement" are part
of it. The title states the general principle, which is the reasoning, and names
neither decision, so it does not meet W-31 (a decision's title states the decision).
The second decides that a pairing's three endings leave different state: an
abandoned pairing leaves a resumable unverified row, a match writes the chosen
label, and a mismatch removes only what this pairing created, never an existing
relationship. It sits in the paragraph that opens "The three endings are now
distinct, and were not before:" (the table) and the paragraph that opens "A mismatch
removes only what the pairing created." The alternatives it weighs are leaving the
mismatched key recorded under the colleague's name ("Abandoning and mismatching left
identical state") and discarding an existing relationship on a mismatch ("not a
reason to discard it and every admission it holds (D-073)"). It could change without
changing when the label is collected.

Four parts are not decisions. The paragraph that opens "Optional did not mean
unlabelled." is history: `Allow` filled an empty name with the derived one. The
paragraph that opens "Asking after the words matched was worse, and was the first
plan." is history, and its argument is an alternative rejected for the first
decision, written as prose, since the entry has no Rejected field. The paragraph
that opens "A hazard found while moving the write:" is a finding: `SetPeerEndpoint`
is an `UPDATE` that affects no row when the row does not exist yet. The Context
paragraph is history.

At f11e871, 12 lines outside the entry cite D-093. None is a test fixture. Eight
mean the first decision: `cmd/cogmer/main.go` lines 1122, 1130, 1263 and 1319,
`cmd/cogmer/membership.go:291`, `cmd/cogmer/pairview.go:54`, and `docs/decisions.md`
lines 5007 and 5036, in D-097 (a pairing string carries a chosen name). Three mean
the second decision, and one could mean either:

- `cmd/cogmer/pairview_test.go:212`, "Abandoning is not mismatching, and neither
  used to be distinguishable from the other (D-093)"
- `cmd/cogmer/verify.go:421`, "the three endings here are genuinely different
  (D-093)"
- `cmd/cogmer/offer_test.go:78`, "which is the state an abandoned pairing leaves
  behind (D-093)"
- `CLAUDE.md:236`, in the reading-list row "pairing, verification, the two words",
  which could cover both

### D-094: The name you chose leads the view; the unverified marker becomes a fact

D-094 holds two decisions. The first decides that the view leads with the label the
user gave a colleague at pairing, with the derived name beside it. The view is the
browser page the daemon serves where a user watches the room, and the derived name is
a word pair computed from the colleague's key. A label equal to the derived name
counts as absent. It sits at "The label now leads, with the derived name kept beside
it". The
second decides that the view carries whether a turn's peer is verified as a field,
and shows the unverified marker, a mark on a turn whose peer has not been verified,
only when that field says so. It sits at "And the
unverified marker was on every remote turn." The alternative the entry weighs for
it is a marker on every remote turn, which it rejects with the reasoning of D-089
(the reachability warning that fired always): "a warning that is always on trains
somebody to ignore the one that matters". The second could be reversed without
reversing the first: the view could drop the marker entirely, or show it on every
turn, and the label would still lead. The title joins the two with a semicolon,
which W-40 (an entry records one decision) names as a sign of two decisions.

The Context paragraph and its table say which name reached the view before the
decision, and "`known_peers.name` was read only by the CLI `peers` listing" says
what read the label then. Both are history under W-32 (a decision describes only the
present). Inside the marker paragraph, "a comment written before D-054 explained
that every remote peer was unverified" and "the marker was permanently on" are
history as well (D-054, verification gates synchronization). The paragraph
"Deferred: carrying the label into the injected block." records work left undone,
which D-099 (the injected block carries the label) has decided since. The paragraph
"Not done, and blocked: defaulting the label to the name the peer chose." records
work left undone, which D-095 (a name is chosen or guessed) and D-097 (a pairing
string carries a chosen name) have decided since. Within it, the account of
`UserDisplayName` as `$USER` capitalised with no way to change it describes the code
before D-095, which is history, and "which is why two daemons asserted the same one
during the first two-peer run" records an observed event, which is a finding. The
Revisit when names a condition that D-095 and D-097 have met, so it is
history too, as is the status line's "with one half deliberately<!-- writing: quotes
the decision log --> deferred".

At f11e871, 12 lines outside the entry cite D-094. None is a test fixture. Six mean
the first decision: `cmd/cogmer/ui.go:31`, `cmd/cogmer/ui_test.go:191`,
`cmd/cogmer/identity.go:146`, `cmd/cogmer/membership.go:312`,
`cmd/cogmer/main.go:934` and `docs/decisions.md:4956`. None means the second
decision. Six mean the deferred or blocked parts: `docs/decisions.md:4918` ("D-094
stopped short of defaulting a peer's label"), `docs/decisions.md:4960` ("D-094's
revisit condition"), `docs/decisions.md:5005` ("Completes D-094's revisit"),
`docs/decisions.md:5008` ("D-094 wanted the default to be the name the peer
picked"), `docs/decisions.md:5110` ("Completes D-094") and `docs/decisions.md:5112`
("D-094 put the label in the view and deferred the same field"). Five of the six sit
in the Context paragraphs or status lines of D-095 (a name is chosen or guessed),
D-097 (a pairing string carries a chosen name) and D-099 (the injected block carries
the label), and `docs/decisions.md:4960` sits in D-095's paragraph "Now
unblocked". All six places are history themselves.

### D-095: A name is chosen or guessed, and the difference is recorded

D-095 holds three decisions. The first decides that whether the display name was
chosen is recorded as a flag, `NameChosen`, which is false at creation, is set by
any deliberate set, and is never inferred by comparing the name with the guess, the
name computed from `$USER`. It
sits at "Chosen is recorded, not inferred." The second decides that a user is offered
the choice of name where their identity first travels, in `printPairingInvitation`,
and that `whoami` shows the name plainly, with a line saying it was guessed only
while it is. It sits at "So the moment is the first time it travels" and "`whoami`
does both, not one or the other." The alternatives the entry weighs for it are
offering the choice "with a command" or at identity creation ("There is no earlier
candidate"), and, for `whoami`, showing only one of the two. It could be reversed
without reversing the first: the offer could move to a command while the flag
stays. The third decides that the command that sets a user's own name is
`/self-name`, under the prefix `self-` for commands about the user. It sits at
"Prefixes name the scope, and `peer-` is for other people." The alternative it
weighs is `/peer-name`. The same paragraph states the prefix rule, including the
"names the activity" reading. That rule is the one D-096 (a command prefix names its
target) holds. The choice of `/self-name` is one application of D-096's rule.

The sentence "A prefix names the activity, so `/peer-pair` with no arguments
printing your own string is not a violation" is corrected by D-096 (a command prefix
names its target), which makes it a partial reversal under W-38 (a partial change
rewrites the earlier entry). The paragraph "Renaming is safe, and by construction
rather than luck." is support for the first or second decision, not a decision of
its own. The paragraph "Now unblocked: the pairing string can carry a chosen name"
records an event, which D-097 (a pairing string carries a chosen name) carried out,
and the Context paragraph ("D-094 stopped short ... Asked what the natural point
is") is history under W-32 (a decision describes only the present). "which is
exactly<!-- writing: quotes the decision log --> why `Ec2-user` could travel for
weeks" records an observed event, which is a
finding.

At f11e871, 11 lines outside the entry cite D-095. None is a test fixture. Three
mean the first decision: `cmd/cogmer/keys_test.go:234`, `cmd/cogmer/identity.go:39`
and `docs/decisions.md:5024`. `CLAUDE.md:243` is a row of the reading list in
`CLAUDE.md` ("Where to read before changing something") and names the entry as a
whole. The other seven mean something else. `cmd/cogmer/main.go:1578`, "The offer
appears exactly<!-- writing: quotes the code --> while it is true and stops at the
first deliberate act (D-095)", means
the second decision. `cmd/cogmer/main.go:1590`, "a command that sets your own name
has no business in that namespace (D-095)", means the third. `CLAUDE.md:149`,
"`self-` you (D-095, D-096)", means the prefix rule, which D-096 (a command prefix
names its target) holds. `docs/decisions.md:4967`, D-096's status "Corrects D-095",
and `docs/decisions.md:4969`, "D-095 wrote the prefix rule down as naming the
activity", mean the corrected activity reading, and both sit in D-096's status line
and Context, which are history themselves. `docs/decisions.md:5009`, "D-095 made
picking one possible", means that a user can choose a name, which is the entry as a
whole rather than the recorded flag. `cmd/cogmer/main.go:1126`, "A GUESSED name is
never carried ... (D-095)", states the rule of D-097 (a pairing string carries a
chosen name, never a guessed one) and no part of D-095.

### D-096: A command prefix names its target; `/self-status` is where you are

D-096 holds two decisions. The first decides that a command prefix names the
command's target: `peer-` another user, `room-` a room, `self-` the user. The user's
own identity and pairing string are shown by `/self-status`, and no `peer-` command
prints them. It sits in the title and at "`/self-status` now owns it". The only
alternative the entry weighs for `/self-status`, `/peer-pair` printing the user's
own string, is a violation of the rule, so `/self-status` belongs to the first
decision. The second decides that there is no `/help` and no overview command, and
that no placeholder name is invented for one. It sits at "No `/help`, and not for
want of noticing." The alternative it weighs is a placeholder command name ("Do not
invent a placeholder"). It could be reversed without reversing the first: an
overview command could be added and the prefix rule would stand. D-118 (the manifest
name is the command namespace) has changed what the second decision rests on, since
Claude Code owns `/help` but not `/cogmer:help`.

The status line's "Corrects D-095" and the Context paragraph, which says what D-095
(a name is chosen or guessed) wrote and that the error was pointed out, are history
under
W-32 (a decision describes only the present). So are "That was a rule bent to fit an
exception, and the exception was the defect.", "It replaces nothing, because until
now there was no slash command for your own identity at all" and
"`printPairingInvitation` has one caller again". The first of these holds the
reason D-095's activity reading was wrong, which W-38 (a partial change rewrites the
earlier entry) says is recorded as a finding. The clauses "which is what Phase 13 is
blocked on" and "the name gets one chance" are overtaken by D-117 (the product is
named `cogmer`), which settled the name, and so is the Revisit when ("the name is
settled"). "the slash menu already lists all twelve commands" is a count that D-119
(no slash command is reachable by the model) contradicts at f11e871, where it
counts fourteen.

At f11e871, 10 lines outside the entry cite D-096. None is a test fixture. Five mean
the first decision: `CLAUDE.md:149`, `cmd/cogmer/main.go:650`,
`cmd/cogmer/main.go:1202`, `docs/decisions.md:6370` and `docs/decisions.md:6450`.
Four mean the second: `docs/decisions.md:6337` ("Also D-096's overview command"),
`docs/decisions.md:6338` ("not in the form D-096 planned for it"),
`docs/decisions.md:6383` ("D-096's overview command cannot be what D-096 reserved",
one line that names D-096 twice) and `docs/decisions.md:6386` ("D-096's stated
blocker is gone - Claude Code owns `/help`"). `CLAUDE.md:243` is a row of the reading
list in `CLAUDE.md` and names the entry as a whole.

### D-097: A pairing string carries a chosen name, never a guessed one

D-097 holds one decision and history. The decision is that a pairing string, the
text one user sends another to pair, may end in `#<name>`, carrying the sender's
chosen name and never a guessed one. The receiver uses it as the default for the
label, the name the receiver gives the sender, after what they typed and before
asking, and says when it does. It sits from "The format gains an optional trailing
name:" to "The receiver's order is". The reduction by `sanitizeName` and "Do not
move that write earlier." are conditions of the same decision, not separate choices.
The name is untrusted input, and it is written only after the two users' words match
in the two-word comparison, the check in which both read aloud two words computed
from their keys.

The Context paragraph ("D-093 made a label compulsory ... This connects them") and
"anything issued before today still parses" are history under W-32 (a decision
describes only the present), and so is the status line's "Completes D-094's
revisit". D-093 is the entry that requires a label at pairing, and D-094 the entry
in which the chosen label leads the view.

At f11e871, 1 line outside the entry cites D-097: `cmd/cogmer/sas_test.go:449`, "A
GUESSED name is never carried (D-097)", which is not a test fixture and means the
decision. `cmd/cogmer/main.go:1126` states this entry's rule and cites D-095 (a name
is chosen or guessed).

### D-098: A decision that changes what the system is updates the specification in the same pass

D-098 holds two decisions. The first decides that a decision that changes what the
system is updates the specification in the same pass. It sits at "The division of
labour was right and incompletely applied." The second decides that the
specification states only what the system is: a correction is a clean replacement,
with no superseded text and no note of what the old text said. It sits at "The
specification says what the system is, and nothing about what it was." The
alternative it weighs is annotating each correction with the reasoning it replaced
("this specification previously concluded that they did"), rejected in the same
paragraph. It could be reversed without reversing the first: the specification
could carry annotated corrections and still be updated in the same pass.

The Context ("nine decisions recorded in one day had produced no specification
edits") and "The worst of it was stated as a requirement" record what was observed,
which is a finding. The second names §29 (the experience we want) and §31
(implementation order) as contradicting the code. "This is the same failure as
review finding A1" is a finding too, about A1 in `docs/spec-review.md` (§19 never
says when delivery state advances). "What the pass changed" and its six-item list
are history under W-32 (a decision describes only the present), and so is "The
first version of this pass annotated each correction", the event behind the second
decision. The status line's "(spec pass done)" records an act completed, which is
history. The Revisit when reads "never", where the decision template in
`docs/writing.md` asks for "a condition somebody could observe". The
sentences after it ("this is a working rule rather than a decision with a
condition", the symptom, and the `git log --name-only` check) are a limit.

At f11e871, 2 lines outside the entry cite D-098. Neither is a test fixture, and
both mean the first decision: `CLAUDE.md:58` ("updates the spec in the same pass")
and `docs/decisions.md:5151`, the status of D-100 (each document answers one
question), "Generalises D-098", which means the
division between documents that the first decision rests on. `CLAUDE.md:28` states
the second decision ("Corrections are clean replacements: no superseded text") and
cites no decision.

### D-099: The injected block carries the label, and says it is the name to use

D-099 holds one decision and history. The decision is that the injected block, the
block of colleagues' turns the daemon adds to a user's prompt as context, carries the
label, the name the user gave that colleague at pairing, beside `speaker` and
`peerName`, omitted when absent, and that its framing says the label is the name to
prefer. It sits at "The framing says what the field is
for.", "All three travel." and "An absent label is an absent field". The `PeerFacts`
interface, at "`PeerFacts` groups the two lookups", is how the lookup reaches
`FormatTeamContext`. It is the mechanism of this decision, not a second one.

The Context paragraph (what D-094, the entry in which the label leads the view,
deferred, and the twenty-two call sites) is history under W-32 (a decision describes
only the present). So are "The gap was an inconsistency, not an omission.", which
says what the view and the model said before, "Six callers passed a function and
now pass the membership", and the status line's "Completes D-094". "The last check
initially scanned the whole block and matched the framing's own use of the word,
which is a reminder that a test over rendered text should read the payload."
records what happened when a test was written, which is a finding.

At f11e871, 5 lines outside the entry cite D-099. None is a test fixture. Four mean
the decision: `cmd/cogmer/transcript_test.go:318`, `cmd/cogmer/daemon.go:447`,
`cmd/cogmer/daemon.go:473` and `CLAUDE.md:237`, a row of the reading list in
`CLAUDE.md`. `docs/decisions.md:5981`, "D-099 prefers the label because a word pair
means nothing to a person weeks later", credits D-099 with a reason it does not give;
the reason is in D-094's Context.

### D-100: Each document answers one question, and a finding is not a commitment

D-100 holds two decisions. The first decides that each document answers one
question: a finding is evidence and stays out of the specification, which states the
requirement the finding justifies; a finding about someone else's software goes in
the behaviour registry; and current-state defects go in none of them. It sits at "A
finding is evidence; a specification statement is a commitment.", "A finding about
someone else's software belongs in the behaviour registry." and "Current-state
defects belong in none of them." The second decides that an always-loaded file,
`CLAUDE.md`, holds only what no check covers, and that where `doctor` checks a fact
it points at the behaviour rather than restating it. It sits at "`CLAUDE.md` is not
an exception, though it was first written as one." and "The rule that actually holds
is sharper: what earns a place in an always-loaded file is what no check covers."
The alternative it weighs is exempting `CLAUDE.md` as the place where a duplicate
"is a warning where warnings work", which it rejects. It could be reversed without
reversing the first: `CLAUDE.md` could hold recitals again while every other
document keeps its one question.

"This was caught in the draft of D-091, which listed what the code gets wrong
today", "though it was first written as one" and "Applying it removed fifteen lines
... The MCP section survived for the same reason", which says what applying the rule
did to `CLAUDE.md`, are history under W-32 (a decision describes only the present).
D-091 is the entry in which identity is advertised and location is discovered. The
Context ("Asked whether findings belong in the specification") is history as well.
The compaction example ("was true of one Claude Code version under one test")
records a test result, which is a finding. "Review finding C2 is valid and is not
asking for this." and the remark on C1 comment on two findings in
`docs/spec-review.md`: C2 (compaction has a phase but no standing requirement) and C1
(nothing in the specification acknowledges that it depends on undocumented
behaviour). They are support, not decisions.

At f11e871, no line outside the entry cites D-100.

### D-101: Peer connections are TLS pinned to the key already verified

D-101 holds one decision. It decides that every peer connection, over every
transport including the overlay (the tailcat network, which reaches a peer on
another network through a tunnel), is TLS with a required client certificate, and
that each side accepts only a key this machine has recorded: the named peer's key
when verifying, and any recorded peer's key when syncing. It sits in the title and in
the subsections "What TLS changes", "What "pinned" means", "The two situations a dial
can be in" and "Why the URL scheme is …". "The test is that the key is recorded, not
that it is verified" is part of this decision, and its rejected alternative, pinning
to verified keys, deadlocks. The last sentence of the subsection "Why this superseded
the interim guard" states that no setting disables encryption between peers: "There is
now no setting that disables encryption, which is …: a security property with an off
switch is one somebody eventually switches off." The environment variable that
subsection describes turned off the interim guard, the rule refusing bare-TCP
connections to other machines, and never disabled encryption. The entry weighs no
alternative of its own for the rule, so under the test this document applies it is
part of the decision.

The subsection "What was wrong" says what peer connections were before, plain HTTP,
which is history under W-32 (a decision describes only the present). So are the
subsection "Why this superseded the interim guard" apart from its last sentence, "A
peer built before this cannot connect at all" in the subsection "A note for whenever
a second person runs this", and the Context ("led to noticing that `peerURL` was
`http://`"). The entry's subsections are headings of its own rather than template
fields: "Considered and rejected" holds what the template calls Rejected, and "What
this does not do" holds what it calls Limits. "Note that being recorded is per
machine ... The attempt retries" is support for the first decision, not a decision.

At f11e871, 14 lines outside the entry cite D-101. None is a test fixture. Thirteen
mean the first decision: `cmd/cogmer/transport.go:89`, `cmd/cogmer/transport.go:136`,
`cmd/cogmer/peertls_test.go:60`, `cmd/cogmer/peertls_test.go:76`,
`cmd/cogmer/peertls.go:16`, `cmd/cogmer/offer_test.go:77`,
`cmd/cogmer/sas_test.go:388`, `cmd/cogmer/daemon.go:90`,
`cmd/cogmer/offline_test.go:58`, `docs/what-leaves-findings.md:27`,
`docs/what-leaves-findings.md:31`, `docs/decisions.md:4708` and
`docs/decisions.md:5377`. Of these, `cmd/cogmer/peertls_test.go:60` and `:76` and
`cmd/cogmer/offer_test.go:77` mean the part that pins to a recorded key rather than
a verified one. `CLAUDE.md:241` is a row of the reading list in `CLAUDE.md` and names
the entry as a whole.

### D-102: The reveal step tolerates a peer that has already finished

D-102 holds one decision and history. The decision concerns the exchange behind the
two-word comparison, in which each side first sends a commit (a hash that binds it to
a random value, its nonce, without showing it) and then a reveal (the nonce itself),
inside a loop that repeats until a ninety-second deadline. A reveal that finds the
other side no longer expecting a verification goes back to the loop, as a failed
commit does. At the top of the loop, the inbound check looks for a nonce the other
side has already revealed. It finds the nonce the finished side sent. It
sits at "It recovers by looping rather than by retrying that call."

The Context (TLS changed the timing, and a test "failed about one run in six") is
history and a finding. "The two halves of the exchange were not treated alike."
says what the code did before the decision, which is history under W-32 (a decision
describes only the present). "A test that passes is not a test that holds." states a
lesson from what happened, which is a finding rather than a decision.

At f11e871, 1 line outside the entry cites D-102: `cmd/cogmer/verify.go:269`, which
is not a test fixture and means the decision.

### D-103: An address belongs to a peer, and is stored in one place

D-103 holds one decision, findings and history. The decision is that a peer's
address is stored once, in `known_peers`, and that a room's sync targets are its
guests other than self joined to that table, with self excluded explicitly. It sits
at "The replacement is a join that already has both halves." and "One wrinkle,
deliberately<!-- writing: quotes the decision log --> made explicit." The explicit
exclusion of self is how the join is defined,
not a separate choice.

"Addresses live in two tables." and "Both writers hold the identity and discard it."
describe the schema before the decision, which is the evidence a finding holds.
"Five problems, one cause." is support whose facts are about that earlier schema.
"Checked: no room holds an address for a non-guest." records a check performed,
which is a finding. "This removes work rather than adding it." says what became
unnecessary, and "What the §4 lifecycle assumed." says what §4 (initial networking
strategy) described before; both are history under W-32 (a decision describes only
the present), as is the Context.

At f11e871, 13 lines outside the entry cite D-103. None is a test fixture. Twelve
mean the decision: `cmd/cogmer/sync.go:88`, `cmd/cogmer/verify_test.go:7`,
`cmd/cogmer/membership_test.go:812`, `cmd/cogmer/membership_test.go:848`,
`cmd/cogmer/membership.go:169`, `cmd/cogmer/membership.go:248`,
`cmd/cogmer/membership.go:622`, `cmd/cogmer/main.go:819`,
`cmd/cogmer/verify.go:172`, `docs/decisions.md:5650`, `docs/decisions.md:6234` and
`CLAUDE.md:241`, a row of the reading list in `CLAUDE.md`. One means something D-103
does not state: `docs/decisions.md:5523`, in D-104 (the overlay address is public and
stable), reads "This makes the ordering in
D-103 … rather than tidy: dialling one known address must come before any sweep".
D-103 states no order of dials. Its only mention of the sweep is that one store
removes "the argument for verification's fallback sweep".

### D-104: The overlay address is public and stable; admission moves to a list

D-104 holds three decisions. The first decides that the overlay (the tailcat
network, which reaches a peer on another network through a tunnel) runs with no
pre-shared key, so its published address holds no secret and is the same at every
start. It sits from "The address changes on every restart, and the cause was
measured." through "The two cannot both hold at that layer.", and at "The hedge is
genuinely lost, and is recoverable elsewhere." The second decides that the overlay
admits only recorded peers, through the library's allow-list of client node keys,
which is rebuilt from the peer list and updated without a restart. It sits at "So the
access-control half moves to a list, where it is better." The alternative it weighs
is the pre-shared key as the means of admission, in the table comparing the two. It
could be
reversed without reversing the first: without the allow-list the address would stay
public and stable. The third decides that the overlay's relay region is chosen once
and stored with the node key. It sits at "Also pin the region." The alternative it
weighs is choosing a region by latency at every start, which is the library's
default. It could be reversed without reversing the others, and `docs/open.md:38`,
under "Decided and not built", records a partial reversal: "Re-pick the overlay
relay when it cannot be reached." The title joins two statements with a semicolon,
which W-40 (an entry records one decision) names as a sign of more than one.

The restart measurement and its table ("Restarting a daemon twice and comparing the
published endpoints"), "Measured: a peer absent from the list cannot open a tunnel"
and "A measured cost: the list refuses by silence." record measurements, which are
findings. The rest of that last paragraph, on the order in which a verification
dials, is support. The Context (the question about sending a pairing string two
hours later from home, and "found the address is not stable at all") and "Today an
unrecorded peer is refused by TLS in milliseconds" are history under W-32 (a
decision describes only the present).

At f11e871, 11 lines outside the entry cite D-104. None is a test fixture. Three mean
the first decision: `cmd/cogmer/tailcat.go:187`, `docs/open.md:210` and
`docs/decisions.md:5556`. Four mean the second: `cmd/cogmer/pairview.go:167`
("rather than being refused by silence (D-104)"), `cmd/cogmer/daemon.go:94`
("through the tunnel without a restart (D-104)"), `cmd/cogmer/main.go:392` ("A peer
not on it cannot open a tunnel at all ... needs no secret in the published address
(D-104)"), which names the first decision as well, and `docs/decisions.md:5436` ("a
tunnel-level allow-list cannot be built from them (D-104)"). Three mean the third:
`cmd/cogmer/tailcat.go:237` ("strands everybody holding the old one (D-104)", in the
comment on the region), `docs/open.md:38` ("D-104 pins it so the address is
stable") and `docs/what-leaves-findings.md:31` ("D-104 pins a region").
`CLAUDE.md:241` is a row of the reading list in `CLAUDE.md` and names the entry as a
whole.

### D-105: An invitation travels over the channel pairing already established

D-105 holds one decision and history. The decision is that the host's daemon
delivers an invitation to a paired guest as an offer, a message telling the guest
that they have been admitted to a room, sent over the connection pairing set up. It
hands the host the string to send by hand only when it cannot reach the guest. It
sits at "An invitation becomes an offer delivered over the paired channel". The
guest polling for offers is its rejected alternative.

The Context, which records the exchange about one-way reachability, is history under
W-32 (a decision describes only the present). "§12 already describes the intended
shape, and the implementation is what diverged" is support in part and history in
part (§12, forming a room). "Today the string is the only path" is history. "This
mostly dissolves Phase 14." is a consequence for the plan of work, which is current
state of the kind `docs/open.md` holds.

At f11e871, 10 lines outside the entry cite D-105. None is a test fixture, and all
ten mean the decision: `cmd/cogmer/offer.go:19`, `cmd/cogmer/offer.go:201`,
`cmd/cogmer/offer_test.go:28`, `cmd/cogmer/membership.go:49`,
`cmd/cogmer/membership.go:821`, `cmd/cogmer/main.go:745`, `cmd/cogmer/main.go:798`,
`docs/decisions.md:5598`, `docs/decisions.md:5600` and `docs/decisions.md:5740`.

### D-106: An offer is delivered when it can work, not when it is made

D-106 holds three decisions. The first decides that inviting an unverified peer
records the admission at once and withholds the offer (the message telling a guest
they have been admitted) and the fallback string until the two have verified, that
completing verification delivers what was waiting, and that the invite reads as
admitted and queued, never as a refusal. It sits at "So the admission is recorded and
the offer is withheld.", "Why the gate stays open at all", "Verification gains an
effect", "The fallback string is withheld on the same terms." and "Which decides the
wording". The second decides that an offer waiting to be accepted is listed with
rooms, apart from joined ones, and that a withheld invitation is shown beside the
peer who can release it, naming the colleague and what to type rather than a count.
It sits at "Where the two waiting things surface" and "Name the person, do not count
the rooms." The alternative it weighs is a tally of withheld rooms ("A tally was
tried first and is close to useless here"). It could be reversed without reversing
the first: the waiting state could surface anywhere, or as a count, and offers would
still be delivered when verification completes. The third decides that `/peer-pair
<name>` resolves a recorded peer and resumes its pairing ceremony, the two-word
comparison, without asking for a label. It sits at "The appeal is to finish pairing,
not to verify." and "That required the command to accept a name." The alternative it
weighs is `/peer-pair` taking only a pairing string. It could be reversed without
reversing the others: the prompt could say "verify" and offer another command. D-107
(pairing again does nothing, loudly) builds on it: "the reason the command accepts a
name at all".

"What today's friction was hiding." says what hosts did before offers were pushed,
and "A tally was tried first" says what was tried; both are history under W-32 (a
decision describes only the present). "It previously took only a pairing string, so
`/peer-pair alice` hashed the literal text into an identifier and invented a peer"
records a defect, which is a finding. "Self is a guest of its own rooms and is never
in its own peer list (D-103)" is a fact the decision needs, which is support (D-103,
an address is stored in one place).

At f11e871, 10 lines outside the entry cite D-106. None is a test fixture. Seven mean
the first decision: `cmd/cogmer/offer.go:123`, `cmd/cogmer/offer.go:178`,
`cmd/cogmer/offer_test.go:69`, `cmd/cogmer/offer_test.go:106`,
`cmd/cogmer/membership.go:884`, `cmd/cogmer/main.go:940` and
`cmd/cogmer/verify.go:447`. Three mean the second: `cmd/cogmer/offer_test.go:140`
("The count is what lets /cogmer:peer-list say what verifying would release
(D-106)"), `cmd/cogmer/membership.go:905` ("WithheldFor counts the rooms a peer has
been admitted to but cannot be told about ... (D-106)") and `cmd/cogmer/main.go:671`
("Named, not counted ... which colleague it is and what to type (D-106)"). None means
the third.

### D-107: Pairing again with somebody already paired does nothing, loudly

D-107 holds one decision, a partial reversal and history. The decision is that
`/peer-pair` for a peer already paired says so, says since when, and stops, before
asking for any name, and that re-verification needs `--again`, which the message
offers only for the case that needs it. It sits at "So the three states answer
differently.", "Re-verification stays available and must be asked for." and "Checked
before anything is named."

The paragraph "A string for somebody already paired changes nothing, including the
address." states what holds now in its first sentence. Its next two sentences ("This
entry first took the address out of it ... That was wrong twice over, and is
superseded by D-108.") say what the entry first decided and that D-108 (pairing is
not how an address is updated) reverses it. That is history under W-32 (a decision
describes only the present) and a partial reversal under W-38 (a partial change
rewrites the earlier entry), and D-108 holds the reason. The Context ("running it on
a finished one silently began the whole ceremony again") is history.

At f11e871, 5 lines outside the entry cite D-107. None is a test fixture. Two mean
the decision: `docs/open.md:295` ("`/cogmer:peer-pair` repeated on a completed
pairing was worked in D-107 and D-108") and `docs/decisions.md:5749` ("What remains
true from D-107: already paired says so and stops"), which sits in a paragraph of
D-108 (pairing is not how an address is updated) that is history. Two mean the
reversed address write, and both sit in D-108's status line and Context:
`docs/decisions.md:5722` ("Supersedes part of D-107") and
`docs/decisions.md:5724` ("D-107 made a pairing string offered for an already-paired
peer update the stored address"). One is `docs/decisions.md:6081`, in D-114 (§3.1 is a
duty not to harm the session). It says that `common.sh` and D-107 both attribute
"silence toward the person" to §3.1 (first, do no harm). D-107 cites no § section at
f11e871.

### D-108: Pairing is not how an address is updated

D-108 holds one decision and history. The decision is that neither pairing nor any
dedicated command updates a peer's stored address, and that the peer's next
synchronization repairs it. It sits at "It is the wrong shape." and "An address
matters only when it is used, and every use repairs itself." The half about a
dedicated command is part of the same decision: the entry calls it the general form
of the same point ("That is the general form, and it is why no command for setting
one is needed either"), and it rests on the same support.

The Context (what D-107, in which pairing again does nothing, did, and "Rejected on
sight") is history under W-32 (a decision describes only the present), and so is
"What remains true from D-107: ... Only the address write is withdrawn." The status
line's "Supersedes part of D-107" records that this entry partly reverses D-107.

At f11e871, 3 lines outside the entry cite D-108. None is a test fixture, and all
three mean the decision: `cmd/cogmer/main.go:1255`, `docs/open.md:295` and
`docs/decisions.md:5705`, D-107's "superseded by D-108".

### D-109: Two is the target and nothing rules out more

D-109 holds one decision and history. The decision is to build for two people, spend
nothing on a third, and treat a design that cannot extend past two as a defect to be
argued for. It sits at "Decision. Build for two people in a room."

The Context, which says the rule "has been applied consistently and never written
down" and was recorded only in an aside of D-032 (the order of work), is history
under W-32 (a decision describes only the present).

At f11e871, 1 line outside the entry cites D-109: `CLAUDE.md:123`, which is not a
test fixture and means the decision.

### D-110: Host order: Claude Code, CoWork soon after, ChatGPT Desktop much later

D-110 holds one decision and history. The decision is that the hosts are Claude
Code, then Claude CoWork after a short interval, then ChatGPT Desktop after a long
one, and that the ordering licenses no adapter machinery. It sits at "Decision.
Three hosts, in order" and "This licenses nothing."

"A consequence for the specification, recorded and not resolved." records a question
left open, which D-111 (§3.8 is a test about hosts) has resolved since, so it is
history. The Context ("recorded nowhere ... appears nowhere in the repository") is
history under W-32 (a decision describes only the present).

At f11e871, 6 lines outside the entry cite D-110. None is a test fixture. Three mean
the decision: `docs/decisions.md:5918` ("D-110's note that a host lacking a hook
equivalent needs a different design", which is in the paragraph "What the long
interval settles"), `docs/decisions.md:6009` and `CLAUDE.md:130`. `CLAUDE.md:242`
is a row of the reading list in `CLAUDE.md` and names the entry, whose one decision
it is. Two mean the resolved paragraph, and both sit in D-111 (§3.8 is a test about
hosts):
`docs/decisions.md:5884`, its status "resolves the question D-110 left open", and
`docs/decisions.md:5886`, its Context "D-110 (host order) recorded that §3.8 ...".

### D-111: §3.8 is a test about hosts, not about Claude Code

D-111 holds one decision, a partial reversal, history and support. The decision is
that §3.8 reads "The host is launched and used unchanged", defines the host, and
states every clause of the test about the host, and that a host with no way in is
out of reach. It sits at "Decision. §3.8 now reads" and "Second, what the test says
about a host offering no way in."

The paragraph opening "First, 'things the host already loads' was ambiguous" decides
what the install clause means: "something the host would load anyway". D-113 (the
install test names a documented type) replaces that reading and rejects it by name,
so the paragraph is a partial reversal under W-38 (a partial change rewrites the
earlier entry). "The test itself did not change.", "The original said "already loads
on its own", and dropping it in summary produced a phrase that could not be read" and
the Context are history under W-32 (a decision describes only the present). "The rule
generalises; the evidence for it does not." is support.

At f11e871, 4 lines outside the entry cite D-111. None is a test fixture.
`CLAUDE.md:242` is a row of the reading list in `CLAUDE.md` and names the entry as a
whole. The other three mean the reversed reading of the install clause, and all three
sit in D-113 (the install test names a documented type): `docs/decisions.md:6004`, its
status "tightens D-111",
`docs/decisions.md:6006` ("D-111 stated the install clause as "something the host
would load anyway"") and `docs/decisions.md:6049` (""Something the host would load
anyway" (D-111's phrasing)").

### D-112: Several sessions from one machine are not told apart in the block

D-112 holds one decision. The decision is that the injected block, the block of
colleagues' turns the daemon adds to a user's prompt as context, does not mark which
session of one machine a turn came from. It sits at "Considered: a per-block
discriminator derived from `originSessionId`" and "Rejected, because it distinguishes
without informing."

Its Context paragraphs describe the schema and the block as they stand at f11e871
("Nothing stops one machine having two live sessions in a room"), which is support
rather than history under W-32 (a decision describes only the present). Its status
line, "active (considered and not built)", joins the two values W-30 (a decision has
its fields, in order) allows.

At f11e871, 1 line outside the entry cites D-112: `docs/open.md:238` ("the same
defect as the rejected discriminator (D-112), one level up: it distinguishes without
informing"), which is not a test fixture and means the decision.

### D-113: The install test names a type, and the type must be documented

D-113 holds one decision and history. The decision is that the project installs only
artifacts of a type among the host's documented and demonstrated extension
mechanisms. It sits at "Decision. This project installs only artifacts of a type
included in the host's demonstrated, documented extension mechanisms." Its two
halves, "Type, not instance." and "Documented, and demonstrated.", are parts of the
one decision.

The Context says what D-111 (§3.8 is a test about hosts) stated about the install
clause and why that phrasing fails, which is history under W-32 (a decision describes
only the present). The paragraph "What this catches that the old phrasing did not."
compares the rule with the phrasing it replaced, which is history as well. Its
example, `NODE_OPTIONS`, is also the ground of the rejected alternative "Demonstrated
alone".

At f11e871, 2 lines outside the entry cite D-113. Neither is a test fixture.
`CLAUDE.md:242` is a row of the reading list in `CLAUDE.md` and names the entry,
whose one decision it is. `CLAUDE.md:127` cites D-113 beside D-043 (the wire format is
defined separately from the stored row) for "No adapter machinery for a second host",
which D-113 does not state. D-110 (host order) states it, in "This licenses nothing."

### D-114: §3.1 is a duty not to harm the session, not a claim about data locality

D-114 holds one decision, a finding and history. The decision is that §3.1 is "First,
do no harm", stated as a general duty with known cases rather than as a list. It sits
at "Decision. §3.1 is First, do no harm." and "Stated as a duty, not a list".

"Twelve citations of §3.1 across the repository are about hooks (6), silence toward
the person (4)" records a count made at a date, and "the section did not contain what
they cite" records what that count was compared with. Both are findings. "§3.1 was
titled "Local first" and said two things"
and "(now including a dead daemon and a failed hook)" say what §3.1 was before, which
is history under W-32 (a decision describes only the present).

At f11e871, no line outside the entry cites D-114.

### D-115: A centralized component must trace to a disclosed tradeoff that benefits the person

D-115 holds one decision, a finding, history and current state. The decision is that
a compromise to privacy or decentralization is acceptable when it traces to a
recorded decision, is disclosed to the user it concerns, and benefits them, and that
failing a test names what is missing rather than forbidding the thing. It sits at
"Decision. A compromise to privacy or to decentralization is acceptable when all
three hold." and "Failing a test is not a prohibition on building the thing."

"Applied to what exists" scores the relay, the model provider and the release host
against the three tests, which is an assessment at a date and so a finding or current
state; `docs/what-leaves-findings.md` records the same components. Within it, the
release-host item and "It exists because there is nowhere else to publish from ...
retired when a public release host exists" describe a component that D-121 (the
repository is the release host) has retired, which makes them history. "The
disclosure test cannot currently be satisfied by anything." is current state, of the
kind `docs/open.md` holds. The Context ("three centralized or third-party components
already exist with nothing governing them") is history under W-32 (a decision
describes only the present).

At f11e871, 6 lines outside the entry cite D-115. None is a test fixture. Three mean
the decision: `docs/writing.md:254`, `docs/decisions.md:6578` ("still failing
D-115's benefit test") and `CLAUDE.md:134`. Three mean the part "Applied to what
exists". Two of them are about the release host, which D-121 retired:
`docs/what-leaves-findings.md:32` ("A pre-GA tag, and nothing else", ending "Retired
when a public release host exists (D-115)") and `docs/what-leaves-findings.md:59`
("It is tagged pre-GA rather than defended (D-115)"). The third,
`docs/decisions.md:6552` ("D-115 counts three centralized components"), means the
tally.

### D-116: Verification dials the peer it is verifying, and nobody else

D-116 holds one decision and history. The decision is that `verifyTargets(peerID)`
returns the address recorded for that peer plus `COGMER_PEERS`, with no fallback
sweep across other addresses. It sits at "Decision. `verifyTargets(peerID)` returns
the address recorded for that peer" and "Why there is no fallback sweep."

The Context ("`handleVerifyStart` passed `d.syncTargets()`"), "The sweep spent a dial
and a timeout per peer" and "the sweep was what made the sentence false" describe the
system before the decision, which is history under W-32 (a decision describes only
the present).

At f11e871, no line outside the entry cites D-116.

### D-117: The product is named `cogmer`, and nothing is carried forward

D-117 holds three decisions. The first decides that the name `cogmer` reaches the
module path, with the organisation's casing, the binary, the command directory, the
plugin manifest, the state directory, the local API header, the environment variable
prefix and the release path, and nothing cryptographic. It sits at "Decision.", "The
organisation casing is fixed at the same time." and "Nothing cryptographic moved".
The second decides that the rename carries no compatibility: the superseded signing
scheme is deleted, `signingBytes` refuses every other version, and no state directory
under the former name is migrated. It sits at "There is no compatibility surface,
because there is nothing to be compatible with." The alternative it weighs is keeping
the old scheme and migrating the state directory, which "every instinct here says".
It could be reversed without reversing the first: the old scheme could have been kept
under the new name. The third decides that the former name is struck from the
decision log rather than preserved. It sits at "The former name is struck from this
log rather than preserved." The alternative it weighs is preserving the name where it
appeared. It could be reversed without reversing the others, since it is a rule about
the log, independent of how far the rename reaches.

"What this unblocks." (Phase 13, and the overview command of D-096, in which a
command prefix names its target) and "the zero value that used to mean "stored before
the column existed"" are history under W-32 (a decision describes only the present).
"The only installation that ever existed ... was deleted rather than migrated"
records an event, and the fact it shows is support for the second decision. "The
command prefixes did not change" is history, and its reason is held by D-118 (the
manifest name is the command namespace).

At f11e871, 3 lines outside the entry cite D-117. None is a test fixture.
`docs/decisions.md:6488`, in D-120 (the manifest version is the release version),
"Noticed during the rename (D-117)", means the first decision.
`docs/decisions.md:3545`, in D-069 (signing namespaces are decoupled from the name),
"D-117 later deleted that scheme, on the ground that no such event existed
anywhere", means the second.
`CLAUDE.md:243` is a row of the reading list in `CLAUDE.md` and names the entry as a
whole.

### D-118: The plugin manifest name is the command namespace, and Claude Code forces it

D-118 holds one decision, findings and history. The decision is that commands keep
their `room-`, `peer-` and `self-` prefixes under the manifest namespace
(`/cogmer:room-create`), and that a test requires every command reference to carry
the manifest's namespace. It sits at "Decision. The prefixes stay." and "Test." The
title states the finding that makes the decision necessary. W-31 (a decision's title
states the decision) asks a title to state the decision instead.

"The manifest's `name` is documented as the namespace." and "Confirmed rather than
read.", with the throwaway plugin `nstest`, record documentation read and a behaviour
observed. They are support for the decision, and the observation is a finding. The
Context, which says what D-070 (slash commands carry a distinctive prefix) chose and
why, is history under W-32 (a decision describes only the present), as are "D-070's
protection becomes belt and braces" and "The manifest name is now … in a way it was
not.", a consequence stated against the time before. "D-096's overview command
cannot be what D-096 reserved." is a consequence for the overview command, which is
the second decision of D-096 (a command prefix names its target). "The half of this
finding that does belong there ... is in `open.md`" is history, since D-119 (no slash
command is reachable by the model) has decided that half.

At f11e871, 10 lines outside the entry cite D-118. None is a test fixture. Seven mean
the decision or the namespace fact it rests on: `cmd/cogmer/commandname_test.go:13`,
`cmd/cogmer/commandname_test.go:36`, `Shared Claude Sessions.md:2415`,
`docs/decisions.md:6334`, `docs/decisions.md:6468`, `docs/decisions.md:6533` and
`CLAUDE.md:150`. Of these, `docs/decisions.md:6468` and `docs/decisions.md:6533`
("for D-118's reason: registry checks observe a live session through hook payloads")
mean the paragraph "Not in the behaviour registry, and the reason is a limitation",
which is a limit of the decision's test. `CLAUDE.md:243` is a row of the reading list
in `CLAUDE.md` and names the entry as a whole. Two mean other parts:
`docs/decisions.md:6338`, in D-117 (the product is named `cogmer`), "D-118 says why",
means the consequence for the overview command of D-096 (a command prefix names its
target), and `docs/decisions.md:6417`, in D-119 (no slash command is reachable by
the model), "D-118 established that `plugin/commands/*.md` is loaded identically to
`skills/<name>/SKILL.md`", means the half of the finding that the entry hands to
`open.md`.

### D-119: No slash command is reachable by the model

D-119 holds one decision, history, a finding and current state. The decision is that
every file in `plugin/commands/` carries `disable-model-invocation: true`, and that a
test requires it. It sits at "Decision. `disable-model-invocation: true` on all
fourteen" and "Test."

"this was a live violation of the duty ... on every session, from the moment the
plugin shipped" is history under W-32 (a decision describes only the present).
"every command was a model-invocable skill: confirmed with a throwaway plugin"
records a confirmation, which is a finding. "Not done here. Migrating to the
`skills/<name>/SKILL.md` layout" is current state, and `docs/open.md:170` holds the
same item at f11e871 ("Move `plugin/commands/*.md` to the `skills/<name>/SKILL.md`
layout.").

At f11e871, 2 lines outside the entry cite D-119. Neither is a test fixture.
`cmd/cogmer/commandinvocation_test.go:12` means the decision. `docs/open.md:171`
means the part "Not done here": it says D-119 did not move the layout "in the same
pass".

### D-120: The manifest version is the release version, and `release.sh` writes it

D-120 holds one decision and history. The decision is that a release has one
version, which `release.sh` writes into `plugin.json` along with `VERSION` and
`checksums.txt`, and that a test compares the two. The title's "and" joins what the
number is with who writes it, which is one decision. It sits at "Decision. One number
for a release." and "The check."

The Context (`0.1.0` against `0.7.0`, noticed during the rename), "The manifest is
now `0.7.0`", which sits inside the Decision paragraph, and "So the stale number was
a release nobody would receive.", which is support stated as an event, are history
under W-32 (a decision describes only the present).

At f11e871, 2 lines outside the entry cite D-120. Neither is a test fixture, and both
mean the decision: `docs/open.md:119` and `CLAUDE.md:111`.

### D-121: The repository is both the release host and the marketplace

D-121 holds two decisions. The first decides that releases are served from the
repository's GitHub releases, with `release-url.txt` and `publish.sh` pointing there,
and that `publish.sh` checks every asset against `checksums.txt` over the public URL.
It sits at "The release host." and "The check." The second decides that the
repository carries `.claude-plugin/marketplace.json` beside `plugin/`, so that the
marketplace and the plugin share one repository name. It sits at "The marketplace."
The alternative it weighs is "A separate marketplace repository." It could be
reversed without reversing the first: the marketplace could move to its own
repository while releases stay on GitHub.

"The address is also removed from the history" records an act performed on the
repository, which is history under W-32 (a decision describes only the present).
"D-041's count is superseded; nothing else in it is" records a change to another
entry, which W-38 (a partial change rewrites the earlier entry) says is made in D-041
(ship as a Claude Code plugin). The Context ("Phase 13 was blocked on one thing") is
history. In the support "What is not gained is trust - the sha256 in `checksums.txt`
is still what authorises a binary, exactly<!-- writing: quotes the decision log --> as
it was when the transport was plain
HTTP", the closing clause describes the system before the decision, which is history.

At f11e871, 3 lines outside the entry cite D-121. None is a test fixture.
`scripts/publish.sh:8` means the first decision. Two name both decisions: `Shared
Claude Sessions.md:2793` ("carries the marketplace manifest beside the plugin, and
serves the release assets ... (D-121)") and `docs/open.md:25` ("carries the
marketplace manifest beside the plugin, and serves releases ... (D-121)").

### D-122: The two READMEs are split by whether you have installed it

D-122 holds one decision and history. The decision is that `README.md` serves
somebody who has not installed cogmer and `plugin/README.md` somebody who has, that
neither repeats the other, and that current state lives in `open.md`. It sits at
"Decision. `README.md` is what somebody who has not installed it needs" and "What was
deliberately<!-- writing: quotes the decision log --> kept at the top rather than
buried." Leading `README.md` with the
result of §30 (the important experimental question) is part of what `README.md`
contains, and its alternative is under Rejected.

The Context ("Until the repository was public ... a lab notebook") is history under
W-32 (a decision describes only the present). "The second reason is rot, and it had
already happened." records an event, which is a finding, and the fact it shows, that
two documents owning one fact both end up wrong, is support. "One README serving
both. It is what we had." and "That promise was already made and already broken" are
history.

At f11e871, 1 line outside the entry cites D-122: `CLAUDE.md:50`, which is not a
test fixture and means the decision.

### D-123: `stop` finds a daemon by the addresses it holds, and signals only cogmer

D-123 holds one decision and a finding. The decision is that `cogmer stop` finds the
process listening on each of this installation's addresses and stops it only if its
executable is named `cogmer`, with SIGTERM and then SIGKILL, reporting what it
stopped and what it declined to stop. The title's two statements are how the process
is found and which process is signalled, and the rejected alternatives cover both
together. It sits at "Decision. `cogmer stop` takes the addresses this installation
uses".

The 09-22 incident in the Context ("a daemon left from a test ... held that port")
records an observed event, which is a finding, and the entry uses it again in "on
09-22 it was not" and in the first rejected alternative.

At f11e871, 4 lines outside the entry cite D-123. One is a test fixture that uses the
number as a sample heading: `cmd/cogmer/structure_test.go:516` ("## D-123 - Old").
Two mean the decision: `docs/open.md:53` and `docs/open.md:137`. One,
`docs/writing.md:320`, uses the number as the cutoff of a check ("every decision
after D-123") and means no part of the entry.

### D-124: The writing test reads documents as Markdown, through goldmark

D-124 holds one decision. The decision is that `writing_test.go` parses documents
with goldmark, a Markdown parser written in Go, and checks only prose. It sits at
"Decision."

At f11e871, 19 lines outside the entry cite D-124. Seventeen are test fixtures that
use the number as a sample heading: `cmd/cogmer/writing_test.go:297-299`,
`cmd/cogmer/structure_test.go:227` and `cmd/cogmer/structure_test.go:517-529`. The
other two mean the decision: `cmd/cogmer/structure_test.go:269` and
`docs/writing.md:278`.

### D-125: A permitted use of a judgement word carries a marker with its reason

D-125 holds one decision. The decision is that a permitted use of one of the three
judgement words carries `<!-- writing: <reason> -->`, and that the test fails every
other use. It sits at "Decision."

At f11e871, 1 line outside the entry cites D-125: `docs/writing.md:283`, which is not
a test fixture and means the decision.

### D-126: Rules that need a reader are reviewed by a skill a maintainer runs

D-126 holds one decision. The decision is that the rules no test can decide are
reviewed by the `writing-review` skill, run by a maintainer, which reports quoted
findings, changes nothing and never blocks a commit. It sits at "Decision." One
support item holds a choice of the skill's: it reports a long item by naming the kind
of detail that does not belong, never by proposing a shorter wording. That
alternative is named only inside the same item, and the item applies a rule the guide
states, W-17 (keep an item short by leaving out kinds of detail), so it is support
rather than a second decision.

At f11e871, 3 lines outside the entry cite D-126. Two are test fixtures that use it as
a sample replacement number: `cmd/cogmer/structure_test.go:510` and
`cmd/cogmer/structure_test.go:523`. The third, `docs/writing.md:314`, means the
decision.

## What this does not show

Each range had one reader, and whether a choice counts as a second decision is a
judgement another reader could make differently. Three writing reviews applied the
same test to
every entry, and for five they still read it differently: the unsigned `have` map in D-044 (sync requests are signed), the unsigned version number in D-058 (signature schemes are
kept), the fetch-back check in `publish.sh` in D-067 (release assets are served from
the droplet), the operator-surface exemption in D-092 (an address is learned once, out of band), and the §30 result leading
`README.md` in D-122 (the two READMEs are split by whether you have installed it).
Each could be read as one more decision than this document counts. A grep finds a
citation only by its
number, so a citation that names an entry by its title alone is not counted.
