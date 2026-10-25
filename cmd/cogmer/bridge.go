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
	"strconv"
	"strings"
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
	// bridgeCommandDeadline is the longest a call waits when the prompt is a command the
	// user typed, which the daemon runs for up to typedCommandTimeout. A command is what
	// the user asked for, and the plugin's own command files wait for it.
	bridgeCommandDeadline = typedCommandTimeout + 5*time.Second
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
	// commandDeadline is the deadline of a call whose prompt is a typed command.
	commandDeadline time.Duration
	grace           time.Duration
	exit            func(int)

	// The daemon the bridge keeps running (D-208). With no control, the bridge only relays.
	control     daemonControl
	startEvery  time.Duration // the least time between two starts
	startWait   time.Duration // how long a typed command waits for a daemon it started
	restartWait time.Duration // how long a restart waits for the new daemon to answer
	startMu     sync.Mutex
	lastStart   time.Time

	// The prompts the bridge answered itself, so that the stop call of the same turn is not
	// relayed and the model's account of the command is not published.
	handledMu sync.Mutex
	handled   map[string]bool

	mu sync.Mutex // serializes writes to out

	// refusal, when set, is the bridge's answer to a command the user typed, and every
	// other call is answered with nothing and relayed nowhere.
	refusal string
}

func runBridge() {
	b := &bridge{
		in: os.Stdin, out: os.Stdout,
		post:     postToDaemon,
		deadline: bridgeDeadline, commandDeadline: bridgeCommandDeadline, grace: bridgeGrace,
		exit:    os.Exit,
		control: localDaemon{}, startEvery: 10 * time.Second, startWait: 6 * time.Second,
		restartWait: restartWait(),
	}
	go b.startAtLaunch()
	b.serve()
}

// runRefusingBridge is the bridge on data newer than this program (D-244, a program
// that finds a newer data version touches nothing). It starts no daemon, relays
// nothing, and answers a command the user typed with refusal, so the host is answered
// as it would be without cogmer.
func runRefusingBridge(refusal string) {
	b := &bridge{
		in: os.Stdin, out: os.Stdout,
		post:     postToDaemon,
		deadline: bridgeDeadline, grace: bridgeGrace,
		exit:    os.Exit,
		refusal: refusal,
	}
	b.serve()
}

// refuse is the reply of a bridge that relays nothing: the refusal for a command the
// user typed, which the model relays, and nothing for any other call.
func (b *bridge) refuse(tool string, args map[string]any) string {
	cmd, typed := isTypedCommand(args)
	host, ok := hostByName(bridgeHost)
	if tool != "prompt" || !typed || !ok {
		return ""
	}
	return string(host.PromptHookOutput("## Output of " + typedCommandName(host, cmd.name) + "\n\n" + b.refusal))
}

