#!/usr/bin/env bash
# What the slash commands invoke.
#
# A command cannot say `claude-team` and expect it to resolve: the installer puts
# the binary in ~/.claude-team/bin, which is not on PATH, and §29 forbids modifying
# a shell profile to put it there. So commands go through here, which uses the same
# resolution the hooks use.
#
# Unlike the hook wrapper, this does NOT fail open. A hook that cannot find the
# binary must leave the session alone; a command a person typed must say why
# nothing happened.
set -u
. "$(dirname "$0")/hooks-handlers/common.sh"

if ! bin="$(claude_team_binary)"; then
  echo "claude-team is not installed yet."
  echo
  echo "The plugin fetches it in the background when a session starts, so it may still"
  echo "be arriving — this usually resolves within a minute of the first session."
  echo "If it does not, ~/.claude-team/install.log says what stopped it."
  exit 1
fi

exec "$bin" "$@"
