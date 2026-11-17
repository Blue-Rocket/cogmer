First-run probe: what a person sees of a hook's systemMessage, and which hook forms run

The plugin in this folder has, for each of SessionStart and UserPromptSubmit, three hooks that
each append a line to cogmer-first-run-probe.log in the working directory and answer with a
systemMessage and an additionalContext carrying a marker (SYS-FRP-<form>-<event> and
CTX-FRP-<form>-<event>):
  bash         a shell-form command line that runs hooks/probe.sh through bash
  powershell   a shell-form command with "shell": "powershell", written inline
  exec         an exec-form hook: command "powershell" with args, no shell between
It holds no secret, reads nothing and sends nothing. Delete the run folder afterwards.

Part 1, print mode (run by the maintainer's assistant): run.sh on macOS and Linux, run.ps1 on Windows
(with -NoGit for the machine without Git for Windows). The report shows which hooks ran, what the
harness said of each, and which markers the model quoted from its context.

Part 2, interactive screens (a person at the keyboard, which an assistant cannot see). On each of
the machines below, in a new empty folder, with that machine's own Claude Code:
  claude --plugin-dir <this folder> --setting-sources project
Say what the screen shows between the start of the session and the first prompt, then type
"What lines in your context contain FRP-?" and say what the screen shows after it, in your own
words and with any text copied exactly:
  a) macOS, where PowerShell is not installed
  b) Windows with Git for Windows on the PATH
  c) Windows with no bash.exe on the PATH (start the session from a command prompt after
     removing Git's folders from PATH for that prompt only: set PATH=...)
The questions this answers: does a systemMessage appear anywhere a person looks, in which of the
three forms; and does a hook that fails to run (the bash form without Git, the PowerShell and exec
forms on macOS) put anything on the screen.

Not covered: Codex, Cowork, the VS Code extension, and a plugin installed from a marketplace.
