# Capture and injection options under vendor change

Material for the open items in `docs/open.md` on capture and injection, deleted with the last of them. Written on 2026-11-28 by a research run and checked against its sources the same day. Everything is in this folder: `params.json` (every rate and its source), `sim.py` (the simulation and its sanity checks), `sens.py` (sensitivity), `out-base/` (results.json, results.csv, run.log, sensitivity.txt/json), `out-cw/` (the Cowork rows of the sensitivity sweep after a fix described below), and `history/` (the changelog bullets the rates were counted from).

(Corrected on 2026-11-28 after a check of its sources: D-270 is withdrawn; Mods date from 2.1.287; the HookMetadata change is the Python SDK's; 2.1.332 is not labelled breaking; MessageDisplay carries no prompt_id, its firing for subagents is undocumented, and it holds each batch until the hook returns; B30 to B32 are in cowork_probe.go; `e7ddab7` does not show which copy of the plugin ran, and plugin commands were refused there; "Cloud orchestration supports only Global MCP hooks" is documented for Work Cloud, not Codex cloud tasks.)

## The answer

Keep the current design on every host. It is the best option the model found for Claude Code and Codex on a user's own computer, and no transcript-free design beats it there. Spend the next effort on three cheap changes that turn silent failures into loud ones, and on the probes that would remove the largest unknowns.

1. **Cross-check the transcript against the hook payload (CC3, CX3, CL2), about 1 person-week per host.** When the transcript's final text block disagrees with `last_assistant_message`, publish the payload's text, treat the transcript as unreadable, and say so. In the simulation this cut expected months of silent failure over 24 months on a local Claude Code session from 0.30 to 0.24 (paired difference −0.057 ± 0.004 months, 19%). It also raised the fraction of time capture works by 0.9 points, and it costs nothing else. For Codex, the same guard plus counting delivery at Stop while the rollout is unreadable (CX3) cut silent months from 0.31 to 0.22 (−0.087 ± 0.005, 28%). It raised the share of time that injection is clean (no re-offering) from 65% to 82%. That revises D-202 (host-specific hooks use the shared peer protocol), whose Limits say Codex has no trust fallback, so it is a decision to take, not a fix.
2. **For Cowork on the user's computer, capture and inject through the plugin's own command hooks (CW2), and keep the bridge for commands only.** The run of 2026-11-18 (`e7ddab7`, recorded in open.md) showed the plugin's hooks firing on the Mac, with plain prompt-hook output reaching the model. The bridge adds the `mcp_tool` hook path, which changed or was fixed six times in seven months, and a failure there loses context silently (D-206 Limits). CW2 against CW1: capture works 86.0% vs 78.6% of the time, silent months 0.40 vs 0.61 (−0.21 ± 0.01), 1.5 person-weeks cheaper. This contradicts nothing decided, since D-205 chose the bridge before the hooks were seen to run on the computer, but it needs a decision.
3. **Add MessageDisplay as a source of text written before a tool call where no transcript exists (KC2, CW3), about 1.5 to 2.5 person-weeks, but only after a latency probe.** A Cowork cloud turn today records only the final text block (D-295, a turn with no transcript path is recorded from the Stop hook's last message). With MessageDisplay, the simulation gave full capture 60% of the time instead of 0%, and silent months fell by 0.12 ± 0.01. The hook runs on the display path, so it fails the values unless it is shown to add no delay to what the user sees.

Not recommended: hook payloads alone on a local session (CC2, CX2). They are cheaper by 2 to 5 person-weeks, but they lose the text before a tool call every month and they raise silent months. OpenTelemetry capture (CC7) gives the most complete text, but it raises silent months and needs settings outside the plugin (D-037, Claude Code is launched and used unchanged). The Agent SDK or a wrapper is rejected by D-034 (no terminal wrapper) and D-037, and nothing found here changes that.

What moves the result most: the vendor redesign rate, how long a silent failure goes unnoticed by users, and how soon `cogmer doctor` meets a new version. None of those changes the ranking: every recommended option beats the current one on silent months in all 16 sensitivity cases.

The weakest part of every portfolio is the cloud. Claude Code cloud and Cowork cloud sessions fail silently for about 1.0 to 1.1 months in 24, against 0.2 to 0.3 on a user's computer. The reason is that no behavior check runs there (D-297, the behavior checks are not run in a cloud container), so users are the only detector. The model says the most valuable single thing to build is a canary that runs inside a cloud session.

## 1. Options

The B-numbers are from `docs/relied-on-behaviors.md`. "New" marks a vendor behavior that no check covers yet. It would need an entry in `behaviors.go` with a negative test before anything relied on it. Build cost is an estimate in person-weeks (pw) from today's code.

### Capture

| option | hosts | vendor behaviors | build | how it fails | values | status |
|---|---|---|---|---|---|---|
| Hook payloads plus transcript reassembly (union) | Claude Code CLI and desktop, Claude Code cloud | B01 B03 B04 B06 B07 B08 B09 B12 B14 B15 B17 B18 B33 | 0 | transcript changes fail silently with wrong text (B06, B07, B17); a missing transcript degrades to the final block | meets | current, D-003 (reassemble turns by unioning two incomplete sources) |
| Same, cross-checked: the transcript's final text block must equal `last_assistant_message` | same | same | 1 | turns the transcript's silent corruption into a known degradation | meets | new |
| Hook payloads alone (prompt + Stop `last_assistant_message`) | all Claude Code surfaces; Codex | B01 B03 B04 (Codex B25 B26) | 0.5 | loses every word before a tool call, every month; fails loudly if Stop changes | meets | rejected for a session with a transcript by D-003; used where there is none, D-295 |
| Transcript alone | Claude Code | B06 to B17 | 0 | lost the final block before 2.1.331 | meets | rejected, D-003 ("the transcript alone") |
| MessageDisplay hook plus Stop (new in Claude Code 2.1.152, 2026-05-26): every assistant text message as displayed | all Claude Code surfaces, if it fires there | new: fires once per assistant message that streams text, with session_id and turn_id but no prompt_id, and a message_id that cannot be matched to the transcript; whether it fires for subagents is undocumented; Claude Code holds each displayed batch until the hook returns | 2 | unknown; it runs on the display path, so a slow hook delays what the user sees | only if latency is shown to be nil; in `-p` and the SDK it fires once per message after completion, interactively once per batch of lines | new |
| OpenTelemetry log events (`user_prompt`, `assistant_response`, 2.1.193) to a receiver in the daemon | Claude Code CLI and cloud | new: event names and attributes; `OTEL_LOG_USER_PROMPTS`, `OTEL_LOG_ASSISTANT_RESPONSES` | 3 | 2.1.332 changed `service.name` for the Code tab, and 2.1.323 fixed events recording another agent's reply | fails: needs environment settings outside the plugin, and takes the exporter slot that an organization's own telemetry may use (D-037) | new |
| `http` hook type (2.1.63) posting payloads straight to the daemon | Claude Code | new: an unreachable URL fails open with nothing shown | 1 | unknown; must behave like B11 (a hook whose daemon is unreachable exits 0 with empty stdout) | meets if it fails open | new, transport only; would remove the dependency of D-277 (Claude Code and Codex on Windows run cogmer through Git for Windows) on a shell |
| Codex hooks plus rollout reassembly | Codex CLI and desktop, ChatGPT Work local | B25 B26 B28 | 0 | OpenAI says the transcript "isn't a stable interface for hooks"; rollouts were compressed (0.137), paginated and migrated (0.147-0.148), and some threads have no local rollout (0.130) | meets | current, D-201, D-202 |
| Same, cross-checked | same | same | 1.5 (with the injection change) | known degradation | meets | new |
| Cowork bridge (`mcp_tool` hooks to a server on the computer) | Cowork local | B29 B30 B31 B32 | 0 | the reply silently not reaching the model while Stop arrives (D-206 Limits) | silent loss accepted in D-206 | current, D-205 |
| Plugin command hooks in local Cowork | Cowork local | B01 B03 B29 B02 | 1 | as hook payloads; commands still need the bridge, since a `!` line is disabled in Cowork | meets | observed 2026-11-18 (`e7ddab7`), undecided |
| Daemon in the sandbox, hook payloads | Cowork cloud, Claude Code cloud | B01 B03 B29, the container's environment | 0 | first prompt empty; transcript path empty; plugin copy stale for a month | meets | current, D-288 (a Cowork cloud session that holds a cloud value starts the daemon as its user), D-295 |
| Codex cloud or Work Cloud | Codex cloud | command hooks do not run there; "Cloud orchestration supports only Global MCP hooks" | 3 + a hosted endpoint | not available | a global MCP hook would need a reachable server, which is a centralized component | open (open.md, "Whether a Codex cloud task can be a cloud session is not known") |
| Agent SDK or `claude -p` wrapper running the session | any | Agent SDK | 8+ | reliable capture, but the user no longer starts the host unchanged | fails §3.8 | rejected, D-034, D-037 |
| Model reports its own turns through an MCP tool or skill | any | model compliance | 1 | depends on the model; adds tool calls nobody asked for | fails | rejected in substance by D-206 ("asking the model to acknowledge each injection with a tool call") and D-119 (no slash command is reachable by the model) |
| `/export`, session-sharing links | Claude Code | manual | — | not automatic | — | out of scope |

### Injection

| option | hosts | vendor behaviors | build | how it fails | values | status |
|---|---|---|---|---|---|---|
| UserPromptSubmit stdout, confirmed from the transcript (hash of the block) | Claude Code local and cloud | B02 B20 B22 B34 | 0 | missing evidence re-offers turns (noisy, announces itself); a host limit under the budget fails silently until users see late turns (B34, `1df545d`) | meets | current, D-014 (derive delivery state from transcript evidence), D-131 (trust without evidence only once the evidence is known broken) |
| Same in the JSON `additionalContext` form | Claude Code | documented; recorded as `hook_additional_context` (`5ac6450`) | 0.5 | as above | meets | already used for Cowork, D-206 |
| Delivery counted at Stop by prompt_id | all Claude Code surfaces; Codex by turn_id | B29 (Codex B25) | 0.5 | a context that did not arrive is lost silently | silent-loss risk | rejected for a session with a transcript by D-131; current where there is none, D-296 (a stop call that carries no transcript path counts the turns offered at its prompt as delivered) |
| Codex: evidence while the rollout is readable, delivery counted at Stop while it is not | Codex | B25 B27 | 1.5 with the guard | silent loss only while the rollout is broken | revises D-202 Limits | new |
| SessionStart `additionalContext` | all | documented | 0.5 | delivers only at the start, so not a live channel | meets | useful only for a resumed session; not an alternative |
| Stop `additionalContext` (2.1.163) | Claude Code | documented | — | continues the turn; a probe saw nine Stop calls in 25 seconds (`d306243`) | fails: makes the session take a turn | ruled out by D-035 (a remote event never makes an interactive session take a turn) |
| Writing CLAUDE.md or AGENTS.md | all | instructions files | 1 | read at start, not live; writes into the user's project | fails: noise in the user's repository | not decided; fails §3.1 |
| MCP tool, resource or prompt the model pulls from, or a skill | all | model choice | 1 | model-dependent, adds tool calls | fails | D-119; sampling banned by D-147 (no MCP server cogmer ships exposes sampling) |
| MCP as a display channel | — | — | — | shows nothing anywhere | — | rejected, D-036 (MCP logging notifications are not a display channel) |
| Codex PreModelRequest or background-hook `additionalContext` "at the next safe point" | Codex | new | 2 | unknown; mid-turn context | needs a decision against D-035 | new, not simulated |
| Cowork `mcp_tool` JSON reply, counted at Stop | Cowork local | B29 B30 | 0 | silent loss if the harness drops the reply | accepted in D-206 | current |

## 2. History: what changed, and how often

### Claude Code

Sources: the changelog (https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md), 448 versions, read 2026-11-28. Each version was dated by its publish time in the npm registry (https://registry.npmjs.org/@anthropic-ai/claude-code, read 2026-11-28; 574 releases from 0.2.6 on 2025-02-24 to 2.1.332 on 2026-11-24, about 27 a month in 2026). `history/mine_changelog.py` extracted 1,012 bullets that mention hooks, transcripts, compaction, plugins or MCP tool hooks, into `history/cc_hook_bullets.csv`. The hooks reference (https://code.claude.com/docs/en/hooks, read 2026-11-28) was also read.

Changes that touched what cogmer relies on, or could:

- Additions, about one every two months: UserPromptSubmit (1.0.54, 2025-07-16) and its `additionalContext` (1.0.59, 2025-07-23); `last_assistant_message` on Stop (2.1.47, 2026-02-18); `http` hooks (2.1.63, 2026-02-28); `mcp_tool` hooks (2.1.118, 2026-04-22); `CLAUDE_CODE_SESSION_ID` in Bash (2.1.132, 2026-05-06), now B21; MessageDisplay (2.1.152, 2026-05-26); `additionalContext` on Stop (2.1.163, 2026-06-04); the OTel `assistant_response` event (2.1.193, 2026-06-25).
- Intentional changes: hook output over 10,000 characters saved to a file with a 2,000-character preview (2.1.89, 2026-03-31). Six months later this silently broke injection for cogmer users, whose budget was 60,000 (`1df545d`, 2026-10-14). It is now B34. UserPromptSubmit runs before a command expands (2.1.299, 2026-10-14), which touches B33. Code Mode no longer runs UserPromptSubmit on script output (2.1.328, 2026-11-18). The transcript gets the final block before Stop fires (2.1.331, 2026-11-23); cogmer's B05 reported this improvement as a failure and was retired (`b35c5ea`).
- Regressions and bugs: a wrong `transcript_path` for resumed or forked sessions (fixed 2.1.72, 2026-03-09) and after EnterWorktree (2.1.141, 2026-05-13); plugin Stop and UserPromptSubmit hooks failing after cache cleanup (2.1.136, loud); `system-reminder` tags escaped in hook output (2.1.292) and hook output altered by text resembling a hint tag (fixed 2.1.296, 2026-10-09). Both of those change the exact text whose hash D-014 matches. Edit and Retry broken while UserPromptSubmit ran (a 2.1.296 regression fixed in 2.1.299 five days later). Six `mcp_tool` changes or fixes between 2.1.281 and 2.1.328 (2026-09-23 to 2026-11-18).
- Removals: none of a hook event or a field cogmer uses in 16 months.
- Documentation direction: the hooks reference now says the transcript "is written asynchronously and may lag", and that hooks needing the final text "should use `last_assistant_message` … instead of reading the transcript". It caps a hook's output at 10,000 characters "with no setting or environment variable to raise it".
- Signals of a redesign: Claude Mods, with hooks such as `prompt.submit`, arrive beside hooks in 2.1.287 (2026-10-01); `PreSkillShellCommand` arrived in 2.1.332; marketplace auto-update becomes the default on or after 2026-11-30 (2.1.331).

Count used for the rates: about 8 adverse changes to documented hook behavior in 16 months, spread over about 9 capabilities. That is 0.055 per capability-month, halved to 0.02 to 0.025 because most hit only resumed, worktree or Code Mode sessions.

### Codex

Sources: GitHub releases of openai/codex (https://api.github.com/repos/openai/codex/releases, 600 releases read 2026-11-28, of which 79 stable, from 0.123.0 on 2026-04-23 to 0.172.0 on 2026-11-24), extracted to `history/codex_bullets.csv`. The hooks page (https://learn.chatgpt.com/docs/hooks, redirected from developers.openai.com/codex/hooks, read 2026-11-28).

- Hooks were marked stable in 0.124 (2026-04-23). Plugin-bundled hooks arrived in 0.128 and were enabled by default in 0.131 (2026-05-18).
- Changes to what cogmer relies on: 0.131 "use new session IDs instead of thread IDs for hooks", which bears on B28; 0.129 large hook outputs spilled from context, and the default limit is "roughly 2,500 tokens", which cogmer turns off with `additionalContextLimit: 0`; 0.149 "Run hooks with the captured session environment"; 0.154 of the Python SDK, not the CLI, wraps `HookMetadata`'s handler in `.root`.
- Rollout files, which cogmer reads: threads without local rollout paths (0.130); compressed cold rollouts (0.137, 2026-06-04), then active ones (0.142); paginated history with a migration of older rollouts (0.147 to 0.148, 2026-08); obsolete turn-context fields removed (0.162). OpenAI states "transcript format isn't a stable interface for hooks and may change over time". A counter-signal is PR #28632, "Tell codex to avoid changing rollout format" (0.142). B26 and B27 still passed in October and November in cogmer's runs.
- Cloud: "Cloud orchestration supports only Global MCP hooks, even when a tool runs on a connected computer", and Work Cloud without local computer access does not run these hooks.

### Deprecations by either vendor

- OpenAI ended ChatGPT plugins with about three weeks' notice. New installs and conversations stopped on 2024-03-19, and existing conversations on 2024-04-09 (https://community.openai.com/t/plugin-store-and-new-chats-with-plugins-closed-march-19-2024/689877, read 2026-11-28).
- OpenAI announced in 2025 that the Assistants API would sunset on 2026-08-26, and it did (https://developers.openai.com/api/docs/assistants and https://community.openai.com/t/assistants-api-beta-deprecation-august-26-2026-sunset/1354666, read 2026-11-28).
- Anthropic: no extension surface removed since hooks shipped (1.0.38, 2025-06-30, per npm). Fast mode for Opus 4.8 was deprecated on 2026-10-30 for removal on 2026-11-30 (changelog), which shows the vendor gives about a month's notice.

### What cogmer itself saw (repository)

In about 2.4 months (2026-09-16 to 2026-11-28):

- B34 failed silently in use. Users reported it two days after the symptom began, and it had been latent since 2.1.89 (`1df545d`).
- B05 changed for the better in 2.1.331 and was retired (`b35c5ea`).
- On new surfaces: Cowork cloud sends an empty `transcript_path` (D-295, D-296) and an empty first prompt (open.md); a plugin shipped with `defaultEnabled: false` runs no hooks (`9477067`); the cloud proxy needs a WebSocket relay (`e415f1d`); the account's marketplace copy sat at v0.10.4 for a month (`db8eced`, D-290); `!` command lines are disabled in Cowork (`e7ddab7`); the plain form of an `mcp_tool` reply does not reach the model (`5ac6450`).
- A hook's `systemMessage` is shown to the person, and a failed hook is loud (`11b1294`).
- None of B01 to B21 has been recorded failing.

## 3. The simulation

### Model

24 months, 20,000 runs for the main results, standard library only, run with `python3 -I`. 188 s for the main run plus three stability seeds, and about 150 s for the sensitivity sweep.

- Capabilities: 17, one per group of B-numbers (`params.json`, `caps`), each on the surfaces where it exists. Each month a capability can break silently with nothing produced (absent), silently with wrong output (corrupt), or loudly; it can be removed; or it can be absent from the start and arrive by improvement (Codex cloud hooks).
- Scope: a break hits every surface with probability 0.75, otherwise one surface. Removals always hit every surface.
- Correlated shocks: each month a vendor may redesign, Anthropic 0.015 and OpenAI 0.03. A redesign hits each of that vendor's capabilities with probability 0.5: 50% loud, 30% removal, 20% silent. It is announced, so noticed within about 0.1 months, and adapting takes about 2 months.
- Detection is the earliest of four channels: `cogmer doctor` (mean 0.3 months, 90% of breaks in a checked capability, only on the local surface where doctor runs, never in a cloud container, D-297); users (mean 1.5 months for a silent failure, 0.25 for a loud one); the portfolio's own noise when re-offering announces a missing injection (mean 0.25 months); and a cross-check, for portfolios that have one (80% of corrupt transcripts within about 0.1 months).
- End of a break: cogmer adapts at detection plus a mean of 0.5 months (plus a 1-month release lag for Cowork cloud), succeeding 90% of the time; half of all breaks are vendor regressions fixed in a mean of 0.3 months; a removal ends only if a replacement arrives (0.10 a month) and cogmer adapts.
- Common random numbers across portfolios; state sampled four times a month.
- Capture is "full" (prompt, text before tool calls, final text), "partial" (prompt and final text), "none" or "wrong"; it works when full or partial. Injection is "ok", "noisy" (delivered but offered again), "none" (not delivered, offered again, so it announces itself) or "lost" (counted delivered and never received, or misframed); it works when ok or noisy.
- A month counts as silent while capture or injection fails and no active break behind it is known to anyone.
- Cost: build cost plus 1 pw per adaptation, 3 per removal and 4 per shock, all assumed.

Rates that rest only on judgment: the transcript hazard (0.05), MessageDisplay and OTel hazards, the share of regressions, all detection and adaptation means, both shock rates, and all costs.

### Sanity checks (run before the simulation, in `sim.py`)

| check | result |
|---|---|
| Every hazard zero: every portfolio works in every substep, has clean injection, the "full" level its rules give with nothing broken, and no silent month; portfolios resting on a capability absent at the start (Codex cloud) do not work in month 1 | pass |
| Each of the 17 capabilities removed with probability 1 from month 1: every portfolio whose capture or injection needs it fails in every month, every other portfolio is unaffected | pass after one fix (below) |
| Seeds: three seeds at 5,000 runs each differ by at most 0.011 in a fraction, 0.042 months in silent months and 0.28 pw in cost | stable within ±0.012 and ±0.045 months at 5,000 runs |
| Intervals against differences: 95% half-widths at 20,000 runs are 0.003 for a fraction and 0.006 to 0.020 months for silent months; paired differences have half-widths of 0.0005 to 0.009, smaller than every difference reported except where marked indistinguishable | pass |

Three faults were found and fixed; all results come from the fixed model:

1. A removal could be drawn as hitting one surface, so a capability removed with certainty still worked elsewhere in month 1. Removals now hit every surface.
2. A payload-only option reported "full" capture in months with no breaks. It now uses the level its rules give with nothing broken, and the zero-hazard check tests it.
3. The bridge's capability (B30, B31) was modeled as existing only in Cowork, so `doctor --host cowork` could never catch a break, overstating CW1's silent months (1.10 before, 0.61 after). `out-base/sensitivity.json` predates this fix, so its CW rows are superseded by `out-cw/`.

### Results (base parameters, 20,000 runs, ± is the 95% half-width)

| id | host | option | capture works | capture full | injection works | injection clean | silent months / 24 | cost pw |
|---|---|---|---|---|---|---|---|---|
| CC1 | Claude Code local | current | 0.881 ± 0.003 | 0.794 | 0.876 | 0.815 | 0.30 ± 0.01 | 10.7 |
| CC2 | | payloads only, delivery at Stop | 0.862 | 0 | 0.872 | 0.826 | 0.35 ± 0.01 | 8.6 |
| CC3 | | current + cross-check | 0.890 | 0.796 | 0.876 | 0.815 | 0.24 ± 0.01 | 11.7 |
| CC4 | | Stop + MessageDisplay, evidence | 0.885 | 0.775 | 0.876 | 0.815 | 0.26 | 12.8 |
| CC5 | | Stop + MessageDisplay, delivery at Stop | 0.885 | 0.775 | 0.872 | 0.826 | 0.31 | 13.0 |
| CC6 | | cross-check + MessageDisplay as third source | 0.887 | 0.845 | 0.876 | 0.815 | 0.26 | 16.1 |
| CC7 | | OTel capture | 0.873 | 0.873 | 0.876 | 0.815 | 0.34 | 10.9 |
| CL1 | Claude Code cloud | current | 0.744 | 0.675 | 0.743 | 0.694 | 1.12 ± 0.02 | 14.0 |
| CL2 | | + cross-check | 0.752 | 0.676 | 0.743 | 0.694 | 1.06 ± 0.02 | 15.0 |
| CW1 | Cowork local | current (bridge) | 0.786 | 0 | 0.795 | 0.755 | 0.61 ± 0.01 | 10.6 |
| CW2 | | plugin command hooks | 0.860 | 0 | 0.870 | 0.824 | 0.40 ± 0.01 | 9.1 |
| CW3 | | CW2 + MessageDisplay | 0.883 | 0.773 | 0.870 | 0.824 | 0.36 ± 0.01 | 13.0 |
| KC1 | Cowork cloud | current | 0.678 | 0 | 0.686 | 0.644 | 1.14 ± 0.02 | 11.4 |
| KC2 | | + MessageDisplay | 0.696 | 0.600 | 0.686 | 0.644 | 1.02 ± 0.02 | 15.3 |
| CX1 | Codex local | current | 0.855 | 0.648 | 0.820 | 0.645 | 0.31 ± 0.01 | 12.9 |
| CX2 | | payloads only, delivery at Stop | 0.824 | 0 | 0.818 | 0.818 | 0.38 | 8.1 |
| CX3 | | cross-check; delivery at Stop while rollout unreadable | 0.868 | 0.646 | 0.820 | 0.820 | 0.22 ± 0.01 | 14.4 |
| XC1 | Codex cloud | command hooks if they ever arrive | 0.229 | 0 | 0.227 | 0.227 | 0.34 | 13.6 |

Paired differences against the current option on the same host (`out-base/results.json`, `paired_diffs`):

- CC3 − CC1: capture works +0.0089 ± 0.0005, silent months −0.057 ± 0.004, cost +1.0 pw; injection unchanged.
- CC6 − CC1: full capture +0.051 ± 0.002, silent −0.042 ± 0.005, cost +5.4 pw. CC6 and CC3 cannot be told apart on silence, and CC6 costs 4.4 pw more for 5 points of full capture.
- CC2 − CC1: silent +0.052 ± 0.007; full capture lost entirely.
- CC7 − CC1: silent +0.037 ± 0.009.
- CW2 − CW1: capture +0.074 ± 0.002, injection +0.075 ± 0.002, silent −0.21 ± 0.01, cost −1.5 pw.
- KC2 − KC1: full capture +0.60, silent −0.12 ± 0.01, cost +3.9 pw.
- CX3 − CX1: clean injection +0.175 ± 0.003, silent −0.087 ± 0.005, cost +1.5 pw.
- CX2 − CX1: silent +0.072 ± 0.008.
- CL2 − CL1: silent −0.060 ± 0.004.

On a local Claude Code session, injection works 87.6% of the time under every option; the options differ in capture and in whether a failure is seen.

### Sensitivity (one at a time, 2,500 runs each, `out-base/sensitivity.txt` and `out-cw/`)

Silent months for CC1, then CC3:

| case | CC1 | CC3 |
|---|---|---|
| base | 0.29 | 0.23 |
| users notice silence in 0.5 months | 0.18 | 0.15 |
| users notice silence in 4 months | 0.42 | 0.35 |
| doctor lag 1 month | 0.51 | 0.41 |
| transcript hazard 0.15 | 0.43 | 0.26 |
| transcript hazard 0.02 | 0.23 | 0.21 |
| all hazards ×2 | 0.45 | 0.36 |
| all hazards ×0.5 | 0.16 | 0.13 |

- Cloud silent months range from 0.48 to 2.2 across the cases, depending almost entirely on how fast users notice and on the cloud-environment hazard. At a cloud-environment hazard of 0.20 a month, Cowork cloud capture works only 57% of the time.
- Shocks ×4 cut capture-works on every host by 13 to 18 points, the largest single effect on availability. Silent months fall in that case, because announced redesigns mask silent breaks.
- Codex rollout hazard 0.30 raises CX1's silent months from 0.30 to 0.51 and CX3's only from 0.22 to 0.25.
- MessageDisplay hazard at 0.12 a month erases CC6's advantage (0.30 silent, against CC1's 0.29).
- Ranking: CC3 ≤ CC1, CX3 < CX1, CW2 < CW1 and KC2 < KC1 on silent months in all 16 cases.

## 4. Recommendation

Ranked by benefit per person-week, all compatible with the values:

1. CC3, CL2, CX3: cross-check the transcript or rollout against `last_assistant_message`. About a fifth to a quarter fewer silent months, and for Codex an end to re-offering whenever the rollout format moves; 1 to 1.5 pw; CX3's injection half revises D-202's Limits.
2. CW2: plugin command hooks for capture and injection in local Cowork, with the bridge kept for commands. About 7 points of availability for capture and injection and a third fewer silent months; about 1 pw to build and 1.5 pw less upkeep; needs a decision beside D-205 and a probe that the hooks fire across Desktop versions.
3. KC2 and CW3: MessageDisplay for text before a tool call where no transcript exists, after experiment 1. Full capture in Cowork, which has none today, and 0.12 fewer cloud silent months; about 2 to 4 pw.
4. Not now: CC6, OTel, payload-only on a local session.

What to monitor:

- A canary inside cloud sessions: the daemon checks its own hooks' invariants every turn (Stop arrives with a `prompt_id` matching an open prompt, B29; `transcript_path` empty or present as expected for the surface; the prompt non-empty after the first turn; the last message non-empty) and reports a violation once through a hook's `systemMessage` (`11b1294`).
- A behavior check on every Claude Code and Codex release, not only when a room is formed, triggered by the npm registry's `time` field and the GitHub releases API.
- Keyword watches on the Claude Code changelog and Codex release notes for `UserPromptSubmit`, `Stop`, `last_assistant_message`, `transcript`, `additionalContext`, `hook output`, `mcp_tool`, `MessageDisplay`, `rollout` and `session_id`; `history/mine_changelog.py` is a start.
- New B-numbers before relying on them: MessageDisplay fires per text message, with session_id, not for subagents, at no latency; an `http` hook to an unreachable daemon shows nothing (B11's analogue); the transcript's final text block equals `last_assistant_message`; in Codex, a hook's `session_id` equals `CODEX_THREAD_ID` after 0.131 (B28).

Experiments, in order:

1. MessageDisplay latency and coverage in interactive Claude Code, desktop, a cloud session and Cowork.
2. Local Cowork with plugin command hooks over several Desktop versions, and whether a transcript path exists there.
3. What an `http` hook does when the daemon is down.
4. How long a silent failure goes unnoticed in practice: record, for each future vendor break, when it began, when doctor saw it and when a user did.
5. A Codex cloud task and a ChatGPT Work local thread.

## What the history shows and what was assumed

Shown: Claude Code adds a relevant hook capability about every two months and has removed none; about 8 adverse changes to documented hook behavior in 16 months; `mcp_tool` changed six times in seven months; Codex reworked its rollout format at least four times in seven months and says it is unstable; OpenAI has twice retired a developer surface, once with three weeks' notice; cogmer's one silent failure in use (B34) came from an intentional, documented change.

Assumed (each marked in `params.json`): how those counts become per-capability monthly rates; the transcript, MessageDisplay and OTel rates; the redesign shock rates and spread; every detection, adaptation and vendor-fix delay; all costs. The model treats breaks as independent apart from shocks, lets a break last until cogmer adapts or the vendor fixes it, samples four times a month, counts noisy re-offering as working, and does not model display latency or the model's behavior (B19, B22).
