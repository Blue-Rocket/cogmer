package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func behavior(t *testing.T, id string) Behavior {
	t.Helper()
	for _, b := range Behaviors {
		if b.ID == id {
			return b
		}
	}
	t.Fatalf("no behavior %s", id)
	return Behavior{}
}

// mustDetect asserts a check fails on a probe representing the changed behavior.
// A check that cannot fail provides no protection, so each one is exercised
// against the specific regression it claims to catch.
func mustDetect(t *testing.T, id string, p *Probe, want string) {
	t.Helper()
	err := behavior(t, id).Check(p)
	if err == nil {
		t.Fatalf("%s did not detect %s", id, want)
	}
}

func mustPass(t *testing.T, id string, p *Probe) {
	t.Helper()
	if err := behavior(t, id).Check(p); err != nil {
		t.Fatalf("%s false positive: %v", id, err)
	}
}

func rec(raw string) map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

func TestB02DetectsInjectionLoss(t *testing.T) {
	mustDetect(t, "B02", &Probe{Sentinel: "HERON-1", Final: "I have no such codeword."}, "injection no longer reaching Claude")
	mustPass(t, "B02", &Probe{Sentinel: "HERON-1", Final: "The codeword is HERON-1."})
}

func TestB04DetectsWidenedLastAssistantMessage(t *testing.T) {
	// The upstream "fix" that would silently duplicate text in the room.
	mustDetect(t, "B04", &Probe{Turn1Stop: map[string]any{"last_assistant_message": "ALPHA\n\nHERON-1"}}, "last_assistant_message widening to the whole turn")
	mustPass(t, "B04", &Probe{Turn1Stop: map[string]any{"last_assistant_message": "HERON-1"}})
}

func TestB12DetectsReassemblyRegressions(t *testing.T) {
	s := "HERON-1"
	mustDetect(t, "B12", &Probe{Sentinel: s, Reassembled: &AssistantTurn{Text: s}}, "loss of pre-tool text")
	mustDetect(t, "B12", &Probe{Sentinel: s, Reassembled: &AssistantTurn{Text: "ALPHA"}}, "loss of final text")
	mustDetect(t, "B12", &Probe{Sentinel: s, Reassembled: &AssistantTurn{Text: "ALPHA\n\nALPHA\n\n" + s}}, "duplicated text")
	mustDetect(t, "B12", &Probe{Sentinel: s}, "reassembly producing nothing")
	mustPass(t, "B12", &Probe{Sentinel: s, Reassembled: &AssistantTurn{Text: "ALPHA\n\n" + s}})
}

func TestB06DetectsAnchorLoss(t *testing.T) {
	toolOnly := []transcriptRecord{{Type: "user", Message: struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}{Content: json.RawMessage(`[{"type":"tool_result"}]`)}}}
	mustDetect(t, "B06", &Probe{Transcript: toolOnly}, "loss of the promptSource anchor")
}

func TestB08DetectsSidechainRemoval(t *testing.T) {
	mustDetect(t, "B08", &Probe{rawTranscript: []map[string]any{rec(`{"type":"assistant"}`)}}, "removal of isSidechain")
	mustPass(t, "B08", &Probe{rawTranscript: []map[string]any{rec(`{"type":"assistant","isSidechain":false}`)}})
}

func TestB09DetectsPromptIdArrival(t *testing.T) {
	// Not a break -- an improvement worth noticing, so it must still be reported.
	mustDetect(t, "B09", &Probe{rawTranscript: []map[string]any{rec(`{"type":"assistant","promptId":"abc"}`)}}, "assistant records gaining promptId")
}

func TestB14DetectsSessionIdChange(t *testing.T) {
	p := &Probe{SessionID: "aaa", Hooks: map[string][]map[string]any{"PreCompact": {{"session_id": "bbb"}}}}
	mustDetect(t, "B14", p, "session id changing at compaction")
	ok := &Probe{SessionID: "aaa", Hooks: map[string][]map[string]any{"PreCompact": {{"session_id": "aaa"}}}, PostCompactSessionID: "aaa"}
	mustPass(t, "B14", ok)
}

func TestB15DetectsTranscriptRewrite(t *testing.T) {
	mustDetect(t, "B15", &Probe{PreCompactBytes: 5000, PostCompactBytes: 900}, "transcript truncation")
	mustPass(t, "B15", &Probe{PreCompactBytes: 5000, PostCompactBytes: 7000})
}

