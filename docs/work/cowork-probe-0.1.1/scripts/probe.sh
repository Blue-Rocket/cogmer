#!/usr/bin/env bash
# A throwaway probe for one question: does a merged Claude conversation fire this plugin's
# hooks, and where does a command hook run?
#
# It does nothing at all outside a sandbox: Claude Code on a person's own machine has no
# CLAUDE_CODE_REMOTE, so there it reads its input and exits 0 without writing anything.
# In a sandbox it appends one block to ~/.cogmer-probe/log.txt with the event, the machine
# name, a few environment variable NAMES and the names of the hook's input fields, never
# any content. It never fails and never delays a session.
event="${1:-unknown}"
input="$(cat 2>/dev/null)"
[ -n "${CLAUDE_CODE_REMOTE:-}" ] || exit 0
dir="${HOME:-/tmp}/.cogmer-probe"
mkdir -p "$dir" 2>/dev/null || { dir="/tmp/.cogmer-probe"; mkdir -p "$dir" 2>/dev/null; }
now="$(date -u +%FT%TZ 2>/dev/null)"
fields=""
if command -v python3 >/dev/null 2>&1 && [ -n "$input" ]; then
  fields="$(printf '%s' "$input" | python3 -c '
import json,sys
try: d=json.load(sys.stdin)
except Exception as e: print("not JSON"); sys.exit()
print("session8=%s fields=%s" % (str(d.get("session_id",""))[:8], ",".join(sorted(d.keys()))))' 2>&1)"
fi
{
  echo "=== $now event=$event host=$(uname -n 2>/dev/null) kernel=$(uname -srm 2>/dev/null) user=$(id -un 2>/dev/null)"
  echo "CLAUDE_CODE_REMOTE=${CLAUDE_CODE_REMOTE:-} cwd=$(pwd)"
  echo "input: $fields"
  echo "env names: $(env | cut -d= -f1 | grep -iE 'claude|ccr|cowork|anthropic|session|sandbox' | sort | tr '\n' ' ')"
} >> "$dir/log.txt" 2>&1
case "$event" in
  SessionStart) echo "[cogmer-probe] SessionStart fired in a sandbox (host $(uname -n 2>/dev/null))." ;;
  UserPromptSubmit) echo "[cogmer-probe] UserPromptSubmit fired in a sandbox (host $(uname -n 2>/dev/null), $now)." ;;
esac
exit 0
