---
description: Stop recognizing a colleague on all your machines; their place in rooms is unchanged
argument-hint: <peer>
disable-model-invocation: true
---

## What happened

The output of this command is in your context, under the heading "Output of /cogmer:peer-forget". Nothing else produced it.

If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.

## Your task

Report what the output says. Forgetting discards the colleague's identity on every one
of the user's machines, so the user's machines stop recognizing them and their turns
stop reaching the user, since a colleague the user does not know is unverified. It
leaves every room's guest list as it is, and the other guests keep working with them.
Pairing with them again makes them known and verified again, in every room they were
never revoked from.

To remove somebody from one room while still knowing them, that is
`/cogmer:room-revoke`.
