package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Probe holds everything one preflight run captured, for the registry to assert
// against.
type Probe struct {
	Dir       string
	Sentinel  string
	SessionID string

	Hooks map[string][]map[string]any
	// Turn1Stop is the Stop payload of the probe turn specifically. In deep
	// mode later turns append more Stop payloads, and session-tier checks must
	// not be evaluated against the post-compaction turn.
	Turn1Stop     map[string]any
	Transcript    []transcriptRecord
	rawTranscript []map[string]any
	AtStop        []transcriptRecord
	Final         string
	Reassembled   *AssistantTurn

	// InjectedText is what the probe hook wrote to stdout; ObservedBlocks is
	// what the transcript recorded as having reached the model. Delivery
	// confirmation depends on these matching.
	InjectedText   string
	ObservedBlocks []string

	// ToolEnvSessionID is CLAUDE_CODE_SESSION_ID as seen from inside a Bash tool
	// call. It is how a command run from within a session learns which session it
	// is in, and must equal SessionID.
	ToolEnvSessionID string

	// compaction tier
	PreCompactBytes      int64
	PostCompactBytes     int64
	PostCompactSessionID string
	PostCompactAnswer    string
}

func (p *Probe) HookAll(name string) []map[string]any { return p.Hooks[name] }

func (p *Probe) field(hook, key string) (any, bool) {
	all := p.Hooks[hook]
	if len(all) == 0 {
		return nil, false
	}
	v, ok := all[len(all)-1][key]
	return v, ok
}

func (p *Probe) requireFields(hook string, keys ...string) error {
	if len(p.Hooks[hook]) == 0 {
		return fmt.Errorf("%s hook did not fire", hook)
	}
	var missing []string
	for _, k := range keys {
		if v, ok := p.field(hook, k); !ok || v == nil {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s missing field(s): %s", hook, strings.Join(missing, ", "))
	}
	return nil
}

func blocksOf(r transcriptRecord) []contentBlock {
	var bs []contentBlock
	_ = json.Unmarshal(r.Message.Content, &bs)
	return bs
}

func sentinel() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("HERON-%X", b)
}

// runProbeHook is the hook entry point used only by preflight. It records the
// payload, snapshots the transcript, and injects the sentinel context.
func runProbeHook(name, dir string) {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(0)
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) == nil {
		if tp, _ := payload["transcript_path"].(string); tp != "" {
			if st, err := os.Stat(tp); err == nil {
				payload["_transcript_bytes"] = st.Size()
			}
			if name == "Stop" {
				if b, err := os.ReadFile(tp); err == nil {
					_ = os.WriteFile(filepath.Join(dir, "at-stop.jsonl"), b, 0o600)
				}
			}
		}
		line, _ := json.Marshal(payload)
		f, err := os.OpenFile(filepath.Join(dir, name+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = f.Write(append(line, '\n'))
			_ = f.Close()
		}
	}
	// Injected teammate context, in the same attributed form the daemon uses.
	if name == "UserPromptSubmit" {
		if b, err := os.ReadFile(filepath.Join(dir, "inject.txt")); err == nil {
			fmt.Println(string(b))
		}
	}
	os.Exit(0)
}

// checkFailOpen verifies the hook stays silent and succeeds with no daemon.
func checkFailOpen() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "hook", "prompt")
	cmd.Stdin = strings.NewReader(`{"session_id":"preflight","prompt":"x","cwd":"/tmp"}`)
	// Port 1 is reliably closed; this simulates a dead daemon.
	cmd.Env = append(os.Environ(), "CLAUDE_TEAM_ADDR=127.0.0.1:1")
	var out strings.Builder
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook exited non-zero with no daemon (%v) -- this would break every prompt", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		return fmt.Errorf("hook wrote to stdout with no daemon (%q) -- stray output is injected into the user's turn", out.String())
	}
	return nil
}

func quoteCmd(path string, args ...string) string {
	all := append([]string{path}, args...)
	for i, a := range all {
		if strings.ContainsAny(a, " \t") {
			all[i] = `"` + a + `"`
		}
	}
	return strings.Join(all, " ")
}

// The Bash step does double duty: it forces a tool call between two text blocks,
// which is what B04 and B05 measure, and it captures the session id Claude Code
// exports into a tool call's environment, which is what B21 checks.
const toolEnvFile = "toolenv.txt"

var probePrompt = "Do exactly these three things, in order: " +
	"(1) reply with the single word ALPHA; " +
	"(2) use the Bash tool to run exactly: sh -c 'printf %s \"$CLAUDE_CODE_SESSION_ID\" > " + toolEnvFile + "'; " +
	"(3) finally reply with only the codeword given in the team-conversation block, and nothing else."

