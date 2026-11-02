#!/bin/sh
# Runs one isolated `claude -p` turn with the probe plugin loaded from this folder, in a fresh
# working directory, with only project settings, and reports three things: which hooks ran (the
# probe's own log), which markers the model quoted from its context, and which markers appear in
# the stream of hook events the harness printed, which is what a program reading `claude -p`
# sees. It touches no cogmer state, reads nothing of the user's, and sends the model only the
# markers. The first argument is the folder for the run's files and must exist and be empty.
# Run it with Git for Windows' bash, or in a Linux or macOS shell. It runs `claude` from PATH.
set -u
here="$(cd "$(dirname "$0")" && pwd)"
out="${1:?usage: run.sh <existing empty folder for this run>}"
cd "$out" || exit 1
rm -f cogmer-first-run-probe.log
claude --version 2>&1 | head -1
prompt="List, exactly as written, every line in your context that contains the text FRP-. If there are none, reply with only the word NONE."
claude -p "$prompt" --plugin-dir "$here" --setting-sources project \
  --output-format stream-json --verbose --include-hook-events > stream.json 2> stderr.txt < /dev/null
echo "exit status $?"
echo "--- the probe's log (which hooks ran, in order)"
cat cogmer-first-run-probe.log 2>/dev/null || echo "(no log: no hook of the probe ran)"
echo "--- markers in the stream of hook events, by hook form and field"
grep -o 'SYS-FRP-[a-z]*-[A-Za-z]*\|CTX-FRP-[a-z]*-[A-Za-z]*' stream.json | sort | uniq -c
echo "--- the model's answer"
grep '"type":"result"' stream.json | sed 's/.*"result":"\([^"]*\)".*/\1/' | head -c 2000
echo
echo "--- stderr"
head -c 1000 stderr.txt
