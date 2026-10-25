//go:build docker

package main

// --- machines in containers ---
//
// Each machine here is a Docker container with its own state directory, key, daemon
// and address, on a network made for one test. A test drives it through the entry
// points a user's machine has: the cogmer commands a slash command runs, the hooks a
// host runs around each turn, and the requests the daemon's browser pages make. No
// real Claude Code runs; a turn is a made-up transcript fed through the hooks.
//
// The images are built FROM scratch, from the working tree and from main, with the
// cogmer binary and testdata/containerdriver, which makes the page's requests from
// inside the container, since the local routes listen on loopback only. Run with:
//
//	go test -tags 'docker ts_debug_websockets' -run Container ./cmd/cogmer

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- images ---

const (
	imageWork = "cogmer-harness-work"
	imageMain = "cogmer-harness-main"
	// imageOlderData is the working tree built to know data version 0, to stand for a
	// program older than the data it finds (D-244, a program that finds a newer data
	// version touches nothing).
	imageOlderData = "cogmer-harness-older-data"
)

var (
	imagesMu    sync.Mutex
	imagesBuilt = map[string]string{} // repository -> tag built by this process
	runTag      = randomHex(4)
	dockerOK    bool
	dockerWhy   string
	dockerOnce  sync.Once
)

func TestMain(m *testing.M) {
	code := m.Run()
	imagesMu.Lock()
	for repo, tag := range imagesBuilt {
		_ = exec.Command("docker", "image", "rm", "-f", repo+":"+tag).Run()
	}
	imagesMu.Unlock()
	os.Exit(code)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// requireDocker skips the test when no Docker daemon answers.
func requireDocker(t *testing.T) {
	t.Helper()
	dockerOnce.Do(func() {
		out, err := exec.Command("docker", "info", "--format", "{{.ServerVersion}}").CombinedOutput()
		dockerOK, dockerWhy = err == nil, strings.TrimSpace(string(out))
	})
	if !dockerOK {
		t.Skipf("docker is not answering, so no container test runs (start dockerd first): %s", dockerWhy)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("finding the repository: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// image returns the image of repo, building it the first time this process asks.
// imageWork is built from the working tree, and imageMain from the commit main names.
// imageOlderData is the working tree with the tag that lowers its data version.
func harnessImage(t *testing.T, repo string) string {
	t.Helper()
	imagesMu.Lock()
	defer imagesMu.Unlock()
	if tag, ok := imagesBuilt[repo]; ok {
		return repo + ":" + tag
	}
	root := repoRoot(t)
	src := root
	if repo == imageMain {
		dir, err := os.MkdirTemp("", "cogmer-main-src-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dir)
		archive := exec.Command("git", "-C", root, "archive", "--format=tar", "main")
		untar := exec.Command("tar", "-x", "-C", dir)
		pipe, err := archive.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		untar.Stdin = pipe
		if err := untar.Start(); err != nil {
			t.Fatal(err)
		}
		if err := archive.Run(); err != nil {
			t.Fatalf("git archive main: %v", err)
		}
		if err := untar.Wait(); err != nil {
			t.Fatalf("extracting main: %v", err)
		}
		src = dir
	}
	ctx, err := os.MkdirTemp("", "cogmer-image-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ctx)
	tags := "ts_debug_websockets"
	if repo == imageOlderData {
		tags += " olderdataversion"
	}
	goBuild(t, src, filepath.Join(ctx, "cogmer"), "./cmd/cogmer", tags)
	goBuild(t, root, filepath.Join(ctx, "containerdriver"), "./cmd/cogmer/testdata/containerdriver", "ts_debug_websockets")
	dockerfile := "FROM scratch\nCOPY cogmer /cogmer\nCOPY containerdriver /containerdriver\nENTRYPOINT [\"/cogmer\"]\n"
	if err := os.WriteFile(filepath.Join(ctx, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("docker", "build", "-q", "-t", repo+":"+runTag, ctx).CombinedOutput(); err != nil {
		t.Fatalf("docker build %s: %v\n%s", repo, err, out)
	}
	imagesBuilt[repo] = runTag
	return repo + ":" + runTag
}

func goBuild(t *testing.T, dir, out, pkg, tags string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-tags", tags, "-o", out, pkg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s in %s: %v\n%s", pkg, dir, err, b)
	}
}

// --- a network of machines ---

// cluster is one test's network and every machine on it.
type cluster struct {
	t        *testing.T
	id       string
	network  string
	mu       sync.Mutex
	machines []*box
}

// newCluster makes a network for t, and removes it, its containers and their state
// volumes when t ends, whether it passed or not.
func newCluster(t *testing.T) *cluster {
	t.Helper()
	requireDocker(t)
	c := &cluster{t: t, id: "ch" + randomHex(4)}
	c.network = c.id + "-net"
	if out, err := exec.Command("docker", "network", "create", "--label", "cogmer-harness="+c.id, c.network).CombinedOutput(); err != nil {
		t.Fatalf("docker network create: %v\n%s", err, out)
	}
	t.Cleanup(c.remove)
	return c
}

func (c *cluster) remove() {
	if c.t.Failed() {
		for _, m := range c.machines {
			c.t.Logf("--- log of %s ---\n%s", m.name, tail(m.logs(), 80))
		}
	}
	ids, _ := exec.Command("docker", "ps", "-aq", "--filter", "label=cogmer-harness="+c.id).Output()
	if f := strings.Fields(string(ids)); len(f) > 0 {
		_ = exec.Command("docker", append([]string{"rm", "-f"}, f...)...).Run()
	}
	vols, _ := exec.Command("docker", "volume", "ls", "-q", "--filter", "label=cogmer-harness="+c.id).Output()
	if f := strings.Fields(string(vols)); len(f) > 0 {
		_ = exec.Command("docker", append([]string{"volume", "rm", "-f"}, f...)...).Run()
	}
	_ = exec.Command("docker", "network", "rm", c.network).Run()
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// machine is one container. Its state lives in a volume, so the machine can be
// stopped and started again, on another image, with the same key and rooms.
type box struct {
	c         *cluster
	name      string // its hostname, and the name it is reached by on the network
	container string
	volume    string
	image     string
	oldLogs   string // what earlier containers of this machine logged
	running   bool

	mu       sync.Mutex
	sessions map[string]*boxSession
}

// session is an agent session on a machine, whose transcript the hooks read.
type boxSession struct {
	id         string
	transcript string
	lines      []string
}

// start makes a machine called name, running a daemon from img, and waits until the
// daemon answers.
func (c *cluster) start(name, img string) *box {
	c.t.Helper()
	m := &box{c: c, name: name, container: c.id + "-" + name, volume: c.id + "-" + name, sessions: map[string]*boxSession{}}
	if out, err := exec.Command("docker", "volume", "create", "--label", "cogmer-harness="+c.id, m.volume).CombinedOutput(); err != nil {
		c.t.Fatalf("docker volume create: %v\n%s", err, out)
	}
	c.mu.Lock()
	c.machines = append(c.machines, m)
	c.mu.Unlock()
	m.run(img)
	return m
}

func (m *box) run(img string) {
	t := m.c.t
	t.Helper()
	m.image = img
	args := []string{"run", "-d", "--name", m.container, "--hostname", m.name,
		"--label", "cogmer-harness=" + m.c.id,
		"--network", m.c.network, "--network-alias", m.name,
		"-v", m.volume + ":/h"}
	for _, e := range m.env() {
		args = append(args, "-e", e)
	}
	args = append(args, img, "daemon")
	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("starting %s: %v\n%s", m.name, err, out)
	}
	m.running = true
	m.waitHealthy()
}

// env is the machine's environment: its state under /h, no relay, no behavior checks,
// a peer address other containers can reach, and Claude Code as the host.
func (m *box) env() []string {
	return []string{
		"HOME=/h", "COGMER_HOME=/h/.cogmer", "COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off",
		"COGMER_PEER_ADDR=0.0.0.0:4783", "COGMER_PEER_ENDPOINT=" + m.name + ":4783", "COGMER_HOST=claude",
	}
}

func (m *box) waitHealthy() {
	t := m.c.t
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, err := m.drive("get", "/healthz", nil)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the daemon on %s did not answer within 30s: %v\n%s", m.name, err, tail(m.logs(), 40))
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// stop ends the daemon with SIGTERM, as `cogmer stop` does, and removes the
// container. The state volume stays.
func (m *box) stop() { m.end("stop") }

// kill ends the daemon with SIGKILL, as a crash or a power cut would.
func (m *box) kill() { m.end("kill") }

func (m *box) end(how string) {
	t := m.c.t
	t.Helper()
	if !m.running {
		return
	}
	if out, err := exec.Command("docker", how, m.container).CombinedOutput(); err != nil {
		t.Fatalf("docker %s %s: %v\n%s", how, m.name, err, out)
	}
	m.oldLogs = m.logs()
	if out, err := exec.Command("docker", "rm", "-f", m.container).CombinedOutput(); err != nil {
		t.Fatalf("removing %s: %v\n%s", m.name, err, out)
	}
	m.running = false
}

// restart starts the machine again on img, with the state it had.
func (m *box) restart(img string) {
	m.c.t.Helper()
	m.end("stop")
	m.run(img)
}

// disconnect takes the machine off the network, and connect puts it back under its
// name, so a partition can be made and healed.
func (m *box) disconnect() {
	if out, err := exec.Command("docker", "network", "disconnect", m.c.network, m.container).CombinedOutput(); err != nil {
		m.c.t.Fatalf("disconnecting %s: %v\n%s", m.name, err, out)
	}
}

func (m *box) connect() {
	if out, err := exec.Command("docker", "network", "connect", "--alias", m.name, m.c.network, m.container).CombinedOutput(); err != nil {
		m.c.t.Fatalf("connecting %s: %v\n%s", m.name, err, out)
	}
}

// logs is everything the machine's daemons have logged, across restarts.
func (m *box) logs() string {
	if !m.running {
		return m.oldLogs
	}
	out, _ := exec.Command("docker", "logs", m.container).CombinedOutput()
	return m.oldLogs + string(out)
}

// exec runs a program in the container. stdin may be nil.
func (m *box) exec(env []string, stdin []byte, argv ...string) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	args := []string{"exec"}
	if stdin != nil {
		args = append(args, "-i")
	}
	for _, e := range env {
		args = append(args, "-e", e)
	}
	args = append(args, m.container)
	args = append(args, argv...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err = cmd.Run()
	return o.String(), e.String(), err
}

// cli runs a cogmer command as a slash command would, from inside the session named,
// or at a terminal when session is empty. It returns stdout and stderr together.
func (m *box) cli(session string, args ...string) (string, error) {
	var env []string
	if session != "" {
		env = append(env, "CLAUDE_CODE_SESSION_ID="+session)
	}
	out, errOut, err := m.exec(env, nil, append([]string{"/cogmer"}, args...)...)
	return out + errOut, err
}

// mustCLI is cli that fails the test when the command fails.
func (m *box) mustCLI(session string, args ...string) string {
	m.c.t.Helper()
	out, err := m.cli(session, args...)
	if err != nil {
		m.c.t.Fatalf("%s: cogmer %s: %v\n%s", m.name, strings.Join(args, " "), err, out)
	}
	return out
}

// drive runs the driver, which makes a request the daemon's page makes, or writes a file.
func (m *box) drive(act, arg string, stdin []byte) (string, error) {
	if stdin == nil {
		stdin = []byte{}
	}
	out, errOut, err := m.exec(nil, stdin, "/containerdriver", act, arg)
	if err != nil {
		return out, fmt.Errorf("%v: %s", err, strings.TrimSpace(errOut))
	}
	return out, nil
}

// post makes a request of a page, with body as JSON, and decodes the reply into out.
func (m *box) post(path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	reply, err := m.drive("post", path, raw)
	if err != nil {
		return fmt.Errorf("%s %s: %w", m.name, path, err)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal([]byte(reply), out)
}

func (m *box) mustPost(path string, body, out any) {
	m.c.t.Helper()
	if err := m.post(path, body, out); err != nil {
		m.c.t.Fatal(err)
	}
}

// --- what a session is given, and what the view shows ---

// viewEvent is a turn as the room view serves it.
type viewEvent struct {
	PeerID    string `json:"peerId"`
	Signer    string `json:"signer"`
	MachineID string `json:"machineId"`
	EventType string `json:"eventType"`
	Content   string `json:"content"`
}

// view is the room's turns as the browser view on this machine shows them.
func (m *box) view(room string) []viewEvent {
	m.c.t.Helper()
	raw, err := m.drive("get", "/events?room="+room, nil)
	if err != nil {
		m.c.t.Fatalf("%s: reading the view of %s: %v", m.name, room, err)
	}
	var out struct {
		Events []viewEvent `json:"events"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		m.c.t.Fatalf("%s: the view of %s did not read: %v", m.name, room, err)
	}
	return out.Events
}

func (m *box) viewHas(room, text string) bool {
	for _, e := range m.view(room) {
		if e.Content == text {
			return true
		}
	}
	return false
}

// injectedTurnSeen is a turn as the injected block gives it to a session.
type injectedTurnSeen struct {
	Speaker  string `json:"speaker"`
	Label    string `json:"label"`
	PeerName string `json:"peerName"`
	Verified bool   `json:"verified"`
	YourUser bool   `json:"yourUser"`
	Machine  *struct {
		Name string `json:"name"`
	} `json:"machine"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// injectedTurns reads the turns out of what a prompt hook gave its session.
func injectedTurns(block string) []injectedTurnSeen {
	i := strings.Index(block, `{"turns":`)
	if i < 0 {
		return nil
	}
	var out struct {
		Turns []injectedTurnSeen `json:"turns"`
	}
	if err := json.NewDecoder(strings.NewReader(block[i:])).Decode(&out); err != nil {
		return nil
	}
	return out.Turns
}

func findTurn(block, text string) (injectedTurnSeen, bool) {
	for _, tr := range injectedTurns(block) {
		if tr.Text == text {
			return tr, true
		}
	}
	return injectedTurnSeen{}, false
}

// turn runs one turn of a session as Claude Code would: the prompt hook, then the
// transcript the host writes, holding the user's prompt and the hook's output, then
// the stop hook with the assistant's reply. It returns what the prompt hook gave the
// session, which is what the model would read.
func (m *box) turn(sessionID, prompt, reply string) string {
	t := m.c.t
	t.Helper()
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	if !ok {
		s = &boxSession{id: sessionID, transcript: "/h/transcripts/" + sessionID + ".jsonl"}
		m.sessions[sessionID] = s
	}
	m.mu.Unlock()
	promptID := randomHex(8)
	payload, _ := json.Marshal(map[string]any{
		"session_id": sessionID, "prompt": prompt, "prompt_id": promptID, "cwd": "/h",
		"transcript_path": s.transcript, "hook_event_name": "UserPromptSubmit",
	})
	given, errOut, err := m.exec(nil, payload, "/cogmer", "hook", "prompt")
	if err != nil || strings.Contains(errOut, "daemon unreachable") {
		t.Fatalf("%s: the prompt hook failed: %v %s", m.name, err, errOut)
	}
	user, _ := json.Marshal(map[string]any{
		"type": "user", "promptSource": "typed", "isSidechain": false, "sessionId": sessionID,
		"promptId": promptID, "message": map[string]any{"role": "user", "content": prompt},
	})
	s.lines = append(s.lines, string(user))
	if strings.TrimSpace(given) != "" {
		att, _ := json.Marshal(map[string]any{"type": "attachment", "attachment": map[string]any{
			"type": "hook_success", "hookName": "UserPromptSubmit", "hookEvent": "UserPromptSubmit",
			"content": strings.TrimSpace(given)}})
		s.lines = append(s.lines, string(att))
	}
	if _, err := m.drive("write", s.transcript, []byte(strings.Join(s.lines, "\n")+"\n")); err != nil {
		t.Fatalf("%s: writing the transcript: %v", m.name, err)
	}
	stop, _ := json.Marshal(map[string]any{
		"session_id": sessionID, "prompt_id": promptID, "transcript_path": s.transcript, "cwd": "/h",
		"last_assistant_message": reply, "hook_event_name": "Stop",
	})
	if _, errOut, err := m.exec(nil, stop, "/cogmer", "hook", "stop"); err != nil || strings.Contains(errOut, "daemon unreachable") {
		t.Fatalf("%s: the stop hook failed: %v %s", m.name, err, errOut)
	}
	return given
}

// eventually polls cond until it holds or the time is up, and then fails with what
// cond last said.
func eventually(t *testing.T, within time.Duration, what string, cond func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(within)
	for {
		ok, why := cond()
		if ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s did not happen within %s: %s", what, within, why)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// --- acts a person does ---

var pairingLine = regexp.MustCompile(`Send your colleague this[^\n]*\n\s*\n\s*(\S+)`)

// pairingString is what the machine's user sends a colleague, as self-status prints it.
func (m *box) pairingString() string {
	m.c.t.Helper()
	out := m.mustCLI("", "whoami")
	match := pairingLine.FindStringSubmatch(out)
	if match == nil {
		m.c.t.Fatalf("%s printed no pairing string:\n%s", m.name, out)
	}
	return match[1]
}

// pair has the users of a and b pair, both at once as on a call: each runs the pair
// command with the other's string, reads the two words their own machine shows, and
// says yes only when the words are the same. aCallsB is what a's user calls b's.
func pairBoxes(t *testing.T, a, b *box, aCallsB, bCallsA string) {
	t.Helper()
	sa, sb := a.pairingString(), b.pairingString()
	pa := startContainerPairing(t, a, sb, aCallsB)
	pb := startContainerPairing(t, b, sa, bCallsA)
	wa, wb := pa.words(t), pb.words(t)
	if wa == "" || wa != wb {
		pa.answer("n")
		pb.answer("n")
		t.Fatalf("the two words differ: %s shows %q and %s shows %q", a.name, wa, b.name, wb)
	}
	pa.answer("y")
	pb.answer("y")
	for _, p := range []*boxPairing{pa, pb} {
		out := p.wait(t)
		if !strings.Contains(out, "paired with") {
			t.Fatalf("%s did not record the pairing:\n%s", p.m.name, out)
		}
	}
}

// pairing is a pair command running in a container, waiting for its person's answer.
type boxPairing struct {
	m      *box
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan string
	output strings.Builder
	mu     sync.Mutex
	done   chan error
}

func startContainerPairing(t *testing.T, m *box, theirs, name string) *boxPairing {
	t.Helper()
	cmd := exec.Command("docker", "exec", "-i", m.container, "/cogmer", "pair", theirs, name, "--terminal")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout
	p := &boxPairing{m: m, cmd: cmd, stdin: stdin, lines: make(chan string, 100), done: make(chan error, 1)}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			p.mu.Lock()
			p.output.WriteString(sc.Text() + "\n")
			p.mu.Unlock()
			p.lines <- sc.Text()
		}
		close(p.lines)
		p.done <- cmd.Wait()
	}()
	return p
}

// words reads the two words the command shows, which it prints on a line of their
// own before asking whether they match.
func (p *boxPairing) words(t *testing.T) string {
	t.Helper()
	timeout := time.After(2 * time.Minute)
	prev := ""
	for {
		select {
		case line, ok := <-p.lines:
			if !ok {
				t.Fatalf("%s: pairing ended without showing words:\n%s", p.m.name, p.text())
			}
			if strings.HasPrefix(line, "say those two words aloud") {
				return strings.TrimSpace(prev)
			}
			if strings.TrimSpace(line) != "" {
				prev = line
			}
		case <-timeout:
			t.Fatalf("%s: no words within 2m:\n%s", p.m.name, p.text())
		}
	}
}

func (p *boxPairing) answer(a string) {
	_, _ = io.WriteString(p.stdin, a+"\n")
	_ = p.stdin.Close()
}

func (p *boxPairing) text() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.output.String()
}

func (p *boxPairing) wait(t *testing.T) string {
	t.Helper()
	go func() {
		for range p.lines {
		}
	}()
	select {
	case <-p.done:
	case <-time.After(time.Minute):
		_ = p.cmd.Process.Kill()
		t.Fatalf("%s: pairing did not finish:\n%s", p.m.name, p.text())
	}
	return p.text()
}

// createUser makes the machine's user identity on the identity page: the page draws
// the phrase and shows it, the user writes it down and types back the words the page
// asks for, and names themselves and this machine. It returns the user's identifier.
func (m *box) createUser(display, machineName string) string {
	t := m.c.t
	t.Helper()
	var ph phraseResponse
	m.mustPost("/identity/phrase", struct{}{}, &ph)
	if ph.Error != "" {
		t.Fatalf("%s: the identity page drew no phrase: %s", m.name, ph.Error)
	}
	typed := map[int]string{}
	for _, pos := range ph.Confirm {
		typed[pos] = ph.Words[pos-1]
	}
	var res identityResult
	m.mustPost("/identity/create", createRequest{CreateID: ph.CreateID, Words: typed, DisplayName: display, MachineName: machineName}, &res)
	if res.Error != "" || res.User == "" {
		t.Fatalf("%s: the identity page made no user: %s", m.name, res.Error)
	}
	return res.User
}

// addMachine adds fresh, a machine with no user identity, to the user of existing,
// with the machine words: fresh's page shows its string, the user enters it in
// existing's page, both pages show two words, and the user confirms on both only when
// the words are the same. name is what the user calls fresh.
func addBoxMachine(t *testing.T, existing, fresh *box, name string) {
	t.Helper()
	var js joinStartResponse
	fresh.mustPost("/identity/join/start", struct{}{}, &js)
	if js.Error != "" {
		t.Fatalf("%s: the page offered no string: %s", fresh.name, js.Error)
	}
	var as addStartResponse
	existing.mustPost("/machines/add/start", addStartRequest{String: js.String}, &as)
	if as.Error != "" {
		t.Fatalf("%s: adding %s did not start: %s", existing.name, fresh.name, as.Error)
	}
	var st joinStatus
	eventually(t, 30*time.Second, fresh.name+"'s page showing words", func() (bool, string) {
		st = joinStatus{}
		if err := fresh.post("/identity/join/status", struct{}{}, &st); err != nil {
			return false, err.Error()
		}
		return st.State == "compare", st.State + " " + st.Error
	})
	if st.Words != as.Words {
		_ = existing.post("/machines/add/confirm", addConfirmRequest{AddID: as.AddID, Matched: false}, nil)
		t.Fatalf("the machine words differ: %s shows %q and %s shows %q", existing.name, as.Words, fresh.name, st.Words)
	}
	var wg sync.WaitGroup
	var confirm addConfirmResponse
	var confirmErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		confirmErr = existing.post("/machines/add/confirm", addConfirmRequest{AddID: as.AddID, Matched: true}, &confirm)
	}()
	var jc joinStatus
	fresh.mustPost("/identity/join/confirm", joinConfirmRequest{Matched: true, MachineName: name}, &jc)
	wg.Wait()
	if confirmErr != nil || confirm.Status != "added" {
		t.Fatalf("%s did not add %s: %v %s", existing.name, fresh.name, confirmErr, confirm.Error)
	}
	eventually(t, 30*time.Second, fresh.name+" taking its certificate", func() (bool, string) {
		st = joinStatus{}
		_ = fresh.post("/identity/join/status", struct{}{}, &st)
		return st.State == "added", st.State + " " + st.Error
	})
}

var createdLine = regexp.MustCompile(`(?m)^created (\S+)`)

// createRoom makes a room from session, and returns its name.
func (m *box) createRoom(session string) string {
	m.c.t.Helper()
	out := m.mustCLI(session, "create")
	match := createdLine.FindStringSubmatch(out)
	if match == nil {
		m.c.t.Fatalf("%s: create printed no room:\n%s", m.name, out)
	}
	return match[1]
}

// --- scenarios ---

// room is two users, each with a session in one room.
type sharedRoom struct {
	name         string
	alice, bob   *box
	aliceS, bobS string
}

// pairedRoom starts alice and bob on img, makes their user identities when withUsers,
// pairs them, and has alice create a room, invite bob, and bob join it.
func pairedRoom(t *testing.T, c *cluster, img string, withUsers bool) *sharedRoom {
	t.Helper()
	r := &sharedRoom{alice: c.start("alice", img), bob: c.start("bob", img), aliceS: "session-alice-1", bobS: "session-bob-1"}
	if withUsers {
		r.alice.createUser("Alice", "alice-laptop")
		r.bob.createUser("Bob", "bob-desktop")
	}
	pairBoxes(t, r.alice, r.bob, "bob", "alice")
	r.name = r.alice.createRoom(r.aliceS)
	out := r.alice.mustCLI(r.aliceS, "invite", "bob")
	if !strings.Contains(out, "Told them") {
		t.Fatalf("alice's invitation did not reach bob:\n%s", out)
	}
	out = r.bob.mustCLI(r.bobS, "join", r.name)
	if !strings.Contains(out, "joined "+r.name) {
		t.Fatalf("bob did not join %s:\n%s", r.name, out)
	}
	return r
}

// handOver has from say text in its session, waits until to's view holds it, and
// returns what to's session is given at its next prompt.
func handOver(t *testing.T, roomName string, from *box, fromS string, to *box, toS, text string) string {
	t.Helper()
	from.turn(fromS, text, "noted: "+text)
	eventually(t, 60*time.Second, "the turn "+text+" reaching "+to.name, func() (bool, string) {
		return to.viewHas(roomName, text), "not in the view"
	})
	return to.turn(toS, "what is new from "+from.name+"?", "nothing to add")
}

func mustSee(t *testing.T, who, block, text string) injectedTurnSeen {
	t.Helper()
	tr, ok := findTurn(block, text)
	if !ok {
		t.Fatalf("%s's next prompt was not given %q; it was given:\n%s", who, text, block)
	}
	return tr
}

// TestContainerMainPairsAndSharesATurnEachWay runs main's binary on two machines: two
// users pair, one creates a room and invites the other, and each one's turn is given
// to the other's session at its next prompt.
func TestContainerMainPairsAndSharesATurnEachWay(t *testing.T) {
	c := newCluster(t)
	r := pairedRoom(t, c, harnessImage(t, imageMain), false)

	got := handOver(t, r.name, r.alice, r.aliceS, r.bob, r.bobS, "alice says the parser is done")
	if tr := mustSee(t, "bob", got, "alice says the parser is done"); !tr.Verified {
		t.Fatalf("bob was given alice's turn as unverified")
	}
	got = handOver(t, r.name, r.bob, r.bobS, r.alice, r.aliceS, "bob says the tests pass")
	if tr := mustSee(t, "alice", got, "bob says the tests pass"); tr.Label != "bob" || !tr.Verified {
		t.Fatalf("alice was given bob's turn labeled %q, verified %v", tr.Label, tr.Verified)
	}
}

// TestContainerMainTurnCrossesAPartitionAndACrash takes bob's machine off the network
// while alice speaks, then kills it and starts it again on the same state, and alice's
// turns reach bob's session once each is over.
func TestContainerMainTurnCrossesAPartitionAndACrash(t *testing.T) {
	c := newCluster(t)
	img := harnessImage(t, imageMain)
	r := pairedRoom(t, c, img, false)

	r.bob.disconnect()
	r.alice.turn(r.aliceS, "alice says this while bob is cut off", "noted")
	time.Sleep(3 * time.Second)
	if r.bob.viewHas(r.name, "alice says this while bob is cut off") {
		t.Fatal("a turn reached bob while his machine was off the network")
	}
	r.bob.connect()
	eventually(t, 60*time.Second, "the turn reaching bob once the network is back", func() (bool, string) {
		return r.bob.viewHas(r.name, "alice says this while bob is cut off"), "not in bob's view"
	})
	mustSee(t, "bob", r.bob.turn(r.bobS, "anything new?", "no"), "alice says this while bob is cut off")

	r.bob.kill()
	r.alice.turn(r.aliceS, "alice says this while bob is down", "noted")
	r.bob.restart(img)
	got := handOver(t, r.name, r.alice, r.aliceS, r.bob, r.bobS, "alice says this after bob is back")
	mustSee(t, "bob", got, "alice says this while bob is down")
	mustSee(t, "bob", got, "alice says this after bob is back")
}

// TestContainerUsersMachineAddedWithWordsSharesTheRoom runs this tree's binary: two
// users make their identities in the identity page, pair, and share a room, and bob
// adds a second machine with the machine words. A turn from bob's second machine
// reaches alice as bob's, and reaches bob's first machine marked as bob's own (D-232,
// a user's own turn from another session is injected as a report marked as the user's).
func TestContainerUsersMachineAddedWithWordsSharesTheRoom(t *testing.T) {
	c := newCluster(t)
	r := pairedRoom(t, c, harnessImage(t, imageWork), true)

	got := handOver(t, r.name, r.alice, r.aliceS, r.bob, r.bobS, "alice says the parser is done")
	mustSee(t, "bob", got, "alice says the parser is done")

	laptop := c.start("bob-laptop", harnessImage(t, imageWork))
	addBoxMachine(t, r.bob, laptop, "bob-laptop")
	laptopS := "session-bob-2"
	out := laptop.mustCLI(laptopS, "join", r.name)
	if !strings.Contains(out, "joined "+r.name) {
		t.Fatalf("bob's laptop did not join %s by name:\n%s", r.name, out)
	}

	laptop.turn(laptopS, "bob on the laptop says the deploy is green", "noted")
	for _, to := range []struct {
		m *box
		s string
	}{{r.alice, r.aliceS}, {r.bob, r.bobS}} {
		eventually(t, 60*time.Second, "the laptop's turn reaching "+to.m.name, func() (bool, string) {
			return to.m.viewHas(r.name, "bob on the laptop says the deploy is green"), "not in the view"
		})
	}
	got = r.alice.turn(r.aliceS, "anything from bob?", "no")
	tr := mustSee(t, "alice", got, "bob on the laptop says the deploy is green")
	if tr.Label != "bob" || !tr.Verified || tr.YourUser {
		t.Fatalf("alice was given the laptop's turn labeled %q, verified %v, as her own %v", tr.Label, tr.Verified, tr.YourUser)
	}
	if tr.Machine == nil || tr.Machine.Name != "bob-laptop" {
		t.Fatalf("alice was given the laptop's turn with machine %+v", tr.Machine)
	}
	got = r.bob.turn(r.bobS, "what did I do on the laptop?", "deployed")
	tr = mustSee(t, "bob", got, "bob on the laptop says the deploy is green")
	if !tr.YourUser || tr.Label != "you" {
		t.Fatalf("bob's desktop was given his laptop's turn labeled %q, as his own %v", tr.Label, tr.YourUser)
	}

	got = handOver(t, r.name, r.alice, r.aliceS, laptop, laptopS, "alice says welcome, laptop")
	mustSee(t, "bob's laptop", got, "alice says welcome, laptop")
}

// TestContainerUpgradeKeepsAPairedRoom pairs two users and shares a room on main's
// binary, then moves each machine to this tree's binary on the same state, one at a
// time, and makes its user identity. An old machine key carries its verification,
// label and turns to its user (D-242), and the old room takes one guest list (D-243),
// so the room keeps working without pairing again, and nothing is lost while only one
// side is upgraded.
func TestContainerUpgradeKeepsAPairedRoom(t *testing.T) {
	c := newCluster(t)
	r := pairedRoom(t, c, harnessImage(t, imageMain), false)
	handOver(t, r.name, r.alice, r.aliceS, r.bob, r.bobS, "before: alice")
	handOver(t, r.name, r.bob, r.bobS, r.alice, r.aliceS, "before: bob")

	// Bob upgrades first and makes his identity. Each says something while only bob is
	// upgraded, and alice's machine, which cannot read what bob's now writes, says to
	// upgrade: main says "upgrade the older side" for a protocol it does not speak, and
	// "upgrade rather than discard it" for an event under a scheme it does not know.
	r.bob.restart(harnessImage(t, imageWork))
	r.bob.createUser("Bob", "bob-desktop")
	r.bob.turn(r.bobS, "during: bob, upgraded", "noted")
	r.alice.turn(r.aliceS, "during: alice, not upgraded", "noted")
	// main logs a peer's failure when it starts and every ten minutes after, so the
	// reason is read where the room status reads it, the daemon's answer about its
	// peers, as well as in the log.
	eventually(t, 60*time.Second, "alice's machine saying to upgrade", func() (bool, string) {
		peers, _ := r.alice.drive("get", "/peers", nil)
		l := r.alice.logs() + peers
		return strings.Contains(l, "upgrade the older side") || strings.Contains(l, "upgrade rather than discard it"), "not in alice's log or peer status"
	})

	// Alice upgrades and makes hers; nobody pairs again, and neither turn from while
	// only bob was upgraded is lost.
	r.alice.restart(harnessImage(t, imageWork))
	r.alice.createUser("Alice", "alice-laptop")
	eventually(t, 60*time.Second, "bob's turn from while only he was upgraded reaching alice", func() (bool, string) {
		return r.alice.viewHas(r.name, "during: bob, upgraded"), "not in alice's view"
	})
	eventually(t, 60*time.Second, "alice's turn from while only bob was upgraded reaching bob", func() (bool, string) {
		return r.bob.viewHas(r.name, "during: alice, not upgraded"), "not in bob's view"
	})
	mustSee(t, "bob", r.bob.turn(r.bobS, "what did alice say meanwhile?", "noted"), "during: alice, not upgraded")
	got := r.alice.turn(r.aliceS, "what did bob say meanwhile?", "noted")
	if tr := mustSee(t, "alice", got, "during: bob, upgraded"); tr.Label != "bob" || !tr.Verified {
		t.Fatalf("alice was given bob's turn labeled %q, verified %v", tr.Label, tr.Verified)
	}
	got = handOver(t, r.name, r.alice, r.aliceS, r.bob, r.bobS, "after: alice")
	if tr := mustSee(t, "bob", got, "after: alice"); !tr.Verified {
		t.Fatalf("bob was given alice's turn as unverified")
	}
	got = handOver(t, r.name, r.bob, r.bobS, r.alice, r.aliceS, "after: bob")
	if tr := mustSee(t, "alice", got, "after: bob"); tr.Label != "bob" || !tr.Verified {
		t.Fatalf("alice was given bob's turn labeled %q, verified %v", tr.Label, tr.Verified)
	}
}
