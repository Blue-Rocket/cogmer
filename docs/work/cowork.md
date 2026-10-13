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
at 1 in each of four new sessions, each with a different boot id, so nothing written under
`HOME` survives into the next session. The only writable mount is `/mnt/attach/outputs`, an
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

**Run:** the probe server's processes killed. Claude Desktop started a new copy 41 seconds
later, and a later copy quit when it started, by a mode file the server reads, so that
every restart failed. Prompts were then sent in a Cowork session.

**Result:** the model's call of the tool failed at once with `Server cogmer-host-probe
unavailable`, and the tool stayed in its list. The hook's call returned nothing and the model's
context held no line from it, no error and no delay the user noticed. Desktop retried the
server at 15:28:45 and 15:28:50, and `mcp-server-cogmer-host-probe.log` records each as
"Couldn't start for Cowork and Code sessions. Error: Connection closed". With the mode file
emptied, Desktop started a copy at 15:29:58, the moment the next hook called, and the hook and
the model's ping both succeeded in the same session, with no reload.

**Run:** the mode file set so that the server took a call and never answered, the running
copy killed, and one prompt sent in the same session.

**Result:** the server logged one call taken at 15:31:39 and nothing after it, since it
handles one request at a time, and Desktop logged no timeout of its own between 15:31:39
and the kill at about 15:34. The model reported that its own call to the tool got no answer
for 60 seconds and then timed out, and that the hook's line was missing. The user
reported that the hang slowed the session for the full 60 second timeout, not timed with a
stopwatch. The probe's hooks set a timeout of 15 seconds, so the wait was four times the limit
the hook declared, and the prompt had also asked the model to call the tool, so the 60 seconds
may be that call's. Desktop did not restart the hung server, so a second prompt would have
waited behind the first.

**Run:** probe 0.0.7, which adds to the session-start report the server names read from the
harness's command line and the output of `claude mcp list`, and a third host call on each prompt
naming `server: "remote-*"`. One session was started, and then resumed after the VM had
restarted.

**Result:** the harness's command line had no `--mcp-config` argument, only `--preload`, and
`claude mcp list` in the VM printed "No MCP servers configured", so a hook in the VM cannot list
the servers of its session, although the model's tool list holds the host program's tool. The
call naming `remote-*` never reached the host, in either run. A `SessionStart` call reached the host once, at
14:21:02 in the probe 0.0.4 session, and did not in the sessions of 14:33, 16:07 and 16:18, the last of
them the resumed one, although the `UserPromptSubmit` and `Stop` calls of those sessions did.
The host program was running at 14:33 and 16:18 and had not yet been started at 16:07. After the resume the VM had a new boot identifier and the same
session identifier and transcript (662,014 bytes, five `hook_success` lines), and the probe's
run counter continued from 3 to 4, so files under `HOME` survive a restart of the VM within a
session and did not survive into a new session. The transcript holds the command hook's output
as a `hook_success` line: at the end of the first prompt it held 4 such lines and the probe's own
text once.

**Run:** probe 0.0.8, with the host program already running and the mode file empty, and
three brand-new sessions, each given the one message "Quote every line in your context that
starts with `SessionStart`. Then stop."

**Result:** in each session the plugin's command hook fired and the model quoted
`SessionStart:startup hook success: [cogmer-probe] SessionStart fired...`. The host logged the
`UserPromptSubmit` and `Stop` calls of sessions `127a86a2` (16:33:44), `907de45c` (16:34:01)
and `7ed422fc` (16:34:19), and no `SessionStart` call for any of them. Counting the earlier
sessions, a `SessionStart` call reached the host once in seven sessions that had one configured.
The hooks reference says why: `SessionStart` at launch, including on resume, fires before the
session's MCP servers are available to hooks, so Claude Code skips its `mcp_tool` hooks without
calling the tool, and its `mcp_tool` hooks run when `SessionStart` fires again after `/clear` or a
compaction. The call at 14:21:02 does not fit that and has no explanation.

Other observations from the Desktop logs on 2026-10-13: Desktop connects a configured server
for Cowork and Code sessions through what `mcp-server-<name>.log` calls a shared pool, and
`main.log` shows a `[remote-tools-device]` component that holds a keep-awake claim on the
machine while remote tools are in use and released it after 1,224 seconds of quiet. The
model's error text named the machine `pushover-local`, which is the computer's name in Cowork,
and the host program saw its own hostname, `macbookpro.lan`. The tool name carries no computer
name, so which computer receives a call for a user with two was not tested.

## Read, not observed

Anthropic's documentation says `mcp_tool` hooks on `SessionStart` at launch are skipped because
no MCP client exists yet, and recommends a command hook for anything needed at launch
(code.claude.com/docs/en/hooks, read on 2026-10-13). It says plugin hooks run in Cowork sessions, that a hook may have the
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
