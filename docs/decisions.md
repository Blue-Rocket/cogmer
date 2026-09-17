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
