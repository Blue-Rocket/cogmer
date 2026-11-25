#!/usr/bin/env bash
# Publishes the static page that reads a cloud session's room to the pages branch of this
# repository's origin, from which GitHub Pages serves it at
# https://blue-rocket.github.io/cogmer/view/ (D-302). It replaces that branch with one commit
# that holds the built page and nothing else. Run it from a clean checkout of the commit
# a release was made from.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
origin=$(git -C "$root" remote get-url origin)
from=$(git -C "$root" rev-parse --short HEAD)
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT

"$root/scripts/build-view.sh" "$dir/view"
touch "$dir/.nojekyll"
cd "$dir"
git init -q
git checkout -q -b pages
git add -A
git -c user.name="${GIT_AUTHOR_NAME:-$(git -C "$root" config user.name)}" \
    -c user.email="${GIT_AUTHOR_EMAIL:-$(git -C "$root" config user.email)}" \
    commit -q -m "Publish the view page built from $from"
git push -q --force "$origin" pages
echo "published the view page built from $from to the pages branch of $origin"
