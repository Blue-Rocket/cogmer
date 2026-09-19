#!/usr/bin/env bash
# Publish the built artifacts to the host named in plugin/release-url.txt.
#
# Separate from release.sh on purpose. Building and hashing are host-agnostic and
# will outlive any particular host; putting the files somewhere is not. When this
# moves to a public release host, this script is what gets replaced and
# release.sh is what does not.
set -euo pipefail
cd "$(dirname "$0")/.."

version="$(cat plugin/VERSION)"
host="${CLAUDE_TEAM_PUBLISH_HOST:-root@<droplet>}"
key="${CLAUDE_TEAM_PUBLISH_KEY:-$HOME/.ssh/droplet}"
remote="${CLAUDE_TEAM_PUBLISH_DIR:-/srv/claude-team/claude-team}"

[ -d dist ] || { echo "no dist/ — run scripts/release.sh $version first" >&2; exit 1; }

echo "publishing v${version} to ${host}:${remote}/v${version}"
ssh -n -i "$key" "$host" "mkdir -p ${remote}/v${version}"
scp -q -i "$key" dist/* "${host}:${remote}/v${version}/"

# Verify what landed, against the same hashes the installer will check. A publish
# that succeeded and served something else is the one failure a person would not
# think to look for.
echo "verifying what the host is serving:"
base="$(grep -v '^#' plugin/release-url.txt | head -1 | tr -d '[:space:]')"
fail=0
while read -r sum name; do
  case "$sum" in \#*|"") continue ;; esac
  got="$(curl -fsSL --max-time 120 "${base}/v${version}/${name}" | shasum -a 256 | awk '{print $1}')"
  if [ "$got" = "$sum" ]; then
    printf '  ok       %s\n' "$name"
  else
    printf '  MISMATCH %s\n           served %s\n           pinned %s\n' "$name" "$got" "$sum"
    fail=1
  fi
done < plugin/checksums.txt
[ "$fail" -eq 0 ] || { echo "the host is not serving what the plugin will accept" >&2; exit 1; }
echo "published and verified v${version}"
