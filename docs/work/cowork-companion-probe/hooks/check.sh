#!/bin/sh
# Says whether the file cloud.value is in this plugin's folder, and its length, mode, owner and the
# first eight hex digits of its sha256, never its content, plus the plugin's version, the plugin folder, the user id the
# hook runs as, whether CLAUDE_CODE_REMOTE is set and the host. It appends the line to
# /tmp/cogmer-companion-probe.txt and returns it as additionalContext, so that a session can be asked
# what the probe says without running a command. It runs only where uname -s is Linux, reads nothing
# else and sends nothing anywhere. The file holds random bytes made for this test and no secret.
[ "$(uname -s 2>/dev/null)" = Linux ] || exit 0
cat > /dev/null 2>&1
EVENT="${1:-unknown}"
ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
F="$ROOT/cloud.value"
OUT=/tmp/cogmer-companion-probe.txt
now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

if [ -f "$F" ]; then
  len="$(wc -c < "$F" | tr -d ' ')"
  mode="$(stat -c '%a' "$F" 2>/dev/null)"
  owner="$(stat -c '%U:%G uid %u' "$F" 2>/dev/null)"
  sum="$( (sha256sum "$F" 2>/dev/null || shasum -a 256 "$F" 2>/dev/null) | cut -c1-8)"
  case "$(head -c 16 "$F" 2>/dev/null)" in
    cogmer-cloud-v1:) prefix="starts with the public prefix cogmer-cloud-v1:" ;;
    *) prefix="does not start with cogmer-cloud-v1:" ;;
  esac
  state="present, $len bytes, $prefix, mode $mode, owner $owner, sha256 starts $sum"
else
  state="absent"
fi
ver="$(sed -n 's/.*"version" *: *"\([^"]*\)".*/\1/p' "$ROOT/.claude-plugin/plugin.json" 2>/dev/null | head -1)"
de="$(sed -n 's/.*"defaultEnabled" *: *\([a-z]*\).*/\1/p' "$ROOT/.claude-plugin/plugin.json" 2>/dev/null | head -1)"
line="$now $EVENT companion probe: plugin version ${ver:-unknown}, defaultEnabled ${de:-not set}, cloud.value $state; plugin folder $ROOT; hook runs as uid $(id -u); CLAUDE_CODE_REMOTE ${CLAUDE_CODE_REMOTE-not set}; host $(hostname 2>/dev/null)"
printf '%s\n' "$line" >> "$OUT" 2>/dev/null
line="$(printf '%s' "$line" | tr -d '\\"' | tr '\r\n\t' '   ')"
printf '{"hookSpecificOutput":{"hookEventName":"%s","additionalContext":"The companion probe says: %s. If the user asks what the companion probe says, repeat it exactly."}}\n' "$EVENT" "$line"
exit 0
