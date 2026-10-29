package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeDaemon stands in for the daemon behind the bridge, and records what it is sent.
type fakeDaemon struct {
	mu     sync.Mutex
	calls  []fakeCall
	answer func(path string, body []byte, deadline time.Duration) ([]byte, error)
}

type fakeCall struct {
	path string
	body map[string]any
}

func (f *fakeDaemon) post(path string, body []byte, deadline time.Duration) ([]byte, error) {
	var m map[string]any
	_ = json.Unmarshal(body, &m)
	f.mu.Lock()
	f.calls = append(f.calls, fakeCall{path, m})
	f.mu.Unlock()
	if f.answer == nil {
		return []byte(`{}`), nil
	}
	return f.answer(path, body, deadline)
}

func (f *fakeDaemon) recorded() []fakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakeCall(nil), f.calls...)
}

// bridgeSession runs a bridge on pipes, and reads its replies a line at a time.
type bridgeSession struct {
	t       *testing.T
	in      *io.PipeWriter
	replies chan map[string]any
	raw     chan string
	exited  chan int
	done    chan struct{}
}

func startBridge(t *testing.T, d *fakeDaemon, deadline, grace time.Duration) *bridgeSession {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	s := &bridgeSession{t: t, in: inW, replies: make(chan map[string]any, 64), raw: make(chan string, 64),
		exited: make(chan int, 4), done: make(chan struct{})}
	b := &bridge{in: inR, out: outW, post: d.post, deadline: deadline, grace: grace,
		exit: func(code int) { s.exited <- code }}
	go func() {
		b.serve()
		outW.Close()
		close(s.done)
	}()
	go func() {
		sc := bufio.NewScanner(outR)
		sc.Buffer(make([]byte, 1<<20), 16<<20)
		for sc.Scan() {
			s.raw <- sc.Text()
			var m map[string]any
			if json.Unmarshal([]byte(sc.Text()), &m) == nil {
				s.replies <- m
			}
		}
		close(s.replies)
	}()
	t.Cleanup(func() { inW.Close() })
	return s
}

func (s *bridgeSession) send(v any) {
	s.t.Helper()
	buf, _ := json.Marshal(v)
	if _, err := s.in.Write(append(buf, '\n')); err != nil {
		s.t.Fatalf("write to the bridge: %v", err)
	}
}

func (s *bridgeSession) request(id int, method string, params any) {
	s.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
}

func (s *bridgeSession) callTool(id int, name string, args map[string]any) {
	s.request(id, "tools/call", map[string]any{"name": name, "arguments": args})
}

// next waits for the next reply.
func (s *bridgeSession) next(within time.Duration) map[string]any {
	s.t.Helper()
	select {
	case m, ok := <-s.replies:
		if !ok {
			s.t.Fatal("the bridge closed its output while a reply was expected")
		}
		return m
	case <-time.After(within):
		s.t.Fatalf("no reply from the bridge within %s", within)
	}
	return nil
}

// toolText is the text of a tool result, and fails if the reply is an error.
func toolText(t *testing.T, reply map[string]any) string {
	t.Helper()
	if reply["error"] != nil {
		t.Fatalf("the reply is an error, which a hook shows a user: %v", reply["error"])
	}
	res, _ := reply["result"].(map[string]any)
	if res == nil {
		t.Fatalf("no result: %v", reply)
	}
	if res["isError"] == true {
		t.Fatalf("the result is flagged as an error: %v", res)
	}
	content, _ := res["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("the result holds %d content blocks, want 1: %v", len(content), res)
	}
	block, _ := content[0].(map[string]any)
	text, _ := block["text"].(string)
	return text
}

