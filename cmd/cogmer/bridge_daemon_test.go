package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeControl stands in for the daemon's process: whether it answers, what version, what
// has been started and stopped.
type fakeControl struct {
	mu          sync.Mutex
	answering   bool
	version     string
	stopped     bool
	starts      int
	stops       []bool
	startErr    error
	stopErr     error
	stopOut     string
	startAnswer bool // a start makes the daemon answer
}

func (f *fakeControl) Answering() (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.version, f.answering
}
func (f *fakeControl) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts++
	if f.startErr != nil {
		return f.startErr
	}
	if f.startAnswer {
		f.answering = true
		f.version = "dev"
	}
	return nil
}
func (f *fakeControl) Stop(forRestart bool) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops = append(f.stops, forRestart)
	if f.stopErr == nil {
		f.answering = false
	}
	return f.stopOut, f.stopErr
}
func (f *fakeControl) Stopped() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopped
}
func (f *fakeControl) SetStopped(s bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = s
}
func (f *fakeControl) startCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.starts
}

func startBridgeWithControl(t *testing.T, d *fakeDaemon, c *fakeControl) *bridgeSession {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	s := &bridgeSession{t: t, in: inW, replies: make(chan map[string]any, 64), raw: make(chan string, 64),
		exited: make(chan int, 4), done: make(chan struct{})}
	b := &bridge{in: inR, out: outW, post: d.post, deadline: time.Second, commandDeadline: 5 * time.Second,
		grace: time.Second, exit: func(code int) { s.exited <- code },
		control: c, startEvery: 10 * time.Second, startWait: 2 * time.Second, restartWait: 2 * time.Second}
	go func() { b.serve(); outW.Close(); close(s.done) }()
	go func() {
		sc := bufio.NewScanner(outR)
		sc.Buffer(make([]byte, 1<<20), 16<<20)
		for sc.Scan() {
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

func downDaemon() *fakeDaemon {
	return &fakeDaemon{answer: func(string, []byte, time.Duration) ([]byte, error) { return nil, errors.New("connection refused") }}
}

// A hook that finds no daemon starts one and goes without, so a session is never held up,
// and a second call a moment later does not start another.
func TestACallThatFindsNoDaemonStartsOneAndGoesWithout(t *testing.T) {
	c := &fakeControl{}
	s := startBridgeWithControl(t, downDaemon(), c)
	start := time.Now()
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "hello"})
	if text := toolText(t, s.next(2*time.Second)); text != "" {
		t.Errorf("reply %q, want nothing", text)
	}
	if took := time.Since(start); took > 500*time.Millisecond {
		t.Errorf("the call waited %s for a daemon it started", took)
	}
	if c.startCount() != 1 {
		t.Errorf("%d daemons started, want 1", c.startCount())
	}
	s.callTool(2, "stop", map[string]any{"session_id": "s", "prompt_id": "p"})
	s.next(2 * time.Second)
	if c.startCount() != 1 {
		t.Errorf("a second call a moment later started another daemon: %d", c.startCount())
	}
}

func TestACallStartsNoDaemonWhileTheUsersStopStands(t *testing.T) {
	c := &fakeControl{stopped: true}
	s := startBridgeWithControl(t, downDaemon(), c)
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "hello"})
	s.next(2 * time.Second)
	if c.startCount() != 0 || !c.Stopped() {
		t.Errorf("starts %d, stopped %v: an ordinary prompt ended the user's stop", c.startCount(), c.Stopped())
	}
}

// A command the user typed ends their stop, starts a daemon, waits for it and runs.
func TestATypedCommandStartsTheDaemonItFindsDownAndRuns(t *testing.T) {
	c := &fakeControl{stopped: true, startAnswer: true}
	var calls int
	d := &fakeDaemon{answer: func(string, []byte, time.Duration) ([]byte, error) {
		calls++
		if !c.answering {
			return nil, errors.New("connection refused")
		}
		return []byte(`{"context":"## Output of /cogmer:self-status\n\nthe output"}`), nil
	}}
	s := startBridgeWithControl(t, d, c)
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "/cogmer:self-status"})
	text := toolText(t, s.next(5*time.Second))
	if !strings.Contains(text, "the output") {
		t.Errorf("the command's output did not arrive after the daemon was started: %q", text)
	}
	if c.startCount() != 1 || c.Stopped() {
		t.Errorf("starts %d, stopped %v; a typed command should end the stop and start one daemon", c.startCount(), c.Stopped())
	}
}

