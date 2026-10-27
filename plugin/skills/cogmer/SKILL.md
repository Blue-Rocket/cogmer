---
name: cogmer
description: Run cogmer room and peer commands in this Codex session to share a conversation with colleagues using Codex or Claude Code.
---

Run the command the user requested using `bash <plugin-root>/codex-cli.sh`, where
the plugin root is two directories above this skill directory. Resolve that path
from this skill's installed location. Commands identify this session from
`CODEX_THREAD_ID`; never supply or guess a session identifier.

Use this mapping for the command name. Pass only arguments the user supplied,
as separate quoted shell arguments. If no command was supplied, show the available
commands and ask which the user wants.

| Command | Binary arguments |
|---|---|
| `self-status` | `whoami` |
| `self-machines` | `machines` |
| `self-name` | `name [name]` |
| `self-stop` | `stop` |
| `self-restart` | `restart` |
| `self-cowork` | `cowork-setup` |
| `peer-list` | `peers` |
| `peer-pair` | `pair <pairing-string> [label]` |
| `peer-rename` | `rename <peer> <new name>` |
| `peer-forget` | `forget <peer>` |
| `room-create` | `create [--show-only] [--another]` |
| `room-join` | `join <room> [--show-only]` |
| `room-leave` | `leave` |
| `room-list` | `rooms` |
| `room-status` | `guests` |
| `room-log` | `log` |
| `room-invite` | `invite <peer>` |
| `room-revoke` | `revoke <peer>` |
| `room-conflicts` | `conflicts [room]` |

Relay the command's result. If it says installation is in progress or a command
failed, report that result and stop. Do not repeat a command that creates a room
or changes membership. Pairing opens a browser where the users compare the words
on their call; never read, compare or confirm those words through the model.

`create` refuses a session that already has a room and says which. Ask the user before adding
`--another`, which makes a second room anyway, and never add it yourself.

Room creation and joining apply to this session. A session can never join a second
room. `--show-only` is fixed for a room on this machine: the room appears in its
browser view and none of its conversation reaches this machine's agent sessions.
Colleagues receive the user's turns unless their own room is show-only too.
