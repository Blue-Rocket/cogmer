//go:build !windows

package main

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShouldReplace(t *testing.T) {
	cases := []struct {
		running, mine string
		want          bool
	}{
		{"0.7.2", "0.7.3", true},
		{"0.7.3", "0.7.2", true}, // the plugin may move a machine back
		{"", "0.7.3", true},      // a daemon from before /healthz carried a version
		{"0.7.3", "0.7.3", false},
		{"dev", "0.7.3", false}, // a maintainer's hand-started build
		{"0.7.3", "dev", false},
	}
	for _, c := range cases {
		if got := shouldReplace(c.running, c.mine); got != c.want {
			t.Errorf("shouldReplace(%q, %q) = %v, want %v", c.running, c.mine, got, c.want)
		}
	}
}

// The helper a test runs as a separate process: something listening on a port
// that is not a cogmer daemon, because its executable is cogmer.test.
func TestHelperListener(t *testing.T) {
	addr := os.Getenv("COGMER_TEST_LISTEN")
	if addr == "" {
		t.Skip("run as a helper process only")
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		os.Exit(3)
	}
	os.Stdout.WriteString("listening\n")
	for {
		c, err := l.Accept()
		if err != nil {
			os.Exit(0)
		}
		c.Close()
	}
}

func TestStopDeclinesAProcessNotNamedCogmer(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("no lsof on this machine")
	}
	addr := freeAddr(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperListener$")
	cmd.Env = append(os.Environ(), "COGMER_TEST_LISTEN="+addr)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	if line, _ := bufio.NewReader(out).ReadString('\n'); strings.TrimSpace(line) != "listening" {
		t.Fatalf("helper did not start listening: %q", line)
	}

	outcomes, err := stopDaemonsAt([]string{addr})
	if err != nil {
		t.Fatalf("stopDaemonsAt: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].PID != cmd.Process.Pid {
		t.Fatalf("expected one outcome for pid %d, got %+v", cmd.Process.Pid, outcomes)
	}
	if o := outcomes[0]; o.Stopped || o.Err != nil || o.Name == "cogmer" {
		t.Fatalf("stop acted on a process that is not a cogmer daemon: %+v", o)
	}
	if conn, err := net.Dial("tcp", addr); err != nil {
		t.Fatalf("the declined process stopped listening: %v", err)
	} else {
		conn.Close()
	}
}

// After an update, the daemon the hook starts replaces one of another version
// that is still serving, and `stop` then ends it. Two real builds, two real
// processes, with the overlay off so nothing leaves the machine.
func TestADaemonOfAnotherVersionIsReplaced(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary twice")
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("no lsof on this machine")
	}
	dir := t.TempDir()
	old := buildVersion(t, dir, "0.0.1")
	cur := buildVersion(t, dir, "0.0.2")

	hooks, peer := freeAddr(t), freeAddr(t)
	env := append(os.Environ(),
		"COGMER_HOME="+filepath.Join(dir, "home"),
		"COGMER_ADDR="+hooks, "COGMER_PEER_ADDR="+peer,
		"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_PEERS=")

	first := exec.Command(old, "daemon")
	first.Env = env
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- first.Wait() }()
	defer func() { _ = first.Process.Kill() }()
	waitForVersion(t, hooks, "0.0.1")

	second := exec.Command(cur, "daemon")
	second.Env = env
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Process.Kill(); _ = second.Wait() }()
	waitForVersion(t, hooks, "0.0.2")

	select {
	case <-exited:
	case <-time.After(10 * time.Second):
		t.Fatal("the 0.0.1 daemon was still running after 0.0.2 took its address")
	}

	stop := exec.Command(cur, "stop")
	stop.Env = env
	if out, err := stop.CombinedOutput(); err != nil || !strings.Contains(string(out), "stopped the cogmer daemon") {
		t.Fatalf("cogmer stop: %v\n%s", err, out)
	}
	if _, ok := daemonVersionAt(hooks); ok {
		t.Fatal("a daemon still answers after cogmer stop")
	}
}

func buildVersion(t *testing.T, dir, v string) string {
	t.Helper()
	out := filepath.Join(dir, v, "cogmer")
	b, err := exec.Command("go", "build", "-ldflags", "-X main.version="+v, "-o", out, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build %s: %v\n%s", v, err, b)
	}
	return out
}

func waitForVersion(t *testing.T, addr, want string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if v, ok := daemonVersionAt(addr); ok && v == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	got, ok := daemonVersionAt(addr)
	t.Fatalf("no daemon reporting %s on %s within 20s (got %q, answering %v)", want, addr, got, ok)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}
