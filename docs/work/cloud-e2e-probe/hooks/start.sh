#!/bin/sh
# Starts the daemon bundled in this plugin, detached, in a state directory of its own under /tmp, on
# high ports, when it is not already answering. It runs only where uname -s is Linux and
# CLAUDE_CODE_REMOTE is true, which is a cloud session, so it does nothing on a user's own machine.
# The daemon is a build of the user-identity branch, which reads a cloud value from a plugin's file
# where the released program does not. It holds no secret and sends nothing but what the daemon
# sends: the cloud value is another plugin's file, which the daemon reads itself.
[ "$(uname -s 2>/dev/null)" = Linux ] || exit 0
[ "${CLAUDE_CODE_REMOTE:-}" = true ] || exit 0
cat > /dev/null 2>&1
D=/tmp/cogmer-e2e
mkdir -p "$D" 2>/dev/null
export COGMER_HOME="$D" COGMER_ADDR=127.0.0.1:47840 COGMER_PEER_ADDR=127.0.0.1:47841
if curl -s -m 1 "http://$COGMER_ADDR/healthz" > /dev/null 2>&1; then
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $1: the daemon already answers" >> "$D/start.txt"
  exit 0
fi
bin="${CLAUDE_PLUGIN_ROOT}/cogmer"
chmod +x "$bin" 2>/dev/null
if command -v setsid > /dev/null 2>&1; then
  setsid nohup "$bin" daemon >> "$D/daemon.log" 2>&1 < /dev/null &
else
  nohup "$bin" daemon >> "$D/daemon.log" 2>&1 < /dev/null &
fi
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $1: started $bin (pid $!)" >> "$D/start.txt"
exit 0
