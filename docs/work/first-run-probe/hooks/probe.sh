#!/usr/bin/env bash
# Appends a line to cogmer-first-run-probe.log in the working directory, then answers with a
# systemMessage and an additionalContext, each carrying a marker that names the hook's form and
# event, so that a run shows which of the two reached the person and which the model. It reads
# nothing else and sends nothing anywhere. $1 is the form, $2 the event.
cat > /dev/null 2>&1
form="${1:-unknown}"; event="${2:-unknown}"
printf '%s %s ran\n' "$form" "$event" >> cogmer-first-run-probe.log 2>/dev/null
printf '{"systemMessage":"SYS-FRP-%s-%s","hookSpecificOutput":{"hookEventName":"%s","additionalContext":"CTX-FRP-%s-%s"}}\n' \
  "$form" "$event" "$event" "$form" "$event"
exit 0
