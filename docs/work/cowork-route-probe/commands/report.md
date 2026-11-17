---
description: Print what the route probe recorded
disable-model-invocation: true
---

Run these three commands with your shell tool and show each output to the user exactly as printed, in a code block of its own. Add nothing and draw no conclusions.

1. `cat /tmp/cogmer-route-probe-relay.txt`
2. `sh "${CLAUDE_PLUGIN_ROOT}/hooks/where.sh" --plain`
3. `tail -n 20 "$HOME/cogmer-route-probe/server.log"`
