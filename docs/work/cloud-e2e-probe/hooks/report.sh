#!/bin/sh
# Returns, as additionalContext, what the bundled daemon did: whether it answers, how many cloud.value
# files the account's synced plugins hold, and the daemon's log lines about a cloud session and about
# the user's lists. The log holds names and times and never a key. It runs only in a cloud session,
# reads nothing else and sends nothing anywhere.
[ "$(uname -s 2>/dev/null)" = Linux ] || exit 0
[ "${CLAUDE_CODE_REMOTE:-}" = true ] || exit 0
cat > /dev/null 2>&1
D=/tmp/cogmer-e2e
sleep 4
files="$(find "${HOME:-/root}/.claude/plugins/synced" -name cloud.value 2>/dev/null | wc -l | tr -d ' ')"
health="$(curl -s -m 2 http://127.0.0.1:47840/healthz 2>/dev/null | tr -d '\n"{}\\')"
lines="$(grep -E 'cloud:|lists|session key|certified' "$D/daemon.log" 2>/dev/null | tail -n 8 | tr -d '"\\' | tr '\n' '|' | cut -c1-1500)"
msg="The cloud e2e probe says: cloud.value files found $files; daemon health ${health:-none}; log: ${lines:-none}"
printf '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"%s If the user asks what the cloud e2e probe says, repeat it exactly."}}\n' "$msg"
exit 0
