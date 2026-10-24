---
description: Add, rename or remove your machines, or recover, in your browser
disable-model-invocation: true
---

## What happened

The output of this command is in your context, under the heading "Output of /cogmer:self-machines". Nothing else produced it.

If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.

## Your task

The output is one of three things and says which.

**A page was opened.** Adding, renaming and removing machines, and recovering, happen
in that page in their browser. Say so in one line and stop; the instructions are on the
page and repeating them competes with it. If they want to add a machine, the one thing
worth adding is that the new machine's page shows a string to enter in this page, and
both pages then show two words to compare.

**The identity is not made yet.** The page that opened asks whether they have used
cogmer on another machine. Say so in one line, and that they come back and run the
command again once the page is done.

**The daemon is not running.** Relay what the output says, and offer
`/cogmer:self-restart` when it names it.

Never ask for, repeat or suggest typing the recovery phrase here. It is typed only in
the page, because anything typed into this session reaches the model and the service
that runs it, and whoever holds the phrase is the user.

Never summarize, invent, or repeat two words yourself. The comparison is between the
two pages; anything that looks like it came from you undermines the only check that
makes it mean anything.
