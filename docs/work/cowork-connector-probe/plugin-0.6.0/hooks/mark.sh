#!/bin/sh
# Appends one line to /tmp/cprobe-hooks.log in the machine it runs on: the time, the event, the
# host name, and the values of CLAUDE_CODE_ENTRYPOINT and CLAUDE_CODE_REMOTE. On PostToolUse it
# adds the tool's name, the names of the fields the hook was given, and the mcp_server field, so
# the server a connector's tool belongs to can be named. It never records the tool's input or
# answer, prints nothing and sends nothing anywhere.
in="$(cat 2>/dev/null)"
extra=""
if [ "$1" = PostToolUse ] && command -v python3 > /dev/null 2>&1; then
  extra="$(printf '%s' "$in" | python3 -c '
import json, sys
try:
    d = json.load(sys.stdin)
except Exception:
    print("unreadable input"); sys.exit()
print("tool=%s keys=%s mcp_server=%s" % (d.get("tool_name"), ",".join(sorted(d)), json.dumps(d.get("mcp_server"))))
' 2>&1)"
fi
printf '%s %s host=%s entrypoint=%s remote=%s %s\n' "$(date -u +%H:%M:%S)" "$1" "$(hostname 2>/dev/null)" \
  "${CLAUDE_CODE_ENTRYPOINT-unset}" "${CLAUDE_CODE_REMOTE-unset}" "$extra" >> /tmp/cprobe-hooks.log 2>/dev/null
exit 0