// The model sees every tool a server offers, and a call by the model reaches the
// server as a hook's call does, so a tool that runs a command would be reachable by
// the model (D-207).
func TestTheBridgeOffersOnlyThePromptAndStopTools(t *testing.T) {
	s := startBridge(t, &fakeDaemon{}, time.Second, time.Second)
	s.request(1, "tools/list", nil)
	res := s.next(2 * time.Second)["result"].(map[string]any)
	var names []string
	for _, tool := range res["tools"].([]any) {
		names = append(names, tool.(map[string]any)["name"].(string))
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "prompt,stop" {
		t.Errorf("the bridge offers %v; only prompt and stop may be offered (D-207)", names)
	}
}

// A tool the bridge does not offer is refused and never reaches the daemon, so naming a
// command as a tool runs nothing.
func TestACommandNamedAsAToolIsRefusedAndRunsNothing(t *testing.T) {
	d := &fakeDaemon{}
	s := startBridge(t, d, time.Second, time.Second)
	for i, name := range []string{"room-create", "room-invite", "peer-forget", "command", ""} {
		s.callTool(i+1, name, map[string]any{"prompt": "/cogmer:room-create"})
		if reply := s.next(2 * time.Second); reply["error"] == nil {
			t.Errorf("the tool %q was not refused: %v", name, reply)
		}
	}
	if n := len(d.recorded()); n != 0 {
		t.Errorf("a refused tool reached the daemon %d times", n)
	}
}

// A Stop hook that returns context starts the turn again, and the hooks reference caps
// such continuations at eight, so the bridge answers a stop call with nothing whatever
// the daemon says. A probe plugin whose Stop hook returned context made nine calls in
// 25 seconds (d306243).
func TestTheBridgeAnswersAStopCallWithNothingWhateverTheDaemonSays(t *testing.T) {
	d := &fakeDaemon{answer: func(string, []byte, time.Duration) ([]byte, error) {
		return []byte(`{"context":"a colleague's turn"}`), nil
	}}
	s := startBridge(t, d, time.Second, time.Second)
	s.callTool(1, "stop", map[string]any{"session_id": "s", "prompt_id": "p", "last_assistant_message": "done"})
	if text := toolText(t, s.next(2*time.Second)); text != "" {
		t.Errorf("a stop call was answered with %q; a Stop hook that returns context starts the turn again", text)
	}
	if calls := d.recorded(); len(calls) != 1 || calls[0].path != "/hook/stop" {
		t.Errorf("the stop call reached the daemon as %v, want one call to /hook/stop", calls)
	}
}

// D-147: a server that cogmer ships asks the client for no inference. The bridge answers
// requests and never makes one, and declares only the tools capability.
func TestTheBridgeDeclaresOnlyToolsAndNeverMakesARequest(t *testing.T) {
	d := &fakeDaemon{answer: func(string, []byte, time.Duration) ([]byte, error) {
		return []byte(`{"context":"a colleague's turn"}`), nil
	}}
	s := startBridge(t, d, time.Second, time.Second)
	s.request(1, "initialize", map[string]any{"protocolVersion": "2025-11-25"})
	init := s.next(2 * time.Second)["result"].(map[string]any)
	caps := init["capabilities"].(map[string]any)
	if len(caps) != 1 || caps["tools"] == nil {
		t.Errorf("capabilities = %v, want only tools", caps)
	}
	if init["protocolVersion"] != "2025-11-25" {
		t.Errorf("the protocol version the client asked for was not offered back: %v", init["protocolVersion"])
	}
	s.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	s.request(2, "tools/list", nil)
	s.next(2 * time.Second)
	s.callTool(3, "prompt", map[string]any{"session_id": "s", "prompt": "hi", "prompt_id": "p"})
	s.next(2 * time.Second)
	s.in.Close()
	<-s.done
	close(s.raw)
	for line := range s.raw {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) != nil {
			t.Errorf("the bridge wrote something that is not a message: %q", line)
			continue
		}
		if m["method"] != nil {
			t.Errorf("the bridge made a request of the client, %v, which could cause inference (D-147)", m["method"])
		}
		if strings.Contains(line, "sampling") {
			t.Errorf("a message names sampling: %q", line)
		}
	}
}

