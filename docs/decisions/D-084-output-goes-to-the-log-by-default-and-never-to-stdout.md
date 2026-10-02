# D-084 — Output goes to the log by default, and never to stdout

**Date:** 2026-09-20 · **Status:** active · **Areas:** daemon, release

**Decision.** In the hooks and the installer, output goes to a log by default. It goes to
`/dev/null` only when the discarded output can be said in advance, and never to stdout.

**Support.**
- §3.1 asks for silence toward the user and not toward the log, and a decision that is
  recorded nowhere gives "the daemon is not running" no next question. §3.1 (first, do no
  harm), `plugin/hooks-handlers/common.sh`, `ct_say`.
- Output that is noise by construction may be dropped, as an exit status that is all that is
  wanted, a stdin being drained, or a `mkdir` whose failure the next line catches. Output that
  is merely usually empty must not be, because the one time it has something to say it looks
  normal. `plugin/hooks-handlers/session-start.sh`.
- A hook's stdout is injected into the user's turn, so a stray line there corrupts a prompt,
  and a detail assembled from error text must be made safe for a JSON string or the hook
  emits malformed JSON. `plugin/hooks-handlers/common.sh`, `json_safe`.
- The installer writes `install.log` and reports a failure to create its own directory
  before it can log anything else. `plugin/hooks-handlers/install.sh`.

**Rejected.**
- *`/dev/null` for output that is usually empty.* It hid a missing `setsid` for an hour,
  since the discarded error was the only sign.

**Revisit when** the logs grow enough to be worth rotating, or a failure is found that none
of `daemon.log`, `install.log` and the state files would have caught.
