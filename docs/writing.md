# Writing

How every document in this repository is written. Each kind of item has a template
below. The rules before the templates apply to all of them.

Every rule has an ID, and a failing check ends its message with the ID of the rule
it enforces, so a failure leads to the rule. Each rule is either checked, meaning a
test fails when the rule is broken, or judgement, meaning no program can tell and
the writer applies it. The `writing-review` skill reviews the judgement rules. IDs
are grouped by section, with gaps, so that a new rule never renumbers an old one.

## Rule index

| ID | rule | enforcement |
|---|---|---|
| W-01 | the words for the people involved | judgement |
| W-02 | put the answer first | judgement |
| W-03 | plain, complete sentences, one idea each | judgement |
| W-04 | state the fact, not a saying about it | judgement |
| W-05 | leave out the process | judgement |
| W-06 | describe the system, not the change | judgement |
| W-07 | numbers with units, dates for observations | judgement |
| W-08 | name a mechanism by what it does | judgement |
| W-09 | cite only what exists | checked |
| W-10 | § names a section of the specification | checked |
| W-11 | give every reference a few words | judgement |
| W-12 | bullets only for real lists | judgement |
| W-13 | backticks only for code | judgement |
| W-14 | bold only for template fields and `open.md` openers | checked |
| W-15 | no header over a section of three lines or fewer | checked |
| W-16 | do not restate in a closing sentence | judgement |
| W-17 | keep an item short by leaving out kinds of detail | judgement |
| W-18 | every claim is true of what it names | judgement |
| W-19 | nothing durable cites working material | checked |
| W-20 | no em-dash | checked |
| W-21 | none of the words in the table | checked |
| W-22 | a permitted judgement word carries a marker | checked |
| W-23 | none of the other listed words | checked |
| W-24 | no status decoration | checked |
| W-30 | a decision has its fields, in order | checked |
| W-31 | a decision's title states the decision | judgement |
| W-32 | a decision describes only the present | judgement |
| W-33 | a decision uses none of the history words | checked |
| W-34 | every supporting fact names a source | checked |
| W-35 | a supporting fact holds now | judgement |
| W-36 | a decision always has **Rejected.** | checked |
| W-37 | a reversed decision becomes a tombstone | checked |
| W-38 | a partial change rewrites the earlier entry | judgement |
| W-39 | a rewrite keeps every fact | judgement |
| W-40 | an entry records one decision | judgement |
| W-41 | a decision never points to open work | checked |
| W-50 | an `open.md` item names the problem | judgement |
| W-51 | a long `open.md` item holds detail of another kind | judgement |
| W-52 | delete a finished `open.md` item | judgement |
| W-55 | a pattern applies beyond this project, and this project follows it | judgement |
| W-56 | a pattern names nothing in this repository | checked |
| W-57 | a pattern states one rule, then its reason | judgement |
| W-60 | a findings document has its fields and sections | checked |
| W-61 | findings are observations, never instructions | judgement |
| W-70 | a behaviour's `Title` is an observed fact | judgement |
| W-71 | a behaviour's `Reliance` starts with what breaks | checked |
| W-72 | a behaviour check's error says what changed | judgement |
| W-80 | one requirement per paragraph of the specification | judgement |
| W-81 | the specification carries no dates | checked |
| W-82 | every requirement is grounded in a benefit | judgement |
| W-83 | the specification carries no status and no plan | judgement |
| W-90 | the form of a commit message | judgement |

## The reader

The reader was not here when the work was done. They may be reading months later,
with no memory of the session, looking for one answer. They may not write code.

W-01. Say "user" for someone using cogmer, "colleague" for another user in the same
room, and "maintainer" for someone changing cogmer. Say "person" only where what
matters is a human as distinct from the model or a program: "a person sees only
what the model says". Never say "developer", since a user may not write code. Where
"user" could be read as the role of a message in a Claude Code conversation, as in
a user turn, say which is meant.

## Rules

W-02. Put the answer first. The first sentence of any item says what is true, what
was decided or what is wrong. Background, evidence and history come after it, and
only if the reader needs them to trust or apply the answer.

