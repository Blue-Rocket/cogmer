#!/usr/bin/env bash
# Says where it runs, so that a run can tell a hook's machine from the machine of a
# command's `!` line or of the model's shell tool, then prints the observer's log from
# here. Usage: where.sh <label>, where the label says what started it: bang for a
# command's `!` line, tool for the model's shell tool.
label="${1:-unlabelled}"
dir="${HOME:-/tmp}/.cogmer-host-observer"
mkdir -p "$dir" 2>/dev/null
now="$(date -u +%FT%TZ 2>/dev/null)"
line="=== $now  where.sh label=$label  on $(uname -sm 2>&1)  hostname=$(hostname 2>/dev/null)  user=$(id -un 2>/dev/null)  HOME=$HOME
  CLAUDE_CODE_REMOTE=${CLAUDE_CODE_REMOTE:-unset}  CLAUDE_CODE_SESSION_ID=$(printf '%s' "${CLAUDE_CODE_SESSION_ID:-}" | cut -c1-8)  CODEX_THREAD_ID=$(printf '%s' "${CODEX_THREAD_ID:-}" | cut -c1-8)
  cogmer here: dir=$([ -d "$HOME/.cogmer" ] && echo yes || echo no) bin=$([ -x "$HOME/.cogmer/bin/cogmer" ] && "$HOME/.cogmer/bin/cogmer" version 2>/dev/null || echo none) daemon-127.0.0.1:4782=$(curl -s -m 2 -o /dev/null -w '%{http_code}' http://127.0.0.1:4782/healthz 2>/dev/null || echo fail)"
echo "$line" >> "$dir/log.txt" 2>/dev/null
echo "$line"
echo "----- the observer's log as seen from here ($dir/log.txt), last 200 lines:"
tail -n 200 "$dir/log.txt" 2>&1
exit 0
