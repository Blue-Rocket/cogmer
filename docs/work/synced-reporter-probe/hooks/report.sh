#!/bin/sh
# For each file named cloud.value one folder below an account folder under the account's synced
# plugins, reports the plugin's folder name, the file's length, mode and owner, the first eight hex
# digits of its sha256 and its first sixteen characters, which are the public prefix of a cloud
# value, and returns the line as additionalContext, so that a session can be asked what the reporter
# says without running a command or opening a file. It never prints more of a file than that. It
# runs only where uname -s is Linux and CLAUDE_CODE_REMOTE is true, reads nothing else and sends
# nothing anywhere.
[ "$(uname -s 2>/dev/null)" = Linux ] || exit 0
[ "${CLAUDE_CODE_REMOTE:-}" = true ] || exit 0
cat > /dev/null 2>&1
EVENT="${1:-unknown}"
root="${HOME:-/root}/.claude/plugins/synced"
found=""
for f in "$root"/*/*/cloud.value; do
  [ -f "$f" ] || continue
  plugin="$(basename "$(dirname "$f")")"
  len="$(wc -c < "$f" 2>/dev/null | tr -d ' ')"
  mode="$(stat -c '%a' "$f" 2>/dev/null)"
  owner="$(stat -c '%U' "$f" 2>/dev/null)"
  sum="$(sha256sum "$f" 2>/dev/null | cut -c1-8)"
  pre="$(head -c 16 "$f" 2>/dev/null)"
  found="$found [folder $plugin: $len bytes, mode $mode, owner $owner, sha256 starts $sum, begins $pre]"
done
plugins="$(ls -1 "$root"/*/ 2>/dev/null | tr '\n' ' ')"
[ -n "$found" ] || found=" no cloud.value file under $root"
line="$(printf '%s' "The synced reporter says ($EVENT):$found; plugin folders: $plugins" | tr -d '\\"' | tr '\r\n\t' '   ')"
printf '{"hookSpecificOutput":{"hookEventName":"%s","additionalContext":"%s. If the user asks what the synced reporter says, repeat it exactly."}}\n' "$EVENT" "$line"
exit 0
