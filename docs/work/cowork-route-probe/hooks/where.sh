#!/bin/sh
# Says where this hook runs: the host, the system, whether CLAUDE_CODE_REMOTE is set, whether
# /sessions exists and what is mounted under it, and HOME. With --plain it prints lines for a
# person. Without it, it prints one line of the hook's additionalContext JSON for the event it
# was run for. It reads nothing else and sends nothing anywhere.
plain=no
[ "${1:-}" = "--plain" ] && plain=yes

host="$(hostname 2>/dev/null)"
sys="$(uname -sm 2>/dev/null)"
if [ -n "${CLAUDE_CODE_REMOTE+x}" ]; then remote="set, value ${CLAUDE_CODE_REMOTE}"; else remote="not set"; fi
if [ -d /sessions ]; then sessions="exists"; else sessions="absent"; fi
mounts="$(ls -d /sessions/*/mnt/* 2>/dev/null | tr '\n' ' ')"
[ -n "$mounts" ] || mounts="none"
home="${HOME:-unset}"

if [ "$plain" = yes ]; then
  echo "where.sh ran on host: $host"
  echo "system: $sys"
  echo "CLAUDE_CODE_REMOTE: $remote"
  echo "/sessions: $sessions"
  echo "mounts under /sessions/*/mnt: $mounts"
  echo "HOME: $home"
  exit 0
fi

# The event the host ran this hook for, from the JSON it sends on stdin.
event="$(cat 2>/dev/null | sed -n 's/.*"hook_event_name" *: *"\([^"]*\)".*/\1/p' | head -1)"
[ -n "$event" ] || event=UserPromptSubmit
text="where.sh ran on host: $host; system: $sys; CLAUDE_CODE_REMOTE: $remote; /sessions: $sessions; mounts: $mounts; HOME: $home"
text="$(printf '%s' "$text" | tr -d '\\"' | tr '\r\n\t' '   ')"
printf '{"hookSpecificOutput":{"hookEventName":"%s","additionalContext":"%s"}}\n' "$event" "$text"
exit 0
