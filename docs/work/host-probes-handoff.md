# Host probes: where the work stands

Written 2026-11-18 so that a session on the droplet can pick the work up. Material for the open items
under "Claude Cowork as a host" and "ChatGPT Work as a host" in `docs/open.md`. Delete it with the
last of them.

## Branch and state

Work is on branch `host-probes`, which is `user-identity` plus the host-probe commits. `main` does not
carry the user identity. `docs/work/host-probes.md` is the runbook, `docs/work/host-observer/` holds the
observer plugin, `build.sh`, `probe.py` and `colleague.py`, and
`docs/work/host-probe-1-20261118T095854Z.txt` records probe 1 (`e7ddab7`).

The droplet can read the web and the repository, build with `build.sh`, and edit documents. It cannot
run probe 1, 2 or 3, because each needs Claude Desktop or the ChatGPT app on the Mac.

## What probe 1 found

In a local Cowork session on the Mac with "Only on this computer" on, no bridge:

- The plugin's hooks run on the Mac, and a hook's plain output reaches the model.
- Capture and injection work when the session is in a room. The room was joined from a terminal.
- A command's `!` line is refused ("[shell command execution disabled by policy]"), and the model's
  shell tool runs in a Linux VM that cannot reach the Mac's daemon. So no `/cogmer:` command runs.
- Anthropic documents the refusal: https://code.claude.com/docs/en/skills.md says a Cowork session on
  the desktop replaces every `!` command line with that placeholder. Managed skills are exempt.

The open item "A Cowork session that runs on the user's computer captures and injects…" in
`docs/open.md` states this with its options for commands.

## What the reading suggests, and has not been shown

Hooks look like the channel every host honors, and the command channel varies. A plugin's bundled local
MCP server might be the one command route Cowork allows without the bridge: it loaded on the Mac on
2026-10-31 (`78477ee`), once. Nothing here says what Codex or ChatGPT Work do for commands. Do not design
a uniform command channel before the host matrix below is filled, since the project's patterns say not to
abstract for a consumer nobody has examined.

## Next, in order

1. Read the Codex and ChatGPT Work documentation for what was asked of Cowork: does a plugin command
   run, where does its shell run, does a plugin's MCP server load, what policy blocks which part. Start
   from https://learn.chatgpt.com/docs/hooks and the Codex plugin pages. Record each claim with its URL
   and the date read.
2. Write a host matrix: capture, injection, command route, shell location and trust, one row for each of
   Claude Code, local Cowork, cloud Cowork, Codex, ChatGPT Work. Keep it in this folder until a decision
   uses it.
3. Probe 3 (ChatGPT Work) and probe 2 (merged Chat and Cowork) on the Mac, with `probe.py`. Probe 3 also
   settles what the docs reading of step 1 could not.
4. Try the bundled-MCP-server route for commands in a local Cowork session, and add a behavior check
   to `cmd/cogmer/behaviors.go` if it is adopted.

## What is left on the Mac

The real state is back at `~/.cogmer` ("wonka" on `macbookpro.lan`, four rooms). Things the person still
decides about:

- In Claude Desktop: remove the observer and the probe's cogmer plugin, reinstall the usual cogmer plugin,
  and set "Only on this computer" back to what it was.
- `~/.cogmer.probe-20261118` holds the probe's state, which includes the person's own prompts and replies
  from the session. Delete it when the record is no longer needed.
- `~/.cogmer.stray-0749` is an empty state that a hook created while the real one was moved aside.
- `~/cogmer-mock`, `~/host-probe-build`, `~/.cogmer-host-observer`, and Desktop's
  `claude_desktop_config.json.before-probe` are leftovers of the run.
- A daemon started on 2026-10-26 (`/tmp/cogmer-test daemon`) still listens on 127.0.0.1:4792. It belongs to
  nobody's current work, and the mock now avoids that port.
- `docs/work/cloud-e2e-probe/` is untracked and unused.

## Things learned about running the probes

- `probe.py before` prints what it moves aside. Check it says the real state, with its rooms, before
  going on: a hand-made move earlier left it moving an empty state.
- A Claude Code session may run beside `probe.py`. It refuses only for Claude Desktop, the ChatGPT app and
  Codex.
- The mock colleague (`colleague.py`) needs session identifiers that differ from the session under test.
  A daemon gives a prompt nothing from a session with its own identifier.
- A session that cannot run commands can still be put in a room from a terminal: its full session identifier
  is the file name of its transcript under Desktop's `local-agent-mode-sessions` folder, and
  `CLAUDE_CODE_SESSION_ID=<it> cogmer join <room>` binds it.
