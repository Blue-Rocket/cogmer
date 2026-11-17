#!/usr/bin/env bash
# Run a cogmer subcommand as a hook, failing open.
#
# Every path exits 0 with whatever the binary wrote, or with nothing. A dead
# daemon, a missing binary and an unreadable payload all degrade to "no
# collaboration" and never to a broken session (§3.1).
. "$(dirname "$0")/common.sh"

# A session in no room ends here, before the program is found or started, so its
# prompts and responses reach nothing of cogmer's (D-284).
#
# A first install is announced by the first hook after the installer leaves its mark
# (D-282), which is usually the prompt hook of the session that started the install,
# and that session is in no room. So a Claude Code prompt that finds the mark runs
# `cogmer notice`, which takes it, and is shown the line. The prompt still reaches
# nothing, and once the mark is gone the hook costs one file test.
payload="$(cat)"
if hook_for_session_in_no_room "$payload" "$@"; then
  if [ "$*" = "hook prompt" ] && [ -f "$(install_notice_file)" ] && bin="$(cogmer_binary)"; then
    line="$("$bin" notice 2>/dev/null < /dev/null | head -1)"
    if [ -n "$line" ]; then
      printf '{"systemMessage":"%s"}\n' "$(json_safe "$line")"
    fi
  fi
  exit 0
fi

bin="$(cogmer_binary)" || exit 0
printf '%s' "$payload" | "$bin" "$@" || true
exit 0
