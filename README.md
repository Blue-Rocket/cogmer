# claude-team

Local-first peer-to-peer shared Claude Code conversations. Several developers,
each on their own Claude Code subscription and session, collaborate in one
replicated conversation with no central server.

Full specification: [`Shared Claude Sessions.md`](Shared%20Claude%20Sessions.md)

## Status: Phase 0 complete

The integration spike is done and **both required directions work**. See
[`docs/phase0-findings.md`](docs/phase0-findings.md) for the full write-up,
including two non-obvious defects in the Claude Code integration surface and how
they are worked around.

Phase 0a (compaction probe) is also done — see
[`docs/phase0a-findings.md`](docs/phase0a-findings.md). The feared silent failure
**did not reproduce**: injected teammate context survived compaction with
attribution intact, even when incidental to the conversation. No remediation is
needed and the delivery watermark is unchanged. Compaction also never interrupts
a turn, and `/compact` never reaches the room.

Not yet built: peer networking (Phases 1–7).

## Try it

```sh
go build -o bin/claude-team ./cmd/claude-team

CLAUDE_TEAM_ROOM=demo ./bin/claude-team daemon &   # localhost:4782
CLAUDE_TEAM_ROOM=demo ./bin/claude-team seed       # simulated teammate turns
```

Register the hooks (`--settings` keeps this out of your real config):

```json
{"hooks":{
 "UserPromptSubmit":[{"hooks":[{"type":"command","command":"/abs/path/bin/claude-team hook prompt"}]}],
 "Stop":[{"hooks":[{"type":"command","command":"/abs/path/bin/claude-team hook stop"}]}]
}}
```

Then run a session whose prompt refers to something only the teammate said:

```sh
CLAUDE_TEAM_ROOM=demo claude -p "I don't think her explanation is right. Check the retry path instead." \
  --settings /abs/path/settings.json

CLAUDE_TEAM_ROOM=demo ./bin/claude-team log
```

## Commands

| Command | Purpose |
|---|---|
| `daemon` | Local daemon on `127.0.0.1:4782` |
| `hook prompt` | `UserPromptSubmit` — captures the prompt, injects unseen teammate turns |
| `hook stop` | `Stop` — reassembles and stores the completed response |
| `seed` | Insert a simulated teammate conversation |
| `log` | Print the room transcript |
| `whoami` | Show peer identity and active room |

`CLAUDE_TEAM_ROOM` overrides the room; `CLAUDE_TEAM_ADDR` the daemon address.

## Cross-platform

Go with pure-Go SQLite, so every target builds from one machine with no cgo and
no runtime dependency — which matters because Claude Code now ships as a native
binary and teammates may have no Node installed.

```sh
GOOS=windows GOARCH=amd64 go build -o bin/claude-team.exe ./cmd/claude-team
```

Windows binaries build but are **unverified at runtime** — hook shell semantics
differ and must be re-probed before Phase 2.
