#!/bin/sh
# Appends one line to /tmp/cprobe-hooks.log in the machine it runs on: the time, the event, the
# host name, and the values of CLAUDE_CODE_ENTRYPOINT and CLAUDE_CODE_REMOTE. It reads nothing
# else, prints nothing and sends nothing anywhere.
cat > /dev/null 2>&1
printf '%s %s host=%s entrypoint=%s remote=%s\n' "$(date -u +%H:%M:%S)" "$1" "$(hostname 2>/dev/null)" \
  "${CLAUDE_CODE_ENTRYPOINT-unset}" "${CLAUDE_CODE_REMOTE-unset}" >> /tmp/cprobe-hooks.log 2>/dev/null
exit 0
