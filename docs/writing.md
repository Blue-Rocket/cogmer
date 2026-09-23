# Writing

How every document in this repository is written. Each kind of item has a template
below. The rules before the templates apply to all of them.

## The reader

The reader was not here when the work was done. They may be reading months later,
with no memory of the session, looking for one answer. They may not write code. Say
"person" or "colleague" for anyone using cogmer, and "maintainer" for someone
changing it.

## Rules

Put the answer first. The first sentence of any item says what is true, what was
decided or what is wrong. Background, evidence and history come after it, and only
if the reader needs them to trust or apply the answer.

Write plain, complete sentences in paragraphs. Use one idea per sentence and the
active voice. When a
sentence needs a second reading, split it.

State the fact rather than a saying about it. "A check that cannot fail reads as
protection" makes the reader work out the rule. "Every behaviour check has a test
that makes it fail" states the rule.

Leave out the process. Do not describe how you got to the answer ("Asked whether…",
"Reading it closely turned up…", "The honest result is…"). If the route matters,
because something was tried and failed, record what was tried and what happened.

Describe what the system is, not what you changed. "The parser reads the config at
startup", not "Updated the parser to read the config at startup". Change history
belongs in commit messages.

Give numbers with units, and dates for observations. "16s on 09-22", not "a few
seconds". Say whether something was observed or only read from the code.

Name a mechanism by what it does, not by our word for it. If a term of our own has
to appear ("watermark", "fence", "the view"), say what it is the first time it
appears in that item.

Give every reference a few words. "D-054 (verification gates sync)", not "D-054".
Cite only what you have checked exists.

The mark § always names a section of the specification. Refer to a section of the
same document in words: "section 4", not "§4".

Use bullets only for real lists: options, steps in order, items to compare. Do not
break an explanation into bullets.

Use backticks only for things that appear in code or at a terminal: identifiers,
file names, commands, environment variables. Not for emphasis.

Use bold only for an item's opening sentence in `open.md`, and for the field names
the templates define. Nowhere else. In particular, do not open a paragraph with a
bold label and a fragment ("**Where the code lives.** Signalling needs…"). If a
thing needs a name and an explanation, write a sentence.

Use headers only on sections long enough to need one. A three-line section does not.

Stop when the item has said what it needs to. Do not restate it in a closing sentence.

## Words and marks to avoid

These are the habits this repository overuses, counted on 09-22 across the five main
documents; the counts include every use. The enforcement test checks for them. The
one em-dash allowed is the one in a decision's heading, `## D-NNN — <title>`.

The em-dash is banned for what it usually does here: it lets a finished sentence
carry a second idea, often as a twist or a reveal ("handshakes — seconds of work"),
or holds a sentence open around an aside. Replacing it with a comma or a semicolon
keeps that structure and defeats the rule. Split the sentence instead.

| avoid | count | use instead |
|---|---|---|
| em-dash (—) | 1,020 | usually two sentences; a colon only when what follows explains or lists what came before |
| "load-bearing" | 18 | say what depends on it |
| "honest", "honestly" | 18 | drop it; state the fact |
| "precisely", "exactly" | 114 | drop it when it only adds emphasis; a use that carries meaning takes a marker |
| "the point", "is the point" | 19 | say what the thing is for |
| "not merely" | 5 | "also", or two sentences |
| "deliberately" | 39 | keep only where it contrasts with an accident, with a marker |

A use of "precisely", "exactly" or "deliberately" that the table permits carries a
marker directly after the word, giving the reason it stays. The marker is an HTML
comment, so it does not show when the document is rendered:

```markdown
The relay was chosen deliberately<!-- writing: contrasts with an accident -->.
```

No other word takes a marker.

Also avoid: leverage, utilize, robust, seamless, comprehensive, ensure, nuanced,
testament, tapestry, delve, crucial. Say what actually happens.

No status decoration: no check marks, emoji, "Note:" or "Important:".

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

**Limits.** <What this does not cover, what it assumes that nothing checks, and
what is left open and where it is tracked.>

**Revisit when** <a condition somebody could observe>.
```

The title is the decision, not the topic: "`stop` finds a daemon by the addresses
it holds", not "Stopping the daemon".

An entry describes the decision as it stands now, so that a reader can understand
what we do with as little to hold in mind as possible. It says nothing about what
was there before: not what the system used to do, not what this replaced, not who
asked or what happened on the way. Words that describe the system as it stood when
the decision was made ("stays", "is kept", "continues to", "already") are history
whatever their tense. A decision states only what it decides. Something decided
elsewhere appears only when the reader needs it to apply this one, and then as a
citation with a few words, not restated.

Every item under **Support.** has a source, and the source is where the fact is
kept. If the only record of a fact would be the decision itself, write the findings
document first.

**Rejected.** is required whenever a real alternative was weighed, which is the
reason for writing an entry at all. Omit **Limits.** when there are none.

When a decision reverses an earlier one, the earlier entry is replaced by a
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
When a later decision changes only part of an earlier one, rewrite the earlier
entry so that it states only what still holds, and record the removed part's
reason as a finding in the same way.

When rewriting an entry, keep every fact, reference and number from the original,
either in the entry or in the findings document it now cites. Remove one only by
saying in the commit message what was removed and why.

### An item in `docs/open.md`

```markdown
**<The problem or task, in one sentence a reader could act on.>** <What happens
now, observed or read from the code, with the date. Who it affects and how.>

<What would fix it. If there are several options, list them, each with its cost.
If something is unknown, say what would answer it.>
```

The opening sentence names the problem, not the fix: "`/cogmer:self-status` on
first use fails instead of answering", not "Make cli.sh exit 0". Keep an item under
about 25 lines. Delete it when it is done; never mark it done.

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

Findings record what was seen. The requirement a finding justifies goes in the
specification, and a finding about someone else's software goes in the behaviour
registry.

### A behaviour-registry entry (`cmd/cogmer/behaviors.go`)

`Title` states the behaviour as an observed fact, in the present tense:
"UserPromptSubmit stdout is injected into the pending turn".

`Reliance` is for somebody debugging at 2am. It starts with what breaks: "If this
changes, <what breaks>. <How it shows up, or that it fails silently>."

The error the check returns says what changed and where to look next.

### A requirement in the specification (`Shared Claude Sessions.md`)

State one requirement per paragraph, in the present tense: "The daemon stops only a
process it has identified as a cogmer daemon." Give at most one sentence of reason,
and point to the decision for the rest. No findings, no history, no dates.

### A commit message

```text
<What changed, in the imperative, under about 60 characters>

<What was wrong or missing, and how it showed. What changed. How it was
checked, if it was.>

Co-Authored-By: …
```

Write for somebody reading `git log` who wants to know whether this commit matters
to them. Wrap at 72 columns. The body is usually 3 to 10 lines.

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

`TestLaterDecisionsFollowTemplate` checks that every decision after D-123 has
**Date:**, **Decision.**, **Rejected.** and **Revisit when**. A tombstone is exempt.
When no alternative was weighed, **Rejected.** says why there was none.
