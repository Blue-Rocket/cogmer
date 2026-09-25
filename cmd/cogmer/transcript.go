package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// transcriptRecord is the subset of a Claude Code transcript JSONL line that
// turn reassembly needs. Field names verified empirically against Claude Code
// 2.1.273; B06 to B09 in behaviors.go check them against the installed version.
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
		// Subagent traffic belongs to a nested session, not the room (§15).
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
	turn.Text = mergeTail(strings.Join(parts, "\n\n"), lastMessage)
	return turn, nil
}

// mergeTail closes the race window without assuming how much of the turn
// lastMessage contains.
//
// Today lastMessage is only the final text block, so it must be appended. But if
// Claude Code ever widens it to the whole turn, blind appending would duplicate
// everything already read from the transcript -- an upstream bugfix would
// silently corrupt the room. Behavior check B04 detects that change; this
// function survives it either way.
func mergeTail(joined, lastMessage string) string {
	tail := strings.TrimSpace(lastMessage)
	switch {
	case tail == "":
		return joined
	case joined == "":
		return tail
	case strings.Contains(tail, joined):
		return tail // lastMessage is a superset: it already holds the whole turn
	case strings.HasSuffix(joined, tail):
		return joined // transcript won the race and already flushed the final block
	default:
		return joined + "\n\n" + tail
	}
}

// attachmentRecord is how Claude Code records a hook's stdout once it has been
// added to the conversation. Its presence is proof the injected block reached
// the model -- the evidence this project's delivery tracking is derived from,
// rather than assuming delivery succeeded because the hook was called.
//
// Verified against 2.1.273; checked by behavior B20.
type attachmentRecord struct {
	Type       string `json:"type"`
	Attachment struct {
		Type      string `json:"type"`
		HookName  string `json:"hookName"`
		HookEvent string `json:"hookEvent"`
		Content   string `json:"content"`
	} `json:"attachment"`
}

// InjectedBlocks returns every UserPromptSubmit hook output recorded in a
// transcript, in file order.
//
// Attachments accumulate across turns and survive compaction, so a block missed
// at its own Stop is still observable later. That is what makes confirmation
// self-healing rather than single-shot.
func InjectedBlocks(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 64*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.Contains(line, "hook_success") {
			continue
		}
		var r attachmentRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		if r.Type != "attachment" || r.Attachment.Type != "hook_success" {
			continue
		}
		if r.Attachment.HookName != "UserPromptSubmit" && r.Attachment.HookEvent != "UserPromptSubmit" {
			continue
		}
		if c := strings.TrimSpace(r.Attachment.Content); c != "" {
			out = append(out, c)
		}
	}
	return out, sc.Err()
}