W-03. Write plain, complete sentences in paragraphs. Use one idea per sentence and
the active voice. When a sentence needs a second reading, split it.

W-04. State the fact rather than a saying about it. "A check that cannot fail reads
as protection" makes the reader work out the rule. "Every behaviour check has a
test that makes it fail" states the rule.

W-05. Leave out the process. Do not describe how you got to the answer ("Asked
whether…", "Reading it closely turned up…", "The honest result is…"). If the route
matters, because something was tried and failed, record what was tried and what
happened.

W-06. Describe what the system is, not what you changed. "The parser reads the
config at startup", not "Updated the parser to read the config at startup". Change
history belongs in commit messages.

W-07. Give numbers with units, and dates for observations. "16s on 09-22", not "a
few seconds". Say whether something was observed or only read from the code.

W-08. Name a mechanism by what it does, not by our word for it. If a term of our
own has to appear ("watermark", "fence", "the view"), say what it is the first time
it appears in that item.

W-09. Cite only what you have checked exists.

W-10. The mark § always names a section of the specification. Refer to a section of
the same document in words: "section 4", not "§4".

W-11. Give every reference a few words. "D-054 (verification gates sync)", not
"D-054".

W-12. Use bullets only for real lists: options, steps in order, items to compare.
Do not break an explanation into bullets.

W-13. Use backticks only for things that appear in code or at a terminal:
identifiers, file names, commands, environment variables. Not for emphasis.

W-14. Use bold only for an item's opening sentence in `open.md`, and for the field
names the templates define. Nowhere else. In particular, do not open a paragraph
with a bold label and a fragment ("**Where the code lives.** Signalling needs…").
If a thing needs a name and an explanation, write a sentence.

W-15. Use headers only on sections long enough to need one. A three-line section
does not.

W-16. Stop when the item has said what it needs to. Do not restate it in a closing
sentence.

W-17. Keep an item short by leaving out kinds of detail, never by compressing what
belongs in it. Each template says what an item holds, and process, history and
restated decisions are not among them. An item that holds only what belongs is not
too long, whatever its length. An allusion or an aphorism in place of an
explanation makes an item shorter and fails the reader.

W-18. Every claim is true of what it names. A count, a line number, a quotation, and
what a citation says its source holds must each match the source. When the source
can change, as a line number or a count can, say which version the claim describes:
a commit, or a date. A citation that resolves to a real entry but attributes to it
something it does not say breaks this rule, although W-09's test passes.

W-19. Working material lives in `docs/work/`. It serves one open item, such as a
checkpoint of a rewrite in progress, and is deleted with that item. It may use the
findings template's **Run:** and **Result:** fields. Nothing durable cites it, because
the citation would outlive it: only `docs/open.md` and other working material may.
A fact in it that stays true after its item is done moves to a durable record or an
open item before it is deleted.

## Words and marks to avoid

These are the habits this repository overuses, counted on 09-22 across the five main
documents; the counts include every use.

W-20. Do not use an em-dash. The one allowed is the one in a decision's heading,
`## D-NNN — <title>`. The em-dash is banned for what it usually does here: it lets
a finished sentence carry a second idea, often as a twist or a reveal ("handshakes
— seconds of work"), or holds a sentence open around an aside. Replacing it with a
comma or a semicolon keeps that structure and defeats the rule. Split the sentence
instead. It appeared 1,020 times; use two sentences, or a colon only when what
follows explains or lists what came before.

W-21. Do not use the words in this table.

| avoid | count | use instead |
|---|---|---|
| "load-bearing" | 18 | say what depends on it |
| "honest", "honestly" | 18 | drop it; state the fact |
| "the point", "is the point" | 19 | say what the thing is for |
| "not merely" | 5 | "also", or two sentences |

W-22. Use "precisely" and "exactly" (114 uses) only where the word carries meaning,
and "deliberately" (39 uses) only where it contrasts with an accident; otherwise
drop it. A use that stays carries a marker directly after the word, giving the
reason it stays. The marker is an HTML comment, so it does not show when the
document is rendered:

```markdown
The relay was chosen deliberately<!-- writing: contrasts with an accident -->.
```

No other word takes a marker.

W-23. Also avoid: leverage, utilize, robust, seamless, comprehensive, ensure,
nuanced, testament, tapestry, delve, crucial. Say what actually happens.

W-24. No status decoration: no check marks, emoji, "Note:" or "Important:".

## Templates

### A decision (`docs/decisions.md`)

A decision entry is a log of what was decided and what supports it. It is not the
story of how the decision was reached. What was observed, measured or tried goes in
a findings document, and the decision cites it.

```markdown
## D-NNN — <what was decided, as a statement>

**Date:** YYYY-MM-DD · **Status:** active | not built

**Decision.** <One to three sentences saying what we do. The first sentence must
make sense on its own.>

**Support.**
- <A fact the decision rests on.> <Its source: a specification section, another
  decision, a findings document and section, a file and function, or external
  documentation with its URL.>

**Rejected.**
- *<Alternative.>* <Why not, in one or two sentences, citing a source where one
  exists.>

**Limits.** <What this does not cover or decide, and what it assumes that nothing
checks.>

**Revisit when** <a condition somebody could observe>.
```

W-30. An entry follows this template: a **Date:** line whose status is "active" or
"not built", then **Decision.**, **Support.** with a list under it, **Rejected.**,
an optional **Limits.** and **Revisit when**, in that order. Omit **Limits.** when
there are none.

W-31. The title is the decision, not the topic: "`stop` finds a daemon by the
addresses it holds", not "Stopping the daemon".

W-32. An entry describes the decision as it stands now, so that a reader can
understand what we do with as little to hold in mind as possible. It says nothing
about what was there before: not what the system used to do, not what this
replaced, not who asked or what happened on the way. A decision states only what it
decides. Something decided elsewhere appears only when the reader needs it to apply
this one, and then as a citation with a few words, not restated.

W-33. Words that describe the system as it stood when the decision was made
("stays", "is kept", "continues to", "already", "no longer", "used to",
"previously", "replaced") are history whatever their tense, and an entry uses none
of them.

W-34. Every item under **Support.** has a source, and the source is where the fact
is kept: a § section of the specification, a decision, a behaviour such as B04, a
file named in backticks, or a URL. If the only record of a fact would be the
decision itself, write the findings document first.

W-35. A support item states a fact that holds now. When the evidence is something
that happened, such as a failure or an earlier design, the item states the fact it
showed, and the finding it cites holds the event. "An environment variable has the
same value in every session a machine starts", not "the variable made a command act
on the wrong room".

W-36. **Rejected.** is required whenever a real alternative was weighed, which is
the reason for writing an entry at all. When no alternative was weighed,
**Rejected.** says why there was none.

W-37. When a decision reverses an earlier one, the earlier entry is replaced by a
tombstone, and the reason for the reversal is recorded as a finding. The new
decision cites neither: it stands on its own support. A tombstone keeps the number
and the title, so that references to it still resolve, and nothing else:

```markdown
## D-NNN — <original title>

**Status:** withdrawn YYYY-MM-DD. Replaced by D-MMM (<words>). Why:
`docs/<topic>-findings.md`, "<section>".
```

The finding says what the withdrawn decision was, what showed it wrong, and the
evidence, so that nobody has to reconstruct the argument to avoid repeating it.

W-38. When a later decision changes only part of an earlier one, rewrite the earlier
entry so that it states only what still holds, and record the removed part's
reason as a finding in the same way.

W-39. When rewriting an entry, keep every fact, reference and number from the
original, either in the entry or in the findings document it now cites. Remove one
only by saying in the commit message what was removed and why.

W-40. An entry records one decision. A second decision, one that has alternatives
of its own and could be reversed without reversing the first, gets its own entry.
Two signs that an entry holds more than one: a title joining two statements with a
semicolon or "and", and a paragraph opening with a bold statement that is not a
template field. When an entry is split, the decision most citations mean keeps the
number, each other decision takes a new number at the bottom of the log, and every
citation of the old number is checked against its few words (W-11) and pointed at
the entry that now holds what it describes.

W-41. A decision never points to open work: not to `docs/open.md`, not to working
material in `docs/work/`, and not to a task in a tracker. Open work is deleted or closed when it is resolved, so a pointer to it
is written knowing it will go stale. **Limits.** states the decision's scope, what it
does not decide, which stays true after a later decision settles the question. The
open item points the other way, citing the decision it concerns: "D-123 (finding a
daemon by the addresses it holds) leaves undecided whether…". The same holds for the
specification and for findings, which W-80 and W-61 already keep free of plans.

### An item in `docs/open.md`

```markdown
**<The problem or task, in one sentence a reader could act on.>** <What happens
now, observed or read from the code, with the date. Who it affects and how.>

<What would fix it. If there are several options, list them, each with its cost.
If something is unknown, say what would answer it.>
```

W-50. The opening sentence names the problem, not the fix: "`/cogmer:self-status`
on first use fails instead of answering", not "Make cli.sh exit 0".

W-51. An item past about 25 lines usually holds detail of another kind: what
happened on the way belongs in a finding, and the reasoning for a choice in a
decision.

W-52. Delete an item when it is done; never mark it done.

### A pattern (`docs/patterns.md`)

```markdown
P-NN. <The rule, as an instruction any project could follow.> <Why it holds, in
general terms.>
```

W-55. A pattern belongs in the list only if it applies beyond this project and this
project follows it. A choice that makes sense only here is a decision, in
`docs/decisions.md`. A pattern this project does not follow belongs in no list of
its.

W-56. A pattern names nothing in this repository: no decision, no section of the
specification, no behaviour, and not the project's name. A pattern stated in this
project's terms goes stale when those terms change, and cannot be read in another
project.

W-57. A pattern states one rule, as an instruction, and then the reason it holds,
never an example from this project. Its ID is never renumbered or reused.

### A findings document (`docs/*-findings.md`)

```markdown
# <What was tested>

**Run:** <date, machines and platforms, versions, anything unusual about the setup>

**Result:** <one sentence.>

## What was run

<A table or numbered steps, enough to repeat it.>

## What we found

### <Each finding, as a statement>

<What was seen, what caused it, and what was done about it, with a commit or
decision reference.>

## What this does not show

<The limits of the test.>
```

W-60. A findings document has **Run:**, **Result:** and the three sections this
template names.

W-61. Findings record what was seen. They are observations, never instructions: a
finding says what is true, and what to do about it goes in `docs/open.md`, where
it is deleted when done. A classification counts as an observation when it applies
a rule the guide states, such as "this entry holds two decisions" under W-40. The
requirement a finding justifies goes in the specification, and a finding about
someone else's software goes in the behaviour registry.

### A behaviour-registry entry (`cmd/cogmer/behaviors.go`)

W-70. `Title` states the behaviour as an observed fact, in the present tense:
"UserPromptSubmit stdout is injected into the pending turn".

W-71. `Reliance` is for somebody debugging at 2am. It starts with what breaks: "If
this changes, <what breaks>. <How it shows up, or that it fails silently>."

W-72. The error the check returns says what changed and where to look next.

### A requirement in the specification (`Shared Claude Sessions.md`)

W-80. State one requirement per paragraph, in the present tense: "The daemon stops
only a process it has identified as a cogmer daemon." Give at most one sentence of
reason, and point to the decision for the rest. No findings and no history.

W-81. The specification carries no dates.

W-83. The specification carries no status and no plan. Nothing in it says what is
built, finished, partial, deferred or next, and nothing orders the work: the phases
the work was planned in are in `docs/phases.md`, actions not yet taken are in
`docs/open.md`, and what has been built is the code. A requirement is stated the same
way whether or not it is met yet. Status in the specification goes stale with every
commit, and a reader cannot tell a requirement from a report.

W-82. Every requirement is grounded in a benefit to the user or to the product
owner, and its sentence of reason says what the benefit is and whose it is. A
requirement that benefits neither does not belong in the specification. Where a
requirement gives up privacy or decentralization, the benefit must be the user's,
and a benefit to the product owner alone does not justify it (D-115, a centralized
component must trace to a disclosed tradeoff that benefits the person).

### A commit message

```text
<What changed, in the imperative, under about 60 characters>

<What was wrong or missing, and how it showed. What changed. How it was
checked, if it was.>

Co-Authored-By: …
```

W-90. Write for somebody reading `git log` who wants to know whether this commit
matters to them. Wrap at 72 columns. The body is usually 3 to 10 lines.

## Enforcement

`TestDocumentsFollowWritingGuide`, in `cmd/cogmer/writing_test.go`, checks every
document a person reads: the Markdown files at the repository root, in `docs/` and
in `plugin/`. It reports each word and mark listed above, with its line. It reads
the documents as Markdown and checks only prose, so code spans and code blocks are
exempt (D-124, the writing test reads documents through goldmark). The files in
`plugin/commands/` are instructions to the model and are not checked.

A use of "precisely", "exactly" or "deliberately" passes only with a marker
directly after it. The test also reports a marker that gives no reason, and a
marker that follows no word it can excuse (D-125, a permitted use carries a
marker).

Documents not yet rewritten are listed in `writingNotYetRewritten` and exempt. The
test fails when a listed document passes, so that it comes off the list in the
commit that rewrote it. This guide is never checked, because it quotes every word
it bans.

`TestCitationsNameThingsThatExist`, in `cmd/cogmer/citations_test.go`, checks
every citation in the documents, the Go sources, the scripts and the plugin. Each
D-NNN must have an entry in `docs/decisions.md`, each § a numbered heading or
numbered step in the specification, and each BNN an entry in the behaviour
registry. No document is exempt. A decision that was withdrawn keeps its tombstone,
so citations of it still resolve.

`structure_test.go` checks the structure the templates set. Bold appears only as a
template's field names, in decisions, findings documents and working material, and
as the opening of an item in `open.md`. A findings document has **Run:**, **Result:**
and the three sections its template names. No header sits over a section of three
lines or fewer; a document's title and the headers a template defines are exempt.
The specification carries no dates. No document other than `open.md` and working
material cites `docs/work/`. The patterns document names no decision, section,
behaviour or the project itself. These checks share the exemption list with the
word check. No length limit is checked, because a limit is met most cheaply by
compressing, which is the failure W-17 describes. The header rule is not such a
limit: what it asks for is removing the header.

`TestBehaviorRelianceStartsWithWhatBreaks`, in `structure_test.go`, checks that
each behaviour's `Reliance` starts "If this changes". Behaviours written before that
template are listed in `relianceNotRewritten`, which only shrinks.

`TestLaterDecisionsFollowTemplate`, in `structure_test.go`, checks every decision
after D-123, reading the log as Markdown so that a field name inside code does not
count as the field. It checks W-30, W-33, W-34, W-36 and W-41 on each; for W-41 it
rejects a mention of `open.md` or `docs/work/`, or a link to a ClickUp, GitHub issue,
Jira or Linear task. Every tombstone,
whatever its number, must hold only its status line, and its "Why:" must name a
findings document and a heading that exist.

`TestWritingRulesAreIndexed`, in `cmd/cogmer/writingrules_test.go`, checks that
the rule index above and the checks agree. Every rule in the index has a paragraph
starting with its ID. Every rule marked checked is cited by some check's message,
and every ID a check cites is in the index and marked checked.

The rules no test can decide are reviewed by the `writing-review` skill in
`.claude/skills/writing-review/` (D-126, rules that need a reader are reviewed by a
skill a maintainer runs). Run it on a document before taking it off the exemption
list, and on a change to a document before committing it. It runs the checks above
on the named documents whether or not they are exempt, reviews the rest, and
reports findings without changing anything.
