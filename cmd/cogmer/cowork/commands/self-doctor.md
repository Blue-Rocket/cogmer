---
description: Check that cogmer and the host's behaviors it relies on are working; `extended` runs the whole check in the background
argument-hint: [extended]
disable-model-invocation: true
---

## What happened

The output of this command is in your context, under the heading "Output of /cogmer:self-doctor". Nothing else produced it.

If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.

## Your task

Report what the output says, briefly. Lead with whether anything is wrong: a failed behavior,
a daemon that is not running, or versions that disagree. Name each failed behavior as the output
does, and relay what the output says resolves a disagreement.

If the output says the whole check is running, say that it takes about 20 seconds and that the
user can run `/cogmer:self-doctor` again to read what it found. Do not run the check yourself,
and do not wait for it.

If the output says the whole check drives real turns on the user's subscription, say that once,
and offer `/cogmer:self-doctor extended` only if something above looks wrong or the user asked
for the whole check. Do not offer it when every check passed.

If the output has a line that begins "Next step", give it as the last sentence of your answer,
in the words it uses. It is the one thing the user does next on the way to a first shared room,
and the command's output reaches them only through you.
