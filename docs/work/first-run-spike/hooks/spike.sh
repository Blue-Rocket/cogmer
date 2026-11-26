#!/bin/sh
# The first-run spike's hook. For docs/work/cowork-first-run-spikes.md, spikes B, C and D.
#
# In the foreground it answers at once, with a systemMessage and an additionalContext that carry
# the markers FRS-SYS-<event> and FRS-CTX-<event> and a link, so the runbook can record what a
# person sees and what the model is given. At SessionStart it also starts, in the background, the
# records of spikes B and C, written to /tmp/first-run-spike.txt:
#   - whether each userConfig option arrived, with its length and first four characters only,
#     since the runbook has the user type a marker and never a secret;
#   - the names, never the values, of the environment variables that bear on cogmer;
#   - for each place a file might persist, the marker an earlier task left there, then a new one;
#   - for each host, curl's exit code, the HTTP status, and the certificate's subject, issuer and
#     SHA-256 fingerprint as the task sees them, to compare with what the Mac sees, and for the
#     probe site's three names the site's own account of the request.
# It sends nothing of the user's anywhere, and downloads nothing to keep.
EVENT="${1:-unknown}"
OUT=/tmp/first-run-spike.txt
LINK="https://example.com/first-run-spike"

if [ "$EVENT" = "--run" ]; then
  tmp="$(mktemp -d)" || exit 0
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  task="${CLAUDE_CODE_REMOTE_SESSION_ID:-${CLAUDE_SESSION_ID:-unknown}}"
  {
    echo "first-run spike, $now"
    echo "host $(hostname 2>/dev/null), $(uname -sm), CLAUDE_CODE_REMOTE ${CLAUDE_CODE_REMOTE-not set}, CLAUDE_CODE_ENTRYPOINT ${CLAUDE_CODE_ENTRYPOINT-not set}"
    echo "task $task"
    echo
    echo "== B. userConfig options"
    for name in CLAUDE_PLUGIN_OPTION_PROBE_SECRET CLAUDE_PLUGIN_OPTION_PROBE_PLAIN; do
      eval "v=\${$name-}"
      eval "set_=\${$name+set}"
      if [ -n "$set_" ]; then
        echo "$name: set, length ${#v}, starts $(printf '%s' "$v" | cut -c1-4)"
      else
        echo "$name: not set"
      fi
    done
    echo
    echo "== C1. environment variable names (values never recorded)"
    env | LC_ALL=C sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' | LC_ALL=C grep -E 'CLAUDE|COGMER|PROXY|proxy|SSL|CERT|ANTHROPIC|PLUGIN|HOME|NODE_EXTRA' | LC_ALL=C sort | LC_ALL=C tr '\n' ' '
    echo
    echo
    echo "== C2. what persists"
    echo "CLAUDE_PLUGIN_DATA=${CLAUDE_PLUGIN_DATA-not set}  CLAUDE_PROJECT_DIR=${CLAUDE_PROJECT_DIR-not set}  cwd=$(pwd)"
    for dir in "$HOME" /tmp "${CLAUDE_PLUGIN_DATA:-}" "${CLAUDE_PROJECT_DIR:-}" "$HOME/.claude" "$(pwd)"; do
      [ -n "$dir" ] || continue
      f="$dir/.first-run-spike-marker"
      if [ -f "$f" ]; then echo "$dir: found [$(head -c 200 "$f")]"; else echo "$dir: no marker"; fi
      if mkdir -p "$dir" 2>/dev/null && printf '%s task %s\n' "$now" "$task" > "$f" 2>/dev/null; then
        echo "  wrote a new marker"
      else
        echo "  could not write"
      fi
    done
    echo "synced plugin folders (names only): $(ls -d "$HOME"/.claude/plugins/synced/*/* 2>/dev/null | sed "s#$HOME/##" | LC_ALL=C tr '\n' ' ')"
    echo "mounts: $(df -P 2>/dev/null | awk 'NR>1 {print $6}' | LC_ALL=C sort -u | LC_ALL=C tr '\n' ' ')"
    echo
    echo "== C3. reach and certificates"
    if ! command -v curl > /dev/null 2>&1; then echo "curl absent"; fi
  } > "$OUT" 2>&1
  for url in \
    https://tailcat.dev/derpmap.json \
    https://tc301a.ipn.dev/ \
    https://tc302a.ipn.dev/ \
    https://blue-rocket.github.io/ \
    https://github.com/ \
    https://raw.githubusercontent.com/ \
    https://cogmer.ai/ \
    https://probe.cogmer.ai/echo \
    https://a.probe.cogmer.ai/echo \
    https://b.probe.cogmer.ai/echo \
    https://api.anthropic.com/ \
    https://example.com/ \
    https://example.org/
  do
    command -v curl > /dev/null 2>&1 || break
    label="$(printf '%s' "$url" | sed -E 's#https://##; s#/.*##')"
    f="$tmp/$label"
    code="$(curl -sS -v -m 8 -o "$f.body" -w '%{http_code}' "$url" 2> "$f.err")"
    rc=$?
    subj="$(sed -n 's/^\* *subject: //p' "$f.err" | head -1)"
    iss="$(sed -n 's/^\* *issuer: //p' "$f.err" | head -1)"
    fp=""
    if command -v openssl > /dev/null 2>&1; then
      proxy="${HTTPS_PROXY:-${https_proxy:-}}"
      hp="$(printf '%s' "$proxy" | sed -E 's#^[a-z]+://##; s#^[^@]*@##; s#/.*$##')"
      if [ -n "$hp" ]; then popt="-proxy $hp"; else popt=""; fi
      # shellcheck disable=SC2086
      if command -v timeout > /dev/null 2>&1; then lim="timeout 8"; else lim=""; fi
      # shellcheck disable=SC2086
      fp="$(echo | $lim openssl s_client $popt -connect "$label:443" -servername "$label" 2>/dev/null \
            | openssl x509 -noout -fingerprint -sha256 2>/dev/null | sed 's/.*=//')"
    fi
    {
      echo "$label: curl exit $rc, http $code"
      echo "  subject: ${subj:-none}"
      echo "  issuer: ${iss:-none}"
      echo "  sha256: ${fp:-not read}"
      [ "$rc" = 0 ] || echo "  error: $(grep -v '^[*<>{}]' "$f.err" | head -c 160 | LC_ALL=C tr '\n' ' ')"
      # The probe site's /echo says what it saw of this request (docs/work/probe-site/Caddyfile).
      case "$url" in */echo) [ "$rc" = 0 ] && { sed 's/^/  echo: /' "$f.body" | head -12; echo; } ;; esac
    } >> "$OUT"
  done
  echo "done $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$OUT"
  rm -rf "$tmp"
  exit 0
fi

cat > /dev/null 2>&1
if [ "$EVENT" = "SessionStart" ] && [ "$(uname -s 2>/dev/null)" = Linux ]; then
  if command -v setsid > /dev/null 2>&1; then
    setsid sh "$0" --run > /dev/null 2>&1 < /dev/null &
  else
    nohup sh "$0" --run > /dev/null 2>&1 < /dev/null &
  fi
fi
printf '{"systemMessage":"FRS-SYS-%s: the first-run spike ran. Open %s","hookSpecificOutput":{"hookEventName":"%s","additionalContext":"FRS-CTX-%s: the first-run spike added this line. If the user asks what the spike said, quote it, and give them the link %s"}}\n' \
  "$EVENT" "$LINK" "$EVENT" "$EVENT" "$LINK"
