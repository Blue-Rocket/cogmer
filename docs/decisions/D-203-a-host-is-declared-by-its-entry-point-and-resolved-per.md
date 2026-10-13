# D-203 — A host is declared by its entry point and resolved per request, and the daemon holds none

**Date:** 2026-10-12 · **Status:** active · **Areas:** hosts, daemon, commands

**Decision.** Each entry point cogmer wrote for a host declares which host it is for. A hook line takes
`--host`, a command's wrapper sets `COGMER_HOST`, and `cli.sh` declares Claude Code when nothing else has. The
hook client puts the host in the payload, and the daemon resolves it against the hosts it supports on every
request, so a hook that declares none is Claude Code's. A command with no declaration takes the host whose
session variable is set, refuses when two are set, and has no host at a terminal. What differs between hosts
is behind one `Host` value, and the daemon holds none, so one daemon serves the sessions of every host at once.
`doctor` checks the host it is given, the host in use, or each installed host.

**Support.**
- Each host loads only the hook file and the wrapper cogmer wrote for it, so a declaration made there cannot
  be for the other host. `plugin/hooks/codex-hooks.json`, `plugin/codex-cli.sh`, `plugin/cli.sh`.
- The session variables of both hosts can be in one process, as when a Claude Code run starts inside a Codex
  shell, and choosing between them binds a command to the wrong session. B21 (CLAUDE_CODE_SESSION_ID is
  exported into a tool call's environment), B28 (CODEX_THREAD_ID equals the hook session identifier),
  `cmd/cogmer/host.go`, `hostFromEnv`, `cmd/cogmer/host_test.go`,
  `TestTheHostOfACommandIsDeclaredOrTheOnlyOneInTheEnvironment`.
- A machine runs both hosts, and one daemon serves a room whose sessions belong to each.
  D-201 (Claude Code and Codex are supported hosts), `cmd/cogmer/codex_test.go`,
  `TestMixedHostRoomCapturesAndDeliversBothDirections`.
- A version held for the daemon would be one host's, read once at start, wrong for any other host and stale
  after an upgrade, so each host's version is looked up in the background and refreshed every ten minutes. An
  unknown version reads as empty and leaves turns pending, and a hook is never delayed by running another
  program. `cmd/cogmer/host.go`, `hostVersion`, `cmd/cogmer/host_test.go`,
  `TestAHostVersionIsLookedUpInTheBackgroundAndRefreshed`.
- A turn with no transcript evidence is counted as delivered only for Claude Code on a version where B20 is
  recorded as failing, and never for Codex. D-202 (host-specific hooks use the shared peer protocol),
  `cmd/cogmer/host_test.go`, `TestDeliveryWithoutEvidenceIsTrustedOnlyWhereThatHostsCheckFailed`.
- A third host adds an entry to the list of hosts and no option or comparison. `cmd/cogmer/host.go`, `hosts`,
  `doctorTargets`.
- Breaking the declaration, the refusal of an ambiguous environment, the default host of a hook, either host's
  delivery rule, the refresh of a version, the choice of hosts for `doctor`, the form of Codex's hook output and
  the default in `cli.sh` each failed a test. `cmd/cogmer/host_test.go`.

**Rejected.**
- *Inferring the host from the shape of a payload or from whichever session variable is set.* Hooks of the two
  hosts carry overlapping fields, and both variables can be set at once.
- *A daemon started for one host.* A machine runs both, and a daemon outlives sessions and host upgrades.
- *A flag for each host on `doctor` and a comparison of the host's name at each place it matters.* Five places
  decided the host and had to agree, so a third host would have been five more edits, and one wrong edit
  misbehaves silently.

**Limits.** Claude Code's preflight is not run, because nothing calls it. `cmd/cogmer/host.go`,
`claudeHost.Preflight`. A host version is read by running the host's executable, so a machine without it
reports unknown. Plain `doctor` at a terminal checks every installed host, and so spends a turn on each.

**Revisit when** a host cannot declare itself in its hook line or its wrapper.
