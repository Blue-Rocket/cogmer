# Host probes

Material for three open items: whether the ordinary cogmer plugin captures and injects in a
Cowork session that runs on the user's computer, whether a merged Chat and Cowork conversation
runs the plugin's hooks, and whether cogmer's Codex plugin captures and injects in a ChatGPT
Work thread. Delete it with the last of them.

Each run needs a Mac with the app it probes, and a person at it, since an assistant cannot see
a desktop app. Each run installs cogmer built from the `host-probes` branch, because `main`
does not carry the user identity.

## What a run installs

`docs/work/host-observer/` is a plugin for Claude Code and for Codex with hooks for
`SessionStart`, `UserPromptSubmit`, `PreToolUse` and `Stop`. Each hook appends to
`~/.cogmer-host-observer/log.txt`, on the machine where the hook runs, the event, the operating
system, the user, `HOME`, the names of the fields the hook received with each value's length,
the names of the environment variables that mention a host, and whether a cogmer daemon answers
at 127.0.0.1:4782. It records no prompt or reply: a prompt appears as its length, and as its
first word when that word is a command. The `SessionStart` and `UserPromptSubmit` hooks also
print a line starting `[host-observer]`, which shows whether a hook's plain output reaches the
model. The command `/host-observer:where` in Claude, and the skill `$host-observer` in Codex,
print where the command's own line and the model's shell tool run, and the log as each sees it.

`docs/work/host-observer/build.sh <empty folder>` builds, from the commit checked out,
`host-observer.zip` and `cogmer-plugin.zip` for Claude Desktop's plugin upload, and the cogmer
program for macOS on Apple silicon, with the version in `plugin/VERSION` so that the plugin's
installer keeps it rather than downloading the release. It prints the commit and the start of
each file's sha256, which the run's record names. It needs Go and runs on the droplet or on the
Mac. Built on the droplet, the files are fetched with `scp`.

## Before and after every run

A program built from this branch migrates the state directory it opens to data version 1, and
a program from `main` then refuses to touch it (D-244, a program that finds newer data touches
nothing). The Mac's `~/.cogmer` holds real pairings and rooms, so a run moves it aside and
works in a fresh one.

Before:

1. Quit Claude Desktop and the ChatGPT app, and end every Claude Code and Codex session.
2. Stop the daemon with `~/.cogmer/bin/cogmer stop`.
3. Move the state aside with `mv ~/.cogmer ~/.cogmer.before-probe`.
4. Make a fresh state directory with the built program in it:
   `mkdir -p ~/.cogmer/bin && cp <built>/cogmer ~/.cogmer/bin/cogmer && chmod 755 ~/.cogmer/bin/cogmer`.
   `~/.cogmer/bin/cogmer version` prints the version in `plugin/VERSION`.
5. Empty the observer's log with `rm -rf ~/.cogmer-host-observer`.

After:

1. Run `/host-observer:where` or `$host-observer` once more, and save its output and
   `~/.cogmer-host-observer/log.txt`.
2. Quit the app, then stop the probe's daemon with `~/.cogmer/bin/cogmer stop`.
3. Keep the probe's state for the record with `mv ~/.cogmer ~/.cogmer.probe-<date>`, and
   put the real state back with `mv ~/.cogmer.before-probe ~/.cogmer`.
4. Remove the observer plugin and the probe's cogmer plugin from the app, and reinstall the
   cogmer plugin the Mac used before.

A run's record is a file `docs/work/host-probe-<probe>-<UTC time>.txt` holding the steps as
clicked, what each showed with any text copied word for word, the observer's log, and what the run does
not show, committed with a message that says when, on what machine and versions, and what was
seen (W-91 in `docs/writing.md`, a commit that records evidence says how it was run).

## A colleague on the same Mac

The two-way part of each run needs a colleague. A second cogmer on the same Mac, with its own
state directory and addresses, is one. The second cogmer has its own user identity and machine
key, and reaches the cogmer under test through the relay, as a colleague on another machine
would. The colleague's session runs in the Claude Code command-line program, started with the
second cogmer's state directory and addresses in its environment, which its hooks inherit:

```sh
mkdir -p ~/cogmer-colleague/bin && cp <built>/cogmer ~/cogmer-colleague/bin/cogmer
mkdir -p ~/cogmer-colleague-work && cd ~/cogmer-colleague-work
git -C <checkout> archive --format=tar HEAD:plugin | (mkdir -p plugin && tar -x -C plugin)
COGMER_HOME=~/cogmer-colleague COGMER_ADDR=127.0.0.1:4792 COGMER_PEER_ADDR=127.0.0.1:4793 \
  claude --plugin-dir ./plugin --setting-sources project
```

`--setting-sources project` keeps the user's own settings out, and with them the cogmer plugin
the Mac already has, so that only this copy's hooks run. In that session `/cogmer:self-status`
opens the colleague's identity page at 127.0.0.1:4792. Pairing the two, verifying the words,
and running `/cogmer:room-invite` from the session under test make the colleague a guest of the
room. Then `/cogmer:room-join <room>` in the colleague's session puts that session in the room.

