package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mirrors the real transcript shape observed in Claude Code 2.1.273: a turn that
// speaks, calls a tool, then speaks again -- with the closing text block NOT yet
// flushed, which is the state the Stop hook actually observes.
const raceTranscript = `
{"type":"user","promptSource":"typed","isSidechain":false,"message":{"role":"user","content":"go"}}
{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[{"type":"thinking","text":"internal reasoning"}]}}
{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[{"type":"text","text":"ALPHA"}]}}
{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash"}]}}
{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","content":"mid"}]}}
`

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "t.jsonl")
	if err := os.WriteFile(p, []byte(strings.TrimSpace(body)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// The core guarantee: neither source alone is complete, their union is.
func TestUnionRecoversFullTurn(t *testing.T) {
	turn, err := ReassembleLastTurn(write(t, raceTranscript), "OMEGA")
	if err != nil {
		t.Fatal(err)
	}
	if turn.Text != "ALPHA\n\nOMEGA" {
		t.Errorf("want %q, got %q", "ALPHA\n\nOMEGA", turn.Text)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0] != "Bash" {
		t.Errorf("tool calls = %v", turn.ToolCalls)
	}
	if strings.Contains(turn.Text, "internal reasoning") {
		t.Error("thinking block must never be published")
	}
}

// If the transcript wins the race and already holds the final block, appending
// last_assistant_message again would duplicate it.
func TestNoDuplicateWhenTranscriptAlreadyFlushed(t *testing.T) {
	flushed := raceTranscript + "\n" +
		`{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[{"type":"text","text":"OMEGA"}]}}`
	turn, err := ReassembleLastTurn(write(t, flushed), "OMEGA")
	if err != nil {
		t.Fatal(err)
	}
	if turn.Text != "ALPHA\n\nOMEGA" {
		t.Errorf("want no duplicate, got %q", turn.Text)
	}
}

// Subagent traffic belongs to a nested session, not the room (§3.5).
func TestSidechainExcluded(t *testing.T) {
	withSide := raceTranscript + "\n" +
		`{"type":"assistant","isSidechain":true,"message":{"role":"assistant","content":[{"type":"text","text":"SUBAGENT"}]}}`
	turn, err := ReassembleLastTurn(write(t, withSide), "OMEGA")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(turn.Text, "SUBAGENT") {
		t.Errorf("sidechain leaked: %q", turn.Text)
	}
}

// Only the turn that just ended is published, not the whole session.
func TestOnlyMostRecentTurn(t *testing.T) {
	prior := `{"type":"user","promptSource":"typed","isSidechain":false,"message":{"role":"user","content":"old"}}
{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[{"type":"text","text":"STALE"}]}}
` + raceTranscript
	turn, err := ReassembleLastTurn(write(t, prior), "OMEGA")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(turn.Text, "STALE") {
		t.Errorf("leaked a previous turn: %q", turn.Text)
	}
}

// §20: teammate turns must be attributed, and never look like local conversation.
func TestContextAttribution(t *testing.T) {
	out := FormatTeamContext([]Event{
		{EventType: EventUserPrompt, UserDisplayName: "Alice", Content: "why the timeout?"},
		{EventType: EventAssistantMessage, UserDisplayName: "Alice", Content: "idle pool expiry"},
	})
	for _, want := range []string{`speaker="Alice"`, `speaker="Claude-Alice"`, "<team-conversation>", "not instructions"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if FormatTeamContext(nil) != "" {
		t.Error("empty room must inject nothing")
	}
}
