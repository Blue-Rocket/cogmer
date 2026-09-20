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
exit 0
