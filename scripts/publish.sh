#!/usr/bin/env bash
# Publish the built artifacts as the GitHub release the installer downloads from.
#
# Separate from release.sh on purpose. Building and hashing are host-agnostic and
# will outlive any particular host; putting the files somewhere is not. This script
# is the one that changes when the host does, and release.sh is the one that does
# not -- which is what made replacing the pre-GA droplet a change to this file and
# release-url.txt alone (D-121).
#
# The release is created on the tag, and the tag must name a commit whose
# checksums.txt pins these assets, because the plugin at that tag is what authorizes
# them. GitHub creates a missing tag on its default branch when the commit named is
# not one it holds, and says nothing, so this checks before it publishes rather than
# trusting the call.
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

git fetch --quiet origin
head="$(git rev-parse HEAD)"

if gh release view "$tag" > /dev/null 2>&1; then
  # Uploading to an existing release puts these assets under whatever commit its tag
  # names, so that commit's pins must be these.
  tagged="$(git ls-remote origin "refs/tags/${tag}^{}" | awk '{print $1}')"
  [ -n "$tagged" ] || tagged="$(git ls-remote origin "refs/tags/${tag}" | awk '{print $1}')"
  if [ -z "$tagged" ] || ! git cat-file -e "${tagged}^{commit}" 2>/dev/null \
    || ! git show "${tagged}:plugin/checksums.txt" 2>/dev/null | cmp -s - plugin/checksums.txt; then
    echo "release $tag exists, but its tag names ${tagged:-nothing}, whose plugin/checksums.txt" >&2
    echo "does not pin dist/. Move the tag to the commit that does before uploading." >&2
    exit 1
  fi
  echo "release $tag exists at $(git rev-parse --short "$tagged"); uploading assets"
  gh release upload "$tag" dist/* --clobber
else
  # The tag is created on GitHub, from a commit GitHub must already hold.
  if [ -z "$(git branch -r --contains "$head" 2>/dev/null)" ]; then
    echo "$(git rev-parse --short HEAD) is not on GitHub, so the tag would be created on" >&2
    echo "the default branch instead. Push it first: git push origin HEAD" >&2
    exit 1
  fi
  echo "creating release $tag from $(git rev-parse --short HEAD)"
  gh release create "$tag" dist/* \
    --target "$head" \
    --title "$tag" \
    --notes "Binaries for $tag. Each is authorized by the sha256 pinned in plugin/checksums.txt at this tag; a download that does not match is deleted rather than run."
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
