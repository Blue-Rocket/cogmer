//go:build docker

package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// stateFiles is every file in the machine's state directory, by path, with a hash of
// its bytes. It reads the volume through a container that is created and never
// started, so nothing runs against the state while it is read.
func (m *box) stateFiles() map[string]string {
	t := m.c.t
	t.Helper()
	id, err := exec.Command("docker", "create", "--label", "cogmer-harness="+m.c.id, "-v", m.volume+":/h", harnessImage(t, imageWork)).Output()
	if err != nil {
		t.Fatalf("creating a container to read %s's state: %v", m.name, err)
	}
	ctr := strings.TrimSpace(string(id))
	defer exec.Command("docker", "rm", "-f", ctr).Run()
	raw, err := exec.Command("docker", "cp", ctr+":/h/.cogmer", "-").Output()
	if err != nil {
		t.Fatalf("reading %s's state: %v", m.name, err)
	}
	files := map[string]string{}
	tr := tar.NewReader(bytes.NewReader(raw))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading %s's state: %v", m.name, err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		files[h.Name] = hex.EncodeToString(sum[:])
		if strings.HasSuffix(h.Name, "/"+dataVersionFile) {
			files[dataVersionFile+" holds"] = strings.TrimSpace(string(b))
		}
	}
	return files
}

// once runs one cogmer process on img against the machine's state, while no daemon of
// the machine runs, and returns what it wrote to stdout and stderr. It fails the test
// when the process has not exited within 30s.
func (m *box) once(img string, env []string, stdin []byte, args ...string) (stdout, stderr string) {
	t := m.c.t
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	argv := []string{"run", "--rm", "-i", "--label", "cogmer-harness=" + m.c.id, "--network", m.c.network, "-v", m.volume + ":/h"}
	for _, e := range append(m.env(), env...) {
		argv = append(argv, "-e", e)
	}
	argv = append(append(argv, img), args...)
	cmd := exec.CommandContext(ctx, "docker", argv...)
	cmd.Stdin = bytes.NewReader(stdin)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	_ = cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("%s: cogmer %s on %s did not exit within 30s\n%s%s", m.name, strings.Join(args, " "), img, o.String(), e.String())
	}
	return o.String(), e.String()
}

// TestContainerAnOlderProgramLeavesUpgradedStateUntouched runs this tree on a machine
// until its state is upgraded, with a user identity, a room and a turn, and then runs a
// program that knows only an older data version on the same state: its daemon does not
// start, its hooks say nothing, its command says to update the plugin, and every file
// in the state directory is byte for byte as it was (D-244, a program that finds a
// newer data version touches nothing). The machine then runs this tree again on the
// same state and still has the turn.
func TestContainerAnOlderProgramLeavesUpgradedStateUntouched(t *testing.T) {
	c := newCluster(t)
	work := harnessImage(t, imageWork)
	m := c.start("alice", work)
	m.createUser("Alice", "alice-laptop")
	session := "session-alice-1"
	room := m.createRoom(session)
	m.turn(session, "said before the older program ran", "noted")
	eventually(t, 30*time.Second, "the turn in alice's view", func() (bool, string) {
		return m.viewHas(room, "said before the older program ran"), "not in the view"
	})
	m.stop()

	before := m.stateFiles()
	if got := before[dataVersionFile+" holds"]; got != strconv.Itoa(dataVersion) {
		t.Fatalf("the upgraded state holds data version %q, want %d", got, dataVersion)
	}

	older := harnessImage(t, imageOlderData)
	out, errOut := m.once(older, nil, nil, "daemon")
	if !strings.Contains(out+errOut, "not starting") {
		t.Errorf("the older daemon did not say why it did not start:\n%s%s", out, errOut)
	}
	payload := []byte(`{"session_id":"session-alice-1","prompt":"said to the older program","prompt_id":"p-older","cwd":"/h","transcript_path":"/h/transcripts/session-alice-1.jsonl","hook_event_name":"UserPromptSubmit"}`)
	for _, kind := range []string{"prompt", "stop"} {
		if out, errOut := m.once(older, nil, payload, "hook", kind); out != "" || errOut != "" {
			t.Errorf("the older %s hook was not silent: stdout %q, stderr %q", kind, out, errOut)
		}
	}
	out, errOut = m.once(older, []string{"CLAUDE_CODE_SESSION_ID=" + session}, nil, "whoami")
	if !strings.Contains(out, "written by a newer cogmer") || !strings.Contains(out, "Updating the cogmer plugin") {
		t.Errorf("the older command did not say the plugin needs updating:\n%s%s", out, errOut)
	}

	after := m.stateFiles()
	for p, h := range before {
		if after[p] != h {
			t.Errorf("the older program changed %s", p)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			t.Errorf("the older program wrote %s", p)
		}
	}

	m.run(work)
	if !m.viewHas(room, "said before the older program ran") {
		t.Errorf("after the older program ran, this tree's view of %s lost the turn", room)
	}
}
