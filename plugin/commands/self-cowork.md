---
description: Set this machine up for Claude Cowork
allowed-tools: Bash("${CLAUDE_PLUGIN_ROOT}/cli.sh" cowork-setup:*)
disable-model-invocation: true
---

## What happened

!`"${CLAUDE_PLUGIN_ROOT}/cli.sh" cowork-setup`

## Your task

Tell the user what the output says changed, in a few lines: that Claude Desktop now starts
cogmer's bridge, and where the plugin for Cowork was written.

Then give the steps it lists, in order, as the user's to do: quit Claude Desktop completely and
open it again, upload the file in Cowork's plugin settings, and start a new Cowork session. Say
that typing `/cogmer:self-status` there shows whether it works. Do not do any of it for them.

If the output says nothing was changed because a file could not be read or written, say what it
said and stop. Do not try again, and do not edit any file yourself.
