# Host probes

Material for three open items: whether the ordinary cogmer plugin captures and injects in a
Cowork session that runs on the user's computer, whether a merged Chat and Cowork conversation
runs the plugin's hooks, and whether cogmer's Codex plugin captures and injects in a ChatGPT
Work thread. Delete it with the last of them.

Each run needs a Mac with the app it probes, and a person at it, since an assistant cannot see
a desktop app. Each run installs cogmer built from the `host-probes` branch, because `main`
does not carry the user identity.

A person does only what needs the app: upload plugins, start a session, send prompts and read
what the screen shows. `docs/work/host-observer/probe.py` does the rest: moving state, starting
the daemon under test and a mock colleague, pairing them with the two words compared in the
script, making a room, and collecting the record. It refuses while Claude Desktop, the ChatGPT app
or Codex is running. A Claude Code session may run alongside it to resolve what comes up: that
session's prompts reach the probe's daemon and are ignored there, since a session in no room hands
cogmer nothing (D-284, a session in no room reaches cogmer only through the commands its user
types).

## What a run installs

`docs/work/host-observer/` is a plugin for Claude Code and for Codex with hooks for
`SessionStart`, `UserPromptSubmit`, `PreToolUse` and `Stop`. Each hook appends to
`~/.cogmer-host-observer/log.txt`, on the machine where the hook runs, the event, the operating
system, the user, `HOME`, the names of the fields the hook received with each value's length,
the names of the environment variables that mention a host, the values of six that say when
and how the session was made, and whether a cogmer daemon answers
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
Mac, and refuses while `plugin`, `cmd` or `docs/work/host-observer` hold uncommitted changes.
Built on the droplet, the files are fetched with `scp`. `probe.py` reads the folder named by
`PROBE_BUILD`, `~/host-probe-build` when unset.

## Before and after every run

A program built from this branch migrates the state directory it opens to data version 1, and
a program from `main` then refuses to touch it (D-244, a program that finds newer data touches
nothing). The Mac's `~/.cogmer` holds real pairings and rooms, so a run moves it aside and
works in a fresh one.

Before, with Claude Desktop, the ChatGPT app and Codex quit:

```sh
python3 docs/work/host-observer/probe.py before <1|2|3>
```

It stops the daemon, moves `~/.cogmer` to `~/.cogmer.before-probe`, installs the built program in
a fresh `~/.cogmer/bin`, empties the observer's log, and for probe 1 takes the `cogmer` and
`cogmer-host-probe` entries out of Desktop's configuration after keeping a copy. It refuses when
either folder it would write already exists.

After, with the app quit again:

```sh
python3 docs/work/host-observer/probe.py report <1|2|3>   # while the state is still the probe's
python3 docs/work/host-observer/probe.py after
```

`report` writes `docs/work/host-probe-<probe>-<UTC time>.txt` from the daemon's room and the
observer's log, with two sections left for the person: the steps as clicked with each screen's
text copied word for word, and what the run does not show. `after` stops both daemons, keeps the
probe's state as `~/.cogmer.probe-<date>`, puts the real state and Desktop's configuration back,
and reminds the person to remove the observer and the probe's cogmer plugin from the app and to
reinstall the cogmer plugin the Mac used before. The record is committed with a message that
says when, on what machine and versions, and what was seen (W-91 in `docs/writing.md`, a commit
that records evidence says how it was run).

## The mock colleague

The two-way part of each run needs a colleague. `docs/work/host-observer/colleague.py` is a
second cogmer on the same Mac, with its own state directory and addresses, its own user identity
and machine key, and a daemon that reaches the one under test through the relay as a colleague on
another machine would. Its turns are written by the script, through the real `hook prompt` and
`hook stop` commands with hand-written transcripts, so no second session is needed.
`probe.py setup` starts it, pairs it and makes a room that invites the user under test, and
`probe.py say` sends a turn from it with a code word in it. It never opens `~/.cogmer`.

`cogmer seed` cannot stand in for a colleague. The turns it writes come from a peer nobody
verified, who is not on the room's guest list, and the daemon gives a prompt no turn of either
kind. On 2026-11-17 on the droplet, a room seeded with two turns gave the next prompt none. A
two-cogmer trial on 2026-11-18 on the Mac gave a prompt the other's turn, once the two sessions
had different session identifiers.

