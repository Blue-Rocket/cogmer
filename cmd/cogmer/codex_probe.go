package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var CodexBehaviors = []Behavior{
	{ID: "B25", Title: "Codex prompt and Stop hooks carry matching session and turn identifiers",
		Reliance: "If this changes, prompts and responses cannot be associated with their session or pending injection. Nothing in the session reaches the room.", Tier: TierSession,
		Check: func(p *Probe) error {
			for _, hook := range []string{"UserPromptSubmit", "Stop"} {
				if err := p.requireFields(hook, "session_id", "turn_id", "transcript_path"); err != nil {
					return err
				}
			}
			if err := p.requireFields("UserPromptSubmit", "prompt"); err != nil {
				return err
			}
			if err := p.requireFields("Stop", "last_assistant_message"); err != nil {
				return err
			}
			for _, field := range []string{"session_id", "turn_id"} {
				a, _ := p.field("UserPromptSubmit", field)
				b, _ := p.field("Stop", field)
				if a != b || a == "" {
					return fmt.Errorf("Codex hooks disagree on %s", field)
				}
			}
			return nil
		}},
	{ID: "B26", Title: "Codex rollouts and Stop together contain commentary, tool calls and final text",
		Reliance: "If this changes, the room records an incomplete response or counts no tools. Reading only the final answer loses what the assistant said before a tool call.", Tier: TierSession,
		Check: func(p *Probe) error {
			if p.Reassembled == nil || !strings.Contains(p.Reassembled.Text, "ALPHA") || !strings.Contains(p.Reassembled.Text, "OMEGA") || len(p.Reassembled.ToolCalls) == 0 {
				return fmt.Errorf("Codex reassembly lost commentary, final text or the tool call")
			}
			return nil
		}},
	{ID: "B27", Title: "Codex records injected context as a hook-marked message and the model receives it",
		Reliance: "If this changes, a colleague's turns are never confirmed as delivered and are offered again. An altered or truncated block cannot confirm delivery of the complete block.", Tier: TierSession,
		Check: func(p *Probe) error {
			if p.Sentinel == "" || !strings.Contains(p.Final, p.Sentinel) {
				return fmt.Errorf("Codex did not return the injected codeword")
			}
			for _, b := range p.ObservedBlocks {
				if HashBlock(b) == HashBlock(p.InjectedText) {
					return nil
				}
			}
			return fmt.Errorf("Codex transcript contains no matching hook-context evidence")
		}},
	{ID: "B28", Title: "CODEX_THREAD_ID in a shell tool equals the hook session identifier",
		Reliance: "If this changes, a room command cannot identify the session that invoked it. The command refuses rather than choosing another session's room.", Tier: TierSession,
		Check: func(p *Probe) error {
			if p.SessionID == "" || p.ToolEnvSessionID != p.SessionID {
				return fmt.Errorf("CODEX_THREAD_ID %q differs from hook session %q", p.ToolEnvSessionID, p.SessionID)
			}
			return nil
		}},
}

func codexExecutable() string {
	if path := os.Getenv("COGMER_CODEX_BIN"); path != "" {
		return path
	}
	return "codex"
}

func CodexVersion() string {
	out, err := exec.Command(codexExecutable(), "--version").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// This entry point is reachable only from a locally requested behavior probe.
func runCodexProbeHook(name, dir string) {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) != nil {
		return
	}
	line, _ := json.Marshal(payload)
	f, err := os.OpenFile(filepath.Join(dir, name+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err == nil {
		_, _ = f.Write(append(line, '\n'))
		_ = f.Close()
	}
	if name == "Stop" {
		if path, _ := payload["transcript_path"].(string); path != "" {
			if b, err := os.ReadFile(path); err == nil {
				_ = os.WriteFile(filepath.Join(dir, "at-stop.jsonl"), b, 0o600)
			}
		}
	}
	if name == "UserPromptSubmit" {
		b, err := os.ReadFile(filepath.Join(dir, "inject.txt"))
		if err == nil {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"hookSpecificOutput": map[string]string{"hookEventName": name, "additionalContext": string(b)}})
		}
	} else {
		fmt.Println("{}")
	}
}

// codexProbeArgs is how the probe starts Codex. It ignores the user's Codex
// configuration, which is where installed plugins are enabled, so the real cogmer
// plugin's hooks cannot fire inside the probe session: they would write to the
// user's own state directory and start nothing useful. The hooks that run are the
// three given here, on the command line. Authentication still comes from
// CODEX_HOME. The session is not ephemeral, because the probe reads its transcript
// afterwards.
func codexProbeArgs(self, dir string) []string {
	args := []string{"exec", "--ignore-user-config", "--dangerously-bypass-hook-trust", "--skip-git-repo-check", "-s", "workspace-write", "-C", dir, "--output-last-message", filepath.Join(dir, "final.txt")}
	for _, event := range []string{"SessionStart", "UserPromptSubmit", "Stop"} {
		command := quoteCmd(self, "codex-probe-hook", event, dir)
		args = append(args, "-c", fmt.Sprintf("hooks.%s=[{hooks=[{type=\"command\",command=%s,timeout=10,additionalContextLimit=0}]}]", event, strconv.Quote(command)))
	}
	return append(args, "This is a cogmer behavior check. Say ALPHA in commentary, then use a shell tool to write the value of CODEX_THREAD_ID to tool-session.txt in the working directory. Reply OMEGA followed by the codeword from the team-conversation block. Do nothing else.")
}

