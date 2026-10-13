package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// A host is the application a session runs in: Claude Code or Codex. A daemon serves
// the sessions of every host on the machine at once, so the host belongs to a
// request or a session and the daemon holds none (D-203).
//
// What differs between hosts is gathered here, so that nothing else compares a host's
// name: how a session is identified, how a completed turn and the evidence of an
// injection are read from the host's transcript, how a prompt hook hands context to
// the model, what a command is called, and what the host's behaviors and checks are.
type Host interface {
	Name() string
	// SessionEnv is the environment variable that carries a session's identifier into
	// a tool the session runs.
	SessionEnv() string
	// SessionID is the identifier a session of this host is stored under. Sessions of
	// different hosts on one machine cannot share one.
	SessionID(raw string) string
	// NormalizeHook checks a hook payload and rewrites the session and prompt
	// identifiers to the form this host's sessions are stored under.
	NormalizeHook(session, prompt *string, turn string) error
	// ReassembleTurn reads a completed assistant turn from the host's transcript.
	ReassembleTurn(path, lastMessage, turnID string) (*AssistantTurn, error)
	// InjectedBlocks reads from the transcript what the host recorded as injected.
	InjectedBlocks(path string) ([]string, error)
	// PromptHookOutput is what a prompt hook writes to its standard output so that the
	// host gives the context to the model.
	PromptHookOutput(context string) []byte
	// CommandText rewrites text that names a command into this host's way of invoking
	// one.
	CommandText(text string) string
	// Executable is what is run to ask the host for its version.
	Executable() string
	Installed() bool
	Version() string
	// MayTrustWithoutEvidence reports whether a turn offered to a session of this host
	// version may be counted as delivered when the transcript shows no evidence.
	MayTrustWithoutEvidence(version string) bool
	Behaviors() []Behavior
	// Preflight runs the host's behavior checks the first time a room is formed under
	// a version this machine has not checked, if the host has such a step.
	Preflight(room string)
	// Doctor runs the host's checks and says whether they all passed.
	Doctor(deep bool) (bool, error)
}

// hosts is every host cogmer supports.
var hosts = []Host{claudeHost{}, codexHost{}}

// hostByName finds a host by the name a hook or a wrapper declares. A hook that
// declares none is Claude Code's, which is every hook installed before there was a
// second host.
func hostByName(name string) (Host, bool) {
	if name == "" {
		name = "claude"
	}
	for _, h := range hosts {
		if h.Name() == name {
			return h, true
		}
	}
	return nil, false
}

// --- finding the host of the process a command runs in ---

// AmbiguousHostError is returned when the environment of a command carries the
// session variables of more than one host and nothing declares which is meant.
type AmbiguousHostError struct{ Hosts []string }

func (e *AmbiguousHostError) Error() string {
	return fmt.Sprintf("cannot tell which host this command was run from: the environment holds the session of %s and nothing says which is meant. "+
		"Run it through the plugin's command.", strings.Join(e.Hosts, " and "))
}

// hostFromEnv decides which host a command is running in. An entry point that cogmer
// wrote says so with COGMER_HOST, and that wins, because each host loads only the
// wrapper cogmer wrote for it. Without it, the host is the one whose session variable
// is set, and a process that holds the variables of two hosts, as a Claude Code run
// started from inside a Codex shell does, is refused and not guessed at. A terminal
// holds neither, and has no host.
func hostFromEnv(getenv func(string) string) (Host, error) {
	if name := strings.TrimSpace(getenv("COGMER_HOST")); name != "" {
		h, ok := hostByName(name)
		if !ok {
			return nil, fmt.Errorf("COGMER_HOST names no host cogmer supports: %q", name)
		}
		return h, nil
	}
	var found []Host
	for _, h := range hosts {
		if getenv(h.SessionEnv()) != "" {
			found = append(found, h)
		}
	}
	switch len(found) {
	case 0:
		return nil, nil
	case 1:
		return found[0], nil
	}
	var names []string
	for _, h := range found {
		names = append(names, h.Name())
	}
	return nil, &AmbiguousHostError{Hosts: names}
}

// currentHost is the host of this process, nil at a terminal.
func currentHost() (Host, error) { return hostFromEnv(os.Getenv) }

// sessionID is the session a command was invoked from, empty at a terminal and
// whenever the host cannot be told. It is stored in the form of the host that owns it.
func sessionID() string {
	h, err := currentHost()
	if err != nil || h == nil {
		return ""
	}
	raw := os.Getenv(h.SessionEnv())
	if raw == "" {
		return ""
	}
	return h.SessionID(raw)
}

// noSessionNote adds to a message about a missing session the reason, when the reason
// is that the host could not be told.
func noSessionNote() string {
	if _, err := currentHost(); err != nil {
		return "\n" + err.Error()
	}
	return ""
}

// commandText writes a command the way the host this process runs in invokes it.
func commandText(text string) string {
	if h, err := currentHost(); err == nil && h != nil {
		return h.CommandText(text)
	}
	return text
}

// --- what to check ---

