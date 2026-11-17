package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
)

// These are observed rollout fields, not a stable Codex API. doctor --host codex
// checks them against a real session. In particular, event_msg duplicates many
// response_item records and must never be used as a second source of messages.
type codexRecord struct {
	Type    string `json:"type"`
	Payload struct {
		Type    string          `json:"type"`
		TurnID  string          `json:"turn_id"`
		Role    string          `json:"role"`
		ID      string          `json:"id"`
		Name    string          `json:"name"`
		CallID  string          `json:"call_id"`
		Source  json.RawMessage `json:"source"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Metadata struct {
			TurnID string   `json:"turn_id"`
			Kinds  []string `json:"content_item_kinds"`
		} `json:"internal_chat_message_metadata_passthrough"`
	} `json:"payload"`
}

func readCodexRecords(path string) ([]codexRecord, error) {
	f, err := openTranscript(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 64*1024*1024)
	var records []codexRecord
	for sc.Scan() {
		var r codexRecord
		if json.Unmarshal(sc.Bytes(), &r) == nil {
			records = append(records, r)
		}
	}
	return records, sc.Err()
}

// ReassembleCodexTurn reads only the requested turn, using task_started rather
// than user-role messages: Codex also records environment context as user text.
// Explicit turn metadata, when available, excludes traffic from other turns.
// Reasoning, tool arguments/results, and event-message copies are never shared.
func ReassembleCodexTurn(path, lastMessage, turnID string) (*AssistantTurn, error) {
	if turnID == "" {
		return nil, fmt.Errorf("cannot capture a Codex turn without turn_id")
	}
	records, err := readCodexRecords(path)
	if err != nil {
		return nil, err
	}
	start := -1
	for i, r := range records {
		if r.Type == "session_meta" && strings.Contains(string(r.Payload.Source), `"subagent"`) {
			return nil, fmt.Errorf("a Codex subagent transcript is not a room session")
		}
		if r.Type == "event_msg" && r.Payload.Type == "task_started" && r.Payload.TurnID == turnID {
			start = i
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("Codex transcript has no task_started for turn %q; run doctor --host codex", turnID)
	}
	turn := &AssistantTurn{}
	var parts []string
	seen := map[string]bool{}
	for _, r := range records[start+1:] {
		p := r.Payload
		if r.Type == "event_msg" && p.Type == "task_started" && p.TurnID != turnID {
			break
		}
		if r.Type != "response_item" || (p.Metadata.TurnID != "" && p.Metadata.TurnID != turnID) {
			continue
		}
		key := p.ID
		if p.Type == "function_call" || p.Type == "custom_tool_call" {
			key = p.CallID
		}
		if key != "" {
			key = p.Type + ":" + key
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		switch p.Type {
		case "message":
			if p.Role == "assistant" {
				for _, c := range p.Content {
					if c.Type == "output_text" && strings.TrimSpace(c.Text) != "" {
						parts = append(parts, strings.TrimSpace(c.Text))
					}
				}
			}
		case "function_call", "custom_tool_call":
			turn.ToolCalls = append(turn.ToolCalls, p.Name)
		}
	}
	turn.Text = mergeTail(strings.Join(parts, "\n\n"), lastMessage)
	return turn, nil
}

// CodexInjectedBlocks accepts only developer messages that Codex marks as hook
// context. A peer's text echoed by the assistant, or pasted by the user, is not
// evidence of delivery. A format we cannot recognize leaves injections pending.
func CodexInjectedBlocks(path string) ([]string, error) {
	records, err := readCodexRecords(path)
	if err != nil {
		return nil, err
	}
	var blocks []string
	for _, r := range records {
		p := r.Payload
		if r.Type != "response_item" || p.Type != "message" || p.Role != "developer" {
			continue
		}
		hook := false
		for _, kind := range p.Metadata.Kinds {
			hook = hook || kind == "hooks.additional_context"
		}
		if !hook {
			continue
		}
		for _, c := range p.Content {
			if c.Type == "input_text" && strings.TrimSpace(c.Text) != "" {
				blocks = append(blocks, c.Text)
			}
		}
	}
	return blocks, nil
}
