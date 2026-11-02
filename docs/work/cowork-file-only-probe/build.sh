#!/bin/sh
# Builds the zips of the file-only probe in the folder given as the first argument, which must exist.
# cogmer-file-probe.zip holds .claude-plugin/plugin.json and cloud.value and nothing else, no hook and
# no command. cogmer-file-probe-with-command.zip also holds commands/about.md, one command file, for the
# case that the upload refuses a plugin with no component. cloud.value is in the form of a cloud value
# (the prefix cogmer-cloud-v1: and the base64url of a JSON object with the fields of a cloud value),
# made of random bytes of the lengths real keys, signatures and relay addresses have, with no real key,
# with mode 600. The same file is in both zips. The script prints its length and the first eight hex
# digits of its sha256, which a session's report is compared with.
set -e
here="$(cd "$(dirname "$0")" && pwd)"
out="${1:?usage: build.sh <existing empty folder for the zips and their staging copies>}"
rm -rf "$out/a" "$out/b" && mkdir -p "$out/a" "$out/b"
python3 - "$out/a/cloud.value" <<'PY'
import base64, json, secrets, sys
def b64(n):
    return base64.urlsafe_b64encode(secrets.token_bytes(n)).decode().rstrip("=")
def key():
    return "ed25519:" + b64(32)
def endpoint():
    return key() + "@tc://tco2" + b64(200) + "#fake"
value = {
    "key": b64(32),
    "shared": {"scheme": 1, "user": "fake-user", "version": 1, "sharedKey": key(), "signature": b64(64)},
    "cloud": {"scheme": 1, "user": "fake-user", "sharedVersion": 1, "machine": key(), "kind": "cloud",
              "name": "fake-cloud", "serial": 1, "signature": b64(64)},
    "displayName": "fake",
    "machines": [{"machine": key(), "name": n, "endpoint": endpoint()} for n in ("laptop", "desktop", "server")],
}
raw = json.dumps(value, separators=(",", ":")).encode()
open(sys.argv[1], "w").write("cogmer-cloud-v1:" + base64.urlsafe_b64encode(raw).decode().rstrip("="))
PY
chmod 600 "$out/a/cloud.value"
cp "$out/a/cloud.value" "$out/b/cloud.value"
cp -R "$here/.claude-plugin" "$out/a/"
cp -R "$here/.claude-plugin" "$here/commands" "$out/b/"
(cd "$out/a" && COPYFILE_DISABLE=1 zip -qr "$out/cogmer-file-probe.zip" .claude-plugin cloud.value)
(cd "$out/b" && COPYFILE_DISABLE=1 zip -qr "$out/cogmer-file-probe-with-command.zip" .claude-plugin commands cloud.value)
echo "zips: $out/cogmer-file-probe.zip and $out/cogmer-file-probe-with-command.zip"
echo "cloud.value: $(wc -c < "$out/a/cloud.value" | tr -d ' ') bytes, sha256 starts $(shasum -a 256 "$out/a/cloud.value" | cut -c1-8)"
