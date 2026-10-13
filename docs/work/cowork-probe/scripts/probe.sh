#!/usr/bin/env bash
# A throwaway probe for one question: could cogmer run inside Claude Cowork?
#
# It records which hook events fire, what a hook can see (environment variable NAMES,
# the hook's input fields, the transcript's record types, never any content), what
# the sandbox can reach over the network, and whether files survive from one session
# to the next. It never fails and never delays a session: every command is bounded,
# and the script always exits 0.
event="${1:-unknown}"
dir="${HOME:-/tmp}/.cogmer-probe"
mkdir -p "$dir" 2>/dev/null || dir="/tmp/.cogmer-probe"; mkdir -p "$dir" 2>/dev/null
log="$dir/log.txt"
now="$(date -u +%FT%TZ 2>/dev/null)"
input="$(cat 2>/dev/null)"

# Which fields did the hook receive? Names only, apart from three that hold no content.
fields=""
if command -v python3 >/dev/null 2>&1 && [ -n "$input" ]; then
  fields="$(printf '%s' "$input" | python3 -c '
import json,sys
try: d=json.load(sys.stdin)
except Exception as e: print("input is not JSON:", e); sys.exit()
safe={k:(str(v)[:60] if k in ("hook_event_name","transcript_path","cwd","source") else ("<%d chars>"%len(str(v)))) for k,v in d.items()}
sid=str(d.get("session_id",""))
print("session_id first 8:", sid[:8]); print(json.dumps(safe))' 2>&1)"
elif [ -n "$input" ]; then
  fields="(no python3) input is $(printf '%s' "$input" | wc -c) bytes"
fi
tpath=""
command -v python3 >/dev/null 2>&1 && tpath="$(printf '%s' "$input" | python3 -c 'import json,sys
try: print(json.load(sys.stdin).get("transcript_path",""))
except Exception: pass' 2>/dev/null)"

# Persistence: does a file written in an earlier session survive into this one?
first="$dir/first_seen"
[ -f "$first" ] || echo "$now" > "$first" 2>/dev/null
count_file="$dir/count"; n=$(( $(cat "$count_file" 2>/dev/null || echo 0) + 1 )); echo "$n" > "$count_file" 2>/dev/null
boot="$(cat /proc/sys/kernel/random/boot_id 2>/dev/null | cut -c1-8)"
tmpmark="/tmp/.cogmer-probe-marker"; [ -f "$tmpmark" ] && tmpold="yes (from $(cat "$tmpmark"))" || { tmpold=no; echo "$now" > "$tmpmark" 2>/dev/null; }

