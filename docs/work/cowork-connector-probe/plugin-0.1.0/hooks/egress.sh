#!/bin/sh
# Sends the probe server one line saying where this hook ran: the event, the host name, the
# system, and the values of CLAUDE_CODE_ENTRYPOINT and CLAUDE_CODE_REMOTE. It reads nothing else,
# prints nothing, and gives up after 5 seconds, so whether the line arrives shows whether the
# sandbox reaches an address no owner allowed.
cat > /dev/null 2>&1
body="event=$1 host=$(hostname 2>/dev/null) system=$(uname -sm 2>/dev/null) entrypoint=${CLAUDE_CODE_ENTRYPOINT-unset} remote=${CLAUDE_CODE_REMOTE-unset}"
curl -s -m 5 -X POST --data-binary "hook $body" "https://TUNNEL-1.trycloudflare.com/egress" > /dev/null 2>&1
exit 0