// doctorTargets says which hosts `doctor` checks. One that is named is checked.
// Otherwise the host a session is running in, and at a terminal every host that is
// installed.
func doctorTargets(explicit string, current Host, installed func(Host) bool) ([]Host, error) {
	if explicit != "" {
		h, ok := hostByName(explicit)
		if !ok {
			return nil, fmt.Errorf("unknown host %q; cogmer supports %s", explicit, hostNames())
		}
		return []Host{h}, nil
	}
	if current != nil {
		return []Host{current}, nil
	}
	var out []Host
	for _, h := range hosts {
		if installed(h) {
			out = append(out, h)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no supported host is installed here (%s); install one, or name it with --host", hostNames())
	}
	return out, nil
}

func hostNames() string {
	var names []string
	for _, h := range hosts {
		names = append(names, h.Name())
	}
	return strings.Join(names, ", ")
}

// --- host facts the daemon needs ---

// hostVersions remembers each host's version for a few minutes, because the daemon
// outlives the hosts' upgrades and serves several hosts. A version that is not known
// yet is "", and the lookup runs in the background, so that a hook is never delayed by
// running another program. Unknown has to fall on the safe side wherever it is used.
type hostVersions struct {
	mu      sync.Mutex
	entries map[string]versionEntry
	pending map[string]bool
	lookup  func(Host) string // Host.Version when nil
	ttl     time.Duration     // ten minutes when zero
	now     func() time.Time  // time.Now when nil
}

type versionEntry struct {
	version string
	at      time.Time
}

const hostVersionTTL = 10 * time.Minute

// hostVersion is the last version known of a host, which may be stale and may be "".
func (d *Daemon) hostVersion(h Host) string {
	v := &d.versions
	v.mu.Lock()
	defer v.mu.Unlock()
	now := time.Now
	if v.now != nil {
		now = v.now
	}
	ttl := v.ttl
	if ttl == 0 {
		ttl = hostVersionTTL
	}
	name := h.Name()
	e, known := v.entries[name]
	if (!known || now().Sub(e.at) >= ttl) && !v.pending[name] {
		if v.pending == nil {
			v.pending = map[string]bool{}
		}
		v.pending[name] = true
		d.inBackground(func() {
			lookup := v.lookup
			if lookup == nil {
				lookup = func(h Host) string { return h.Version() }
			}
			version := lookup(h)
			v.mu.Lock()
			defer v.mu.Unlock()
			if v.entries == nil {
				v.entries = map[string]versionEntry{}
			}
			v.entries[name] = versionEntry{version: version, at: now()}
			delete(v.pending, name)
		})
	}
	return e.version
}

// --- Claude Code ---

type claudeHost struct{}

func (claudeHost) Name() string                     { return "claude" }
func (claudeHost) SessionEnv() string               { return "CLAUDE_CODE_SESSION_ID" }
func (claudeHost) SessionID(raw string) string      { return raw }
func (claudeHost) CommandText(text string) string   { return text }
func (claudeHost) Executable() string               { return "claude" }
func (claudeHost) Behaviors() []Behavior            { return Behaviors }
func (claudeHost) Version() string                  { return ClaudeVersion() }
func (claudeHost) Installed() bool                  { _, err := exec.LookPath("claude"); return err == nil }
func (claudeHost) PromptHookOutput(c string) []byte { return []byte(c + "\n") }

func (claudeHost) NormalizeHook(session, prompt *string, turn string) error { return nil }

func (claudeHost) ReassembleTurn(path, lastMessage, _ string) (*AssistantTurn, error) {
	return ReassembleLastTurn(path, lastMessage)
}

func (claudeHost) InjectedBlocks(path string) ([]string, error) { return InjectedBlocks(path) }

// MayTrustWithoutEvidence is true only for a version on which B20 is recorded as
// failing. "No evidence" is otherwise as consistent with an injection that never
// arrived as with a changed transcript format, and committing on trust would lose that
// context silently.
func (claudeHost) MayTrustWithoutEvidence(version string) bool {
	return version != "" && BehaviorKnownBroken(version, "B20")
}

// Preflight runs the session-tier checks the first time a room is formed under a
// Claude Code version this machine has not checked (D-129).
func (claudeHost) Preflight(room string) { EnsureVerified(room) }

func (claudeHost) Doctor(deep bool) (bool, error) { return runDoctor(deep), nil }

// --- Codex ---

type codexHost struct{}

func (codexHost) Name() string                { return "codex" }
func (codexHost) SessionEnv() string          { return "CODEX_THREAD_ID" }
func (codexHost) SessionID(raw string) string { return "codex:" + raw }
func (codexHost) Executable() string          { return codexExecutable() }
func (codexHost) Behaviors() []Behavior       { return CodexBehaviors }
func (codexHost) Version() string             { return CodexVersion() }
func (codexHost) Installed() bool             { _, err := exec.LookPath(codexExecutable()); return err == nil }
func (codexHost) Preflight(room string)       { EnsureCodexVerified() }

func (codexHost) CommandText(text string) string {
	return strings.ReplaceAll(text, "/cogmer:", "$cogmer ")
}

// NormalizeHook requires the identifiers Codex's hooks carry. A turn is identified by
// turn_id, which stands in for the prompt identifier.
func (c codexHost) NormalizeHook(session, prompt *string, turn string) error {
	if *session == "" || turn == "" {
		return fmt.Errorf("Codex hooks require session_id and turn_id")
	}
	*session = c.SessionID(*session)
	*prompt = turn
	return nil
}

func (codexHost) ReassembleTurn(path, lastMessage, turnID string) (*AssistantTurn, error) {
	return ReassembleCodexTurn(path, lastMessage, turnID)
}

func (codexHost) InjectedBlocks(path string) ([]string, error) { return CodexInjectedBlocks(path) }

// PromptHookOutput is the JSON Codex reads a hook's context from.
func (codexHost) PromptHookOutput(context string) []byte {
	b, _ := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
		"hookEventName": "UserPromptSubmit", "additionalContext": context,
	}})
	return append(b, '\n')
}

// MayTrustWithoutEvidence is never true for Codex: missing evidence leaves a turn
// pending, whatever a behavior check found (D-202).
func (codexHost) MayTrustWithoutEvidence(string) bool { return false }

var errDeepUnsupported = fmt.Errorf("the compaction checks are not available for Codex")

func (codexHost) Doctor(deep bool) (bool, error) {
	if deep {
		return true, errDeepUnsupported
	}
	return runCodexDoctor(), nil
}
