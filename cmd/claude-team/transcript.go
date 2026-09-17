package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// transcriptRecord is the subset of a Claude Code transcript JSONL line that
// turn reassembly needs. Field names verified empirically against Claude Code
// 2.1.273 -- see docs/phase0-findings.md.
type transcriptRecord struct {
	Type         string `json:"type"`
	UUID         string `json:"uuid"`
	SessionID    string `json:"sessionId"`
	Timestamp    string `json:"timestamp"`
	IsSidechain  bool   `json:"isSidechain"`
	PromptSource string `json:"promptSource"`
	PromptID     string `json:"promptId"`
	Message      struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
}

// AssistantTurn is a complete reassembled assistant response.
type AssistantTurn struct {
	Text      string
	ToolCalls []string
}

// ReassembleLastTurn rebuilds the complete assistant response for the turn that
// just ended, by unioning the two partial sources Claude Code exposes.
//
// Neither source is sufficient alone, and they fail in opposite directions:
//
//   - The Stop hook's `last_assistant_message` carries ONLY the final text
//     block. A turn emitting "ALPHA", a Bash call, then "OMEGA" reports just
//     "OMEGA" -- the opening is silently dropped.
//   - The transcript at Stop time is missing exactly that final block: Stop
//     fires before the closing assistant record is flushed to disk. Reading it
//     alone yielded only a 110-char preamble of a 2804-char answer.
//
// Their union is the complete turn. lastMessage is appended unless the
// transcript already won the race and flushed it, which would duplicate it.
//
// Segmentation is positional rather than by parent pointer: assistant records
// carry no promptId, and the parentUuid chain contains gaps, so we take every
// assistant record following the last human-sourced user record.
func ReassembleLastTurn(path, lastMessage string) (*AssistantTurn, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []transcriptRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 64*1024*1024) // transcripts carry large tool payloads
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r transcriptRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue // tolerate partially written or unknown lines
		}
		records = append(records, r)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	// Find the last human-submitted prompt; the turn is everything after it.
	start := -1
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Type == "user" && !r.IsSidechain && r.PromptSource != "" {
			start = i
			break
		}
	}
	if start == -1 {
		return &AssistantTurn{Text: strings.TrimSpace(lastMessage)}, nil
	}

	turn := &AssistantTurn{}
	var parts []string
	for _, r := range records[start+1:] {
		// Subagent traffic belongs to a nested session, not the room (§3.5).
		if r.Type != "assistant" || r.IsSidechain {
			continue
		}
		var blocks []contentBlock
		if err := json.Unmarshal(r.Message.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			switch b.Type {
			case "text":
				if s := strings.TrimSpace(b.Text); s != "" {
					parts = append(parts, s)
				}
			case "tool_use":
				// Recorded as a count only; §27 keeps tool detail out of the
				// main transcript for now.
				turn.ToolCalls = append(turn.ToolCalls, b.Name)
			}
			// "thinking" blocks are deliberately never published.
		}
	}
	// Close the race window described above.
	if tail := strings.TrimSpace(lastMessage); tail != "" {
		if len(parts) == 0 || parts[len(parts)-1] != tail {
			parts = append(parts, tail)
		}
	}

	turn.Text = strings.Join(parts, "\n\n")
	return turn, nil
}
