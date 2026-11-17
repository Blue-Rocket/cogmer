#!/bin/sh
# Builds the probe's zip in the folder given as the first argument, with the plugin version given as
# the second when there is one, and with defaultEnabled set to false in the manifest when the third is
# "off". The fourth, "real-shape", makes cloud.value in the form of a cloud value: the prefix
# cogmer-cloud-v1: and the base64url of a JSON object with the fields of cloudValueWire, holding
# random bytes of the lengths real keys, signatures and relay addresses have, and no real key.
# It copies the plugin's files,
# adds cloud.value, by default a file of random base64url text of 2400 characters, which is about the
# size of a cloud environment's value and holds no secret, with mode 600, and zips the result. It prints
# the first eight hex digits of the file's sha256, which the probe's report shows, to compare with.
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
if [ "${4:-}" = "real-shape" ]; then
  python3 - "$stage/cloud.value" <<'PY'
import base64, json, secrets, sys
def b64(n):
    return base64.urlsafe_b64encode(secrets.token_bytes(n)).decode().rstrip("=")
def key():
    return "ed25519:" + b64(32)
def endpoint():
    return key() + "@tc://tco2" + b64(200) + "#fake"
machines = [{"machine": key(), "name": n, "endpoint": endpoint()} for n in ("laptop", "desktop", "server")]
value = {
    "key": b64(32),
    "shared": {"scheme": 1, "user": "fake-user", "version": 1, "sharedKey": key(), "signature": b64(64)},
    "cloud": {"scheme": 1, "user": "fake-user", "sharedVersion": 1, "machine": key(), "kind": "cloud",
              "name": "fake-cloud", "serial": 1, "signature": b64(64)},
    "displayName": "fake",
    "machines": machines,
}
raw = json.dumps(value, separators=(",", ":")).encode()
open(sys.argv[1], "w").write("cogmer-cloud-v1:" + base64.urlsafe_b64encode(raw).decode().rstrip("="))
PY
else
  python3 -c "import secrets; print(secrets.token_urlsafe(1800)[:2400], end='')" > "$stage/cloud.value"
fi
chmod 600 "$stage/cloud.value"
(cd "$stage" && COPYFILE_DISABLE=1 zip -qr "$out/cogmer-companion-probe.zip" .claude-plugin hooks commands cloud.value)
echo "zip: $out/cogmer-companion-probe.zip"
echo "cloud.value: $(wc -c < "$stage/cloud.value" | tr -d ' ') bytes, sha256 starts $(shasum -a 256 "$stage/cloud.value" | cut -c1-8)"
