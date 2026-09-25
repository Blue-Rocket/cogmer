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
| W-25 | each fact is written in one document | judgement |
| W-26 | a file loaded every session holds what no check covers | judgement |
| W-27 | a present defect is written only in `open.md` | judgement |
| W-28 | the decision log holds only decisions about the system | judgement |
| W-29 | a rule says what to do and why | judgement |
| W-30 | a decision has its fields, in order | checked |
| W-31 | a decision's title states the decision | judgement |
| W-32 | a decision describes only the present | judgement |
| W-33 | a decision uses none of the history words | checked |
| W-34 | every supporting fact names a source | checked |
| W-35 | a supporting fact holds now | judgement |
| W-36 | **Rejected.** names only an alternative likely to be proposed again | judgement |
| W-37 | a reversed decision becomes a tombstone | checked |
| W-38 | a partial change rewrites the earlier entry | judgement |
| W-39 | a rewrite keeps every fact | judgement |
| W-40 | an entry records one decision | judgement |
| W-41 | a decision never points to open work | checked |
| W-42 | evidence is cited by the commit that recorded it | judgement |
| W-50 | an `open.md` item names the problem | judgement |
| W-51 | a long `open.md` item holds detail of another kind | judgement |
| W-52 | delete a finished `open.md` item | judgement |
| W-55 | a pattern applies beyond this project, and this project follows it | judgement |
| W-56 | a pattern names nothing in this repository | checked |
| W-57 | a pattern states one rule, then its reason | judgement |
| W-58 | a value states a commitment, not a technique | judgement |
| W-59 | nothing cites a value or a pattern | checked |
| W-65 | an explanation cites nothing | checked |
| W-70 | a behaviour's `Title` is an observed fact | judgement |
| W-71 | a behaviour's `Reliance` starts with what breaks | checked |
| W-72 | a behaviour check's error says what changed | judgement |
| W-75 | `README.md` is for somebody who has not installed it | judgement |
| W-76 | `plugin/README.md` is for somebody who has | judgement |
| W-80 | one requirement per paragraph of the specification | judgement |
| W-81 | the specification carries no dates | checked |
| W-82 | every requirement is grounded in a benefit | judgement |
| W-83 | the specification carries no status and no plan | judgement |
| W-84 | the specification cites no decision | checked |
| W-85 | a change to what the user experiences changes the specification with it | judgement |
| W-90 | the form of a commit message | judgement |
| W-91 | a commit that records evidence says how it was run | judgement |

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
**Run:** and **Result:** fields to record a run. Nothing durable cites it, because
the citation would outlive it: only `docs/open.md` and other working material may.
A fact in it that stays true after its item is done moves to a durable record or an
open item before it is deleted.

## Words and marks to avoid

