#!/bin/sh
# Prints what the sandbox probe recorded. It reads only the probe's own files under /tmp.
D=/tmp/cogmer-sandbox-probe
echo "== now: $(date -u +%Y-%m-%dT%H:%M:%SZ) on $(hostname 2>/dev/null), $(uname -sm 2>/dev/null)"
echo; echo "== the relay probe (relay.sh)"; cat /tmp/cogmer-relay-probe.txt 2>&1
echo; echo "== the network probe (net.sh)"; cat /tmp/cogmer-network-probe.txt 2>&1
echo; echo "== hook payloads and where hooks ran"; cat "$D/payloads.txt" 2>&1
echo; echo "== markers the hooks returned"
for f in "$D"/marker-*; do [ -f "$f" ] && echo "$(basename "$f"): $(cat "$f")"; done 2>&1
echo; echo "== heartbeat (one line every 30s from SessionStart)"
if [ -f "$D/heartbeat.txt" ]; then
  echo "lines: $(wc -l < "$D/heartbeat.txt" | tr -d ' ')"
  echo "first: $(head -1 "$D/heartbeat.txt")"
  echo "last: $(tail -1 "$D/heartbeat.txt")"
else
  echo "no heartbeat file"
fi