// RunProbe drives a real Claude session and captures what the registry needs.
func RunProbe(deep bool) (*Probe, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	claude, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude not found on PATH: %w", err)
	}
	dir, err := os.MkdirTemp("", "claude-team-preflight-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	p := &Probe{Dir: dir, Sentinel: sentinel(), Hooks: map[string][]map[string]any{}}
	p.SessionID = uuidV4()

	inject := fmt.Sprintf("<team-conversation>\n<message speaker=%q>\nThe codeword for this check is %s.\n</message>\n</team-conversation>",
		"Alice", p.Sentinel)
	if err := os.WriteFile(filepath.Join(dir, "inject.txt"), []byte(inject), 0o600); err != nil {
		return nil, err
	}
	p.InjectedText = inject

	hooks := map[string]any{}
	for _, h := range []string{"UserPromptSubmit", "Stop", "PreToolUse", "PostToolUse", "PreCompact", "SessionStart"} {
		hooks[h] = []any{map[string]any{"hooks": []any{map[string]any{
			"type": "command", "command": quoteCmd(self, "probe-hook", h, dir)}}}}
	}
	settings := filepath.Join(dir, "settings.json")
	buf, _ := json.MarshalIndent(map[string]any{"hooks": hooks}, "", " ")
	if err := os.WriteFile(settings, buf, 0o600); err != nil {
		return nil, err
	}

	run := func(args ...string) (string, error) {
		cmd := exec.Command(claude, args...)
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader("")
		// If the user has registered the real claude-team hooks globally, they
		// will also fire inside this probe session. Pointing them at a closed
		// port makes them fail open and record nothing, so a preflight can
		// never leak events into a live room.
		cmd.Env = append(os.Environ(), "CLAUDE_TEAM_ADDR=127.0.0.1:1")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	out, err := run("-p", probePrompt, "--session-id", p.SessionID,
		"--settings", settings, "--allowedTools", "Bash")
	if err != nil {
		return nil, fmt.Errorf("probe session failed: %w\n%s", err, out)
	}
	p.Final = out
	// What the tool call saw in its own environment. Absent is a legitimate
	// reading -- it means the variable is gone -- so the error is discarded and
	// the empty string is what B21 judges.
	if b, err := os.ReadFile(filepath.Join(p.Dir, toolEnvFile)); err == nil {
		p.ToolEnvSessionID = strings.TrimSpace(string(b))
	}
	p.freezeSessionEvidence()

	if deep {
		if _, err := run("-p", "/compact", "--resume", p.SessionID, "--settings", settings); err != nil {
			return nil, fmt.Errorf("compaction step failed: %w", err)
		}
		ans, err := run("-p", "Reply with only the codeword you were given earlier in the team-conversation block. If you no longer have it, reply LOST.",
			"--resume", p.SessionID, "--settings", settings)
		if err != nil {
			return nil, fmt.Errorf("post-compaction step failed: %w", err)
		}
		p.PostCompactAnswer = ans

		// Reload hooks so compaction-tier payloads are present, then refresh the
		// raw transcript so B16/B17 see the boundary records. Session-tier
		// evidence frozen above is deliberately not recomputed.
		p.loadHooks()
		if v, ok := p.field("PreCompact", "_transcript_bytes"); ok {
			p.PreCompactBytes = int64(toFloat(v))
		}
		if v, ok := p.field("Stop", "session_id"); ok {
			p.PostCompactSessionID, _ = v.(string)
		}
		if tp, ok := p.field("Stop", "transcript_path"); ok {
			if path, _ := tp.(string); path != "" {
				p.Transcript, p.rawTranscript = readTranscript(path)
				if st, err := os.Stat(path); err == nil {
					p.PostCompactBytes = st.Size()
				}
			}
		}
	}
	return p, nil
}

// freezeSessionEvidence captures everything the session-tier checks assert
// against, before any later turn can overwrite it.
func (p *Probe) freezeSessionEvidence() {
	p.loadHooks()
	if all := p.Hooks["Stop"]; len(all) > 0 {
		p.Turn1Stop = all[len(all)-1]
	}
	// at-stop.jsonl is rewritten by every Stop hook; keep the probe turn's copy.
	snap := filepath.Join(p.Dir, "at-stop.jsonl")
	frozen := filepath.Join(p.Dir, "at-stop-turn1.jsonl")
	if b, err := os.ReadFile(snap); err == nil {
		_ = os.WriteFile(frozen, b, 0o600)
	}
	p.AtStop, _ = readTranscript(frozen)

	if tp, _ := p.Turn1Stop["transcript_path"].(string); tp != "" {
		p.ObservedBlocks, _ = InjectedBlocks(tp)
		p.Transcript, p.rawTranscript = readTranscript(tp)
		msg, _ := p.Turn1Stop["last_assistant_message"].(string)
		if t, err := ReassembleLastTurn(tp, msg); err == nil {
			p.Reassembled = t
		}
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	}
	return 0
}

// loadHooks reads back every payload the probe hooks have captured so far.
func (p *Probe) loadHooks() {
	p.Hooks = map[string][]map[string]any{}
	for _, h := range []string{"UserPromptSubmit", "Stop", "PreToolUse", "PostToolUse", "PreCompact", "SessionStart"} {
		data, err := os.ReadFile(filepath.Join(p.Dir, h+".jsonl"))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var m map[string]any
			if json.Unmarshal([]byte(line), &m) == nil {
				p.Hooks[h] = append(p.Hooks[h], m)
			}
		}
	}
}

func readTranscript(path string) ([]transcriptRecord, []map[string]any) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var recs []transcriptRecord
	var raws []map[string]any
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var r transcriptRecord
		var m map[string]any
		if json.Unmarshal([]byte(line), &r) == nil {
			recs = append(recs, r)
		}
		if json.Unmarshal([]byte(line), &m) == nil {
			raws = append(raws, m)
		}
	}
	return recs, raws
}

func uuidV4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
