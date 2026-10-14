# D-195 — A restart stops the daemon and starts it with the hook's own function

**Date:** 2026-10-12 · **Status:** active · **Areas:** daemon, commands

**Decision.** `/cogmer:self-restart` runs `cli.sh restart`, which runs `cogmer stop --for-restart`, starts a
daemon with `start_daemon_if_needed`, waits up to five seconds for it to answer, and says whether it does.
When it does not, it relays the reason the daemon recorded.

**Support.**
- A start has to be detached and logged, and the hook's function settles how, with and without
  `setsid`, without costing the daemon its GUI session. A second path in Go would be one more thing to keep
  in step with it on every platform. D-150 (the session-start hook starts the daemon, and never delays the
  session), `plugin/hooks-handlers/common.sh`, `start_daemon_if_needed`.
- Started and answering are different facts, and somebody who asks for a restart wants the second, so the
  command waits for it. The wait is bounded, so a command never holds a session up. D-191 (a command answers
  at once and says what to do when the install is not finished), `plugin/cli.sh`, `restart_daemon`.
- When something that is not cogmer holds an address, `stop` leaves it and the start fails, and the daemon
  has recorded why, so the command reads that and says it. D-123 (`stop` finds a daemon by the addresses it
  holds, and signals only cogmer), `cmd/cogmer/main.go`, `reportDaemonBlocked`.
- `stop` says that nothing runs until a session starts one, which a restart contradicts, so `--for-restart`
  turns that note off. `cmd/cogmer/main.go`, `runStop`.
- A restart over a real daemon ends it and leaves one that answers, and with the stop, the start or the flag broken each test
  failed. `cmd/cogmer/cli_test.go`, `TestARestartReplacesARunningDaemonWithOneThatAnswers` and
  `TestARestartStopsThenStartsAndSaysWhenNothingAnswers`.

**Rejected.**
- *A Go subcommand that starts the daemon.* It would be a second start path beside the hook's.
- *`stop` starting the daemon.* Somebody who wants it stopped would have to stop it twice. D-193 (`stop` ends
  the daemon and starts nothing, and `/cogmer:self-stop` runs it).

**Limits.** A Cowork user's daemon is started by the bridge's own Go function, which this command does not share (D-208). The five-second wait may end before a slow machine's daemon answers, and the command then says
it was started and is not answering yet, which is true and may be gone a moment later.

**Revisit when** the hook's start function moves into the binary, since both starts would then be one.
