package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// The bridge is the MCP server a Claude Cowork session reaches through its hooks
// (D-205). A Cowork session runs in a sandbox that cannot reach the daemon, and a plugin
// hook of type mcp_tool can call a server that runs on the user's machine, so the
// bridge relays each of those calls to the daemon as the hook client does for the other
// hosts.
//
// It offers two tools, prompt and stop, and no tool that runs a command, because the
// model sees every tool a server offers and a call by the model reaches it as a hook's
// call does (D-207). A typed command arrives as the text of the prompt.
//
// It fails open as the hook client does (D-011): whatever goes wrong, a call is answered
// with an empty reply and no error, because a hook that reports a failure shows the user
// one, and an empty reply adds nothing to the session. Nothing it does may hold a
// session up, and a call that held one for 60 seconds was observed with a server that
// took a call and never answered, so every call is answered within its own deadline.
//
// It declares only the tools capability and never sends the client a request, so no
// peer's event can cause inference through it (D-147).

const (
	// bridgeDeadline is the longest a call waits for the daemon. It is the deadline the
	// hook client uses for the other hosts.
	bridgeDeadline = 3 * time.Second
	// bridgeGrace is how long past its deadline a call may still be running before the
	// bridge exits. An exit is recovered from, since Claude Desktop starts the server
	// again at the next call, and a hung bridge is not.
	bridgeGrace = 2 * time.Second
	// bridgeMaxLine is the longest message the bridge reads. A longer one is dropped.
	bridgeMaxLine = 16 << 20
	// bridgeProtocol is offered when the client names no version.
	bridgeProtocol = "2025-06-18"
)

// bridgeTools are the only tools the bridge offers. A test requires exactly these.
var bridgeTools = []map[string]any{
	{
		"name":        "prompt",
		"description": "Called by cogmer's prompt hook, not by the model.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{"type": "string"},
				"prompt":     map[string]any{"type": "string"},
				"prompt_id":  map[string]any{"type": "string"},
				"cwd":        map[string]any{"type": "string"},
			},
			"required": []string{"session_id", "prompt_id"},
		},
	},
	{
		"name":        "stop",
		"description": "Called by cogmer's stop hook, not by the model.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id":             map[string]any{"type": "string"},
				"prompt_id":              map[string]any{"type": "string"},
				"last_assistant_message": map[string]any{"type": "string"},
				"transcript_path":        map[string]any{"type": "string"},
				"cwd":                    map[string]any{"type": "string"},
			},
			"required": []string{"session_id", "prompt_id"},
		},
	},
}

// bridgeEndpoints maps a tool to the daemon route it relays to, and the fields of its
// input that are passed on. Nothing else a caller supplies reaches the daemon.
var bridgeEndpoints = map[string]struct {
	path   string
	fields []string
}{
	"prompt": {"/hook/prompt", []string{"session_id", "prompt", "prompt_id", "cwd"}},
	"stop":   {"/hook/stop", []string{"session_id", "prompt_id", "last_assistant_message", "transcript_path", "cwd"}},
}

// bridgeHost is the host the bridge declares to the daemon with every call.
const bridgeHost = "cowork"

type bridge struct {
	in       io.Reader
	out      io.Writer
	post     func(path string, body []byte, deadline time.Duration) ([]byte, error)
	deadline time.Duration
	grace    time.Duration
	exit     func(int)

	mu sync.Mutex // serializes writes to out
}

func runBridge() {
	b := &bridge{
		in: os.Stdin, out: os.Stdout,
		post:     postToDaemon,
		deadline: bridgeDeadline, grace: bridgeGrace,
		exit: os.Exit,
	}
	b.serve()
}

