---
description: Stop the cogmer daemon on this machine
disable-model-invocation: true
---

## What happened

The output of this command is in your context, under the heading "Output of /cogmer:self-stop". Nothing else produced it.

If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.

## Your task

Report what the output says, in a line or two.

If it stopped a daemon, say that nothing is captured or shared, in any room and any
session on this machine, until a daemon runs again. A new agent session with cogmer enabled starts
one. This command does not, because somebody who stops the daemon may want it stopped.

If it left a process alone, name the process and say it is not cogmer's, so the user
knows `stop` did not touch it.

If nothing was listening, say so.
