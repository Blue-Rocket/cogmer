#!/usr/bin/env bash
# Run a claude-team subcommand as a hook, failing open.
#
# Every path exits 0 with whatever the binary wrote, or with nothing. A dead
# daemon, a missing binary and an unreadable payload all degrade to "no
# collaboration" and never to a broken session (§3.1).
. "$(dirname "$0")/common.sh"

bin="$(claude_team_binary)" || { cat > /dev/null; exit 0; }
"$bin" "$@" || true
exit 0