{
echo "=== $now  event=$event  run#$n  boot=$boot"
echo "hook input: $fields"
if [ "$event" != "SessionStart" ] && [ -n "$tpath" ] && [ -f "$tpath" ]; then
  echo "transcript now $(wc -c < "$tpath") bytes; hook_success lines: $(grep -c hook_success "$tpath" 2>/dev/null); our UserPromptSubmit text recorded: $(grep -c 'cogmer-probe. UserPromptSubmit fired' "$tpath" 2>/dev/null)"
fi
if [ "$event" = "SessionStart" ]; then
  echo "uname: $(uname -srm 2>&1)"
  echo "user: $(id 2>&1 | cut -c1-80)  HOME=$HOME  cwd=$(pwd)"
  echo "home first seen (earlier sessions' file survives?): $(cat "$first" 2>/dev/null)   count so far: $n"
  echo "/tmp marker from an earlier session: $tmpold"
  echo "disk under HOME: $(df -h "$HOME" 2>&1 | tail -1 | cut -c1-100)"
  echo "mounts that may be host folders: $(mount 2>/dev/null | grep -iE 'virtiofs|9p|fuse|/mnt|/sessions' | cut -c1-110 | head -6 | tr '\n' '|')"
  echo "/sessions: $(ls -la /sessions 2>&1 | head -6 | tr '\n' '|')"
  echo "plugin data dir: ${CLAUDE_PLUGIN_DATA:-unset} -> $(ls -ld "${CLAUDE_PLUGIN_DATA:-/nonexistent}" 2>&1 | cut -c1-100)"
  echo "tools: python3=$(command -v python3 || echo no) curl=$(command -v curl || echo no) go=$(command -v go || echo no) git=$(command -v git || echo no)"
  echo "existing cogmer state here: $(ls -d "$HOME"/.cogmer "$HOME"/.cogmer/bin 2>&1 | tr '\n' ' ')"
  echo "env names (CLAUDE|CODEX|COGMER|ANTHROPIC|COWORK|SESSION): $(env | cut -d= -f1 | grep -iE 'claude|codex|cogmer|anthropic|cowork|session' | sort | tr '\n' ' ')"
  echo "CLAUDE_CODE_SESSION_ID first 8: $(printf '%s' "${CLAUDE_CODE_SESSION_ID:-}" | cut -c1-8)  CLAUDE_PLUGIN_ROOT=${CLAUDE_PLUGIN_ROOT:-}"
  if [ -n "$tpath" ]; then
    if [ -f "$tpath" ]; then
      echo "transcript: $tpath  $(wc -c < "$tpath") bytes"
      echo "transcript record types: $(grep -o '"type":"[a-z_-]*"' "$tpath" 2>/dev/null | sort | uniq -c | sort -rn | head -8 | tr '\n' ' ')"
      echo "transcript hook_success records: $(grep -c hook_success "$tpath" 2>/dev/null)"
    else
      echo "transcript path does not exist from here: $tpath"
    fi
  fi
  if command -v curl >/dev/null 2>&1; then
    c() { curl -s -m 4 -o /dev/null -w "%{http_code}" "$@" 2>/dev/null || echo fail; }
    echo "this sandbox's own 127.0.0.1:4782 (a cogmer daemon here?): $(c http://127.0.0.1:4782/healthz)"
    gw=""; command -v ip >/dev/null 2>&1 && gw="$(ip route 2>/dev/null | awk '/default/ {print $3; exit}')"
    echo "default gateway: ${gw:-unknown}"
    [ -n "$gw" ] && echo "gateway:4782 (a host daemon reachable from the sandbox?): $(c "http://$gw:4782/healthz")"
    for h in host.docker.internal host.lima.internal host.internal; do
      getent hosts "$h" >/dev/null 2>&1 && echo "name $h resolves: $(getent hosts "$h" | cut -c1-60)  :4782 -> $(c "http://$h:4782/healthz")"
    done
    echo "github release binary (HEAD, 302/200 = binary can be fetched): $(curl -sI -m 6 -o /dev/null -w '%{http_code}' https://github.com/Blue-Rocket/cogmer/releases/download/v0.9.1/cogmer_0.9.1_linux_arm64 2>/dev/null || echo fail)"
    echo "tailscale control plane: $(c https://controlplane.tailscale.com/key?v=71)"
    echo "tailscale relay (DERP) https: $(c https://derp1.tailscale.com/derp/probe)"
    echo "general web (example.com): $(c https://example.com)"
    for h in login.tailscale.com derp2.tailscale.com derp10.tailscale.com api.github.com pypi.org registry.npmjs.org; do
      echo "$h: $(c "https://$h/")"
    done
    echo "proxy settings: $(env | grep -iE '^(https?|all|no)_proxy=' | sed 's/=.*@/=<creds>@/' | cut -c1-120 | tr '\n' ' ')"
  fi
fi
} >> "$log" 2>&1

# What the model is given. Only SessionStart reports in full; UserPromptSubmit says
# only that its output arrived, which is the question for it.
case "$event" in
  SessionStart) echo "[cogmer-probe] SessionStart fired. Report recorded; run /cogmer-probe:probe-report to read it." ;;
  UserPromptSubmit) echo "[cogmer-probe] UserPromptSubmit fired and its output reached the model (run #$n at $now)." ;;
esac
exit 0
