#!/bin/sh
# Asks which hosts the machine this runs on can reach over HTTPS: the relay's server list, the four
# relay servers, and two controls. It returns at once, runs only where uname -s is Linux, makes each
# request in the background with a time limit, and writes what it found to
# /tmp/cogmer-network-probe.txt. For each host it records curl's exit code, the HTTP status and the
# first 120 bytes of the reply, and for one relay server it also sends a WebSocket upgrade request.
# It sends no data of the user's, downloads nothing to keep, and records the names of proxy
# variables but never their values.
OUT=/tmp/cogmer-network-probe.txt

if [ "${1:-}" != "--run" ]; then
  [ "$(uname -s 2>/dev/null)" = Linux ] || exit 0
  cat > /dev/null 2>&1
  if command -v setsid > /dev/null 2>&1; then
    setsid sh "$0" --run > /dev/null 2>&1 < /dev/null &
  else
    nohup sh "$0" --run > /dev/null 2>&1 < /dev/null &
  fi
  exit 0
fi

tmp="$(mktemp -d)" || exit 0
{
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) net.sh ran on host $(hostname 2>/dev/null), $(uname -sm), CLAUDE_CODE_REMOTE ${CLAUDE_CODE_REMOTE-not set}"
  if command -v curl > /dev/null 2>&1; then echo "curl: $(curl --version 2>/dev/null | head -1)"; else echo "curl: absent"; fi
  echo "proxy variables set (names only): $(env | LC_ALL=C sed -n 's/^\([A-Za-z_]*[Pp][Rr][Oo][Xx][Yy][A-Za-z_]*\)=.*/\1/p' | sort | LC_ALL=C tr '\n' ' ')"
} > "$OUT"
command -v curl > /dev/null 2>&1 || { echo "cannot probe without curl" >> "$OUT"; rm -rf "$tmp"; exit 0; }

# check <label> <url> [extra curl args...]
check() {
  label="$1"; url="$2"; shift 2
  f="$tmp/$label"
  code="$(curl -sS -m 8 -o "$f.body" -w '%{http_code}' "$@" "$url" 2> "$f.err")"
  rc=$?
  {
    printf '%s\n  url %s\n  curl exit %s, http status %s\n' "$label" "$url" "$rc" "$code"
    printf '  curl error: %s\n' "$(head -c 200 "$f.err" 2>/dev/null | LC_ALL=C tr '\n' ' ')"
    printf '  reply starts: %s\n' "$(head -c 120 "$f.body" 2>/dev/null | LC_ALL=C tr '\r\n\t' '   ')"
  } > "$f"
}

check 1-derpmap https://tailcat.dev/derpmap.json &
check 2-relay-nyc https://tc301a.ipn.dev/ &
check 3-relay-sfo https://tc302a.ipn.dev/ &
check 4-relay-fra https://tc303a.ipn.dev/ &
check 5-relay-tok https://tc304a.ipn.dev/ &
check 6-relay-sfo-websocket-upgrade https://tc302a.ipn.dev/derp \
  --http1.1 -H 'Connection: Upgrade' -H 'Upgrade: websocket' \
  -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' -H 'Sec-WebSocket-Version: 13' &
check 7-control-github https://github.com/ &
check 8-control-example https://example.com/ &
wait

for f in "$tmp"/[1-8]-*; do
  case "$f" in *.body|*.err) continue ;; esac
  cat "$f" >> "$OUT"
done
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) done" >> "$OUT"
rm -rf "$tmp"
exit 0
