# Claude Cowork as a host

Material for the open item "Whether Claude Cowork becomes a supported host". Delete it with
that item. The probe that produced the runs is `docs/work/cowork-probe/`: a plugin with hooks
for `SessionStart`, `UserPromptSubmit` and `Stop`, a report command, and `host-mcp/server.py`,
a Model Context Protocol server with one tool, `ping`, that runs on the user's machine. All
runs were on 2026-10-13, on one macOS machine, in Claude Desktop with Cowork, with the plugin
uploaded through the Plugins settings and the server added to `claude_desktop_config.json`
under the name `cogmer-host-probe`. `coworkd.log` reports a Claude Code SDK of v2.1.295.

**Run:** probe 0.0.1 and 0.0.2, a new Cowork session with two or three prompts, then
`/cogmer-probe:probe-report`.

**Result:** the plugin's `SessionStart`, `UserPromptSubmit` and `Stop` command hooks fire.
The hook input carries `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `source`,
`scratchpad_dir`, and for a prompt `prompt_id`, `permission_mode` and `prompt`.
`CLAUDE_CODE_SESSION_ID` is set, so the present rule for finding a host would read the
session as Claude Code's. The transcript exists in the VM at
`~/.claude/projects/<dir>/<session>.jsonl` and holds `hook_success` lines once a hook has
run. The VM is Linux x86_64 as root, with `CLAUDE_CODE_REMOTE` and `CCR_*` variables set,
`/dev/vda` as its disk, and no `/sessions` directory. Its network reaches `github.com`
(the release binary answered 302), `api.github.com`, `pypi.org` and `registry.npmjs.org`, and
failed for `controlplane.tailscale.com`, `login.tailscale.com`, `derp1.tailscale.com`,
`derp2.tailscale.com`, `derp10.tailscale.com` and `example.com`, through an HTTPS proxy on
the loopback address. The relay cogmer uses is Tailscale's, so a daemon in the VM could be
installed and could reach no peer. The probe's run counter and first-seen file restarted
at 1 in each of four sessions, each with a different boot id, so nothing written under
`HOME` survives to the next session. The only writable mount is `/mnt/attach/outputs`, an
rclone filestore, and no folder of the user's was mounted.

**Run:** probe 0.0.3 with an `mcp_tool` hook naming the server `cogmer-host-probe`, then the
model asked to call a tool whose name contains "ping".

**Result:** the hook produced no call on the user's machine. The tool the model could call was
`mcp__remote-devices__cogmer-host-probe__ping`, and calling it reached the machine
(`ping event=` at 14:12:53). Cowork presents a server from the Desktop configuration as the
tool `<server>__<tool>` on a server named `remote-devices`.

**Run:** probe 0.0.4 with the hook naming `server: remote-devices` and
`tool: cogmer-host-probe__ping`, then the model asked which lines in its context contained
`[cogmer-`.

**Result:** the user's machine logged `ping event=SessionStart` two seconds after the session
began and `ping event=UserPromptSubmit` for each of four prompts. The model's context held the
command hook's text for each prompt and none of the text the call returned, which was a
plain line.

**Run:** probe 0.0.6, with the hook's fields passed through `${...}` placeholders in the tool
input, a `Stop` hook passing `last_assistant_message`, and two calls on each prompt, one
returning a plain line and one returning
`{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"..."}}`.
Claude Desktop was restarted before the run.

**Result:** every placeholder resolved. The server logged the session identifier (its first
eight characters matched the VM's), the prompt (64, 7, 10 and 58 characters), the prompt
identifier, the transcript path, and on `Stop` the last assistant message (7, 7, 7 and 677
characters) and `stop_hook_active`. The model's context held the five JSON replies, which it
described as extra context a hook added with each prompt, and none of the plain replies. The
tool name was the same after the restart.

**Run:** the probe server's processes killed on the user's machine, then prompts sent in a new
Cowork session.

**Result:** not run yet.

## Read, not observed

Anthropic's documentation says plugin hooks run in Cowork sessions, that a hook may have the
type `mcp_tool` with the fields `server`, `tool` and `input`, and that its text output is read
as a command hook's stdout is (code.claude.com/docs/en/hooks and
claude.com/docs/cowork/3p/extensions). The 0.0.4 run shows the plain-text half of that
statement did not hold in this session. Hooks from the user's own `~/.claude/settings.json`
are reported not to fire in Cowork (anthropics/claude-code issue 63360, closed as not
planned on 2026-06-30).

`github.com/marius-bughiu/cowork-terminal-mcp` is a server on the host that Cowork reaches
through the Desktop configuration, and it gives a shell, which a cogmer bridge must not.
`github.com/sharpninja/mcpserver-claude-cowork-plugin` uses hooks with a local server and says
local servers may be disabled by an administrator in Cowork and that a cloud connector cannot
reach a local stdio process. Anthropic's `knowledge-work-plugins` repository uses remote
connectors and one `SessionStart` echo, and no local server. Issue 89129 reports the
`remote-devices` bridge stuck in a failed state, on Windows, and issue 26259 reports desktop
extension servers not passed to Cowork sessions. Nothing found documents whether the
`remote-devices` name is stable.

Installing the probe in Cowork showed the notice "it uses the legacy commands/ format. Both
formats work", which is the layout item under "Tidying the commands".
