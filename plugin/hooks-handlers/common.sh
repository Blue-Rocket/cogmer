#!/usr/bin/env bash
# Shared by every entry point: the hooks, the installer, and the slash commands.
#
# One resolution rule, in one place. The slash commands used to invoke a bare
# `cogmer`, which resolves only if the person installed it themselves — and
# the installer puts it in ~/.cogmer/bin, which is on nobody's PATH. So every
# command failed for every installed user, while the hooks worked, because only
# the hooks knew where to look. Two rules meant one of them was wrong.

# cogmer_binary prints the path to the binary, or fails.
#
# Searched in the order a person would expect to win: an explicit override, a
# binary they installed themselves, the plugin's own copy, then PATH.
cogmer_binary() {
  if [ -n "${COGMER_BIN:-}" ] && [ -x "$COGMER_BIN" ]; then
    printf '%s' "$COGMER_BIN"; return 0
  fi
  for candidate in \
    "${COGMER_HOME:-$HOME/.cogmer}/bin/cogmer" \
    "${CLAUDE_PLUGIN_ROOT:-}/bin/cogmer"
  do
    if [ -x "$candidate" ]; then printf '%s' "$candidate"; return 0; fi
  done
  if command -v cogmer > /dev/null 2>&1; then
    command -v cogmer; return 0
  fi
  return 1
}

# ct_say appends one timestamped line to a log under the state directory.
#
# §3.1 requires failure to be silent TO THE PERSON AT THE KEYBOARD. It does not require silence
# to the log, and conflating the two is how a broken start became indistinguishable
# from a working one. Nothing here ever writes to stdout: stdout is injected into
# the user's turn, so a stray line there would corrupt every prompt.
ct_say() {
  local f
  f="${COGMER_HOME:-$HOME/.cogmer}/$1"; shift
  mkdir -p "$(dirname "$f")" 2>/dev/null
  printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >> "$f" 2>/dev/null
}

# json_safe strips what would break a JSON string literal.
#
# These details are assembled from error text -- a path, an OS message -- and land
# in the one line of stdout a hook is allowed to write. A stray quote there does not
# produce a worse message, it produces malformed JSON, and §3.1 says a hook must
# never do that to a session.
json_safe() { printf '%s' "$*" | tr -d '\\"' | tr '\r\n\t' '   '; }

# start_daemon_if_needed starts the daemon unless it is already answering.
#
# Called from the session-start hook and again from the installer when it has just
# fetched the binary. Without the second call the first session installs and the
# SECOND one starts the daemon, because the hook looked for a binary that was still
# downloading — a two-session warm-up nobody would guess at.
start_daemon_if_needed() {
  local bin addr log
  if ! bin="$(cogmer_binary)"; then
    # Ordinary on a first session -- the binary is still downloading -- and a real
    # fault on any later one. install-state tells those apart (D-075); this records
    # that the start was attempted and found nothing, which is the half that used
    # to be missing.
    ct_say daemon.log "hook: no binary yet (searched COGMER_BIN, ${COGMER_HOME:-$HOME/.cogmer}/bin, plugin bin, PATH); not starting"
    return 0
  fi

  addr="${COGMER_ADDR:-127.0.0.1:4782}"
  if command -v curl > /dev/null 2>&1; then
    if curl -s -m 1 "http://${addr}/healthz" > /dev/null 2>&1; then
      return 0   # already running, which is the ordinary outcome, not an error
    fi
  else
    # "curl said no" and "there is no curl" are not the same answer, and reading the
    # second as the first starts a daemon on every single session. Say which it was.
    ct_say daemon.log "hook: no curl, so whether a daemon already answers on ${addr} is unknown; starting one, which exits harmlessly if the port is already ours"
  fi

  log="${COGMER_HOME:-$HOME/.cogmer}/daemon.log"
  mkdir -p "$(dirname "$log")" 2>/dev/null

  # Detached, so the daemon outlives the session that started it: a room may have
  # members in several sessions, and starting it repeatedly is worse than leaving
  # it running.
  #
  # Both branches are correct, and neither costs the daemon its GUI session --
  # which matters, because opening the room view and raising a notification are
  # the only two ways this design ever reaches a person (B23). macOS ships no
  # setsid at all, so it always takes the second branch; and a new POSIX session
  # was measured not to change anything anyway, since the Mach bootstrap
  # namespace is inherited separately. Do not collapse these to one branch on the
  # theory that setsid strands the daemon somewhere it cannot open a window. It
  # does not. That was tested, and the test that appeared to show otherwise was
  # measuring a `setsid` that did not exist.
  if command -v setsid > /dev/null 2>&1; then
    setsid nohup "$bin" daemon >> "$log" 2>&1 < /dev/null &
  else
    nohup "$bin" daemon >> "$log" 2>&1 < /dev/null &
  fi
  # Which binary, which branch, which process. All three were decided here and
  # recorded nowhere, so "the daemon is not running" had no next question.
  ct_say daemon.log "hook: starting $bin on ${addr} (pid $!)"
  disown 2>/dev/null
  return 0
}

# daemon_state prints "<state> <age-seconds> <detail>", mirroring install_state.
#
# Written by the daemon itself when it cannot bind. A busy port used to kill it
# silently: no daemon, no capture, no injection, and nothing anywhere saying why.
daemon_state() {
  local f line state when detail age
  f="${COGMER_HOME:-$HOME/.cogmer}/daemon-state"
  [ -f "$f" ] || { printf 'none 0 '; return 0; }
  IFS=$'\t' read -r state when detail < "$f" || { printf 'none 0 '; return 0; }
  age=$(( $(date +%s) - ${when:-0} ))
  printf '%s %s %s' "$state" "$age" "$detail"
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

install_state_file() { printf '%s' "${COGMER_HOME:-$HOME/.cogmer}/install-state"; }

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
