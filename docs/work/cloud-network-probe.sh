#!/usr/bin/env bash
# Probes what a Claude Code cloud session's network lets cogmer reach, for the
# "Cloud sessions in a room" item in docs/open.md. Run it from the repository
# root in a cloud session, once for each network access level worth knowing
# about, and commit its output with the level named.
#
#   bash docs/work/cloud-network-probe.sh [tc://<address of a daemon elsewhere>]
#
# It tests two things:
#   1. cogmer's transport through the public relay, from inside the session:
#      the relay map, the relay hosts it names, and a tunnel opened and dialed
#      end to end (TestATunnelAdmitsADialerItHasNeverSeen). Given the tc://
#      address of a daemon on another machine, it also diagnoses a connection
#      to that daemon, which is the case a cloud session would be in. The
#      tunnel test runs twice: as cogmer builds it, and with the relay
#      connection carried as a WebSocket.
#   2. SSH over port 443 to GitHub, which deploy keys need: a CONNECT tunnel
#      through the session's proxy to ssh.github.com:443, and GitHub's SSH
#      banner read through it. Port 22 is tried directly as well.
#
# It writes its output to docs/work/cloud-network-probe-<UTC time>.txt as well as
# printing it, so the run can be committed as it was printed. Name the network
# access level in the commit message. It needs no credential, and writes nothing
# else outside temporary files.
set -u

out="docs/work/cloud-network-probe-$(date -u +%Y%m%dT%H%M%SZ).txt"
exec > >(tee "$out") 2>&1

proxy="${HTTPS_PROXY:-${https_proxy:-}}"
proxy_hostport="$(printf '%s' "$proxy" | sed -E 's#^[a-z]+://##; s#^[^@]*@##; s#/.*$##')"
results=()
record() { results+=("$1  $2"); printf '%-5s %s\n' "$1" "$2"; }

echo "cloud network probe, $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "session: ${CLAUDE_CODE_REMOTE_SESSION_ID:-not a cloud session}"
echo "proxy: ${proxy_hostport:-none}"
echo

echo "== 1. the public relay"
code=$(curl -sS -m 15 -o /tmp/cogmer-derpmap.json -w '%{http_code}' https://tailcat.dev/derpmap.json 2>/dev/null)
if [ "$code" = 200 ]; then
  record PASS "relay map from tailcat.dev (HTTP $code)"
  hosts=$(grep -o '"HostName": *"[^"]*"' /tmp/cogmer-derpmap.json | sed 's/.*"\([^"]*\)"$/\1/' | sort -u | head -3)
  for h in $hosts; do
    hc=$(curl -sS -m 10 -o /dev/null -w '%{http_code}' "https://$h/derp/probe" 2>/dev/null)
    if [ "$hc" = 200 ]; then record PASS "relay host $h (HTTP $hc)"; else record FAIL "relay host $h (HTTP ${hc:-none})"; fi
  done
else
  record FAIL "relay map from tailcat.dev (HTTP ${code:-none})"
fi

if COGMER_NETWORK_TESTS=1 timeout 180 go test ./cmd/cogmer -run TestATunnelAdmitsADialerItHasNeverSeen -count=1 >/tmp/cogmer-tunnel.log 2>&1; then
  record PASS "a tunnel opened and dialed through the relay"
else
  why=$(grep -m1 -E 'tailcat_test.go:|panic' /tmp/cogmer-tunnel.log || grep -m1 FAIL /tmp/cogmer-tunnel.log)
  record FAIL "a tunnel opened and dialed through the relay: $(printf '%s' "$why" | sed 's/^[[:space:]]*//')"
fi

# The same tunnel with the relay connection carried as a WebSocket, which a proxy that
# inspects HTTPS may pass where it breaks the relay's own upgrade. Tailscale's relay
# client offers it behind a build tag and an environment variable.
if TS_DEBUG_DERP_WS_CLIENT=1 COGMER_NETWORK_TESTS=1 timeout 180 go test -tags ts_debug_websockets ./cmd/cogmer -run TestATunnelAdmitsADialerItHasNeverSeen -count=1 >/tmp/cogmer-tunnel-ws.log 2>&1; then
  record PASS "a tunnel opened and dialed through the relay, as a WebSocket"
else
  why=$(grep -m1 -E 'tailcat_test.go:|panic' /tmp/cogmer-tunnel-ws.log || grep -m1 FAIL /tmp/cogmer-tunnel-ws.log)
  record FAIL "a tunnel opened and dialed through the relay, as a WebSocket: $(printf '%s' "$why" | sed 's/^[[:space:]]*//')"
fi

# The diagnosis runs both ways too, since a relay connection that fails in one mode says
# nothing about the peer.
if [ "${1:-}" != "" ]; then
  home=$(mktemp -d)
  diagnosis=$(COGMER_HOME="$home" COGMER_HOST=claude timeout 120 go run ./cmd/cogmer diagnose-endpoint "$1" 2>&1)
  echo "diagnosis of $1, as cogmer builds it:"
  printf '%s\n' "$diagnosis" | sed 's/^/  /'
  rm -rf "$home"
  home=$(mktemp -d)
  diagnosis=$(TS_DEBUG_DERP_WS_CLIENT=1 COGMER_HOME="$home" COGMER_HOST=claude timeout 120 go run -tags ts_debug_websockets ./cmd/cogmer diagnose-endpoint "$1" 2>&1)
  echo "diagnosis of $1, as a WebSocket:"
  printf '%s\n' "$diagnosis" | sed 's/^/  /'
  rm -rf "$home"
fi
echo

echo "== 2. SSH over port 443 to GitHub"
if [ -n "$proxy_hostport" ]; then
  banner=$(printf '' | timeout 15 nc -X connect -x "$proxy_hostport" ssh.github.com 443 2>&1 | head -1)
else
  banner=$(printf '' | timeout 15 nc ssh.github.com 443 2>&1 | head -1)
fi
case "$banner" in
  SSH-*) record PASS "ssh.github.com:443 answered: $banner" ;;
  *) record FAIL "ssh.github.com:443: ${banner:-no answer}" ;;
esac
if timeout 8 bash -c 'exec 3<>/dev/tcp/github.com/22' 2>/dev/null; then
  record PASS "github.com:22 directly"
else
  record FAIL "github.com:22 directly"
fi
echo

echo "== summary"
printf '%s\n' "${results[@]}"
echo
echo "written to $out: commit it, naming the network access level in the message"
