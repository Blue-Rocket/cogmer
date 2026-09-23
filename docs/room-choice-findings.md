# Which room a command acts on

**Run:** 2026-09-20, read from the code of v0.3.2 (commit 9c76527, 00:49 -0700) and
v0.4.0 (commit 05f89c4, 00:59 -0700). The failure was found by reading the code,
not by running it. The variable was then named `CLAUDE_TEAM_ROOM`; it is
`COGMER_ROOM` now.

**Result:** An environment variable ranked above the session's own room chose one
room for every session on the machine, so D-076 was withdrawn ten minutes after it
was committed.

## What was run

| commit | version | what `currentRoom` consulted, in order |
|---|---|---|
| 9c76527 | v0.3.2 | `CLAUDE_TEAM_ROOM`, the invoking session's room, the machine-level pointer |
| 05f89c4 | v0.4.0 | the invoking session's room, the machine-level pointer |

## What we found

### An ambient variable ranked above the session answers for every session

D-076 ranked `CLAUDE_TEAM_ROOM` first, on the reasoning that somebody who names a
room means it. The variable is read from the environment, so a single export in a
shell profile, or in the process that starts sessions, gives it the same value in
every session a machine starts. A command run inside a session in one room then
acted on the room the variable named. `/room-invite` would admit a person to that
room without an error. This is the failure D-064 (a session's room is the one
somebody chose inside it) removed, returning at a higher precedence than the
session.

05f89c4 removed the variable from `currentRoom`, with the dead `LoadConfig` that
also read it. D-077 (every room has its own URL, and no ambient value picks one)
states the rule that replaced it: a room is named in the invocation, or it is the
session's.

### The rest of D-076 survived

A command inside a session acts on that session's room, and a session in no room
has no room, however many the machine holds. `TestARoomResolvesOnlyThroughTheSession`
in `membership_test.go` tests both. The machine-level pointer that D-076 kept as a
fallback no longer exists: D-080 (there is no current room) removed it.
`TestNoAmbientRoomOverride` fails if anything reads the variable again.

### The same pattern appeared three times in one day

The view, the `where` command and `currentRoom` each answered "which room" from
machine-wide state where a session could have been asked. Each looked like a
convenience and gave a silent wrong answer. 05f89c4 records all three.

## What this does not show

Nobody is known to have run v0.3.2 with the variable set. The failure was never
observed, only read from the code.
