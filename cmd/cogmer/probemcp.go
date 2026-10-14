package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// runProbeMCP is the MCP server the Cowork behavior check starts for a Claude Code session
// (D-206). It has one tool, ping, that returns the word it is given either as plain text or
// in the JSON form the bridge returns, and it writes the event of each call to a file in the
// check's directory. It talks to no daemon, since the check is about what the harness does
// with a reply and not about what the bridge says in it.
func runProbeMCP(dir string) {
	calls := filepath.Join(dir, "mcp-calls.log")
	in := bufio.NewReaderSize(os.Stdin, 1<<20)
	out := bufio.NewWriter(os.Stdout)
	send := func(v any) {
		buf, _ := json.Marshal(v)
		fmt.Fprintf(out, "%s\n", buf)
		_ = out.Flush()
	}
	for {
		line, err := in.ReadBytes('\n')
		var m struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params struct {
				ProtocolVersion string         `json:"protocolVersion"`
				Name            string         `json:"name"`
				Arguments       map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if len(strings.TrimSpace(string(line))) > 0 && json.Unmarshal(line, &m) == nil {
			switch m.Method {
			case "initialize":
				v := m.Params.ProtocolVersion
				if v == "" {
					v = bridgeProtocol
				}
				send(map[string]any{"jsonrpc": "2.0", "id": m.ID, "result": map[string]any{
					"protocolVersion": v, "capabilities": map[string]any{"tools": map[string]any{}},
					"serverInfo": map[string]any{"name": "cogmer-probe", "version": version}}})
			case "tools/list":
				send(map[string]any{"jsonrpc": "2.0", "id": m.ID, "result": map[string]any{"tools": []any{map[string]any{
					"name": "ping", "description": "Used by cogmer's behavior check.",
					"inputSchema": map[string]any{"type": "object"}}}}})
			case "tools/call":
				event, _ := m.Params.Arguments["event"].(string)
				format, _ := m.Params.Arguments["format"].(string)
				word, _ := m.Params.Arguments["word"].(string)
				if f, err := os.OpenFile(calls, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
					fmt.Fprintf(f, "%s event=%s format=%s\n", time.Now().UTC().Format(time.RFC3339), event, format)
					_ = f.Close()
				}
				text := word
				if format == "json" {
					// the form the bridge returns, built by the host that the bridge uses
					text = string(coworkHost{}.PromptHookOutput(word))
				}
				send(map[string]any{"jsonrpc": "2.0", "id": m.ID, "result": map[string]any{
					"content": []any{map[string]any{"type": "text", "text": text}}}})
			default:
				if len(m.ID) > 0 && string(m.ID) != "null" {
					send(map[string]any{"jsonrpc": "2.0", "id": m.ID, "error": map[string]any{"code": -32601, "message": "method not found"}})
				}
			}
		}
		if err != nil {
			return
		}
	}
}
