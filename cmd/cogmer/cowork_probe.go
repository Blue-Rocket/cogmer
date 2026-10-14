package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// What a Cowork session relies on is what the Claude Code harness does with a hook of type
// mcp_tool, and no check can run inside Cowork, so the checks run against the Claude Code
// on the user's machine, which is the same harness (D-206). One isolated session starts a
// small MCP server of this binary and fires hooks at it.

// CoworkBehaviors are the behaviors a Cowork session relies on, checked on this machine.
var CoworkBehaviors = []Behavior{
	{
		ID:       "B30",
		Title:    "A Claude Code session adds the JSON form of an mcp_tool hook's reply to the model's context",
		Reliance: "If this changes, a colleague's turns reach a Cowork session's model not at all, while every call succeeds and every turn is counted delivered when its stop call arrives, so the loss is silent. Once doctor has recorded this check as failing for the installed version, the daemon stops counting a Cowork turn delivered without evidence, and the turns are offered again at every prompt, which is noisy and loses nothing. The plain form of the same reply is recorded as a successful hook and does not reach the model, and a change there would not break this.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			if p.McpErr != nil {
				return fmt.Errorf("the session did not complete: %v", p.McpErr)
			}
			if !strings.Contains(p.McpAnswer, p.McpJSONWord) {
				return fmt.Errorf("the model's context did not hold the reply an mcp_tool hook returned in the JSON form, so the bridge's reply reaches nobody; the model answered %q", firstLine(p.McpAnswer))
			}
			return nil
		},
	},
	{
		ID:       "B31",
		Title:    "A hook of type mcp_tool that names a server that is not there shows nothing and leaves the session working",
		Reliance: "If this changes, a Cowork user whose bridge is stopped, crashed or not yet started sees an error or a failed session on every prompt, which breaks the rule that cogmer never makes a session worse. A hook that calls an absent server is what a stopped bridge looks like to the harness.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			if p.McpErr != nil {
				return fmt.Errorf("a hook that named an absent server failed the session: %v", p.McpErr)
			}
			if strings.Contains(p.McpAnswer, p.McpAbsentServer) {
				return fmt.Errorf("the session printed something about the absent server %q: %q", p.McpAbsentServer, firstLine(p.McpAnswer))
			}
			return nil
		},
	},
	{
		ID:       "B32",
		Title:    "A hook of type mcp_tool on SessionStart is not called at launch",
		Reliance: "If this changes, nothing breaks: a Cowork bridge needs nothing at launch, because the first prompt's call is the first that reaches it. A call at launch would let the bridge start the daemon, and learn of a session, before its first prompt, which could replace what it does on the first call.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			for _, c := range p.McpCalls {
				if strings.Contains(c, "event=SessionStart") {
					return fmt.Errorf("an mcp_tool hook on SessionStart now reaches its server at launch, which the Cowork bridge could use")
				}
			}
			if len(p.McpCalls) == 0 {
				return fmt.Errorf("no hook reached the server at all, so this check saw nothing")
			}
			return nil
		},
	},
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 160 {
		s = s[:160] + "…"
	}
	return s
}

// coworkAbsentServer is the name a probe hook gives a server that does not exist.
const coworkAbsentServer = "cogmer-absent-server"

// RunCoworkProbe runs one isolated Claude Code session with hooks of type mcp_tool and
// returns what it saw. A session that fails for a reason that has nothing to do with the
// absent server is not a reading of anything, and is returned as an error.
func RunCoworkProbe() (*Probe, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	claude, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude not found on PATH: %w", err)
	}
	dir, err := os.MkdirTemp("", "cogmer-cowork-probe-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	tag := strings.TrimPrefix(sentinel(), "HERON-")
	p := &Probe{Dir: dir, McpJSONWord: "JSONFORM-" + tag, McpAbsentServer: coworkAbsentServer}
	plainWord := "PLAINFORM-" + tag

	hook := func(server string, input map[string]any) map[string]any {
		return map[string]any{"type": "mcp_tool", "server": server, "tool": "ping", "input": input, "timeout": 15}
	}
	settings := map[string]any{"hooks": map[string]any{
		"SessionStart": []any{map[string]any{"hooks": []any{hook("probe", map[string]any{"event": "SessionStart", "format": "plain", "word": plainWord})}}},
		"UserPromptSubmit": []any{map[string]any{"hooks": []any{
			hook("probe", map[string]any{"event": "UserPromptSubmit", "format": "plain", "word": plainWord}),
			hook("probe", map[string]any{"event": "UserPromptSubmit", "format": "json", "word": p.McpJSONWord}),
			hook(coworkAbsentServer, map[string]any{"event": "UserPromptSubmit"}),
		}}},
	}}
	mcp := map[string]any{"mcpServers": map[string]any{"probe": map[string]any{
		"command": self, "args": []string{"probe-mcp", dir}}}}
	write := func(name string, v any) (string, error) {
		path := filepath.Join(dir, name)
		buf, _ := json.MarshalIndent(v, "", " ")
		return path, os.WriteFile(path, buf, 0o600)
	}
	settingsPath, err := write("settings.json", settings)
	if err != nil {
		return nil, err
	}
	mcpPath, err := write("mcp.json", mcp)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf("Quote every line in your context that contains the text FORM-%s. If there are none, reply with only the word NONE.", tag)
	cmd := exec.Command(claude, claudeArgs("-p", prompt, "--settings", settingsPath,
		"--mcp-config", mcpPath, "--strict-mcp-config")...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("")
	// The user's own cogmer hooks are not loaded, and pointing them at a closed port as well
	// means a preflight can never leak an event into a live room.
	cmd.Env = append(os.Environ(), "COGMER_ADDR=127.0.0.1:1")
	out, runErr := cmd.CombinedOutput()
	p.McpAnswer = string(out)
	if runErr != nil {
		if strings.Contains(p.McpAnswer, coworkAbsentServer) {
			p.McpErr = fmt.Errorf("%v: %s", runErr, firstLine(p.McpAnswer))
		} else {
			return nil, fmt.Errorf("probe session failed: %w\n%s", runErr, p.McpAnswer)
		}
	}
	if b, err := os.ReadFile(filepath.Join(dir, "mcp-calls.log")); err == nil {
		for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			if l != "" {
				p.McpCalls = append(p.McpCalls, l)
			}
		}
	}
	return p, nil
}

