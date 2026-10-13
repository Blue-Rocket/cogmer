#!/usr/bin/env bash
# The skill supplies its installed path; room identity comes from Codex's tool
# environment, never from an id guessed by the model or saved for the machine.
export CLAUDE_PLUGIN_ROOT="$(cd "$(dirname "$0")" && pwd)"
export COGMER_HOST=codex
if [ -z "${CODEX_THREAD_ID:-}" ]; then
  echo "No Codex session was supplied. Run the cogmer skill inside the session you want to use."
  exit 0
fi
exec bash "$CLAUDE_PLUGIN_ROOT/cli.sh" "$@"
