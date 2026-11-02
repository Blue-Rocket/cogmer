# D-282 — The person is shown one line when the install finishes and when cogmer needs repair

**Date:** 2026-11-02 · **Status:** not built · **Areas:** release, daemon

**Decision.** The session-start hook shows the person one line, in the hook's `systemMessage`, when the install has finished and when cogmer cannot work and the person can act: an install that failed or stalled (D-075, the install has a state, and absence is not an inference), a daemon that could not bind its address (D-177, a busy daemon port is told apart: one of ours exits quietly, anything else is reported), and an upgrade that failed (D-256, the databases are copied before the state directory is migrated). The line for a finished install is shown once, by the first hook that runs after the installer finishes, and says what to run next. A line for a fault names the reason and is shown at the start of each session while the fault holds, and never at a prompt. The hook shows nothing while the install is in progress and nothing when cogmer is working, so an ordinary session carries nothing. A user whose install failed or whose daemon is blocked loses collaboration with nothing on the screen to say so, and one line that says what to do serves them better than an answer they get only by asking. The hook also gives the model the state, so a session asked why nothing works answers from the fact. Claude Code showed a hook's `systemMessage` on the screen as "<hook> says: ..." on macOS and on Windows and never gave it to the model (`11b1294`).

**Rejected.**
- *Telling the person only through the model.* A user who does not know to ask learns nothing from a failed install or a blocked daemon, since the model speaks of them only when asked.
- *The fault line at each prompt while the fault holds.* The session's first line has said it, and repeating it at every prompt spends the user's attention on nothing new.

**Limits.** The field was shown on macOS in Claude Code 2.1.316 and on Windows in 2.1.312, in sessions started with `--plugin-dir` (`11b1294`). No run shows it for a plugin installed through a marketplace, in Codex, in Cowork or in the VS Code extension, and a host that does not show it leaves the person with the model's note alone. A later version of Claude Code can stop showing it, and nothing checks that.

**Revisit when** a host that cogmer supports is found to show the person no hook message.