W-20. Do not use an em-dash. The one allowed is the one in a decision's heading,
`## D-NNN — <title>`. The em-dash is banned for what it usually does here: it lets
a finished sentence carry a second idea, often as a twist or a reveal ("handshakes
— seconds of work"), or holds a sentence open around an aside. Replacing it with a
comma or a semicolon keeps that structure and defeats the rule. Split the sentence
instead: use two sentences, or a colon only when what follows explains or lists what
came before.

W-21. Do not use the words in this table.

| avoid | use instead |
|---|---|
| "load-bearing" | say what depends on it |
| "honest", "honestly" | drop it; state the fact |
| "the point", "is the point" | say what the thing is for |
| "not merely" | "also", or two sentences |

W-22. Use "precisely" and "exactly" only where the word carries meaning, and
"deliberately" only where it contrasts with an accident; otherwise
drop it. A use that stays carries a marker directly after the word, giving the
reason it stays. The marker is an HTML comment, so it does not show when the
document is rendered:

```markdown
The relay was chosen deliberately<!-- writing: contrasts with an accident -->.
```

No other word takes a marker. Nothing in the text separates a permitted use from an
emphatic one, so without a marker the test could only fail every use or none. A list
of exceptions by file and line in the test would move with every edit above them,
and would keep the reason away from the word it excuses. Nothing checks that a
marker's reason is true.

W-23. Also avoid: leverage, utilize, robust, seamless, comprehensive, ensure,
nuanced, testament, tapestry, delve, crucial. Say what actually happens.

W-24. No status decoration: no check marks, emoji, "Note:" or "Important:".

## Where a thing is written

The table in `CLAUDE.md`, "Where a thing gets written down", names each document and
the question it answers.

W-25. Write each fact in the one document whose question it answers, and nowhere
else.

W-26. A file loaded into every session, such as `CLAUDE.md`, holds only what no check
covers: a prohibition, a judgement, a residual risk. Where a test or a behaviour
check covers a fact, the file points at the check and states the rule that depends
on it, because the unchecked copy is the one that goes wrong without anybody seeing.

W-27. A defect in the present code is written only in `open.md`. A sentence about
what the code gets wrong is false once the defect is fixed, and in a decision or the
specification it makes the text around it false with it.

W-28. The decision log holds only decisions about the system: what it does, and the
detail beneath the specification. How a document is written, where a thing is
written, and how the documents are checked are rules in this guide, where a rule and
its reason can be corrected together.

W-29. A rule in this guide states what to do. It gives a reason only where the rule
would surprise a reader, and the reason says what the rule protects, in general
terms, never what happened while the rule was absent. It names an alternative only
where a maintainer would otherwise be likely to propose it again, in one sentence.

## Templates

### A decision (`docs/decisions.md`)

A decision entry is a log of what was decided and what supports it. It is not the
story of how the decision was reached, and what was observed on the way is cited
under W-42.

W-84 says how a decision relates to the specification.

```markdown
## D-NNN — <what was decided, as a statement>

**Date:** YYYY-MM-DD · **Status:** active | not built

**Decision.** <One to three sentences saying what we do. The first sentence must
make sense on its own.>

**Support.**
- <A fact the decision rests on.> <Its source: a specification section, another
  decision, a commit, a file and function, or external
  documentation with its URL.>

**Rejected.**
- *<Alternative.>* <Why not, in one or two sentences, citing a source where one
  exists.>

**Limits.** <What this does not cover or decide, and what it assumes that nothing
checks.>

**Revisit when** <a condition somebody could observe>.
```

W-30. An entry follows this template: a **Date:** line whose status is "active" or
"not built", then **Decision.**, **Support.** with a list under it, an optional
**Rejected.**, an optional **Limits.** and **Revisit when**, in that order. Omit an
optional field when it has nothing to hold.

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
file named in backticks, a commit, or a URL. A commit is cited as `<commit>` for its
message, or as `<commit>:<path>` for a file as that commit holds it. If the only
record of a fact would be the decision itself, commit it first, usually in a commit
message under W-91, and cite that commit.

W-35. A support item states a fact that holds now. When the evidence is something
that happened, such as a failure or an earlier design, the item states the fact it
showed, and the commit it cites holds the event. "An environment variable has the
same value in every session a machine starts", not "the variable made a command act
on the wrong room".

W-36. **Rejected.** names an alternative only where a maintainer would otherwise be
likely to propose it again, which is the test W-29 sets for a rule. An alternative
that nobody would propose again is left out, and an entry with none omits the field.

W-37. When a decision reverses an earlier one, the earlier entry is replaced by a
tombstone. A tombstone keeps the number and the title, so that references to it
still resolve, and nothing else:

```markdown
## D-NNN — <original title>

**Status:** withdrawn YYYY-MM-DD. Replaced by D-MMM (<words>).
```

The replacing decision's **Rejected.** names the withdrawn approach in words, not
by its number, and says why it does not hold, citing the evidence under W-42. A
withdrawn decision is the alternative most likely to be proposed again, and the
reason against it has to stay where it can be corrected.

A decision that W-28 places in this guide becomes a tombstone of a second kind,
naming the rule that holds it, or the section when no one rule does:

```markdown
**Status:** moved YYYY-MM-DD to `docs/writing.md`, W-NN (<words>).
**Status:** moved YYYY-MM-DD to `docs/writing.md`, "<heading>".
```

W-38. When a later decision changes only part of an earlier one, rewrite the earlier
entry so that it states only what still holds, and name the removed part in the
later decision's **Rejected.** in the same way.

W-39. When rewriting an entry, keep every fact, reference and number from the
original, either in the entry or in the commit it now cites. Remove one
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
specification, which W-83 keeps free of plans.

W-42. Evidence of a past run is cited by the commit that recorded it, as
`<commit>:<path>` for a file or `<commit>` for a message, and no document retells it.
What someone else's software does is a behaviour in `cmd/cogmer/behaviors.go`, and
what cogmer's own code does is a test. A commit never changes, so a citation of it
needs no upkeep, while a retelling is a new set of claims about the run that every
edit has to check against the original again. A findings document kept in the tree
is that retelling. A reader has to run `git show` to read the evidence, and evidence
has to be committed before anything can cite it.

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
happened on the way belongs in a commit message, and the reasoning for a choice in a
decision.

W-52. Delete an item when it is done; never mark it done.

### A pattern (`docs/patterns.md`)

```markdown
<The rule, as an instruction any project could follow.> <Why it holds, in general
terms.>
```

W-55. A pattern belongs in the list only if it could apply to hundreds of other
projects and this project follows it. It is stated in neutral, general terms. A
pattern is a technique: a way of building, with a mechanism a reader can apply to code
or data. A commitment about whom the project serves is a
value, in `docs/values.md`. A choice that makes sense only here is a decision, in
`docs/decisions.md`. A pattern this project does not follow belongs in no list of
its.

W-56. A pattern names nothing in this repository: no decision, no section of the
specification, no behaviour, and not the project's name. A pattern stated in this
project's terms goes stale when those terms change, and cannot be read in another
project.

W-57. A pattern states one rule, as an instruction, and then the reason it holds,
never an example from this project. A pattern has no ID.

### A value (`docs/values.md`)

```markdown
<The commitment, as a statement of what this project holds itself to.>
```

W-58. A value states a commitment about whom this project serves and what it owes
them. It is not a technique, so no mechanism follows from it alone, and it may name
this project. A value has no ID.

W-59. Nothing cites a value or a pattern. No decision, specification
section, open item or line of code refers to one: `CLAUDE.md` imports both files, so
every session reads them whole.

### An explanation (`docs/explanations/`)

An explanation walks a reader through how something works, in whatever form
teaches it best. W-65 is the only rule in this guide that applies to it.

W-65. An explanation cites nothing: no decision, no section of the specification
and no behaviour. It says what the cited entry would have held instead. A citation
has to be kept in step with the entry it names, and an explanation is written once
and read long after.

### A behaviour-registry entry (`cmd/cogmer/behaviors.go`)

W-70. `Title` states the behaviour as an observed fact, in the present tense:
"UserPromptSubmit stdout is injected into the pending turn".

W-71. `Reliance` is for somebody debugging at 2am. It starts with what breaks: "If
this changes, <what breaks>. <How it shows up, or that it fails silently>."

W-72. The error the check returns says what changed and where to look next.

### The two READMEs

W-75. `README.md` is what somebody who has not installed cogmer needs, in this order:
what it is, the two lines that install it, pairing, a room, why it might be worth it,
what it does not do, and then how to work on it. Why it might be worth it is the
two-peer result, stated with its limit: it happened between two sessions one person
was watching. It comes before the commands, because somebody deciding whether to
install it reads the first screen and stops. A README that serves both audiences
serves neither in its first screen, since one reader wants the install lines and the
other wants what is unfinished.

W-76. `plugin/README.md` is what somebody who has installed cogmer needs: the command
reference and where the binary lives. Neither README repeats the other, and neither
says what is built or unfinished, which is in `open.md` (W-25).

### A requirement in the specification (`Shared Claude Sessions.md`)

W-80. State one requirement per paragraph, in the present tense: "The daemon stops
only a process it has identified as a cogmer daemon." Give at most one sentence of
reason. No evidence and no history. Evidence has a method, a date and a version,
and a requirement has none, so evidence written as a requirement reads as timeless
and goes stale without anything in the sentence saying so.

W-81. The specification carries no dates.

W-83. The specification carries no status and no plan. Nothing in it says what is
built, finished, partial, deferred or next, and nothing orders the work: the phases
the work was planned in are in `docs/phases.md`, actions not yet taken are in
`docs/open.md`, and what has been built is the code. A requirement is stated the same
way whether or not it is met yet. Status in the specification goes stale with every
commit, and a reader cannot tell a requirement from a report.

W-84. The specification is upstream of the decisions, and cites none of them. It is
more concerned with describing the system as the user experiences it, constraining
the system's boundaries, and avoiding negative outcomes. Decisions are smaller
grained: they fill in what is too mundane for the specification, or too concerned
with minute technical detail. A decision need not relate to the specification; one
that does extends a requirement's implementation detail or details a necessary
exception to it, and cites the requirement's § section. Nothing in this rule requires
detail to be moved out of the specification. A requirement that pointed to a
decision for its reason would read as unfinished wherever the log says more, and
would point at a tombstone once the decision was withdrawn. Restating decisions in
the specification instead fills it with detail no user experiences, and the two
copies drift apart.

W-85. A change to what the user experiences changes the specification in the same
commit as the decision or the code that makes it.

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

W-91. A commit that records evidence for a decision to cite says, in its message or
in a file it adds, when the run happened and on what machines and versions, what was
done, enough to repeat it, what was seen, and what the run does not show. It records
observations, never instructions: what to do about them goes in `docs/open.md`.

## Enforcement

`TestDocumentsFollowWritingGuide`, in `cmd/cogmer/writing_test.go`, checks every
document a person reads: the Markdown files at the repository root, in `docs/` and
in `plugin/`. It reports each word and mark listed above, with its line. The files
in `plugin/commands/` are instructions to the model and are not checked.

The test parses each document with `github.com/yuin/goldmark` and checks only the
prose, because the rules govern prose and the documents quote code that breaks them:
the decision template's own heading, with its em-dash, sits in a fenced block. Code
spans, code blocks and text inside HTML blocks are not checked. goldmark is pure Go,
so every target still builds with `CGO_ENABLED=0`, and only test files import it, so
the binary does not contain it. goldmark can split one sentence into several text
nodes, so the test joins a document's prose before matching a phrase. Scanning lines
with regular expressions would need its own handling for indented code, code spans
and table cells, and each one missed reports a problem in text no rule governs.

A use of "precisely", "exactly" or "deliberately" passes only with a marker
directly after it (W-22). The test also reports a marker that gives no reason, and a
marker that follows no word it can excuse.

Documents not yet rewritten are listed in `writingNotYetRewritten` and exempt. The
test fails when a listed document passes, so that it comes off the list in the
commit that rewrote it. This guide is never checked, because it quotes every word
it bans. The word and structure checks skip `docs/explanations/`.
`TestExplanationsCiteNothing`, in `structure_test.go`, checks only W-65 there,
reading code spans as well as prose.

`TestCitationsNameThingsThatExist`, in `cmd/cogmer/citations_test.go`, checks
every citation in the documents, the Go sources, the scripts and the plugin. Each
D-NNN must have an entry in `docs/decisions.md`, each § a numbered heading or
numbered step in the specification, each BNN an entry in the behaviour
registry, and each `<commit>:<path>` a file that commit holds. No document is exempt. A decision that was withdrawn keeps its tombstone,
so citations of it still resolve.

`structure_test.go` checks the structure the templates set. Bold appears only as a
template's field names, in decisions and working material, and as the opening of an
item in `open.md`. No header sits over a section of three
lines or fewer; a document's title and the headers a template defines are exempt.
The specification carries no dates and cites no decision. No document other than `open.md` and working
material cites `docs/work/`. The patterns document names no decision, section,
behaviour or the project itself, and no document other than `CLAUDE.md` names the
values or patterns document. These checks share the exemption list with the
word check. No length limit is checked, because a limit is met most cheaply by
compressing, which is the failure W-17 describes. The header rule is not such a
limit: what it asks for is removing the header.

`TestBehaviorRelianceStartsWithWhatBreaks`, in `structure_test.go`, checks that
each behaviour's `Reliance` starts "If this changes". Behaviours written before that
template are listed in `relianceNotRewritten`, which only shrinks.

`TestLaterDecisionsFollowTemplate`, in `structure_test.go`, checks every decision
after D-123, reading the log as Markdown so that a field name inside code does not
count as the field. It checks W-30, W-33, W-34 and W-41 on each; for W-41 it
rejects a mention of `open.md` or `docs/work/`, or a link to a ClickUp, GitHub issue,
Jira or Linear task. Every tombstone,
whatever its number, must hold only its status line. A withdrawn entry must name the
decision that replaced it, and that decision must have a **Rejected.**. A moved
entry must name a rule or a heading this guide holds.

`TestWritingRulesAreIndexed`, in `cmd/cogmer/writingrules_test.go`, checks that
the rule index above and the checks agree. Every rule in the index has a paragraph
starting with its ID. Every rule marked checked is cited by some check's message,
and every ID a check cites is in the index and marked checked.

The rules no test can decide are reviewed by the `writing-review` skill in
`.claude/skills/writing-review/`, which a maintainer runs. Run it on a document
before taking it off the exemption list, and on a change to a document before
committing it. It runs the checks above on the named documents whether or not they
are exempt, reviews the rest, and reports findings without changing anything. It
reads this guide in full each time and restates none of its rules. Every finding
quotes the text it is about, and `TestReviewFindingsQuoteTheirDocuments` in
`review_test.go` drops any finding whose quote is not in the file, because a model's
review can name text that does not exist. The review is not part of `go test`,
because it would need the network and a model, cost money on every run, and give
different results from run to run. Its judgement is not reproducible, and nothing
runs it unless somebody asks.