`cogmer seed` cannot stand in for a colleague. The turns it writes come from a peer nobody
verified, who is not on the room's guest list, and the daemon gives a prompt no turn of either
kind. On 2026-11-17 on the droplet, a room seeded with two turns gave the next prompt none.

## Probe 1: Cowork on the user's computer with no bridge

The question is whether the ordinary cogmer plugin captures and injects in a Cowork session
that runs on the Mac, with no `cogmer` entry in the Desktop configuration. `78477ee` saw a
local session run its command hooks on the Mac and its shell in a Linux VM, so the hooks would
reach the Mac's daemon and a command's `!` line may not. Which side the `!` line runs on is
the first thing the run shows.

1. Do "Before". Look for a `cogmer` entry in
   `~/Library/Application Support/Claude/claude_desktop_config.json`. If there is one, copy the
   file to `claude_desktop_config.json.before-probe` and remove the entry. Remove the
   `cogmer-host-probe` entry if it is still there.
2. Open Claude Desktop. In the plugin settings, disable or remove any cogmer plugin, then upload
   `cogmer-plugin.zip` and `host-observer.zip`.
3. Start a Cowork session that runs on this computer. Record the setting that says so, and what
   the screen shows before the first prompt.
4. Send `hello`. Record whether a line saying cogmer is installed appears, and the reply.
5. Run `/host-observer:where`. Record both outputs. The command's line and the shell tool
   each report their operating system, and the log shows where each hook ran.
6. Run `/cogmer:self-status`. Record what it says and whether a browser page opens on the Mac.
   If the identity page opens, make the identity. It is a throwaway identity, so write its
   recovery phrase on paper and keep the paper with the probe's state.
7. Run `/cogmer:room-create`, then send two prompts, then run `/cogmer:room-log`. Record whether
   the two turns are in the log, and open the watch address `/cogmer:room-status` gives in a
   browser to see the room view.
8. If a colleague is set up, pair, invite and join it, send a prompt from the colleague's
   session, then send one in the Cowork session and ask what the colleague said. Then send one
   in the Cowork session and ask the colleague's session what was said.
9. Do "After", and restore the Desktop configuration from its copy.

## Probe 2: a merged Chat and Cowork conversation

The question is which of the plugin's hooks fire in a conversation that starts as Chat and
in which Claude later uses Cowork. Only the observer is needed, since the question is whether
hooks fire at all.

1. As the Team plan's admin, turn on the setting "Chat and Cowork unified", and record when.
2. Upload `host-observer.zip` in Claude Desktop, if probe 1 has not.
3. Start a new conversation. Send two prompts that need no tool, such as asking for a word that
   rhymes with orange.
4. Send a prompt that needs Cowork, such as asking for a file listing three fruits in a folder
   the conversation can write to. Record what the screen shows as Cowork joins.
5. Send two more prompts that need no tool.
6. Run `/host-observer:where`, and if it is not offered, ask Claude to run
   `bash <plugin folder>/hooks/where.sh tool`. Its output holds the log of the machine the
   shell tool runs on. Also save `~/.cogmer-host-observer/log.txt` from the Mac, which holds
   the lines of every hook that ran on the Mac.
7. Record which `[host-observer]` lines Claude says it can see for each prompt, by asking
   "which lines in your context start with [host-observer]?".
8. Turn the setting off again if it was off before, and remove the observer.

## Probe 3: a ChatGPT Work thread

The question is whether cogmer's Codex plugin captures and injects in a local Work thread, as
Codex's hooks do. OpenAI's page on hooks (https://learn.chatgpt.com/docs/hooks, read on
2026-11-16) says local Work threads run a plugin's hooks once they are trusted.

1. Do "Before".
2. Add two Codex marketplaces: `codex plugin marketplace add <checkout>`, whose
   `.agents/plugins/marketplace.json` names the cogmer plugin, and
   `codex plugin marketplace add <checkout>/docs/work/host-observer`, whose marketplace file
   names the observer. If either is refused, record what Codex said. If the ChatGPT app's
   plugin browser does not show them afterwards, record what it shows.
3. In the ChatGPT app, install cogmer and host-observer, and trust their hooks. Record the
   screens the trust step shows.
4. Start a local Work thread. Send `hello`, and ask "which lines in your context start with
   [host-observer]?".
5. Run `$host-observer`. Record whether `CODEX_THREAD_ID` is set in the shell tool, and whether
   its first eight characters match the `session_id` the hooks logged (B28, `CODEX_THREAD_ID` in
   a shell tool equals the hook session identifier).
6. Run `$cogmer self-status`, make the identity if the page opens, then `$cogmer room-create`,
   two prompts, and `$cogmer room-log`. Record whether the log shows the turns, and whether the
   hook logged the typed `$cogmer` as a prompt's first word.
7. With a colleague, do step 8 of probe 1 from the Work thread.
8. Do "After", and remove both plugins and the marketplace entries the run added.
