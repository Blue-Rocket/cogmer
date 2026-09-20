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
. "$(dirname "$0")/common.sh"

# Obtain or update the binary, detached, so nothing here waits on a download.
# It decides for itself whether there is anything to do, and does nothing in the
# overwhelmingly common case where the right version is already installed.
if command -v setsid > /dev/null 2>&1; then
  setsid nohup bash "$(dirname "$0")/install.sh" > /dev/null 2>&1 < /dev/null &
else
  nohup bash "$(dirname "$0")/install.sh" > /dev/null 2>&1 < /dev/null &
fi
disown 2>/dev/null

# Starting the daemon and fetching the binary are both attempted, because on a
# first session the fetch has not finished and there is nothing to start yet. The
# installer starts it when the download lands, so the two together cover both the
# first session and every one after.
start_daemon_if_needed

# If there is still no binary, tell the MODEL what is happening, because there is
# no other channel: everything that reaches a person arrives by way of what the
# model says (D-033, D-036). This is the difference between a session that answers
# "it is still installing" and one that invents a reason.
#
# Emitted only while the binary is absent, so an ordinary session carries nothing.
if ! claude_team_binary > /dev/null 2>&1; then
  read -r state age detail <<< "$(install_state)"
  case "$state" in
    installing) note="claude-team is downloading in the background (started ${age}s ago). Room commands will not work until it finishes, which is usually seconds." ;;
    failed)     note="claude-team failed to install: ${detail}. It retries about an hour after a failure. Room commands will not work until it succeeds." ;;
    stalled)    note="A claude-team install started ${age}s ago and did not finish. Starting a new session will try again." ;;
    *)          note="claude-team is installed as a plugin but its binary has not been fetched yet. It is fetched in the background when a session starts." ;;
  esac
  printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s Do not speculate about other causes; this is the reason."}}\n' "$note"
fi

exit 0
