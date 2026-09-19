#!/usr/bin/env bash
# Locate the claude-team binary, printing its path on stdout.
#
# Nothing here may fail loudly. §3.1: a hook must never break Claude Code, so a
# missing binary means no collaboration rather than a broken session.
#
# Searched in the order a developer would expect to win: an explicit override, a
# binary they installed themselves, the plugin's own copy, then PATH.
claude_team_binary() {
  if [ -n "$CLAUDE_TEAM_BIN" ] && [ -x "$CLAUDE_TEAM_BIN" ]; then
    printf '%s' "$CLAUDE_TEAM_BIN"; return 0
  fi
  for candidate in \
    "$HOME/.claude-team/bin/claude-team" \
    "${CLAUDE_PLUGIN_ROOT:-}/bin/claude-team"
  do
    if [ -x "$candidate" ]; then printf '%s' "$candidate"; return 0; fi
  done
  if command -v claude-team > /dev/null 2>&1; then
    command -v claude-team; return 0
  fi
  return 1
}