func TestATypedCommandSaysSoWhenNoDaemonCanBeStarted(t *testing.T) {
	c := &fakeControl{startErr: errors.New("no such file")}
	s := startBridgeWithControl(t, downDaemon(), c)
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "/cogmer:room-status"})
	text := toolText(t, s.next(5*time.Second))
	if !strings.Contains(text, "not running and could not be started") || !strings.Contains(text, "/cogmer:room-status") {
		t.Errorf("a typed command with no daemon answered %q", text)
	}
}

// What the session-start hook does for the other hosts, the bridge does when it starts.
func TestTheBridgeStartsADaemonWhenItStartsUnlessTheUserStoppedOne(t *testing.T) {
	defer func(v string) { version = v }(version)
	version = "1.0.0"
	for _, c := range []struct {
		name  string
		ctl   *fakeControl
		start bool
	}{
		{"none answers", &fakeControl{}, true},
		{"none answers and the user stopped it", &fakeControl{stopped: true}, false},
		{"one answers at this version", &fakeControl{answering: true, version: "1.0.0"}, false},
		{"one answers at another version", &fakeControl{answering: true, version: "0.9.0"}, true},
		{"one that is a development build answers", &fakeControl{answering: true, version: "dev"}, false},
	} {
		b := &bridge{control: c.ctl, startEvery: time.Second}
		b.startAtLaunch()
		if got := c.ctl.startCount() == 1; got != c.start {
			t.Errorf("%s: started = %v, want %v", c.name, got, c.start)
		}
	}
}

// A restart and a stop act on the daemon, so the bridge runs them and the daemon does not,
// and the model's account of either is not published.
func TestARestartIsRunByTheBridgeAndNotRelayed(t *testing.T) {
	c := &fakeControl{answering: true, version: "dev", startAnswer: true}
	d := &fakeDaemon{}
	s := startBridgeWithControl(t, d, c)
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "/cogmer:self-restart"})
	text := toolText(t, s.next(5*time.Second))
	var out struct {
		H struct {
			Context string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil || !strings.Contains(out.H.Context, "was restarted and is answering") {
		t.Errorf("the restart answered %q", text)
	}
	if len(c.stops) != 1 || !c.stops[0] || c.startCount() != 1 {
		t.Errorf("stops %v, starts %d; a restart stops with the restart flag and starts one", c.stops, c.startCount())
	}
	if len(d.recorded()) != 0 {
		t.Errorf("the command reached the daemon %d times", len(d.recorded()))
	}
	s.callTool(2, "stop", map[string]any{"session_id": "s", "prompt_id": "p", "last_assistant_message": "I restarted it"})
	s.next(2 * time.Second)
	if len(d.recorded()) != 0 {
		t.Error("the model's account of a restart was relayed to the daemon, which would publish it")
	}
	s.callTool(3, "stop", map[string]any{"session_id": "s", "prompt_id": "other", "last_assistant_message": "an ordinary reply"})
	s.next(2 * time.Second)
	if len(d.recorded()) != 1 {
		t.Errorf("an ordinary stop was held back: %d relayed", len(d.recorded()))
	}
}

func TestARestartSaysWhenTheDaemonDoesNotAnswerOrCannotBeStopped(t *testing.T) {
	slow := &fakeControl{answering: true, version: "dev"} // a start that does not make it answer
	s := startBridgeWithControl(t, &fakeDaemon{}, slow)
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "/cogmer:self-restart"})
	if text := toolText(t, s.next(5*time.Second)); !strings.Contains(text, "not answering") {
		t.Errorf("a restart that did not answer said %q", text)
	}
	stuck := &fakeControl{answering: true, stopErr: errors.New("exit 1"), stopOut: "could not stop the cogmer daemon, pid 7"}
	s = startBridgeWithControl(t, &fakeDaemon{}, stuck)
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p", "prompt": "/cogmer:self-restart"})
	text := toolText(t, s.next(5*time.Second))
	if !strings.Contains(text, "could not be stopped") || !strings.Contains(text, "pid 7") || stuck.startCount() != 0 {
		t.Errorf("a restart that could not stop said %q, starts %d", text, stuck.startCount())
	}
}