// Claude Code adds a hook's reply to the model's context only in this form (D-206).
func TestThePromptReplyCarriesTheContextInTheJSONForm(t *testing.T) {
	d := &fakeDaemon{answer: func(string, []byte, time.Duration) ([]byte, error) {
		return []byte(`{"context":"<team-conversation>a turn</team-conversation>"}`), nil
	}}
	s := startBridge(t, d, time.Second, time.Second)
	s.callTool(1, "prompt", map[string]any{"session_id": "sess-1", "prompt": "hello", "prompt_id": "p-1", "cwd": "/work"})
	text := toolText(t, s.next(2*time.Second))
	var out struct {
		Specific struct {
			Event   string `json:"hookEventName"`
			Context string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("the reply is not the JSON form, which Claude Code drops: %q", text)
	}
	if out.Specific.Event != "UserPromptSubmit" || out.Specific.Context != "<team-conversation>a turn</team-conversation>" {
		t.Errorf("reply = %+v", out.Specific)
	}
}

// The daemon is told the host, and only the fields a hook supplies reach it.
func TestTheBridgeDeclaresTheHostAndPassesOnlyHookFields(t *testing.T) {
	d := &fakeDaemon{}
	s := startBridge(t, d, time.Second, time.Second)
	s.callTool(1, "prompt", map[string]any{"session_id": "sess-1", "prompt": "/cogmer:room-status", "prompt_id": "p-1",
		"cwd": "/work", "host": "claude", "command": "room-create", "room": "other"})
	s.next(2 * time.Second)
	s.callTool(2, "stop", map[string]any{"session_id": "sess-1", "prompt_id": "p-1", "last_assistant_message": "done",
		"transcript_path": "/root/x.jsonl", "role": "admin"})
	s.next(2 * time.Second)
	calls := d.recorded()
	if len(calls) != 2 {
		t.Fatalf("the daemon saw %d calls", len(calls))
	}
	byPath := map[string]map[string]any{calls[0].path: calls[0].body, calls[1].path: calls[1].body}
	for path, want := range map[string][]string{
		"/hook/prompt": {"cwd", "host", "prompt", "prompt_id", "session_id"},
		"/hook/stop":   {"host", "last_assistant_message", "prompt_id", "session_id", "transcript_path"},
	} {
		body := byPath[path]
		var got []string
		for k := range body {
			got = append(got, k)
		}
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s was sent %v, want %v", path, got, want)
		}
		if body["host"] != "cowork" {
			t.Errorf("%s declared the host %v, want cowork", path, body["host"])
		}
	}
	if byPath["/hook/prompt"]["prompt"] != "/cogmer:room-status" {
		t.Errorf("the prompt text was altered: %v", byPath["/hook/prompt"]["prompt"])
	}
}

// D-011 applies to the bridge: a daemon that is down, or says nothing, is an empty
// reply and never an error, since a hook that errs shows the user one.
func TestABridgeFailureIsAnEmptyReplyNotAnError(t *testing.T) {
	for name, answer := range map[string]func(string, []byte, time.Duration) ([]byte, error){
		"daemon down":      func(string, []byte, time.Duration) ([]byte, error) { return nil, errors.New("connection refused") },
		"no context":       func(string, []byte, time.Duration) ([]byte, error) { return []byte(`{"context":""}`), nil },
		"not json":         func(string, []byte, time.Duration) ([]byte, error) { return []byte(`<html>`), nil },
		"empty body":       func(string, []byte, time.Duration) ([]byte, error) { return nil, nil },
		"unexpected shape": func(string, []byte, time.Duration) ([]byte, error) { return []byte(`[1,2]`), nil },
	} {
		d := &fakeDaemon{answer: answer}
		s := startBridge(t, d, time.Second, time.Second)
		for i, tool := range []string{"prompt", "stop"} {
			s.callTool(i+1, tool, map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "x"})
			if text := toolText(t, s.next(2*time.Second)); text != "" {
				t.Errorf("%s, %s: the reply is %q, want nothing", name, tool, text)
			}
		}
	}
}

// A daemon that hangs is answered within the bridge's own deadline, which is what a
// server that took a call and never answered did not do: it held a session for 60
// seconds.
func TestAHungDaemonIsAnsweredWithinTheDeadline(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	d := &fakeDaemon{answer: func(_ string, _ []byte, deadline time.Duration) ([]byte, error) {
		select {
		case <-release:
		case <-time.After(deadline): // an HTTP client that honors its timeout
		}
		return nil, errors.New("timeout")
	}}
	s := startBridge(t, d, 150*time.Millisecond, 400*time.Millisecond)
	start := time.Now()
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "x"})
	text := toolText(t, s.next(2*time.Second))
	if took := time.Since(start); took > 600*time.Millisecond {
		t.Errorf("the reply took %s, with a deadline of 150ms", took)
	}
	if text != "" {
		t.Errorf("the reply is %q, want nothing", text)
	}
	select {
	case code := <-s.exited:
		t.Errorf("the bridge exited (%d) though its call returned at the deadline", code)
	case <-time.After(700 * time.Millisecond):
	}
}