## Probe 1: Cowork on the user's computer with no bridge

The question is whether the ordinary cogmer plugin captures and injects in a Cowork session
that runs on the Mac, with no `cogmer` entry in the Desktop configuration. `78477ee` saw a
local session run its command hooks on the Mac and its shell in a Linux VM, so the hooks would
reach the Mac's daemon and a command's `!` line may not. Which side the `!` line runs on is
the first thing the run shows.

1. Quit Claude Desktop, the ChatGPT app and Codex, then run `probe.py before 1`.
2. Open Claude Desktop. In the plugin settings, disable or remove any cogmer plugin, then upload
   `cogmer-plugin.zip` and `host-observer.zip`.
3. Start a Cowork session that runs on this computer. Record the setting that says so, and what
   the screen shows before the first prompt.
4. Send `hello`. Record whether a line saying cogmer is installed appears, and the reply. Run
   `/host-observer:where` and record both outputs: the command's line and the shell tool each
   report their operating system, and the log shows where each hook ran.
5. In a terminal, run `probe.py check`. It shows whether a daemon is answering on the Mac, which
   is whether the plugin started one, since none was started before this step.
6. Run `probe.py setup`. It makes the identity, pairs the mock and makes a room. Then in the
   session, run `/cogmer:self-status` and record what it says and whether a browser page opens on
   the Mac.
7. In the session, run the join line `setup` printed, `/cogmer:room-join <room>`. Send two
   prompts, then run `/cogmer:room-log` and `/cogmer:room-status`, and open the watch address in a
   browser. Record whether the two turns are in the log. `probe.py check` counts the events
   from the session.
8. Run `probe.py say`. In the session, ask what the colleague just said, word for word. Record
   the reply, which holds the code word when the session was given the turn. Then send one more
   prompt, and run `probe.py check` to see it reach the mock's room.
9. Quit the app, then run `probe.py report 1` and `probe.py after`, and restore the person's
   usual plugins.

## Probe 2: a merged Chat and Cowork conversation

The question is which of the plugin's hooks fire in a conversation that starts as Chat and
in which Claude later uses Cowork. Only the observer is needed, since the question is whether
hooks fire at all.

1. As the Team plan's admin, turn on the setting "Chat and Cowork unified", and record when.
   Quit the apps, then run `probe.py before 2`.
2. Upload `host-observer.zip` in Claude Desktop, if probe 1 has not.
3. Start a new conversation. Send two prompts that need no tool, such as asking for a word that
   rhymes with orange.
4. Send a prompt that needs Cowork, such as asking for a file listing three fruits in a folder
   the conversation can write to. Record what the screen shows as Cowork joins.
5. Send two more prompts that need no tool.
6. Run `/host-observer:where`, and if it is not offered, ask Claude to run
   `bash <plugin folder>/hooks/where.sh tool`. Its output holds the log of the machine the
   shell tool runs on. `probe.py check` summarizes the log of the Mac, which holds the lines of
   every hook that ran on the Mac.
7. Record which `[host-observer]` lines Claude says it can see for each prompt, by asking
   "which lines in your context start with [host-observer]?".
8. Turn the setting off again if it was off before, quit the app, then run `probe.py report 2`
   and `probe.py after`, and remove the observer.

## Probe 3: a ChatGPT Work thread

The question is whether cogmer's Codex plugin captures and injects in a local Work thread, as
Codex's hooks do. OpenAI's page on hooks (https://learn.chatgpt.com/docs/hooks, read on
2026-11-16) says local Work threads run a plugin's hooks once they are trusted.

1. Quit the apps, then run `probe.py before 3`.
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
6. Run `probe.py setup`, then in the thread `$cogmer self-status`, `$cogmer room-join <room>`, two
   prompts, and `$cogmer room-log`. Record whether the log shows the turns, and whether the
   hook logged the typed `$cogmer` as a prompt's first word.
7. With the mock, do step 8 of probe 1 from the Work thread.
8. Quit the app, then run `probe.py report 3` and `probe.py after`, and remove both plugins and
   the marketplace entries the run added.