func TestAStopLeavesTheMarkThatKeepsOrdinaryPromptsFromStartingOne(t *testing.T) {
	c := &fakeControl{answering: true, version: "dev", startAnswer: true}
	d := &fakeDaemon{answer: func(string, []byte, time.Duration) ([]byte, error) {
		if !c.answering {
			return nil, errors.New("connection refused")
		}
		return []byte(`{"context":"output"}`), nil
	}}
	s := startBridgeWithControl(t, d, c)
	s.callTool(1, "prompt", map[string]any{"session_id": "s", "prompt_id": "p1", "prompt": "/cogmer:self-stop"})
	text := toolText(t, s.next(5*time.Second))
	if !strings.Contains(text, "was stopped") || !strings.Contains(text, "Any /cogmer: command starts it") {
		t.Errorf("a stop answered %q", text)
	}
	if !c.Stopped() || len(c.stops) != 1 || c.stops[0] {
		t.Errorf("stopped %v, stops %v", c.Stopped(), c.stops)
	}
	s.callTool(2, "prompt", map[string]any{"session_id": "s", "prompt_id": "p2", "prompt": "an ordinary prompt"})
	s.next(2 * time.Second)
	if c.startCount() != 0 {
		t.Error("an ordinary prompt started the daemon the user had stopped")
	}
	s.callTool(3, "prompt", map[string]any{"session_id": "s", "prompt_id": "p3", "prompt": "/cogmer:self-status"})
	if text := toolText(t, s.next(5*time.Second)); !strings.Contains(text, "output") {
		t.Errorf("a typed command did not run: %q", text)
	}
	if c.startCount() != 1 || c.Stopped() {
		t.Errorf("starts %d, stopped %v; a typed command should start it and end the stop", c.startCount(), c.Stopped())
	}
}

// The mark is a file: `stop` writes it, a daemon removes it when it starts, and the bridge
// reads it.
func TestTheStoppedMarkIsAFileTheDaemonRemoves(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	var l localDaemon
	if l.Stopped() {
		t.Fatal("a fresh state directory has a stop standing")
	}
	l.SetStopped(true)
	if !l.Stopped() {
		t.Error("the mark was not written")
	}
	if _, err := os.Stat(stoppedMarker()); err != nil {
		t.Error(err)
	}
	l.SetStopped(false)
	if l.Stopped() {
		t.Error("the mark was not removed")
	}
}