// Whatever is stuck inside the bridge is recovered by exiting, which Claude Desktop
// follows by starting the server at the next call. A call that outlives its deadline by
// the grace is that case.
func TestACallStuckPastItsGraceMakesTheBridgeExit(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	d := &fakeDaemon{answer: func(string, []byte, time.Duration) ([]byte, error) {
		<-release // ignores every deadline
		return nil, nil
	}}
	s := startBridge(t, d, 100*time.Millisecond, 200*time.Millisecond)
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p"})
	if text := toolText(t, s.next(2*time.Second)); text != "" {
		t.Errorf("the reply is %q, want nothing", text)
	}
	select {
	case code := <-s.exited:
		if code != 1 {
			t.Errorf("exit code %d, want 1", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the bridge did not exit with a call stuck past its grace")
	}
}

// One call waiting on the daemon does not hold up the next, because the pool of
// sessions that share the server is one process.
func TestCallsAreServedConcurrently(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	d := &fakeDaemon{answer: func(_ string, body []byte, _ time.Duration) ([]byte, error) {
		if strings.Contains(string(body), `"slow"`) {
			<-release
		}
		return []byte(`{"context":"fast"}`), nil
	}}
	s := startBridge(t, d, 10*time.Second, 10*time.Second)
	s.callTool(1, "prompt", map[string]any{"session_id": "slow", "prompt_id": "p"})
	s.callTool(2, "prompt", map[string]any{"session_id": "quick", "prompt_id": "p"})
	reply := s.next(2 * time.Second)
	if id, _ := reply["id"].(float64); id != 2 {
		t.Errorf("the first reply was for call %v; the call that was not waiting should have been answered first", reply["id"])
	}
}

// Input that is not a message is survived, and the next message is still answered.
func TestMalformedInputIsSurvived(t *testing.T) {
	s := startBridge(t, &fakeDaemon{}, time.Second, time.Second)
	for _, junk := range []string{"not json at all\n", "\n", "{}\n", `{"method":5}` + "\n", "[1,2,3]\n", `{"id":1,"method":"tools/call","params":"x"}` + "\n"} {
		if _, err := io.WriteString(s.in, junk); err != nil {
			t.Fatal(err)
		}
	}
	// the last junk line is a call with unreadable parameters, which is refused
	if reply := s.next(2 * time.Second); reply["error"] == nil {
		t.Errorf("a call with unreadable parameters was not refused: %v", reply)
	}
	s.request(9, "tools/list", nil)
	if reply := s.next(2 * time.Second); reply["result"] == nil {
		t.Errorf("the bridge stopped answering after malformed input: %v", reply)
	}
}

// The bridge reaches the daemon the way the hook client does: with the header that
// marks the request as ours (D-087), to the address the daemon listens on, and with the
// body it was given.
func TestTheBridgeReachesTheDaemonWithTheLocalHeader(t *testing.T) {
	var gotHeader, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader, gotPath = r.Header.Get(localGuardHeader), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"context":"from the daemon"}`))
	}))
	defer srv.Close()
	t.Setenv("COGMER_ADDR", strings.TrimPrefix(srv.URL, "http://"))
	raw, err := postToDaemon("/hook/prompt", []byte(`{"host":"cowork","session_id":"s"}`), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if gotHeader != localGuardValue || gotPath != "/hook/prompt" || gotBody["host"] != "cowork" {
		t.Errorf("the daemon saw header %q, path %q, body %v", gotHeader, gotPath, gotBody)
	}
	if !strings.Contains(string(raw), "from the daemon") {
		t.Errorf("the daemon's reply was not returned: %q", raw)
	}
}
