#!/usr/bin/env bash
# Builds the static page that reads a cloud session's room into <output directory> (D-302).
#
# The page is the view the daemon serves, cmd/cogmer/ui.html, with web/view/viewer.js added at the
# place the view marks, and tailcat's browser build, which is built here from the version go.mod
# pins. Nothing in it is secret: the link a command prints carries the address and the token.
set -euo pipefail

out=${1:?usage: scripts/build-view.sh <output directory>}
root=$(cd "$(dirname "$0")/.." && pwd)

tailcat=$(cd "$root" && go list -m -f '{{.Dir}}' github.com/tailscale/tailcat)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# The module cache is read-only, and the build writes beside the source.
cp -R "$tailcat" "$work/tailcat"
chmod -R u+w "$work/tailcat"
(cd "$work/tailcat" && GOFLAGS=-mod=mod go run ./cmd/tailcat-webdist -o "$work/dist" -web-dir web)

mkdir -p "$out"
cp "$work/dist/main.wasm" "$work/dist/wasm_exec.js" "$out/"
cp "$root/web/view/viewer.js" "$out/"

marker='<!-- cogmer:view-hooks -->'
if [ "$(grep -cF -- "$marker" "$root/cmd/cogmer/ui.html")" != 1 ]; then
  echo "ui.html does not hold the marker exactly once" >&2
  exit 1
fi
policy=$(cat "$root/web/view/policy.txt")
awk -v marker="$marker" -v policy="$policy" '
  $0 == marker {
    print "<meta http-equiv=\"Content-Security-Policy\" content=\"" policy "\">"
    print "<meta name=\"referrer\" content=\"no-referrer\">"
    print "<script src=\"wasm_exec.js\"></script>"
    print "<script src=\"viewer.js\"></script>"
    next
  }
  { print }
' "$root/cmd/cogmer/ui.html" > "$out/index.html"
echo "built the page in $out ($(du -h "$out/main.wasm" | cut -f1) of WebAssembly)"
