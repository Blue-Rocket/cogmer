#!/usr/bin/env bash
# Records the form of a hook's transcript_path: the path as given, its extension, and whether the
# file exists, what kind it is and how large. It never reads a transcript's content, writes nothing
# to the session, and exits 0 whatever happens.
event="${1:-unknown}"
log=/tmp/cogmer-path-probe.log
input="$(cat 2>/dev/null)"
{
  printf '=== %s event=%s\n' "$(date -u +%FT%TZ)" "$event"
  if command -v python3 >/dev/null 2>&1; then
    printf '%s' "$input" | python3 -c '
import json, os, stat, sys
try:
    d = json.load(sys.stdin)
except Exception as e:
    print("  input is not JSON:", type(e).__name__)
    sys.exit()
print("  fields:", ", ".join(sorted(d)))
p = d.get("transcript_path")
print("  transcript_path:", repr(p))
if isinstance(p, str) and p:
    print("  extension:", repr(os.path.splitext(p)[1]))
    print("  basename:", repr(os.path.basename(p)))
    print("  dirname:", repr(os.path.dirname(p)))
    try:
        st = os.stat(p)
        kind = ("regular" if stat.S_ISREG(st.st_mode) else "fifo" if stat.S_ISFIFO(st.st_mode)
                else "directory" if stat.S_ISDIR(st.st_mode) else "other")
        print("  exists: yes, kind:", kind, ", size:", st.st_size)
    except Exception as e:
        print("  exists: no (", type(e).__name__, ")")
else:
    print("  transcript_path is empty or absent")
print("  session_id, first 8:", str(d.get("session_id", ""))[:8])
' 2>&1
  else
    printf '  no python3 here\n'
  fi
} >> "$log" 2>&1
exit 0
