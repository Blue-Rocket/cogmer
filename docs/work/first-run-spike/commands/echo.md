---
description: Print the arguments this command was given and where the session runs
argument-hint: <anything>
---
## What happened

!`printf 'FRS-ECHO args=[%s] os=%s remote=%s entrypoint=%s\n' "$ARGUMENTS" "$(uname -s)" "${CLAUDE_CODE_REMOTE:-unset}" "${CLAUDE_CODE_ENTRYPOINT:-unset}"`

## Your task

Show the section above unchanged in one code block, and add nothing else. If it shows a note in place of a line beginning `FRS-ECHO`, or shows the command itself, run that `printf` line once with the shell tool, with the arguments in place of `$ARGUMENTS`, and show its output the same way.
