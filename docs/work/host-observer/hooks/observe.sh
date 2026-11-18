#!/usr/bin/env bash
# A throwaway probe for the open host items: records which hook events fire, where each
# hook runs, and the NAMES of what it receives. It never records a prompt, a reply or
# any other content: a field is logged by name and length, except a few that hold no
# content (the event name, the source, the tool name, the permission mode), and a prompt
# is logged only as its first word when that word is a command (/x or $x). The values of six
# environment variables that say when and how the session was made are logged too, cut to 40
# characters; none holds content or a secret.
#
# It never fails and never delays a session: every command is bounded, and it exits 0.
# Usage: observe.sh <claude|codex> <event>
host="${1:-unknown}"
event="${2:-unknown}"
dir="${HOME:-/tmp}/.cogmer-host-observer"
mkdir -p "$dir" 2>/dev/null || { dir="/tmp/.cogmer-host-observer"; mkdir -p "$dir" 2>/dev/null; }
log="$dir/log.txt"
now="$(date -u +%FT%TZ 2>/dev/null)"
input="$(cat 2>/dev/null)"

count_file="$dir/count"
n=$(( $(cat "$count_file" 2>/dev/null || echo 0) + 1 ))
echo "$n" > "$count_file" 2>/dev/null

fields=""
if command -v python3 >/dev/null 2>&1 && [ -n "$input" ]; then
  fields="$(printf '%s' "$input" | python3 -c '
import json, sys
try:
    d = json.load(sys.stdin)
except Exception as e:
    print("input is not JSON:", type(e).__name__); sys.exit()
SAFE = ("hook_event_name", "source", "tool_name", "permission_mode", "model", "stop_hook_active")
out = {}
for k, v in d.items():
    if k in SAFE:
        out[k] = str(v)[:60]
    elif k in ("session_id", "turn_id", "thread_id", "prompt_id"):
        out[k] = str(v)[:8] + "..."
    elif k == "prompt":
        w = str(v).strip().split()
        out[k] = "<%d chars>%s" % (len(str(v)), (" first word " + w[0][:40]) if w and w[0][:1] in "/$" else "")
    elif isinstance(v, (dict, list)):
        out[k] = "<%s of %d>" % (type(v).__name__, len(v))
    else:
        out[k] = "<%d chars>" % len(str(v))
print(json.dumps(out, sort_keys=True))' 2>&1 | cut -c1-900)"
elif [ -n "$input" ]; then
  # No python3: the top-level names only, which may include some nested ones.
  fields="(no python3) $(printf '%s' "$input" | wc -c | tr -d ' ') bytes, names: $(printf '%s' "$input" | grep -oE '"[A-Za-z_]+"[[:space:]]*:' | tr -d '":[:space:]' | sort -u | tr '\n' ' ')"
fi

healthz() {
  command -v curl >/dev/null 2>&1 || { echo "no curl"; return; }
  curl -s -m 2 -o /dev/null -w '%{http_code}' "http://127.0.0.1:4782/healthz" 2>/dev/null || echo fail
}

{
  echo "=== $now  #$n  host=$host  event=$event"
  echo "  where: $(uname -sm 2>&1)  hostname=$(hostname 2>/dev/null)  user=$(id -un 2>/dev/null)  HOME=$HOME  cwd=$(pwd)"
  echo "  fields: ${fields:-<no input>}"
  if [ "$event" = SessionStart ] || [ "$event" = UserPromptSubmit ]; then
    echo "  CLAUDE_CODE_REMOTE=${CLAUDE_CODE_REMOTE:-unset}  CLAUDE_CODE_SESSION_ID=$(printf '%s' "${CLAUDE_CODE_SESSION_ID:-}" | cut -c1-8)  CODEX_THREAD_ID=$(printf '%s' "${CODEX_THREAD_ID:-}" | cut -c1-8)"
    vals=""
    for v in CLAUDE_CODE_SESSION_CREATED_AT_MS CCR_SPAWN_TIMESTAMP_MS CLAUDE_CODE_SESSION_ORIGIN CLAUDE_CODE_ENTRYPOINT CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE CLAUDE_CODE_VERSION; do
      vals="$vals $v=$(printf '%s' "${!v:-unset}" | cut -c1-40)"
    done
    echo "  env values:$vals"
    echo "  env names: $(env | cut -d= -f1 | grep -iE 'claude|codex|cogmer|cowork|ccr|chatgpt|openai|plugin|session|thread' | sort | tr '\n' ' ')"
    echo "  cogmer here: dir=$([ -d "$HOME/.cogmer" ] && echo yes || echo no) bin=$([ -x "$HOME/.cogmer/bin/cogmer" ] && "$HOME/.cogmer/bin/cogmer" version 2>/dev/null || echo none) bound-sessions=$(ls "$HOME/.cogmer/bound-sessions" 2>/dev/null | wc -l | tr -d ' ') daemon-127.0.0.1:4782=$(healthz)"
  fi
} >> "$log" 2>&1

# What the model is given. A marker on SessionStart and UserPromptSubmit shows whether a
# hook's plain output reaches the model on this host. The other events print nothing.
case "$event" in
  SessionStart) echo "[host-observer] SessionStart ran on $(uname -s 2>/dev/null) as event #$n." ;;
  UserPromptSubmit) echo "[host-observer] UserPromptSubmit ran on $(uname -s 2>/dev/null) as event #$n, and this line reached the model." ;;
esac
exit 0
