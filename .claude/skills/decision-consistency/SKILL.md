---
name: decision-consistency
description: Check the decisions a push adds or changes against the other active decisions they touch, for contradictions, revived rejected alternatives and earlier entries left unrewritten. Use before a push that changes docs/decisions/, when the push hook asks for it, or when asked to check a decision's consistency. Runs in a subagent and reports; it edits no decision.
context: fork
agent: general-purpose
background: false
---

# Decision consistency

No test reads two decisions against each other: the tests check that each entry is
well formed and that what it cites exists. This check reads the decisions a push
changes beside the active decisions they touch, and reports where they disagree. It
reports and never edits a decision. The person decides what to change.

## 1. Find the changed decisions

If the arguments name decision files or numbers, check those. Otherwise take the
decisions the push would send:

```
git diff --name-only --diff-filter=AMR @{upstream}..HEAD -- docs/decisions/
```

With no upstream, compare with `origin/main`. Leave out `areas.md` and `README.md`.
If nothing is left, say so and stop.

## 2. Gather what each one touches

For each changed decision, read it whole, then gather:

- the active decisions in each of its areas, found as the `decisions` skill says,
  with `rg -l 'Status:\*\* (active|not built).*Areas:.*\b<area>\b' docs/decisions`;
- every decision it cites;
- every decision that cites it, with `rg -l 'D-NNN\b' docs/decisions`;
- where the decision is built: `rg -l 'D-NNN\b' cmd plugin scripts`, which finds the code and
  tests that cite it, and the open items that name it, with `rg -n 'D-NNN\b' docs/open.md`.

Read the **Decision.**, **Rejected.** and **Limits.** of each. An area with many
decisions is read by title first, and in full only where a title bears on the
changed decision.

## 3. Look for each of these

1. **Contradiction.** The changed decision and an active one cannot both hold: one
   says the system does something the other says it does not, or both decide the same
   question differently.
2. **A revived rejection.** The changed decision does what another active decision
   lists under **Rejected.**, and does not answer that decision's reason.
3. **A rejection now done elsewhere.** An alternative the changed decision rejects is
   what another active decision does.
4. **An earlier entry left as it was.** The changed decision reverses an earlier one
   and the earlier entry is not a tombstone (W-37 in `docs/writing.md`), or it changes
   part of what an earlier one decides and the earlier entry is not rewritten to what
   still holds (W-38). Read W-37 and W-38 in `docs/writing.md` before judging this, in
   full. A decision that makes an exception to an earlier one for a narrower case
   changes no part of it, and W-38 says the earlier entry is left as it is: never
   report that the earlier entry should name the exception, point to the later
   decision, or be rewritten, and never ask whether it should. The question is only
   whether the later entry says it is an exception and cites the earlier one, and
   whether it states only how it departs from it.
5. **A citation whose words do not match.** A cited decision's few words, or the fact
   attributed to it, are not what that decision says (W-18).
6. **A limit crossed.** The changed decision relies on something another decision's
   **Limits.** says does not hold.
7. **A status the code does not bear out.** A changed or cited decision says "not built"
   and code or tests cite it, or says "active" and nothing in the code, the tests or
   `docs/open.md` bears on it. The second is a question and not a finding, since a
   decision can govern a choice that no code names. A decision whose status just changed to
   "active" is read against any open item that says when it becomes active, and reported
   when that condition is not met. Whether a decision is built in full is the person's
   to judge: say what the code covers and what the decision says, and stop there.
8. **A "not built" decision nothing tracks.** A changed decision that is "not built", or a
   "not built" decision a changed one cites or relies on, is named by no open item. An
   open item that mentions it only in passing does not track it: the item must say what
   would build it or why it waits. Report it as a question, with the lines of any item
   that mentions it. A changed decision that relies on a "not built" one, and does not say
   so, is a question too.

Look at only the changed decisions and the "not built" ones they cite or rely on. Do not
sweep the status of every decision on each run.

Report a finding only when you can quote both texts. Report a reading that is
arguable as a question. Do not report wording or style; the writing-review skill
covers those.

## 4. Record that the check ran, and report

Record that this check ran for the commit being pushed, whatever it found, so that
the push hook lets the push through:

```
mkdir -p "$(git rev-parse --git-dir)/decision-consistency" &&
  touch "$(git rev-parse --git-dir)/decision-consistency/$(git rev-parse HEAD)"
```

Then report, grouped by changed decision: each finding's kind from section 3, the two
quotes with their files, and one or two sentences on what disagrees. Say which
decisions were read and how many. End with the questions, if any.
