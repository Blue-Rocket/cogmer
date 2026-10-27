#!/usr/bin/env bash
# What the slash commands invoke.
#
# A command cannot say `cogmer` and expect it to resolve: the installer puts
# the binary in ~/.cogmer/bin, which is not on PATH, and §29 forbids modifying
# a shell profile to put it there. So commands go through here, which uses the same
# resolution the hooks use.
#
# Unlike the hook wrapper, this does NOT fail open. A hook that cannot find the
# binary must leave the session alone; a command a person typed must say why
# nothing happened — and say which of the three reasons it is, rather than guessing
# the hopeful one (D-075).
#
# It always exits 0, and everything it says goes to stdout. Claude Code abandons a
# command whose `!` line exits non-zero: no model turn runs, so the model never sees
# the explanation and the person sees only a failed shell line. Every message below
# is written to be read by the model and relayed, which needs the turn to run.
#
# It never waits for an install. A command answers at once and says what to do, so
# that no command holds a session up (§3.1, D-191).
set -u
. "$(dirname "$0")/hooks-handlers/common.sh"

# The binary reads the plugin's VERSION through this, to report whether the plugin,
# the binary and the running daemon agree.
export CLAUDE_PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")" && pwd)}"

# Which host this command was run from. Claude Code's commands call this script
# directly, and Codex's go through codex-cli.sh, which sets it first, so the host is
# declared by the entry point and not guessed from whatever the environment holds
# (D-203).
export COGMER_HOST="${COGMER_HOST:-claude}"

# restart_daemon stops whichever cogmer daemon holds this installation's addresses
# and starts one with the function the session-start hook uses, so the start is
# detached and logged the same way on every platform (D-195). It waits a few
# seconds for the daemon to answer, because "started" and "answering" are different
# facts and a person asking for a restart wants the second. COGMER_RESTART_WAIT is
# that wait in seconds, and exists so a test need not sit through it.
restart_daemon() {
  local bin="$1" addr="${COGMER_ADDR:-127.0.0.1:4782}" waited=0 limit="${COGMER_RESTART_WAIT:-5}"
  local state_file="${COGMER_HOME:-$HOME/.cogmer}/daemon-state" state when detail
  "$bin" stop --for-restart 2>&1
  start_daemon_if_needed
  while [ "$waited" -lt "$limit" ]; do
    if curl -s -m 1 "http://${addr}/healthz" > /dev/null 2>&1; then
      echo "The daemon was restarted and is answering on ${addr}, version $("$bin" version 2>/dev/null)."
      echo
      echo "Nothing that was captured is lost, because it is written to disk before it is"
      echo "shared. Connecting to colleagues again can take a few seconds more."
      return 0
    fi
    sleep 1
    waited=$((waited + 1))
  done
  echo "The daemon was started and is not answering on ${addr} yet."
  if [ -r "$state_file" ]; then
    IFS=$'\t' read -r state when detail < "$state_file"
    if [ "$state" = blocked ] && [ -n "$detail" ]; then
      echo
      echo "It recorded why: $detail"
    fi
  fi
  echo
  echo "~/.cogmer/daemon.log says what happened."
}

if bin="$(cogmer_binary)"; then
  if [ "${1:-}" = restart ]; then
    restart_daemon "$bin"
    exit 0
  fi
  "$bin" "$@" 2>&1
  exit 0
fi

home_dir="${COGMER_HOME:-$HOME/.cogmer}"

# start_install begins the download the way the session-start hook does, detached,
# so this command returns at once. install.sh takes its own lock and decides for
# itself whether there is anything to do, so starting it twice is harmless.
start_install() {
  local log="$home_dir/install.log"
  ct_state_dir
  (umask 077; : >> "$log") 2>/dev/null
  if command -v setsid > /dev/null 2>&1; then
    setsid nohup bash "$CLAUDE_PLUGIN_ROOT/hooks-handlers/install.sh" >> "$log" 2>&1 < /dev/null &
  else
    nohup bash "$CLAUDE_PLUGIN_ROOT/hooks-handlers/install.sh" >> "$log" 2>&1 < /dev/null &
  fi
  disown 2>/dev/null
}

read -r state age detail <<< "$(install_state)"
case "$state" in
  installing)
    echo "cogmer is still downloading. Nothing is wrong."
    echo
    echo "The download started ${age}s ago (${detail}) and takes about 15 to 20 seconds."
    echo "Wait that long, then run the same command again."
    ;;
  failed)
    echo "cogmer could not be installed."
    echo
    echo "  ${detail}"
    echo
    echo "The installer waits an hour after a failure before trying by itself. To try"
    echo "now, remove ~/.cogmer/install-state and run the same command again."
    echo "~/.cogmer/install.log says what happened."
    ;;
  stalled)
    # Stalled is an install that began more than five minutes ago and never
    # finished, which means the process died. It may have left its lock behind, and
    # every later install finds the lock and leaves, so a new session cannot help.
    rmdir "$home_dir/.install.lock" 2>/dev/null
    clear_install_state
    start_install
    echo "An earlier download started ${age}s ago and never finished, which usually"
    echo "means it was interrupted."
    echo
    echo "This command has cleared what it left behind and started the download again."
    echo "It takes about 15 to 20 seconds. Wait that long, then run the same command"
    echo "again."
    ;;
  *)
    start_install
    echo "cogmer is not installed yet. This command has started downloading it."
    echo
    echo "The download takes about 15 to 20 seconds. Wait that long, then run the same"
    echo "command again. Nothing else is needed."
    echo
    echo "~/.cogmer/install.log says what happened if it does not arrive."
    ;;
esac
exit 0
