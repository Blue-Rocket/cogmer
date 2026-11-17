#!/bin/sh
# Builds what the host probes in docs/work/host-probes.md install, from the commit checked out:
#   host-observer.zip      this folder's plugin, for Claude Desktop's plugin upload
#   cogmer-plugin.zip      the cogmer plugin folder, for Claude Desktop's plugin upload
#   cogmer                 the cogmer program for one platform, built as release.sh builds it,
#                          with the version in plugin/VERSION, so the plugin's installer keeps it
# Both zips come from the commit, not the working tree, so a run names exactly what it installed.
# Usage: build.sh <existing empty folder> [goos/goarch, default darwin/arm64]
set -e
out="${1:?usage: build.sh <existing empty folder> [goos/goarch]}"
target="${2:-darwin/arm64}"
repo="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
cd "$repo"
if [ -n "$(git status --porcelain -- plugin cmd docs/work/host-observer go.mod go.sum)" ]; then
  echo "uncommitted changes under plugin, cmd or docs/work/host-observer: commit them first, so the run can name its commit" >&2
  exit 1
fi
commit="$(git rev-parse --short HEAD)"
version="$(cat plugin/VERSION)"
git archive --format=zip -o "$out/host-observer.zip" HEAD:docs/work/host-observer
git archive --format=zip -o "$out/cogmer-plugin.zip" HEAD:plugin
CGO_ENABLED=0 GOOS="${target%/*}" GOARCH="${target#*/}" \
  go build -tags ts_debug_websockets -ldflags "-s -w -X main.version=$version" -o "$out/cogmer" ./cmd/cogmer
sum() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1"; else shasum -a 256 "$1"; fi | cut -c1-12; }
echo "commit $commit, cogmer $version for $target"
for f in host-observer.zip cogmer-plugin.zip cogmer; do
  printf '%-18s sha256 %s\n' "$f" "$(sum "$out/$f")"
done
