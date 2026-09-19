#!/usr/bin/env bash
# Start the daemon if it is not already running.
#
# Three rules from §29, each easy to get wrong:
#
#   Starting must not delay the session. Nothing here waits for the daemon to be
#   ready; a session that begins first simply has nothing to inject yet.
#
#   Already running is the ordinary outcome. Several sessions begin at once on one
#   machine routinely: each tries, at most one wins, and none reports a problem.
#
#   Failure is silent to the developer. The daemon records its own troubles in its
#   log; the session says nothing about any of it.
. "$(dirname "$0")/find-binary.sh"

# Obtain or update the binary, detached, so nothing here waits on a download.
# It decides for itself whether there is anything to do, and does nothing in the
# overwhelmingly common case where the right version is already installed.
if command -v setsid > /dev/null 2>&1; then
  setsid nohup bash "$(dirname "$0")/install.sh" > /dev/null 2>&1 < /dev/null &
else
  nohup bash "$(dirname "$0")/install.sh" > /dev/null 2>&1 < /dev/null &
fi
disown 2>/dev/null

bin="$(claude_team_binary)" || exit 0

addr="${CLAUDE_TEAM_ADDR:-127.0.0.1:4782}"
if curl -s -m 1 "http://${addr}/healthz" > /dev/null 2>&1; then
  exit 0   # already running, which is not an error
fi

log="$HOME/.claude-team/daemon.log"
mkdir -p "$(dirname "$log")" 2>/dev/null

# Detached, so the daemon outlives the session that started it -- a room may have
# members in several sessions, and starting it repeatedly is worse than leaving it.
if command -v setsid > /dev/null 2>&1; then
  setsid nohup "$bin" daemon >> "$log" 2>&1 < /dev/null &
else
  nohup "$bin" daemon >> "$log" 2>&1 < /dev/null &
fi
disown 2>/dev/null

exit 0
