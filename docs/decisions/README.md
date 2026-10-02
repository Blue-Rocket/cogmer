# Decisions

Each file here records one decision as it stands now: what we do, the facts that
support it with their sources, the alternatives rejected and why, and the condition that
would reopen it. The rejected alternatives stop a later reader from proposing something
already ruled out. `docs/writing.md` gives the template.

A file is named for the decision's number and a slug of its title, such as
`D-048-verification-is-a-live-commit-then-reveal-exchange.md`. A new decision takes the
next number, and a number is never reused.

An entry's revisit condition names the automated check that would show it wrong, where
one exists: `cogmer doctor`, or an entry in the behaviour registry in
`cmd/cogmer/behaviors.go`. When that condition occurs, the decision is due for review.
It is not automatically wrong.
