# D-084 — Silent to the person is not silent to the log

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** A test reported that a detached `open` never brought a browser forward,
and the conclusion stood for an hour before it turned out that macOS ships no
`setsid` — the command had never run. Asked why detection took so long, and whether
logging practice should change.

**Four things stacked, and each alone was survivable.** The error was discarded at
source: `setsid nohup open … >/dev/null 2>&1 &` closes both channels that would have
said the command did not exist — stderr to `/dev/null`, exit status to the
background. The observable was **negative**: "is the browser frontmost" cannot
distinguish *ran and did not focus* from *never ran*. A plausible mechanism was
available — macOS focus-stealing prevention is real and fit the data — so having an
explanation is what ended the inquiry. And the code taught the wrong thing:
`if command -v setsid` reads as "setsid is the preferred path" when it encodes "we do
not know whether this exists."

What broke it was a **positive** observable — counting browser tabs. Zero tabs is not
explicable by a focus theory. One measurement the prevailing story could not
accommodate did what five consistent ones could not.

**§3.1 requires silence toward the person, not toward the log,** and the code had
been conflating the two. `install.sh` kept a record of its decisions; `common.sh` and
`session-start.sh` kept none, so the four load-bearing choices in
`start_daemon_if_needed` — which binary won the search, whether the daemon answered,
which detach branch ran, whether a child started — were decided and recorded nowhere.
"The daemon is not running" had no next question.

**The rule adopted: redirect to the log by default; use `/dev/null` only when you can
say what the discarded output would have said.** Output that is *noise by
construction* may be dropped — an exit status is all that is wanted, stdin is being
drained, a `mkdir` whose failure the next line catches. Output that is merely
*usually empty* must not be, and that distinction is the whole lesson: the setsid
line was usually empty, which is exactly why the one time it had something to say it
looked normal.

**Never to stdout, in any of this.** A hook's stdout is injected into the user's
turn, so a stray line there does not produce a worse message, it corrupts a prompt.
`json_safe` was added for the same reason: details assembled from error text reach a
JSON string literal, and an unescaped quote makes a hook emit malformed JSON.

**What changed.** A shared `ct_say`; the four decisions above recorded; "curl said no"
separated from "there is no curl", which were the same answer and start a daemon on
every session; `install.sh` output to `install.log` rather than `/dev/null`, since its
own logger cannot record what goes wrong before it is defined; and its `mkdir`
failure — the quietest line in the file, and a load-bearing one — made audible.

**A busy port is two different events and now says which.** `net.Listen` failing was
`log.Fatalf` for both. Several sessions starting at once race to start a daemon and
exactly one wins: that is the ordinary outcome (§29), and it now exits quietly after
confirming via `/healthz` that the holder is one of ours. Anything else holding the
port is a real fault, and it now writes `daemon-state`, which the session-start hook
relays to the model — the same mechanism D-075 built for a half-finished install, and
the only route from a detached daemon to a person (D-033, D-036).

**Revisit when:** the logs grow enough to be worth rotating, or a failure is found
that none of these three channels — daemon.log, install.log, the state files —
would have caught.
