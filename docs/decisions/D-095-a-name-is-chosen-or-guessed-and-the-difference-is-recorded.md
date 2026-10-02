# D-095 — A name is chosen or guessed, and the difference is recorded

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** D-094 stopped short of defaulting a peer's label to the name they had
picked, because a peer does not pick one: `UserDisplayName` is `$USER` with the first
letter capitalised, fixed at identity creation, with no way to change it. Asked what
the natural point is for somebody to choose their own.

**The name has one function: to be seen by other people.** You never see yourself
labelled — the view leaves the derived name off your own turns, because it exists to
identify other peers (D-021, peer names are derived from the identity), and your
turns are marked as yours. So there is no moment when its owner notices it is wrong, which is exactly
why `Ec2-user` could travel for weeks.

**So the moment is the first time it travels**, and that is one place:
`printPairingInvitation`. All three paths that hand your identity to somebody else go
through it (D-089 consolidated them), so the offer belongs with the string rather
than with a command. There is no earlier candidate — identity is created by whatever
touches it first, which is a daemon started detached by a hook, and D-150 forbids
delaying a session or speaking to the person. Nobody is watching when the name is
invented.

**Chosen is recorded, not inferred.** `NameChosen` is false at creation and flipped
by a deliberate set — including setting it to the guessed value, because running the
command is the choice. Comparing the name against the guess instead would pester the
person whose username really is their name, for ever, which is the always-on warning
of D-089 in another costume.

**`whoami` does both, not one or the other.** It shows the name plainly, lifted out
of the JSON blob where it was buried next to the key, because it is the literal
answer to the question asked. The provenance line appears only while the name is
guessed; saying "you chose this" otherwise is noise about something already known.

**Prefixes name the scope, and `peer-` is for other people.** `/self-name`, not
`/peer-name`. The rule is now: `peer-` for a command about somebody else, `self-`
for one about you, `room-` for one about a room. A prefix names the **activity**, so
`/peer-pair` with no arguments printing your own string is not a violation — sending
your half is part of pairing with a peer — but that reading should be stated, because
it is the first question anybody will ask of the rule.

**Renaming is safe, and by construction rather than luck.** Turns already sent keep
the name they carried, because events are immutable (§7). And the label a colleague
gave you is theirs (D-094), so your rename cannot change what anybody else calls you.
That is what makes this cosmetic enough to defer and to change.

**Now unblocked:** the pairing string can carry a chosen name, and a label can
default to it (D-094's revisit condition).

**Revisit when:** something other than a person needs a display name, or the name
starts travelling somewhere other than a pairing string and an event.
