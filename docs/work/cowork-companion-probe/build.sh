#!/bin/sh
# Builds the probe's zip in the folder given as the first argument, with the plugin version given as
# the second when there is one, and with defaultEnabled set to false in the manifest when the third is
# "off". It copies the plugin's files,
# adds cloud.value, a file of random base64url text of 2400 characters, which is about the size of a
# cloud environment's value and holds no secret, with mode 600, and zips the result. It prints the
# first eight hex digits of the file's sha256, which the probe's report shows, to compare with.
set -e
here="$(cd "$(dirname "$0")" && pwd)"
out="${1:?usage: build.sh <existing empty folder for the zip and its staging copy>}"
stage="$out/stage"
rm -rf "$stage" && mkdir -p "$stage"
cp -R "$here/.claude-plugin" "$here/hooks" "$here/commands" "$stage/"
if [ -n "${2:-}" ]; then
  sed "s/\"version\": \"[^\"]*\"/\"version\": \"$2\"/" "$here/.claude-plugin/plugin.json" > "$stage/.claude-plugin/plugin.json"
fi
if [ "${3:-}" = "off" ]; then
  python3 - "$stage/.claude-plugin/plugin.json" <<'PY'
import json, sys
p = sys.argv[1]
m = json.load(open(p))
m["defaultEnabled"] = False
json.dump(m, open(p, "w"), indent=2)
PY
fi
python3 -c "import secrets; print(secrets.token_urlsafe(1800)[:2400], end='')" > "$stage/cloud.value"
chmod 600 "$stage/cloud.value"
(cd "$stage" && COPYFILE_DISABLE=1 zip -qr "$out/cogmer-companion-probe.zip" .claude-plugin hooks commands cloud.value)
echo "zip: $out/cogmer-companion-probe.zip"
echo "cloud.value: $(wc -c < "$stage/cloud.value" | tr -d ' ') bytes, sha256 starts $(shasum -a 256 "$stage/cloud.value" | cut -c1-8)"
