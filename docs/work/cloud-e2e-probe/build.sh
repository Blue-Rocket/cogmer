#!/bin/sh
# Builds the starter plugin's zip in the folder given as the first argument, which must exist, from
# the Linux build of cogmer given as the second, which it adds at the plugin's root with the execute
# bit. The binary is not in the repository: build it with
#   GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags ts_debug_websockets -ldflags "-s -w" -o <path> ./cmd/cogmer
set -e
here="$(cd "$(dirname "$0")" && pwd)"
out="${1:?usage: build.sh <existing folder> <linux cogmer binary>}"
bin="${2:?usage: build.sh <existing folder> <linux cogmer binary>}"
stage="$out/stage"
rm -rf "$stage" && mkdir -p "$stage"
cp -R "$here/.claude-plugin" "$here/hooks" "$stage/"
cp "$bin" "$stage/cogmer" && chmod 755 "$stage/cogmer" "$stage/hooks/start.sh" "$stage/hooks/report.sh"
(cd "$stage" && COPYFILE_DISABLE=1 zip -qr "$out/cogmer-e2e-starter.zip" .claude-plugin hooks cogmer)
echo "zip: $out/cogmer-e2e-starter.zip ($(wc -c < "$out/cogmer-e2e-starter.zip" | tr -d ' ') bytes)"
