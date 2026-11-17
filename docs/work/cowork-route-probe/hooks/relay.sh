#!/bin/sh
# Asks whether the machine this runs on can reach the public relay, and whether it does so as a
# WebSocket. It returns at once, runs only where uname -s is Linux, downloads the release program
# for this machine and refuses to run it unless its sha256 is the one pinned in checksums.txt,
# starts a daemon on a state directory made for the run, reads what the daemon logged about the
# relay, stops it, and writes what it found to /tmp/cogmer-route-probe-relay.txt. It reads
# nothing of the user's and keeps no key.
OUT=/tmp/cogmer-route-probe-relay.txt
ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"

if [ "${1:-}" != "--run" ]; then
  [ "$(uname -s 2>/dev/null)" = Linux ] || exit 0
  cat > /dev/null 2>&1
  if command -v setsid > /dev/null 2>&1; then
    CLAUDE_PLUGIN_ROOT="$ROOT" setsid sh "$0" --run > /dev/null 2>&1 < /dev/null &
  else
    CLAUDE_PLUGIN_ROOT="$ROOT" nohup sh "$0" --run > /dev/null 2>&1 < /dev/null &
  fi
  exit 0
fi

note() { printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >> "$OUT"; }
: > "$OUT"
note "relay.sh ran on host $(hostname 2>/dev/null), $(uname -sm), CLAUDE_CODE_REMOTE ${CLAUDE_CODE_REMOTE-not set}"

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) note "unsupported architecture $(uname -m); stopping"; exit 0 ;;
esac
asset="cogmer_0.10.5_linux_${arch}"
url="https://github.com/Blue-Rocket/cogmer/releases/download/v0.10.5/${asset}"
want="$(grep -E "[[:space:]]${asset}\$" "$ROOT/checksums.txt" 2>/dev/null | awk '{print $1}' | head -1)"
[ -n "$want" ] || { note "no pinned checksum for $asset; stopping"; exit 0; }

tmp="$(mktemp -d)" || { note "no temporary directory; stopping"; exit 0; }
bin="$tmp/cogmer"
if command -v curl > /dev/null 2>&1; then
  curl -fsSL --max-time 120 -o "$bin" "$url" 2> "$tmp/dl.err"
elif command -v wget > /dev/null 2>&1; then
  wget -q -T 120 -O "$bin" "$url" 2> "$tmp/dl.err"
else
  note "neither curl nor wget; stopping"; rm -rf "$tmp"; exit 0
fi
if [ ! -s "$bin" ]; then
  note "download failed: $(head -c 300 "$tmp/dl.err" 2>/dev/null | tr '\n' ' ')"; rm -rf "$tmp"; exit 0
fi
got="$(sha256sum "$bin" 2>/dev/null | awk '{print $1}')"
[ -n "$got" ] || got="$(shasum -a 256 "$bin" 2>/dev/null | awk '{print $1}')"
if [ "$got" != "$want" ]; then
  note "checksum mismatch for $asset (got $got); the download was deleted and not run"
  rm -rf "$tmp"; exit 0
fi
chmod +x "$bin"
note "downloaded $asset and its sha256 matches the pin"

home="$(mktemp -d)"
export COGMER_HOME="$home" COGMER_ADDR=127.0.0.1:4892 COGMER_PEER_ADDR=127.0.0.1:4893 COGMER_HOST=claude COGMER_PREFLIGHT=off
"$bin" daemon > "$home/daemon.out" 2>&1 < /dev/null &
dpid=$!

healthy() {
  if command -v curl > /dev/null 2>&1; then curl -s -m 1 "http://$COGMER_ADDR/healthz" > /dev/null 2>&1
  elif command -v wget > /dev/null 2>&1; then wget -q -T 1 -O /dev/null "http://$COGMER_ADDR/healthz" 2>/dev/null
  else return 1; fi
}
waited=0
while [ "$waited" -lt 60 ]; do
  healthy && break
  sleep 1; waited=$((waited + 1))
done
if healthy; then note "the daemon answered on $COGMER_ADDR after ${waited}s"; else note "the daemon did not answer within 60s"; fi

"$bin" whoami > "$home/whoami.txt" 2>&1 < /dev/null
note "daemon log lines that name the relay, a WebSocket or an address:"
grep -i -E 'relay|websocket|reachable' "$home/daemon.out" 2>/dev/null | cut -c1-300 | sed 's/^/    /' >> "$OUT"
note "whoami lines that begin 'Checked just now' or say why there is no string to send:"
grep -E '^(Checked just now|Nothing was done yet|No string|The cogmer daemon is not|.*could not be reached|.*not attached)' "$home/whoami.txt" 2>/dev/null | cut -c1-300 | sed 's/^/    /' >> "$OUT"

"$bin" stop > /dev/null 2>&1
sleep 2
kill "$dpid" 2>/dev/null
note "stopped the daemon"
rm -rf "$tmp" "$home"
exit 0
