---
description: Show the room this session is in, and who is in it
allowed-tools: Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms:*), Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" guests:*), Bash(bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" where:*)
disable-model-invocation: true
---

## Rooms

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms`

## Guests of this session's room

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" guests`

## Where to watch it

!`bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" where`

## If the output above is missing

If a section above shows a note in place of cogmer's output, such as one saying that a shell command was not run or was disabled, or shows the command line itself, the host did not run it. Run each of these yourself with the Bash tool, once and in this order, and take what each prints as its section's output:

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" rooms
```

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" guests
```

```
bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" where
```

If the sections above show cogmer's output, run nothing.

## Your task

Summarize in a few lines: which room this session is in, and who may enter it.
Mark anyone unverified as unverified — their turns are not being exchanged.

If the output says THIS MACHINE CANNOT BE REACHED, say that before anything else, and
how long: colleagues pull each other's turns, so a machine nobody can reach hands its
turns to nobody, and the user may believe they are shared when they are not. Offer
`/cogmer:self-restart`.

If the output says a colleague is NOT REACHABLE, say that first, before anything else
about the room. A room whose colleague cannot be reached looks quiet and is not, and
nobody would otherwise know. Say for how long, and that none of their turns reach this
machine until it can reach them. If the output says they reached this machine
recently, say that their daemon is running and the fault is in reaching it. If it says
they have not, say that this machine has also heard nothing from them. Suggest asking the colleague to run
`/cogmer:self-status` and say whether it tells them colleagues can reach them. Do not
suggest the fault is in this session: the colleague's daemon is the likely place.

Give them the watch address from above verbatim. It is a link they open in a
browser and leave open beside this session, not something to summarize — the room
is read there, and this session is read here.