// With real programs: the bridge starts a daemon for a typed command, stops it, leaves it
// stopped through an ordinary prompt, and starts it again for the next command.
func TestTheBridgeKeepsARealDaemonForACoworkUser(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary and starts daemons")
	}
	needsToFindADaemon(t)
	// It starts daemons on ports and in a state directory of its own and gives each program its own
	// environment, so it can run beside the others.
	t.Parallel()
	dir := t.TempDir()
	bin := buildVersion(t, dir, "0.0.9")
	hooks, peer := freeAddr(t), freeAddr(t)
	home := filepath.Join(dir, "home")
	env := append(os.Environ(), "COGMER_HOME="+home, "COGMER_ADDR="+hooks, "COGMER_PEER_ADDR="+peer,
		"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_PEERS=", "COGMER_RESTART_WAIT=8")
	t.Cleanup(func() {
		stop := exec.Command(bin, "stop", "--for-restart")
		stop.Env = env
		_ = stop.Run()
	})
	br := exec.Command(bin, "bridge")
	br.Env = env
	in, _ := br.StdinPipe()
	out, _ := br.StdoutPipe()
	if err := br.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { in.Close(); _ = br.Wait() }()
	rd := bufio.NewReader(out)
	id := 0
	rpc := func(method string, params any) map[string]any {
		t.Helper()
		id++
		buf, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
		if _, err := in.Write(append(buf, '\n')); err != nil {
			t.Fatal(err)
		}
		line, err := rd.ReadString('\n')
		if err != nil {
			t.Fatalf("no reply to %s: %v", method, err)
		}
		var m map[string]any
		_ = json.Unmarshal([]byte(line), &m)
		return m
	}
	prompt := func(text string) string {
		t.Helper()
		r := rpc("tools/call", map[string]any{"name": "prompt", "arguments": map[string]any{
			"session_id": "cw1", "prompt_id": text, "prompt": text}})
		res, _ := r["result"].(map[string]any)
		content, _ := res["content"].([]any)
		if len(content) == 0 {
			t.Fatalf("no result: %v", r)
		}
		txt, _ := content[0].(map[string]any)["text"].(string)
		return txt
	}
	answering := func() bool { _, ok := daemonVersionAt(hooks); return ok }
	rpc("initialize", map[string]any{"protocolVersion": "2025-11-25"})
	// the bridge starts a daemon at launch, so wait for it to be up and then stop it by hand
	deadline := time.Now().Add(20 * time.Second)
	for !answering() && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}
	if !answering() {
		t.Fatal("the bridge did not start a daemon when it started")
	}
	if got := prompt("/cogmer:self-stop"); !strings.Contains(got, "was stopped") {
		t.Fatalf("a stop answered %q", got)
	}
	time.Sleep(500 * time.Millisecond)
	if answering() {
		t.Fatal("the daemon was still answering after a stop")
	}
	prompt("an ordinary prompt")
	time.Sleep(1500 * time.Millisecond)
	if answering() {
		t.Fatal("an ordinary prompt started the daemon the user had stopped")
	}
	if got := prompt("/cogmer:self-status"); !strings.Contains(got, "Output of /cogmer:self-status") || !answering() {
		t.Fatalf("a typed command did not start the daemon and run: %q", got)
	}
	if got := prompt("/cogmer:self-restart"); !strings.Contains(got, "was restarted and is answering") {
		t.Fatalf("a restart answered %q", got)
	}
	if !answering() {
		t.Error("no daemon answered after a restart")
	}
}

// `stop` leaves the mark, so a Cowork bridge honors a stop typed in another host, and a
// daemon removes it when it starts, whoever started it.
func TestStopLeavesTheMarkAndAStartingDaemonRemovesIt(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary and starts a daemon")
	}
	needsToFindADaemon(t)
	dir := t.TempDir()
	bin := buildVersion(t, dir, "0.0.9")
	hooks, peer := freeAddr(t), freeAddr(t)
	home := filepath.Join(dir, "home")
	env := append(os.Environ(), "COGMER_HOME="+home, "COGMER_ADDR="+hooks, "COGMER_PEER_ADDR="+peer,
		"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_PEERS=")
	marker := filepath.Join(home, "stopped")
	stop := exec.Command(bin, "stop")
	stop.Env = env
	if out, err := stop.CombinedOutput(); err != nil {
		t.Fatalf("stop: %v\n%s", err, out)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("stop left no mark: %v", err)
	}
	restart := exec.Command(bin, "stop", "--for-restart")
	restart.Env = env
	_ = os.Remove(marker)
	if out, err := restart.CombinedOutput(); err != nil {
		t.Fatalf("stop --for-restart: %v\n%s", err, out)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("a stop for a restart left the mark, which would keep the next daemon from being started")
	}
	_ = os.WriteFile(marker, []byte("x\n"), 0o600)
	daemon := exec.Command(bin, "daemon")
	daemon.Env = env
	if err := daemon.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = daemon.Process.Kill(); _ = daemon.Wait() }()
	waitForVersion(t, hooks, "0.0.9")
	if _, err := os.Stat(marker); err == nil {
		t.Error("a daemon that is running left the user's stop standing")
	}
}