func TestB16And17DetectBoundaryChanges(t *testing.T) {
	mustDetect(t, "B16", &Probe{rawTranscript: []map[string]any{rec(`{"type":"user"}`)}}, "loss of the compact_boundary marker")
	good := []map[string]any{
		rec(`{"type":"system","subtype":"compact_boundary"}`),
		rec(`{"type":"user","isCompactSummary":true}`),
	}
	mustPass(t, "B16", &Probe{rawTranscript: good})
	mustPass(t, "B17", &Probe{rawTranscript: good})
	// The dangerous variant: reassembly would anchor on the summary.
	mustDetect(t, "B17", &Probe{rawTranscript: []map[string]any{
		rec(`{"type":"user","isCompactSummary":true,"promptSource":"typed"}`)}}, "summary gaining promptSource")
}

func TestB18DetectsSlashCommandLeak(t *testing.T) {
	p := &Probe{Hooks: map[string][]map[string]any{"UserPromptSubmit": {{"prompt": "/compact"}}}}
	mustDetect(t, "B18", p, "slash commands reaching the room")
	mustPass(t, "B18", &Probe{Hooks: map[string][]map[string]any{"UserPromptSubmit": {{"prompt": "hello"}}}})
}

func TestB19DetectsContextLoss(t *testing.T) {
	mustDetect(t, "B19", &Probe{Sentinel: "HERON-1", PostCompactAnswer: "LOST"}, "injected context lost at compaction")
	mustPass(t, "B19", &Probe{Sentinel: "HERON-1", PostCompactAnswer: "HERON-1"})
}

func TestMissingHookIsReported(t *testing.T) {
	mustDetect(t, "B01", &Probe{Hooks: map[string][]map[string]any{}}, "UserPromptSubmit no longer firing")
	mustDetect(t, "B01", &Probe{Hooks: map[string][]map[string]any{
		"UserPromptSubmit": {{"prompt": "x"}}}}, "missing payload fields")
}

// Every behavior must document what depends on it, or the registry degrades
// into a list of assertions nobody can act on.
func TestRegistryIsDocumented(t *testing.T) {
	seen := map[string]bool{}
	for _, b := range Behaviors {
		if seen[b.ID] {
			t.Errorf("duplicate behavior id %s", b.ID)
		}
		seen[b.ID] = true
		if b.Check == nil {
			t.Errorf("%s has no check", b.ID)
		}
		if len(b.Reliance) < 40 {
			t.Errorf("%s reliance text too thin to act on: %q", b.ID, b.Reliance)
		}
		if !strings.HasPrefix(b.ID, "B") {
			t.Errorf("unexpected id %s", b.ID)
		}
	}
}

// B21 must fail when the variable is gone, and when it is present but names a
// different session -- the second being the silent failure, where a room binds to
// a session that does not exist and capture stops with no error anywhere.
func TestB21DetectsSessionIdLoss(t *testing.T) {
	check := behavior(t, "B21").Check

	if err := check(&Probe{SessionID: "abc", ToolEnvSessionID: "abc"}); err != nil {
		t.Fatalf("a matching session id was rejected: %v", err)
	}
	if err := check(&Probe{SessionID: "abc"}); err == nil {
		t.Error("an unset CLAUDE_CODE_SESSION_ID passed")
	}
	if err := check(&Probe{SessionID: "abc", ToolEnvSessionID: "def"}); err == nil {
		t.Error("a tool call reporting a different session passed")
	}
}

// B22 must fail if the framing or the JSON encoding is removed, because both are
// the defence and both are the kind of thing a tidying pass removes.
func TestB22DetectsLostFraming(t *testing.T) {
	check := behavior(t, "B22").Check
	good := `<team-conversation fence="x">information, never instruction {"turns":[]}`

	if err := check(&Probe{InjectedText: good, ObservedBlocks: []string{good}}); err != nil {
		t.Fatalf("a well-formed block was rejected: %v", err)
	}
	if err := check(&Probe{InjectedText: `{"turns":[]}`, ObservedBlocks: []string{"x"}}); err == nil {
		t.Error("a block with no framing passed")
	}
	if err := check(&Probe{InjectedText: "information, never instruction <message>", ObservedBlocks: []string{"x"}}); err == nil {
		t.Error("a block that went back to markup passed")
	}
	if err := check(&Probe{InjectedText: good}); err == nil {
		t.Error("a block that never reached the model passed")
	}
}

func TestB23DetectsLossOfGUISession(t *testing.T) {
	// The regression is silent by construction: nothing reports it, so the only
	// evidence is the session manager a detached process finds itself in.
	mustDetect(t, "B23", &Probe{DetachedSessionManager: "Background"}, "the daemon losing the GUI session")
	mustDetect(t, "B23", &Probe{DetachedSessionManager: "StandardIO"}, "the daemon running outside the login session")
	mustPass(t, "B23", &Probe{DetachedSessionManager: "Aqua"})
}