// postToDaemon sends one hook call to the local daemon.
func postToDaemon(path string, body []byte, deadline time.Duration) ([]byte, error) {
	resp, err := localPost(&http.Client{Timeout: deadline}, "http://"+addr()+path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

type bridgeMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// serve reads messages until the input ends. Calls run concurrently, so a call that is
// waiting on the daemon does not hold up the next.
func (b *bridge) serve() {
	var wg sync.WaitGroup
	r := bufio.NewReaderSize(b.in, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 && len(line) <= bridgeMaxLine {
			var m bridgeMessage
			if json.Unmarshal(bytes.TrimSpace(line), &m) == nil && m.Method != "" {
				if m.Method == "tools/call" {
					wg.Add(1)
					go func() { defer wg.Done(); b.call(m) }()
				} else {
					b.handle(m)
				}
			}
		}
		if err != nil {
			break
		}
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(b.deadline + b.grace):
	}
}

func (b *bridge) handle(m bridgeMessage) {
	hasID := len(m.ID) > 0 && string(m.ID) != "null"
	switch m.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(m.Params, &p)
		if p.ProtocolVersion == "" {
			p.ProtocolVersion = bridgeProtocol
		}
		b.result(m.ID, map[string]any{
			"protocolVersion": p.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "cogmer", "version": version},
		})
	case "ping":
		b.result(m.ID, map[string]any{})
	case "tools/list":
		b.result(m.ID, map[string]any{"tools": bridgeTools})
	default:
		// A notification has no id and gets no reply. A request for anything else is
		// refused, because the bridge does nothing else.
		if hasID {
			b.failure(m.ID, -32601, "method not found")
		}
	}
}

// call answers one tools/call within the deadline whatever the daemon does.
func (b *bridge) call(m bridgeMessage) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if json.Unmarshal(m.Params, &p) != nil {
		b.failure(m.ID, -32602, "invalid parameters")
		return
	}
	endpoint, ok := bridgeEndpoints[p.Name]
	if !ok {
		b.failure(m.ID, -32602, "unknown tool")
		return
	}
	done := make(chan string, 1)
	go func() { done <- b.relay(p.Name, endpoint.path, endpoint.fields, p.Arguments) }()
	select {
	case text := <-done:
		b.reply(m.ID, text)
	case <-time.After(b.deadline):
		log.Printf("bridge: %s did not finish within %s; answering with nothing", p.Name, b.deadline)
		b.reply(m.ID, "")
		select {
		case <-done:
		case <-time.After(b.grace):
			// Whatever is stuck is stuck in this process, and an exit is what Desktop
			// recovers from.
			log.Printf("bridge: %s is still running %s past its deadline; exiting", p.Name, b.grace)
			b.exit(1)
		}
	}
}

// relay sends the hook's fields to the daemon and returns the text the hook's reply
// carries, empty when there is none or when anything fails.
func (b *bridge) relay(tool, path string, fields []string, args map[string]any) string {
	payload := map[string]any{"host": bridgeHost}
	for _, f := range fields {
		if s, ok := args[f].(string); ok {
			payload[f] = s
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	raw, err := b.post(path, body, b.deadline)
	if err != nil {
		log.Printf("bridge: daemon unreachable (%v); continuing without collaboration", err)
		return ""
	}
	if tool != "prompt" {
		return ""
	}
	var out struct {
		Context string `json:"context"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Context == "" {
		return ""
	}
	// Claude Code adds a reply to the model's context only in this form. A plain line
	// is recorded as a successful hook and never reaches the model (D-206).
	reply, err := json.Marshal(map[string]any{"hookSpecificOutput": map[string]any{
		"hookEventName":     "UserPromptSubmit",
		"additionalContext": out.Context,
	}})
	if err != nil {
		return ""
	}
	return string(reply)
}

func (b *bridge) reply(id json.RawMessage, text string) {
	b.result(id, map[string]any{"content": []map[string]any{{"type": "text", "text": text}}})
}

func (b *bridge) result(id json.RawMessage, result any) {
	b.write(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func (b *bridge) failure(id json.RawMessage, code int, message string) {
	b.write(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}})
}

func (b *bridge) write(v any) {
	buf, err := json.Marshal(v)
	if err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, err := fmt.Fprintf(b.out, "%s\n", buf); err != nil {
		log.Printf("bridge: could not write a reply: %v", err)
	}
}