// runCoworkChecks is RunCoworkChecks, as a variable so a test can stand in for a real
// Claude Code turn.
var runCoworkChecks = RunCoworkChecks

// RunCoworkChecks runs the Cowork behaviors against one isolated Claude Code session.
func RunCoworkChecks() ([]Result, error) {
	p, err := RunCoworkProbe()
	if err != nil {
		return nil, err
	}
	var results []Result
	for _, b := range CoworkBehaviors {
		results = append(results, Result{B: b, Err: b.Check(p)})
	}
	return results, nil
}

func coworkVerificationKey() string { return "cowork:" + ClaudeVersion() }

func recordCoworkVerification(results []Result) verification {
	failures := map[string]string{}
	for _, r := range results {
		if r.Err != nil {
			failures[r.B.ID] = r.Err.Error()
		}
	}
	v := verification{Version: coworkVerificationKey(), When: time.Now().UTC().Format(time.RFC3339), Failures: failures}
	saveVerification(v)
	return v
}

// runCoworkDoctor checks the Claude Code on this machine for what Cowork relies on and
// says whether every check passed.
func runCoworkDoctor() bool {
	version := ClaudeVersion()
	if version == "unknown" {
		fmt.Println("cogmer doctor: there is no Claude Code on this machine, so what Cowork relies on cannot be checked here.")
		fmt.Println("Until it is, a Cowork turn counts as delivered when its stop call arrives, and nothing guards that.")
		return false
	}
	fmt.Printf("cogmer doctor — Claude Code %s, as Cowork's harness\nDriving one isolated Claude turn.\n\n", version)
	results, err := runCoworkChecks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cogmer: the Cowork probe could not run: %v\n", err)
		return false
	}
	v := recordCoworkVerification(results)
	for _, r := range results {
		if r.Err == nil {
			fmt.Printf("  ok    %s  %s\n", r.B.ID, r.B.Title)
		} else {
			fmt.Printf("  FAIL  %s  %s\n        %v\n        relied on for: %s\n\n", r.B.ID, r.B.Title, r.Err, r.B.Reliance)
		}
	}
	fmt.Printf("\n%d checked, %d failed\n", len(results), len(v.Failures))
	return len(v.Failures) == 0
}

// EnsureCoworkVerified runs the Cowork checks the first time a room is formed from a Cowork
// session on a Claude Code version this machine has not checked, as the other hosts' do (D-129).
// A machine with no Claude Code has nothing to check.
func EnsureCoworkVerified(room string) {
	if strings.EqualFold(os.Getenv("COGMER_PREFLIGHT"), "off") || ClaudeVersion() == "unknown" {
		return
	}
	key := coworkVerificationKey()
	if v, ok := loadVerifications()[key]; ok && !probeDue(v, time.Now()) {
		if why, couldNotRun := v.Failures[probeCouldNotRun]; couldNotRun && len(v.Failures) == 1 {
			fmt.Fprintf(os.Stderr, "cogmer: the Cowork behavior check could not run (%s); it is tried again after an hour, or run cogmer doctor --host cowork\n", why)
		} else if len(v.Failures) > 0 {
			fmt.Fprintln(os.Stderr, "cogmer: Cowork behavior checks failed; run cogmer doctor --host cowork for detail")
		}
		return
	}
	fmt.Fprintf(os.Stderr, "cogmer: checking what Cowork relies on in this Claude Code with one isolated turn\n")
	results, err := runCoworkChecks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cogmer: the Cowork probe could not run: %v\n", err)
		recordProbeNotRun(key, err)
		return
	}
	if v := recordCoworkVerification(results); len(v.Failures) > 0 {
		fmt.Fprintln(os.Stderr, "cogmer: Cowork behavior checks failed; collaboration continues, and Cowork turns are offered again until they pass")
	}
}
