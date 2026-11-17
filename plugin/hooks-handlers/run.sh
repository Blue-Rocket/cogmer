#!/usr/bin/env bash
# Run a cogmer subcommand as a hook, failing open.
#
# Every path exits 0 with whatever the binary wrote, or with nothing. A dead
# daemon, a missing binary and an unreadable payload all degrade to "no
# collaboration" and never to a broken session (§3.1).
. "$(dirname "$0")/common.sh"

# A session in no room ends here, before the program is found or started, so its
# prompts and responses reach nothing of cogmer's (D-284).
payload="$(cat)"
if hook_for_session_in_no_room "$payload" "$@"; then
  exit 0
fi

bin="$(cogmer_binary)" || exit 0
printf '%s' "$payload" | "$bin" "$@" || true
exit 0
