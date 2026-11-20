#!/usr/bin/env bash
# Publish the built artifacts as the GitHub release the installer downloads from.
#
# Usage: scripts/publish.sh [commit]     (default HEAD)
#        scripts/publish.sh --where [commit]
#
# Separate from release.sh on purpose. Building and hashing are host-agnostic and
# will outlive any particular host; putting the files somewhere is not. This script
# is the one that changes when the host does, and release.sh is the one that does
# not -- which is what made replacing the pre-GA droplet a change to this file and
# release-url.txt alone (D-121).
#
# Everything it publishes is read from the commit named: the version, the hashes, and
# the address in plugin/release-url.txt, whose repository is where the release is made.
# A release commit names Blue-Rocket/cogmer, and the commit scripts/stage.sh makes on
# top of it names the staging repository, so one script publishes both (D-292).
# --where prints the repository and tag and publishes nothing.
#
# The release is created on the tag, and the tag must name a commit whose
# checksums.txt pins these assets, because the plugin at that tag is what authorizes
# them. GitHub creates a missing tag on its default branch when the commit named is
# not one it holds, and says nothing, so this checks before it publishes rather than
# trusting the call.
set -euo pipefail
cd "$(dirname "$0")/.."

where=""
[ "${1:-}" = "--where" ] && { where=1; shift; }
commit="$(git rev-parse --verify "${1:-HEAD}^{commit}")"

version="$(git show "$commit:plugin/VERSION" | tr -d '[:space:]')"
tag="v${version}"
base="$(git show "$commit:plugin/release-url.txt" | grep -v '^#' | head -1 | tr -d '[:space:]')"
repo="$(printf '%s\n' "$base" | sed -nE 's#^https://github\.com/([^/]+/[^/]+)/releases/download$#\1#p')"
[ -n "$repo" ] || { echo "plugin/release-url.txt at $commit names $base, which is not a GitHub release address" >&2; exit 1; }
if [ -n "$where" ]; then echo "$repo $tag"; exit 0; fi
checksums="$(git show "$commit:plugin/checksums.txt")"

command -v gh > /dev/null 2>&1 || { echo "gh is required to publish" >&2; exit 1; }
[ -d dist ] || { echo "no dist/ -- run scripts/release.sh $version first" >&2; exit 1; }

# The assets must be the ones checksums.txt pins, and the cheapest place to find out
# otherwise is here rather than on somebody's machine at install time.
echo "checking dist/ against plugin/checksums.txt:"
while read -r sum name; do
  case "$sum" in \#*|"") continue ;; esac
  got="$(shasum -a 256 "dist/$name" 2>/dev/null | awk '{print $1}')"
  [ "$got" = "$sum" ] || { echo "  $name does not match its pin; rerun release.sh $version" >&2; exit 1; }
done <<< "$checksums"
echo "  all five match"

if gh release view "$tag" -R "$repo" > /dev/null 2>&1; then
  # Uploading to an existing release puts these assets under whatever commit its tag
  # names, so that commit's pins must be these.
  tagged="$(git ls-remote "https://github.com/${repo}.git" "refs/tags/${tag}^{}" | awk '{print $1}')"
  [ -n "$tagged" ] || tagged="$(git ls-remote "https://github.com/${repo}.git" "refs/tags/${tag}" | awk '{print $1}')"
  if [ -z "$tagged" ] || ! git cat-file -e "${tagged}^{commit}" 2>/dev/null \
    || ! git show "${tagged}:plugin/checksums.txt" 2>/dev/null | cmp -s - <(printf '%s\n' "$checksums"); then
    echo "release $tag exists, but its tag names ${tagged:-nothing}, whose plugin/checksums.txt" >&2
    echo "does not pin dist/. Move the tag to the commit that does before uploading." >&2
    exit 1
  fi
  echo "release $tag exists at $(git rev-parse --short "$tagged"); uploading assets"
  gh release upload "$tag" dist/* --clobber -R "$repo"
else
  # The tag is created on GitHub, from a commit GitHub must already hold.
  if ! gh api "repos/${repo}/commits/${commit}" --silent > /dev/null 2>&1; then
    echo "$(git rev-parse --short "$commit") is not in ${repo} on GitHub, so the tag would be" >&2
    echo "created on the default branch instead. Push it there first." >&2
    exit 1
  fi
  echo "creating release $tag in ${repo} from $(git rev-parse --short "$commit")"
  gh release create "$tag" dist/* -R "$repo" \
    --target "$commit" \
    --title "$tag" \
    --notes "Binaries for $tag. Each is authorized by the sha256 pinned in plugin/checksums.txt at this tag; a download that does not match is deleted rather than run."
fi

# Verify what is actually served, against the same hashes the installer will check.
# A publish that succeeded and served something else is the one failure a person
# would not think to look for.
echo "verifying what GitHub is serving:"
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
done <<< "$checksums"
[ "$fail" -eq 0 ] || { echo "the release is not serving what the plugin will accept" >&2; exit 1; }
echo "published and verified $tag"
