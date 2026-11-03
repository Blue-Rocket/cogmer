#!/bin/sh
# Builds the reporter's zip in the folder given as the first argument, which must exist.
set -e
here="$(cd "$(dirname "$0")" && pwd)"
out="${1:?usage: build.sh <existing folder for the zip>}"
stage="$out/reporter-stage"
rm -rf "$stage" && mkdir -p "$stage"
cp -R "$here/.claude-plugin" "$here/hooks" "$stage/"
chmod 755 "$stage/hooks/report.sh"
(cd "$stage" && COPYFILE_DISABLE=1 zip -qr "$out/cogmer-synced-reporter.zip" .claude-plugin hooks)
echo "zip: $out/cogmer-synced-reporter.zip ($(wc -c < "$out/cogmer-synced-reporter.zip" | tr -d ' ') bytes)"
