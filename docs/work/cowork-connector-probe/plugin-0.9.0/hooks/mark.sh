#!/bin/sh
# Appends one line to /tmp/cprobe-hooks.log in the machine it runs on: the time, the event, the
# host name, and the values of CLAUDE_CODE_ENTRYPOINT and CLAUDE_CODE_REMOTE, and the transcript
# path the hook was given with whether that file exists and its size. On PostToolUse it adds the
# tool's name and the mcp_server field. On Stop it writes to /tmp/cprobe-transcript.txt the
# event, type, error text, reply text and command of each hook outcome the transcript records, so the
# outcome the session recorded for each hook can be read. It never records a tool's input or
# answer beyond those lines, prints nothing and sends nothing anywhere.
in="$(cat 2>/dev/null)"
info="$(printf '%s' "$in" | python3 -c '
import json, os, sys
try:
    d = json.load(sys.stdin)
except Exception:
    print("unreadable input"); sys.exit()
tp = d.get("transcript_path") or ""
out = "transcript=%s exists=%s size=%s" % (tp, os.path.exists(tp), os.path.getsize(tp) if os.path.exists(tp) else "-")
if d.get("hook_event_name") == "PostToolUse":
    out += " tool=%s mcp_server=%s" % (d.get("tool_name"), json.dumps(d.get("mcp_server")))
if d.get("hook_event_name") == "Stop" and os.path.exists(tp):
    with open(tp, errors="replace") as f, open("/tmp/cprobe-transcript.txt", "w") as o:
        for n, line in enumerate(f, 1):
            try:
                a = json.loads(line).get("attachment") or {}
            except Exception:
                continue
            if not str(a.get("type", "")).startswith("hook_"):
                continue
            o.write("%d: %s %s stderr=%s stdout=%s command=%s\n" % (n, a.get("hookEvent"), a.get("type"),
                    json.dumps(str(a.get("stderr", ""))[:200]), json.dumps(str(a.get("stdout", a.get("content", "")))[:120]),
                    json.dumps(str(a.get("command", ""))[:80])))
print(out)
' 2>&1)"
printf '%s %s host=%s entrypoint=%s remote=%s %s\n' "$(date -u +%H:%M:%S)" "$1" "$(hostname 2>/dev/null)" \
  "${CLAUDE_CODE_ENTRYPOINT-unset}" "${CLAUDE_CODE_REMOTE-unset}" "$info" >> /tmp/cprobe-hooks.log 2>/dev/null
exit 0
