---
description: Create a room and put this session in it
argument-hint: [--show-only] [--another]
disable-model-invocation: true
---

## What happened

The output of this command is in your context, under the heading "Output of /cogmer:room-create". Nothing else produced it.

If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.

## Your task

Tell the user the room's name, and that this session is in it. A session they start later
joins only when they run `/cogmer:room-join` with the room's name inside that session.

If the output says the room is show-only, say what that means in a few words: its
conversation appears in the room view and none of it is given to their agent, that
cannot be changed for this room, and their colleagues' sessions still receive the
user's turns unless the colleagues joined show-only too. Do not offer to change it.

If the output says this session is already in a room, or has left one, and that nothing was
created, tell the user which room it is and stop. Run the command with `--another` only
if the user says they want a second room, and never add it yourself. A second room cannot
take this session, so they enter it from another session.

If the output says they are not in a room or that something failed, say what it said
and stop. Do not run the command again: creating a room twice makes two rooms, and
the second is indistinguishable from the first to everyone but them.
