---
name: writing-review
description: Review documents or a commit message in this repository against docs/writing.md, for the rules no test can decide. Use when asked to review writing, before committing a change to a document, and as the last step of rewriting a document. Reports findings for a person to decide on; applies nothing.
---

# Writing review

This review covers the rules in `docs/writing.md` that need a reader. The
deterministic rules are enforced by tests, and this skill runs those tests rather
than repeating their work. It reports; it never edits the documents it reviews.

## 1. Decide what to review

Review what the person named: files, a commit, or a commit message. If they named
nothing, review the Markdown files changed in the working tree and the index
(`git diff --name-only HEAD -- '*.md'`), and say which files that was.

Only documents a person reads are in scope: the Markdown at the repository root,
in `docs/` and in `plugin/`, and commit messages. `plugin/commands/` holds
instructions to the model and is out of scope.

## 2. Read the rules

Read `docs/writing.md` in full, every time. It is the only statement of the
rules; this skill says how to review against them, never what they are. If
something here seems to disagree with the guide, the guide wins, and say so in the
report.

Identify the kind of each document (decision, `open.md` item,
specification, README, commit message) and read the guide's template for it.

## 3. Run the deterministic checks

```
WRITING_REVIEW=<file>,<file> go test ./cmd/cogmer -run TestReviewOneDocument -count=1
go test ./cmd/cogmer -run 'TestCitations|TestLaterDecisions' -count=1
```

The first runs the word, mark and structure checks on the named files even when
they are exempt. Report its output under its own heading, as test results, and do
not report those problems again as findings. The guide's rule index marks each rule
checked or judgement; the checked rules are the tests' job, and every judgement
rule is this review's.

## 4. Review

Read each document whole before judging any part of it. For each rule the guide's
index marks judgement, look for places the text breaks it. Report a finding only
when you can name the rule's ID and point to the sentence of the guide that the
text breaks. When a reading is
arguable, report it as a question rather than a finding.

Check every citation's few words against what it cites. Open the decision, the
section or the file and confirm the words describe it and the fact attributed to
it is there. A citation that resolves but says something the source does not
support is a W-18 finding; the tests only prove the number exists. Check counts,
line numbers and quotations the same way, against the version the document says it
describes.

### Length

A long item is not a finding. An item holds what its template says it holds, and
the guide keeps items short by leaving out kinds of detail, never by compressing
what belongs. When an item is long because it holds detail of another kind, the
finding names that kind and where it belongs: what happened on the way goes in a
finding, the reasoning for a choice in a decision, what changed in the commit
message. Never propose a shorter wording of an explanation. An allusion or an
aphorism in place of an explanation is a failure, and a suggestion that
introduces one is worse than none.

### Suggestions

A suggestion is optional. When you give one, it must follow every rule in the
guide itself, keep every fact and reference of the text it replaces, and be no
less explicit. If a fix needs a fact you do not have, say what is missing instead
of inventing it.

## 5. Verify the findings

Write the findings to a temporary file as a JSON array:

```json
[
  {
    "file": "docs/open.md",
    "quote": "text copied exactly from the source, Markdown included",
    "rule": "the rule's ID from the guide's index, then the sentence it breaks, quoted",
    "problem": "what is wrong, in one or two sentences",
    "suggestion": "optional replacement text"
  }
]
```

`file` is relative to the repository root. For a commit message, save the message
to a temporary file and give its absolute path.

Then run:

```
WRITING_FINDINGS=<path> go test ./cmd/cogmer -run TestReviewFindingsQuoteTheirDocuments -count=1 -v
```

It keeps each finding whose quote appears in its file, adds the line, and writes
the result to `<path>.verified`. It fails, listing them, for any finding whose
quote it could not find. Correct a dropped quote only by copying it again from the
file. If the text is not there, the finding was about text that does not exist:
discard it.

## 6. Report

Report from `<path>.verified`, grouped by file and in line order. Give each finding
its location as `file:line`, the quote, the rule's ID and sentence, the problem
and any suggestion.
Put the test results and any questions after the findings. Say how many findings
were dropped in step 5, so the person knows the review's error rate.

Apply nothing. The person decides what to change.
