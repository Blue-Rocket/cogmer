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
set -u
. "$(dirname "$0")/hooks-handlers/common.sh"

if bin="$(cogmer_binary)"; then
  exec "$bin" "$@"
fi

read -r state age detail <<< "$(install_state)"
case "$state" in
  installing)
    echo "cogmer is still arriving — started ${age}s ago (${detail})."
    echo "Try again in a moment; it is usually seconds."
    ;;
  failed)
    echo "cogmer could not be installed, and is not being retried yet."
    echo
    echo "  ${detail}"
    echo
    echo "It will try again about an hour after the failure. To try now, start a new"
    echo "Claude Code session after removing ~/.cogmer/install-state."
    ;;
  stalled)
    echo "An install started ${age}s ago and never finished, which usually means the"
    echo "process was interrupted. Starting a new Claude Code session will try again."
    ;;
  *)
    echo "cogmer is not installed yet."
    echo
    echo "The plugin fetches it in the background when a session starts, so starting a"
    echo "new session should be enough. If nothing changes, ~/.cogmer/install.log"
    echo "says what happened."
    ;;
esac
exit 1
