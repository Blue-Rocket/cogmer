#!/usr/bin/env bash
# Shared by every entry point: the hooks, the installer, and the slash commands.
#
# One resolution rule, in one place. The slash commands used to invoke a bare
# `claude-team`, which resolves only if the person installed it themselves — and
# the installer puts it in ~/.claude-team/bin, which is on nobody's PATH. So every
# command failed for every installed user, while the hooks worked, because only
# the hooks knew where to look. Two rules meant one of them was wrong.

# claude_team_binary prints the path to the binary, or fails.
#
# Searched in the order a person would expect to win: an explicit override, a
# binary they installed themselves, the plugin's own copy, then PATH.
claude_team_binary() {
  if [ -n "${CLAUDE_TEAM_BIN:-}" ] && [ -x "$CLAUDE_TEAM_BIN" ]; then
    printf '%s' "$CLAUDE_TEAM_BIN"; return 0
  fi
  for candidate in \
    "${CLAUDE_TEAM_HOME:-$HOME/.claude-team}/bin/claude-team" \
    "${CLAUDE_PLUGIN_ROOT:-}/bin/claude-team"
  do
    if [ -x "$candidate" ]; then printf '%s' "$candidate"; return 0; fi
  done
  if command -v claude-team > /dev/null 2>&1; then
    command -v claude-team; return 0
  fi
  return 1
}

# start_daemon_if_needed starts the daemon unless it is already answering.
#
# Called from the session-start hook and again from the installer when it has just
# fetched the binary. Without the second call the first session installs and the
# SECOND one starts the daemon, because the hook looked for a binary that was still
# downloading — a two-session warm-up nobody would guess at.
start_daemon_if_needed() {
  local bin addr log
  bin="$(claude_team_binary)" || return 0

  addr="${CLAUDE_TEAM_ADDR:-127.0.0.1:4782}"
  if curl -s -m 1 "http://${addr}/healthz" > /dev/null 2>&1; then
    return 0   # already running, which is the ordinary outcome, not an error
  fi

  log="${CLAUDE_TEAM_HOME:-$HOME/.claude-team}/daemon.log"
  mkdir -p "$(dirname "$log")" 2>/dev/null

  # Detached, so the daemon outlives the session that started it: a room may have
  # members in several sessions, and starting it repeatedly is worse than leaving
  # it running.
  if command -v setsid > /dev/null 2>&1; then
    setsid nohup "$bin" daemon >> "$log" 2>&1 < /dev/null &
  else
    nohup "$bin" daemon >> "$log" 2>&1 < /dev/null &
  fi
  disown 2>/dev/null
  return 0
}

# --- what the install is doing, as a fact rather than an inference ---
#
# Absence of the binary used to mean three different things, and every reader
# guessed the same one: never started, arriving now, or failed an hour ago and
# waiting out a cooldown. Telling a person "it may still be arriving" when nothing
# is arriving sends them to wait for something that will not happen (D-075).
#
# One file, three readers: the installer sets it and honours the cooldown, the
# command wrapper explains it, and the session-start hook passes it to the model so
# an answer to "why isn't this working" is true rather than invented.

install_state_file() { printf '%s' "${CLAUDE_TEAM_HOME:-$HOME/.claude-team}/install-state"; }

# set_install_state <installing|failed> <detail>
set_install_state() {
  local f; f="$(install_state_file)"
  mkdir -p "$(dirname "$f")" 2>/dev/null
  printf '%s\t%s\t%s\n' "$1" "$(date +%s)" "${2:-}" > "$f" 2>/dev/null
}

clear_install_state() { rm -f "$(install_state_file)" 2>/dev/null; }

# install_state prints "<state> <age-seconds> <detail>".
#
# An `installing` mark older than staleAfter is reported as `stalled`: a crash
# between setting the mark and clearing it would otherwise leave the file saying
# a download is in progress forever, which is the one answer worse than silence.
install_state() {
  local f line state when detail age
  f="$(install_state_file)"
  [ -f "$f" ] || { printf 'none 0 '; return 0; }
  IFS=$'\t' read -r state when detail < "$f" || { printf 'none 0 '; return 0; }
  age=$(( $(date +%s) - ${when:-0} ))
  if [ "$state" = installing ] && [ "$age" -gt 300 ]; then state=stalled; fi
  printf '%s %s %s' "$state" "$age" "$detail"
}
