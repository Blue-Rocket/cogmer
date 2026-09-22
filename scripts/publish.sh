#!/usr/bin/env bash
# Publish the built artifacts as the GitHub release the installer downloads from.
#
# Separate from release.sh on purpose. Building and hashing are host-agnostic and
# will outlive any particular host; putting the files somewhere is not. This script
# is the one that changes when the host does, and release.sh is the one that does
# not -- which is what made replacing the pre-GA droplet a change to this file and
# release-url.txt alone (D-121).
#
# The release is created on the tag, so the tag and the assets cannot disagree: the
# same call that publishes the binaries is the one that names the commit they were
# built from.
set -euo pipefail
cd "$(dirname "$0")/.."

version="$(cat plugin/VERSION)"
tag="v${version}"

command -v gh > /dev/null 2>&1 || { echo "gh is required to publish" >&2; exit 1; }
[ -d dist ] || { echo "no dist/ -- run scripts/release.sh $version first" >&2; exit 1; }

# The assets must be the ones checksums.txt pins, and the cheapest place to find out
# otherwise is here rather than on somebody's machine at install time.
echo "checking dist/ against plugin/checksums.txt:"
while read -r sum name; do
  case "$sum" in \#*|"") continue ;; esac
  got="$(shasum -a 256 "dist/$name" 2>/dev/null | awk '{print $1}')"
  [ "$got" = "$sum" ] || { echo "  $name does not match its pin; rerun release.sh $version" >&2; exit 1; }
done < plugin/checksums.txt
echo "  all five match"

if gh release view "$tag" > /dev/null 2>&1; then
  echo "release $tag exists; uploading assets"
  gh release upload "$tag" dist/* --clobber
else
  echo "creating release $tag from $(git rev-parse --short HEAD)"
  gh release create "$tag" dist/* \
    --title "$tag" \
    --notes "Binaries for $tag. Each is authorised by the sha256 pinned in plugin/checksums.txt at this tag; a download that does not match is deleted rather than run."
fi

# Verify what is actually served, against the same hashes the installer will check.
# A publish that succeeded and served something else is the one failure a person
# would not think to look for.
echo "verifying what GitHub is serving:"
base="$(grep -v '^#' plugin/release-url.txt | head -1 | tr -d '[:space:]')"
fail=0
while read -r sum name; do
  case "$sum" in \#*|"") continue ;; esac
  got="$(curl -fsSL --max-time 120 "${base}/${tag}/${name}" | shasum -a 256 | awk '{print $1}')" || got=""
  if [ "$got" = "$sum" ]; then
    printf '  ok       %s\n' "$name"
  else
    printf '  MISMATCH %s\n           served %s\n           pinned %s\n' "$name" "${got:-nothing}" "$sum"
    fail=1
  fi
done < plugin/checksums.txt
[ "$fail" -eq 0 ] || { echo "the release is not serving what the plugin will accept" >&2; exit 1; }
echo "published and verified $tag"
