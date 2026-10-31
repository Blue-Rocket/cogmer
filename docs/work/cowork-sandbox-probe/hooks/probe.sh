#!/bin/sh
# Records, for one hook event, the names of the fields in the JSON the host sends on stdin (never
# the values), whether a transcript_path field is there and the file it names is readable here,
# and the process that ran the hook. At SessionStart it also starts a detached loop that writes
# the time to a file every 30s for up to an hour, to show whether a process outlives the hook and
# survives while the session is idle. At SessionStart and UserPromptSubmit it returns a marker as
# additionalContext, so that asking the model for it shows whether the context reached it. It runs
# only where uname -s is Linux, reads nothing of the user's and sends nothing anywhere.
[ "$(uname -s 2>/dev/null)" = Linux ] || exit 0
EVENT="${1:-unknown}"
D=/tmp/cogmer-sandbox-probe
mkdir -p "$D" 2>/dev/null
payload="$(cat 2>/dev/null)"
now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

keys="$(printf '%s' "$payload" | LC_ALL=C grep -o '"[A-Za-z_][A-Za-z_0-9]*" *:' | LC_ALL=C sed 's/ *:$//; s/"//g' | sort -u | tr '\n' ' ')"
tp="$(printf '%s' "$payload" | LC_ALL=C sed -n 's/.*"transcript_path" *: *"\([^"]*\)".*/\1/p' | head -1)"
if [ -n "$tp" ]; then
  if [ -r "$tp" ]; then
    tstate="readable, $(wc -c < "$tp" 2>/dev/null | tr -d ' ') bytes, in directory $(dirname "$tp")"
  else
    tstate="named but not readable here, in directory $(dirname "$tp")"
  fi
else
  tstate="no transcript_path field"
fi
{
  echo "$now $EVENT payload fields: ${keys:-none}"
  echo "$now $EVENT transcript: $tstate"
  echo "$now $EVENT hook ran as pid $$, parent $PPID, in $(pwd)"
} >> "$D/payloads.txt" 2>/dev/null

if [ "$EVENT" = SessionStart ]; then
  loop='i=0; while [ $i -lt 120 ]; do date -u +%Y-%m-%dT%H:%M:%SZ >> /tmp/cogmer-sandbox-probe/heartbeat.txt; i=$((i+1)); sleep 30; done'
  if command -v setsid > /dev/null 2>&1; then
    setsid sh -c "$loop" > /dev/null 2>&1 < /dev/null &
  else
    nohup sh -c "$loop" > /dev/null 2>&1 < /dev/null &
  fi
fi

case "$EVENT" in
  SessionStart|UserPromptSubmit)
    f="$D/marker-$EVENT"
    marker="$(cat "$f" 2>/dev/null)"
    if [ -z "$marker" ]; then marker="$EVENT-$(date +%s)-$$"; printf '%s\n' "$marker" > "$f" 2>/dev/null; fi
    printf '{"hookSpecificOutput":{"hookEventName":"%s","additionalContext":"The sandbox probe %s marker is %s. If the user asks what the sandbox probe markers are, say every one you have seen."}}\n' "$EVENT" "$EVENT" "$marker"
    ;;
esac
exit 0