// runCodexChecks is RunCodexChecks, as a variable so a test can stand in for a real
// Codex turn.
var runCodexChecks = RunCodexChecks

func RunCodexChecks() ([]Result, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "cogmer-codex-probe-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	p := &Probe{Dir: dir, Sentinel: sentinel()}
	p.InjectedText = FormatTeamContext([]Event{{PeerID: "ed25519:probe", UserDisplayName: "Alice", EventType: EventUserPrompt,
		Timestamp: time.Now().UTC().Format(time.RFC3339), Content: "The codeword for this check is " + p.Sentinel + "."}}, fixedFacts{verified: true})
	if err := os.WriteFile(filepath.Join(dir, "inject.txt"), []byte(p.InjectedText), 0o600); err != nil {
		return nil, err
	}
	args := codexProbeArgs(self, dir)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, codexExecutable(), args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("")
	cmd.Env = append(os.Environ(), "COGMER_ADDR=127.0.0.1:1", "COGMER_PREFLIGHT=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Codex probe failed: %w\n%s", err, out)
	}
	p.loadHooks()
	if all := p.Hooks["Stop"]; len(all) > 0 {
		p.Turn1Stop = all[len(all)-1]
	}
	p.SessionID, _ = p.Turn1Stop["session_id"].(string)
	turnID, _ := p.Turn1Stop["turn_id"].(string)
	last, _ := p.Turn1Stop["last_assistant_message"].(string)
	p.Reassembled, _ = ReassembleCodexTurn(filepath.Join(dir, "at-stop.jsonl"), last, turnID)
	if path, _ := p.Turn1Stop["transcript_path"].(string); path != "" {
		p.ObservedBlocks, _ = CodexInjectedBlocks(path)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "final.txt")); err == nil {
		p.Final = string(b)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "tool-session.txt")); err == nil {
		p.ToolEnvSessionID = strings.TrimSpace(string(b))
	}
	var results []Result
	for _, b := range CodexBehaviors {
		results = append(results, Result{B: b, Err: b.Check(p)})
	}
	return results, nil
}

func recordCodexVerification(results []Result) verification {
	v := verification{Version: "codex:" + CodexVersion(), When: time.Now().UTC().Format(time.RFC3339), Failures: map[string]string{}}
	for _, r := range results {
		if r.Err != nil {
			v.Failures[r.B.ID] = r.Err.Error()
		}
	}
	saveVerification(v)
	return v
}

// runCodexDoctor checks Codex and says whether every check passed.
func runCodexDoctor() bool {
	fmt.Printf("cogmer doctor: %s\nDriving one isolated Codex turn.\n", CodexVersion())
	results, err := RunCodexChecks()
	if err != nil {
		logCodexProbeFailure(err)
		return false
	}
	v := recordCodexVerification(results)
	for _, r := range results {
		if r.Err == nil {
			fmt.Printf("  ok    %s  %s\n", r.B.ID, r.B.Title)
		} else {
			fmt.Printf("  FAIL  %s  %v\n        %s\n", r.B.ID, r.Err, r.B.Reliance)
		}
	}
	return len(v.Failures) == 0
}

func logCodexProbeFailure(err error) {
	fmt.Fprintf(os.Stderr, "cogmer: Codex probe could not run: %v\n", err)
}

func EnsureCodexVerified() {
	if strings.EqualFold(os.Getenv("COGMER_PREFLIGHT"), "off") {
		return
	}
	key := "codex:" + CodexVersion()
	if v, ok := loadVerifications()[key]; ok && !probeDue(v, time.Now()) {
		if why, couldNotRun := v.Failures[probeCouldNotRun]; couldNotRun && len(v.Failures) == 1 {
			fmt.Fprintf(os.Stderr, "cogmer: the Codex behavior check could not run (%s); it is tried again after an hour, or run cogmer doctor --host codex\n", why)
		} else if len(v.Failures) > 0 {
			fmt.Fprintln(os.Stderr, "cogmer: Codex behavior checks failed; run cogmer doctor --host codex for detail")
		}
		return
	}
	fmt.Fprintln(os.Stderr, "cogmer: checking this Codex version with one isolated turn")
	results, err := runCodexChecks()
	if err != nil {
		logCodexProbeFailure(err)
		// Recorded, so that the next room command does not run another turn.
		recordProbeNotRun(key, err)
		return
	}
	v := recordCodexVerification(results)
	if len(v.Failures) > 0 {
		fmt.Fprintln(os.Stderr, "cogmer: Codex behavior checks failed; collaboration continues unverified")
	}
}