// restartWait is how long a restart waits for the daemon to answer. COGMER_RESTART_WAIT
// sets it in seconds, as it does for the plugin's restart.
func restartWait() time.Duration {
	if v := os.Getenv("COGMER_RESTART_WAIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 5 * time.Second
}

// startAtLaunch does for a Cowork user what the session-start hook does for the others: it
// starts a daemon when none answers and the user has not stopped it, and starts one over an
// older daemon, which gives way (D-158). It waits for nothing.
func (b *bridge) startAtLaunch() {
	if b.control == nil {
		return
	}
	running, ok := b.control.Answering()
	switch {
	case !ok && b.control.Stopped():
		return
	case ok && !shouldReplace(running, version):
		return
	}
	b.startDaemon("at launch", false)
}

// startDaemon starts a daemon unless one was started a moment ago, and says whether it did.
// A command the user typed starts one whatever was started before.
func (b *bridge) startDaemon(why string, force bool) bool {
	b.startMu.Lock()
	defer b.startMu.Unlock()
	if b.control == nil {
		return false
	}
	if !force && !b.lastStart.IsZero() && time.Since(b.lastStart) < b.startEvery {
		return false
	}
	b.lastStart = time.Now()
	if err := b.control.Start(); err != nil {
		log.Printf("bridge: could not start the daemon (%s): %v", why, err)
		return false
	}
	log.Printf("bridge: started the daemon (%s)", why)
	return true
}

// waitAnswering waits up to a limit for a daemon to answer.
func (b *bridge) waitAnswering(limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for {
		if _, ok := b.control.Answering(); ok {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
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
	deadline := b.deadlineFor(p.Name, p.Arguments)
	work := func() string { return b.relay(p.Name, endpoint.path, endpoint.fields, p.Arguments, deadline) }
	if b.refusal != "" {
		work = func() string { return b.refuse(p.Name, p.Arguments) }
	} else if local, ok := b.localCommand(p.Name, p.Arguments); ok {
		work = func() string { return b.runLocal(local, p.Arguments) }
	} else if p.Name == "stop" && b.wasHandled(p.Arguments) {
		work = func() string { return "" }
	}
	done := make(chan string, 1)
	go func() { done <- work() }()
	select {
	case text := <-done:
		b.reply(m.ID, text)
	case <-time.After(deadline):
		log.Printf("bridge: %s did not finish within %s; answering with nothing", p.Name, deadline)
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

// deadlineFor is how long a call may take: the usual deadline, or the longer one for a
// prompt that is a command the user typed.
func (b *bridge) deadlineFor(tool string, args map[string]any) time.Duration {
	if tool != "prompt" || b.commandDeadline == 0 {
		return b.deadline
	}
	host, ok := hostByName(bridgeHost)
	if !ok {
		return b.deadline
	}
	prompt, _ := args["prompt"].(string)
	if _, isCommand := parseTypedCommand(prompt, host.CommandPrefix()); isCommand {
		return b.commandDeadline
	}
	return b.deadline
}

// relay sends the hook's fields to the daemon and returns the text the hook's reply
// carries, empty when there is none or when anything fails.
func (b *bridge) relay(tool, path string, fields []string, args map[string]any, deadline time.Duration) string {
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
	raw, err := b.post(path, body, deadline)
	if err != nil {
		log.Printf("bridge: daemon unreachable (%v); continuing without collaboration", err)
		raw, err = b.afterFailure(tool, path, body, args, deadline)
		if err != nil {
			// A command the user typed has to answer, since the model is told to relay what
			// it printed and would otherwise have nothing to relay.
			if cmd, typed := isTypedCommand(args); typed && tool == "prompt" && b.control != nil {
				if host, ok := hostByName(bridgeHost); ok {
					return string(host.PromptHookOutput("## Output of " + typedCommandName(host, cmd.name) +
						"\n\nThe cogmer daemon on this machine is not running and could not be started. The daemon's log says why."))
				}
			}
			return ""
		}
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
	// Claude Code adds a reply to the model's context only in the form the host writes. A
	// plain line is recorded as a successful hook and never reaches the model (D-206).
	host, ok := hostByName(bridgeHost)
	if !ok {
		return ""
	}
	return string(host.PromptHookOutput(out.Context))
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

// isTypedCommand says whether a prompt call carries a command the user typed.
func isTypedCommand(args map[string]any) (typedCommand, bool) {
	host, ok := hostByName(bridgeHost)
	if !ok {
		return typedCommand{}, false
	}
	prompt, _ := args["prompt"].(string)
	return parseTypedCommand(prompt, host.CommandPrefix())
}

// afterFailure is what a call that found no daemon does. A call that is only a hook starts
// a daemon, unless the user stopped it, and goes without: a session is never held up for
// one, and the next call finds it. A command the user typed is what the user asked for, so
// it ends their stop, starts a daemon, waits for it and tries once more.
func (b *bridge) afterFailure(tool, path string, body []byte, args map[string]any, deadline time.Duration) ([]byte, error) {
	if b.control == nil {
		return nil, errNoDaemon
	}
	_, typed := isTypedCommand(args)
	if tool != "prompt" || !typed {
		if !b.control.Stopped() {
			b.startDaemon("a call found none", false)
		}
		return nil, errNoDaemon
	}
	b.control.SetStopped(false)
	b.startDaemon("a command found none", true)
	if !b.waitAnswering(b.startWait) {
		return nil, errNoDaemon
	}
	return b.post(path, body, deadline)
}

var errNoDaemon = fmt.Errorf("no daemon answered")

// localCommands are the commands the bridge runs itself, because each acts on the daemon
// and a daemon that ran one would end the process that has to answer.
var localCommands = map[string]bool{"self-restart": true, "self-stop": true}

func (b *bridge) localCommand(tool string, args map[string]any) (string, bool) {
	if tool != "prompt" || b.control == nil {
		return "", false
	}
	cmd, ok := isTypedCommand(args)
	if !ok || !localCommands[cmd.name] {
		return "", false
	}
	return cmd.name, true
}

func handledKey(args map[string]any) string {
	session, _ := args["session_id"].(string)
	prompt, _ := args["prompt_id"].(string)
	return session + "\x00" + prompt
}

func (b *bridge) wasHandled(args map[string]any) bool {
	b.handledMu.Lock()
	defer b.handledMu.Unlock()
	return b.handled[handledKey(args)]
}

// runLocal runs a command the bridge answers itself and returns the reply for the hook.
func (b *bridge) runLocal(name string, args map[string]any) string {
	b.handledMu.Lock()
	if b.handled == nil {
		b.handled = map[string]bool{}
	}
	if len(b.handled) >= commandPromptsKept {
		b.handled = map[string]bool{}
	}
	b.handled[handledKey(args)] = true
	b.handledMu.Unlock()

	host, ok := hostByName(bridgeHost)
	if !ok {
		return ""
	}
	var text string
	switch name {
	case "self-restart":
		text = b.restartDaemon()
	default:
		text = b.stopDaemon()
	}
	return string(host.PromptHookOutput("## Output of " + typedCommandName(host, name) + "\n\n" + text))
}

// restartDaemon stops the daemon and starts one, and says whether it answers. Started and
// answering are different facts, and somebody who asks for a restart wants the second, so
// it waits, for a bounded time (D-195).
func (b *bridge) restartDaemon() string {
	if out, err := b.control.Stop(true); err != nil {
		return "The daemon could not be stopped, so it was not restarted.\n\n" + strings.TrimSpace(out)
	}
	b.control.SetStopped(false)
	if !b.startDaemon("a restart", true) {
		return "The daemon was stopped and could not be started again. The daemon's log says why."
	}
	if b.waitAnswering(b.restartWait) {
		v, _ := b.control.Answering()
		return fmt.Sprintf("The daemon was restarted and is answering on %s, version %s.\n\n"+
			"Nothing that was captured is lost, because it is written to disk before it is shared. "+
			"Connecting to colleagues again can take a few seconds more.", addr(), v)
	}
	return fmt.Sprintf("The daemon was started and is not answering on %s yet.\n\nThe daemon's log says what happened.", addr())
}

// stopDaemon stops the daemon and leaves the mark that stops the bridge from starting one
// at the next prompt. A typed command starts it again.
func (b *bridge) stopDaemon() string {
	out, err := b.control.Stop(false)
	if err != nil {
		return "The daemon could not be stopped.\n\n" + strings.TrimSpace(out)
	}
	b.control.SetStopped(true)
	return "The daemon on this machine was stopped.\n\n" +
		"Nothing is captured or shared in Cowork until it runs again. Any /cogmer: command starts it."
}
