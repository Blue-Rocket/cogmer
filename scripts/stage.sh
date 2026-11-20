#!/usr/bin/env bash
# Publish a release commit to the staging repository, so that the release can be
# installed and tested exactly as a user would install it before it reaches anybody.
#
# Usage: scripts/stage.sh [commit]           (default HEAD)
#        scripts/stage.sh --commit-only [commit]
#
# Run after scripts/release.sh and before scripts/publish.sh, from the release commit,
# with dist/ built for it. It makes one commit on top of the release commit that
# changes plugin/release-url.txt to the staging repository's releases and nothing else,
# pushes that commit as the staging repository's main, and publishes dist/ there with
# scripts/publish.sh, which reads where to publish from that commit. A plugin installed
# from the staging repository therefore downloads from it, and one installed from
# Blue-Rocket/cogmer never does (D-292). --commit-only makes the commit, prints it, and
# pushes and publishes nothing.
#
# The staging repository's main is replaced each time, since every staging commit sits
# on a different release commit. A candidate rebuilt after a failed test takes the next
# version: a plugin updates only when its version moves, and the installer replaces a
# program only with a newer one, so a rebuild under the same version never reaches a
# machine that installed the first. scripts/publish.sh refuses different binaries under
# a tag that exists.
set -euo pipefail
cd "$(dirname "$0")/.."

staging="Blue-Rocket/cogmer-staging"
only=""
[ "${1:-}" = "--commit-only" ] && { only=1; shift; }
release="$(git rev-parse --verify "${1:-HEAD}^{commit}")"

url="https://github.com/${staging}/releases/download"
blob="$(git show "$release:plugin/release-url.txt" \
  | sed -E "s#^https://[^[:space:]]+\$#${url}#" | git hash-object -w --stdin)"
index="$(mktemp)"
trap 'rm -f "$index"' EXIT
GIT_INDEX_FILE="$index" git read-tree "$release"
GIT_INDEX_FILE="$index" git update-index --cacheinfo "100644,${blob},plugin/release-url.txt"
tree="$(GIT_INDEX_FILE="$index" git write-tree)"
commit="$(git commit-tree "$tree" -p "$release" \
  -m "Stage $(git rev-parse --short "$release") from ${staging}'s releases")"

if [ -n "$only" ]; then echo "$commit"; exit 0; fi

read -r repo _ <<< "$(scripts/publish.sh --where "$commit")"
[ "$repo" = "$staging" ] || { echo "the staging commit names $repo, not $staging" >&2; exit 1; }
git push --force "git@github.com:${staging}.git" "${commit}:refs/heads/main"
scripts/publish.sh "$commit"
echo "staged $(git rev-parse --short "$release") as ${staging} $(git rev-parse --short "$commit")"
